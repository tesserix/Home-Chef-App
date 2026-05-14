package session

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(t *testing.T) (*Handler, *Manager) {
	t.Helper()
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	return &Handler{Mgr: mgr}, mgr
}

func TestHandler_Session_Cookie_Authenticated(t *testing.T) {
	h, mgr := newTestHandler(t)
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "u1", Email: "a@b.com", Role: "customer", Pool: "customer",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "u1", body["user_id"])
	assert.Equal(t, "a@b.com", body["email"])
	assert.Equal(t, "customer", body["role"])
	assert.Equal(t, "customer", body["pool"])
}

func TestHandler_Session_Bearer_Authenticated(t *testing.T) {
	h, mgr := newTestHandler(t)
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "u-mobile", Email: "m@b.com", Role: "driver", Pool: "business",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, _ := mgr.Encode(p)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.Header.Set("Authorization", "Bearer "+enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":"u-mobile"`)
	assert.Contains(t, w.Body.String(), `"role":"driver"`)
}

func TestHandler_Session_NoCredentials_401(t *testing.T) {
	h, _ := newTestHandler(t)
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Session_BadCookie_401(t *testing.T) {
	h, mgr := newTestHandler(t)
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: "garbage"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Logout_ClearsCookie(t *testing.T) {
	h, mgr := newTestHandler(t)
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	setCookie := w.Header().Get("Set-Cookie")
	assert.Contains(t, setCookie, mgr.CookieName()+"=")
	assert.True(t, strings.Contains(setCookie, "Max-Age=0") || strings.Contains(setCookie, "Expires"))
}

func TestHandler_Refresh_ExtendsExp(t *testing.T) {
	h, mgr := newTestHandler(t)
	r := gin.New()
	h.Register(r)

	originalExp := time.Now().Add(5 * time.Minute).Unix()
	p := &Payload{UID: "u1", Email: "a@b.com", Role: "customer", Pool: "customer",
		IssuedAt: time.Now().Unix(), ExpiresAt: originalExp}
	enc, _ := mgr.Encode(p)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	newExp, _ := body["expires_at"].(float64)
	assert.Greater(t, int64(newExp), originalExp)
}

func TestHandler_Refresh_NoSession_401(t *testing.T) {
	h, _ := newTestHandler(t)
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_CSRF_ReturnsTokenAndCookie(t *testing.T) {
	h, _ := newTestHandler(t)
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest(http.MethodGet, "/auth/csrf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	tok, _ := body["csrf_token"].(string)
	require.NotEmpty(t, tok)
	assert.Len(t, tok, 64) // 32 bytes hex

	cookie := w.Header().Get("Set-Cookie")
	assert.Contains(t, cookie, "hc_csrf="+tok)
	assert.Contains(t, cookie, "SameSite=Strict")
}
