package oidc

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserStateManager_RoundTripAcrossReplicas(t *testing.T) {
	key := stateTestKey(t)
	issuer, err := NewBrowserStateManager(key, true)
	require.NoError(t, err)
	consumer, err := NewBrowserStateManager(key, true)
	require.NoError(t, err)

	login := httptest.NewRecorder()
	state, err := issuer.Begin(login, StateEntry{
		AppName:  "admin-portal",
		Nonce:    "nonce-1",
		ReturnTo: "/orders/123",
	})
	require.NoError(t, err)
	require.NotEmpty(t, state)

	callback := httptest.NewRequest(http.MethodGet, "https://admin.fe3dr.com/auth/callback?state="+state, nil)
	callback.AddCookie(requireStateCookie(t, login.Result()))
	consumed := httptest.NewRecorder()
	entry, ok := consumer.Take(consumed, callback, state)

	require.True(t, ok)
	assert.Equal(t, "admin-portal", entry.AppName)
	assert.Equal(t, "nonce-1", entry.Nonce)
	assert.Equal(t, "/orders/123", entry.ReturnTo)
	assert.Contains(t, consumed.Header().Get("Set-Cookie"), "Max-Age=0")
}

func TestBrowserStateManager_RejectsWrongBrowserAndTampering(t *testing.T) {
	manager, err := NewBrowserStateManager(stateTestKey(t), true)
	require.NoError(t, err)
	login := httptest.NewRecorder()
	state, err := manager.Begin(login, StateEntry{AppName: "web", Nonce: "nonce-1"})
	require.NoError(t, err)

	wrongBrowser := httptest.NewRequest(http.MethodGet, "https://fe3dr.com/auth/callback", nil)
	_, ok := manager.Take(httptest.NewRecorder(), wrongBrowser, state)
	assert.False(t, ok)

	cookie := requireStateCookie(t, login.Result())
	tampered := state[:len(state)-1] + differentStateCharacter(state[len(state)-1])
	tamperedRequest := httptest.NewRequest(http.MethodGet, "https://fe3dr.com/auth/callback", nil)
	tamperedRequest.AddCookie(cookie)
	_, ok = manager.Take(httptest.NewRecorder(), tamperedRequest, tampered)
	assert.False(t, ok)
}

func TestBrowserStateManager_RejectsExpiredAndConsumedState(t *testing.T) {
	manager, err := NewBrowserStateManager(stateTestKey(t), true)
	require.NoError(t, err)
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return now }

	login := httptest.NewRecorder()
	state, err := manager.Begin(login, StateEntry{AppName: "web", Nonce: "nonce-1"})
	require.NoError(t, err)
	cookie := requireStateCookie(t, login.Result())

	manager.now = func() time.Time { return now.Add(5*time.Minute + time.Second) }
	expired := httptest.NewRequest(http.MethodGet, "https://fe3dr.com/auth/callback", nil)
	expired.AddCookie(cookie)
	_, ok := manager.Take(httptest.NewRecorder(), expired, state)
	assert.False(t, ok)

	manager.now = func() time.Time { return now }
	first := httptest.NewRequest(http.MethodGet, "https://fe3dr.com/auth/callback", nil)
	first.AddCookie(cookie)
	consumed := httptest.NewRecorder()
	_, ok = manager.Take(consumed, first, state)
	require.True(t, ok)

	replay := httptest.NewRequest(http.MethodGet, "https://fe3dr.com/auth/callback", nil)
	for _, cleared := range consumed.Result().Cookies() {
		replay.AddCookie(cleared)
	}
	_, ok = manager.Take(httptest.NewRecorder(), replay, state)
	assert.False(t, ok)
}

func TestBrowserStateManager_PreservesParallelLoginBindings(t *testing.T) {
	manager, err := NewBrowserStateManager(stateTestKey(t), true)
	require.NoError(t, err)
	first := httptest.NewRecorder()
	firstState, err := manager.Begin(first, StateEntry{AppName: "web", Nonce: "nonce-1"})
	require.NoError(t, err)
	second := httptest.NewRecorder()
	secondState, err := manager.Begin(second, StateEntry{AppName: "web", Nonce: "nonce-2"})
	require.NoError(t, err)

	assert.NotEqual(t, requireStateCookie(t, first.Result()).Name, requireStateCookie(t, second.Result()).Name)
	request := httptest.NewRequest(http.MethodGet, "https://fe3dr.com/auth/callback", nil)
	request.AddCookie(requireStateCookie(t, first.Result()))
	request.AddCookie(requireStateCookie(t, second.Result()))
	entry, ok := manager.Take(httptest.NewRecorder(), request, firstState)
	require.True(t, ok)
	assert.Equal(t, "nonce-1", entry.Nonce)

	request = httptest.NewRequest(http.MethodGet, "https://fe3dr.com/auth/callback", nil)
	request.AddCookie(requireStateCookie(t, first.Result()))
	request.AddCookie(requireStateCookie(t, second.Result()))
	entry, ok = manager.Take(httptest.NewRecorder(), request, secondState)
	require.True(t, ok)
	assert.Equal(t, "nonce-2", entry.Nonce)
}

func TestBrowserStateManager_UsesHardenedCookieAttributes(t *testing.T) {
	manager, err := NewBrowserStateManager(stateTestKey(t), true)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	_, err = manager.Begin(w, StateEntry{AppName: "web", Nonce: "nonce-1"})
	require.NoError(t, err)

	cookie := requireStateCookie(t, w.Result())
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, "/auth/callback", cookie.Path)
	assert.Empty(t, cookie.Domain)
	assert.True(t, strings.HasPrefix(cookie.Name, "__Secure-hc_oidc_"))
}

func stateTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return key
}

func requireStateCookie(t *testing.T, response *http.Response) *http.Cookie {
	t.Helper()
	cookies := response.Cookies()
	require.Len(t, cookies, 1)
	return cookies[0]
}

func differentStateCharacter(current byte) string {
	if current == 'A' {
		return "B"
	}
	return "A"
}
