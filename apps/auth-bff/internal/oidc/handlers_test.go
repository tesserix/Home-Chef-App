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
	resp     *apiclient.UpsertUserResponse
	err      error
	captured bool
	lastReq  apiclient.UpsertUserRequest
}

func (f *fakeAPI) UpsertUser(ctx context.Context, req apiclient.UpsertUserRequest) (*apiclient.UpsertUserResponse, error) {
	f.captured = true
	f.lastReq = req
	return f.resp, f.err
}

type fakeSessions struct {
	encoded string
}

func (f *fakeSessions) Encode(*session.Payload) (string, error) { return f.encoded, nil }
func (f *fakeSessions) SetCookie(w http.ResponseWriter, name, v string) {
	if name == "" {
		name = "hc_session"
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: v, Path: "/"})
}
func (f *fakeSessions) MaxAge() time.Duration { return time.Hour }

type recordingStateManager struct {
	state string
	entry StateEntry
	err   error
}

func (s *recordingStateManager) Begin(_ http.ResponseWriter, entry StateEntry) (string, error) {
	s.entry = entry
	if s.err != nil {
		return "", s.err
	}
	s.state = "recorded-state"
	return s.state, nil
}

func (s *recordingStateManager) Take(http.ResponseWriter, *http.Request, string) (StateEntry, bool) {
	return StateEntry{}, false
}

func loadReg(t *testing.T) *productregistry.Registry {
	t.Helper()
	r, err := productregistry.Load("../../homechef-products.yaml")
	require.NoError(t, err)
	return r
}

func newHandlers(t *testing.T, ver *fakeVerifier, api *fakeAPI) *Handlers {
	stateManager, err := NewBrowserStateManager(stateTestKey(t), false)
	require.NoError(t, err)
	return &Handlers{
		Registry: loadReg(t),
		OAuthByApp: map[string]*oauth2.Config{
			"admin-portal": {
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				Endpoint:     oauth2.Endpoint{AuthURL: "https://example.com/oauth/authorize", TokenURL: "https://example.com/oauth/token"},
				RedirectURL:  "http://localhost:5173/auth/callback",
				Scopes:       []string{"openid", "email", "profile"},
			},
		},
		GIPVerifier:  ver,
		API:          api,
		Sessions:     &fakeSessions{encoded: "sess-blob"},
		StateManager: stateManager,
	}
}

func TestLogin_Redirects(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "https://example.com/oauth/authorize")
	assert.Contains(t, loc, "state=")
	assert.Contains(t, loc, "tenantId=HomeChef-Internal-gyofe")
	assert.Contains(t, loc, "nonce=")
	assert.Contains(t, w.Header().Get("Set-Cookie"), "hc_oidc_")
	assert.Contains(t, w.Header().Get("Set-Cookie"), "HttpOnly")
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

func TestLogin_DoesNotPersistExternalReturnTo(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	states := &recordingStateManager{}
	h.StateManager = states
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/login?return_to=https%3A%2F%2Fevil.example%2Fsteal", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	assert.NotEmpty(t, states.state)
	assert.Empty(t, states.entry.ReturnTo, "external redirect targets must never enter OAuth state")
}

func TestLogin_StateManagerFailure_500(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	h.StateManager = &recordingStateManager{err: errors.New("random source unavailable")}
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "state_generation_failed")
}

func TestSafeReturnTo_RejectsRedirectAmbiguities(t *testing.T) {
	assert.Equal(t, "/orders/123?tab=details", safeReturnTo("/orders/123?tab=details"))
	for _, unsafe := range []string{
		"https://evil.example/steal",
		"//evil.example/steal",
		`/\evil.example/steal`,
		"/%5c%5cevil.example/steal",
		"/orders\r\nLocation: https://evil.example",
	} {
		assert.Empty(t, safeReturnTo(unsafe), "unsafe return_to %q must be discarded", unsafe)
	}
}

func TestCallbackInvariants_RequireMatchingAppNonceAndTenant(t *testing.T) {
	entry := StateEntry{AppName: "admin-portal", Nonce: "expected-nonce"}
	assert.True(t, stateMatchesApp(entry, "admin-portal"))
	assert.False(t, stateMatchesApp(entry, "web"))
	assert.True(t, nonceMatches(entry, "expected-nonce"))
	assert.False(t, nonceMatches(entry, ""))
	assert.False(t, nonceMatches(entry, "attacker-nonce"))

	claims := map[string]any{
		"firebase": map[string]any{"tenant": "HomeChef-Internal-gyofe"},
	}
	assert.True(t, tenantMatchesApp(claims, "HomeChef-Internal-gyofe"))
	assert.False(t, tenantMatchesApp(claims, "HomeChef-Customer-gyofe"))
	assert.False(t, tenantMatchesApp(map[string]any{}, "HomeChef-Internal-gyofe"))
}

func TestExchange_Happy(t *testing.T) {
	// Internal-tenant admin login: the allowlist must include the email now that
	// the gate fails closed on an unconfigured allowlist.
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "x@y.com")
	ver := &fakeVerifier{
		tok: &gip.VerifiedToken{
			UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Internal-gyofe", Provider: "password",
			Name: "Ada Admin", Picture: "https://example.com/avatar.png", EmailVerified: true,
			Claims: map[string]any{
				"sub":      "g1",
				"email":    "x@y.com",
				"firebase": map[string]any{"sign_in_provider": "password", "tenant": "HomeChef-Internal-gyofe"},
			},
		},
	}
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, ver, api)
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"valid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), `"user_id":"u1"`)
	// Session cookie should be set under the admin app's own cookie name
	// (homechef-products.yaml: admin-portal → hc_admin_session), not the
	// shared hc_session name — that's exactly the isolation this handler
	// exists to preserve (see productregistry.App.SessionCookie).
	// Joined, not Get(): the exchange also sets the browser device cookie, and
	// Get returns whichever Set-Cookie happens to come first.
	assert.Contains(t, strings.Join(w.Header().Values("Set-Cookie"), " "), "hc_admin_session=sess-blob")
	assert.Equal(t, "Ada Admin", api.lastReq.Name)
	assert.Equal(t, "https://example.com/avatar.png", api.lastReq.Avatar)
	assert.True(t, api.lastReq.EmailVerified)
}

func TestExchange_AdminEmailNotInAllowlist_403(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "allowed@fe3dr.com")
	ver := &fakeVerifier{
		tok: &gip.VerifiedToken{
			UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Internal-gyofe", Provider: "password",
			EmailVerified: true,
			Claims:        map[string]any{"sub": "g1", "email": "x@y.com"},
		},
	}
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, ver, api)
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"valid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "email_not_allowed")
}

func TestExchange_AdminUnverifiedEmail_Denied(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "admin@fe3dr.com")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, &fakeVerifier{tok: &gip.VerifiedToken{
		UID: "g1", Email: "admin@fe3dr.com", TenantID: "HomeChef-Internal-gyofe",
		Provider: "password", EmailVerified: false,
		Claims: map[string]any{"sub": "g1", "email": "admin@fe3dr.com", "email_verified": false},
	}}, api)
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"valid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "email_not_verified")
	assert.False(t, api.captured, "an unverified admin identity must never be upserted")
}

func TestExchange_AdminAllowlistUnset_Denied(t *testing.T) {
	// Fail-closed: an unconfigured allowlist must DENY admin login. The k8s
	// secret is mounted optional and the mesh does not strip X-User-* headers, so
	// a missing allowlist would otherwise hand admin to any verified email.
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "")
	ver := &fakeVerifier{
		tok: &gip.VerifiedToken{
			UID: "g1", Email: "admin@fe3dr.com", TenantID: "HomeChef-Internal-gyofe", Provider: "password",
			Claims: map[string]any{"sub": "g1", "email": "admin@fe3dr.com"},
		},
	}
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, ver, api)
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"valid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "email_not_allowed")
	assert.False(t, api.captured, "a denied admin must never be upserted")
}

func TestExchange_AdminEmailInAllowlist_OK(t *testing.T) {
	// Case-insensitive + space-trimmed match.
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", " X@Y.com , other@fe3dr.com ")
	ver := &fakeVerifier{
		tok: &gip.VerifiedToken{
			UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Internal-gyofe", Provider: "password",
			EmailVerified: true,
			Claims:        map[string]any{"sub": "g1", "email": "x@y.com"},
		},
	}
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, ver, api)
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"valid"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"user_id":"u1"`)
}

func TestExchange_InvalidBody_400(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{}, &fakeAPI{})
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExchange_InvalidToken_401(t *testing.T) {
	h := newHandlers(t, &fakeVerifier{err: errors.New("bad")}, &fakeAPI{})
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"bad"}`))
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
