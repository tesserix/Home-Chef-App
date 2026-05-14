package oidc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/gip"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type fakeVerifier struct {
	tok *gip.VerifiedToken
	err error
}

func (f *fakeVerifier) Verify(ctx context.Context, raw, tenant string) (*gip.VerifiedToken, error) {
	return f.tok, f.err
}

type fakeAPI struct {
	resp *apiclient.UpsertUserResponse
	err  error
}

func (f *fakeAPI) UpsertUser(ctx context.Context, req apiclient.UpsertUserRequest) (*apiclient.UpsertUserResponse, error) {
	return f.resp, f.err
}

type fakeSessions struct {
	encoded string
}

func (f *fakeSessions) Encode(*session.Payload) (string, error) { return f.encoded, nil }
func (f *fakeSessions) SetCookie(w http.ResponseWriter, v string) {
	http.SetCookie(w, &http.Cookie{Name: "hc_session", Value: v, Path: "/"})
}
func (f *fakeSessions) MaxAge() time.Duration { return time.Hour }

func loadReg(t *testing.T) *productregistry.Registry {
	t.Helper()
	r, err := productregistry.Load("../../homechef-products.yaml")
	require.NoError(t, err)
	return r
}

func newHandlers(t *testing.T, ver *fakeVerifier, api *fakeAPI) *Handlers {
	return &Handlers{
		Registry: loadReg(t),
		OAuthByApp: map[string]*oauth2.Config{
			"web": {
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				Endpoint:     oauth2.Endpoint{AuthURL: "https://example.com/oauth/authorize", TokenURL: "https://example.com/oauth/token"},
				RedirectURL:  "http://localhost:5173/auth/callback",
				Scopes:       []string{"openid", "email", "profile"},
			},
		},
		GIPVerifier: ver,
		API:         api,
		Sessions:    &fakeSessions{encoded: "sess-blob"},
		StateStore:  NewMemStateStore(),
	}
}

func TestLogin_Redirects(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://fe3dr.com/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "https://example.com/oauth/authorize")
	assert.Contains(t, loc, "state=")
	assert.Contains(t, loc, "tenantId=HomeChef-Customer-xxxxx")
	assert.Contains(t, loc, "nonce=")
}

func TestLogin_UnknownHost_400(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest("GET", "http://attacker.example.com/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExchange_Happy(t *testing.T) {
	ver := &fakeVerifier{
		tok: &gip.VerifiedToken{
			UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Customer-xxxxx", Provider: "password",
			Claims: map[string]any{
				"sub":      "g1",
				"email":    "x@y.com",
				"firebase": map[string]any{"sign_in_provider": "password", "tenant": "HomeChef-Customer-xxxxx"},
			},
		},
	}
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, ver, api)
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("POST", "http://fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"valid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), `"user_id":"u1"`)
	// Session cookie should be set.
	assert.Contains(t, w.Header().Get("Set-Cookie"), "hc_session=sess-blob")
}

func TestExchange_InvalidBody_400(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest("POST", "http://fe3dr.com/auth/exchange", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExchange_InvalidToken_401(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{err: errors.New("bad")}, &fakeAPI{})
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest("POST", "http://fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestExchange_UnknownHost_400(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest("POST", "http://attacker.example.com/auth/exchange", strings.NewReader(`{"id_token":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStateStore_PutTakeRoundTrip(t *testing.T) {
	s := NewMemStateStore()
	s.Put("k1", StateEntry{AppName: "web", Nonce: "n1"})
	e, ok := s.Take("k1")
	require.True(t, ok)
	assert.Equal(t, "web", e.AppName)
	// Take is one-shot.
	_, ok = s.Take("k1")
	assert.False(t, ok)
}
