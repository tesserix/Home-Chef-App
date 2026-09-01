package oidc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

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

func newHandlers(t *testing.T, api *fakeAPI) *Handlers {
	stateManager, err := NewBrowserStateManager(stateTestKey(t), false)
	require.NoError(t, err)
	return &Handlers{
		Registry: loadReg(t),
		OAuthByApp: map[string]*oauth2.Config{
			"admin-portal": {
				ClientID:    "test-client",
				Endpoint:    oauth2.Endpoint{AuthURL: "https://example.com/oauth/v2/authorize", TokenURL: "https://example.com/oauth/v2/token"},
				RedirectURL: "http://localhost:5176/auth/callback",
				Scopes:      []string{"openid", "email", "profile"},
			},
		},
		API:          api,
		Sessions:     &fakeSessions{encoded: "sess-blob"},
		StateManager: stateManager,
	}
}

func TestLogin_Redirects_WithPKCE(t *testing.T) {
	h := newHandlers(t, &fakeAPI{})
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "https://example.com/oauth/v2/authorize")
	assert.Contains(t, loc, "state=")
	assert.Contains(t, loc, "nonce=")
	assert.Contains(t, loc, "code_challenge=")
	assert.Contains(t, loc, "code_challenge_method=S256")
	// Zitadel is a single multi-app instance — the GIP-era tenantId must be gone.
	assert.NotContains(t, loc, "tenantId")
	assert.Contains(t, w.Header().Get("Set-Cookie"), "hc_oidc_")
	assert.Contains(t, w.Header().Get("Set-Cookie"), "HttpOnly")
}

func TestLogin_RegisterScreen_AddsPromptCreate(t *testing.T) {
	h := newHandlers(t, &fakeAPI{})
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/login?screen=register", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "prompt=create")
}

func TestLogin_StoresPKCEVerifierAndConsentInState(t *testing.T) {
	h := newHandlers(t, &fakeAPI{})
	states := &recordingStateManager{}
	h.StateManager = states
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/login?marketing_consent=true", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusFound, w.Code)
	assert.NotEmpty(t, states.entry.CodeVerifier, "PKCE verifier must be sealed into state")
	assert.True(t, states.entry.MarketingConsent)
}

func TestLogin_UnknownHost_400(t *testing.T) {
	h := newHandlers(t, &fakeAPI{})
	r := gin.New()
	h.Register(r)
	req := httptest.NewRequest("GET", "http://attacker.example.com/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_DoesNotPersistExternalReturnTo(t *testing.T) {
	h := newHandlers(t, &fakeAPI{})
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
	h := newHandlers(t, &fakeAPI{})
	h.StateManager = &recordingStateManager{err: errors.New("random source unavailable")}
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "state_failed")
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

func TestCallbackInvariants_RequireMatchingAppAndNonce(t *testing.T) {
	entry := StateEntry{AppName: "admin-portal", Nonce: "expected-nonce"}
	assert.True(t, stateMatchesApp(entry, "admin-portal"))
	assert.False(t, stateMatchesApp(entry, "web"))
	assert.True(t, nonceMatches(entry, "expected-nonce"))
	assert.False(t, nonceMatches(entry, ""))
	assert.False(t, nonceMatches(entry, "attacker-nonce"))
}

// --- issueSession (white-box: the callback path after token verification) ---

func runIssueSession(t *testing.T, h *Handlers, host string, claims map[string]any, entry StateEntry) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "http://"+host+"/auth/callback", nil)
	app, err := h.Registry.ResolveByHost(host)
	require.NoError(t, err)
	h.issueSession(c, app, claims, entry)
	return w
}

// allCookies joins every Set-Cookie header; Get() only returns the first.
func allCookies(w *httptest.ResponseRecorder) string {
	return strings.Join(w.Header().Values("Set-Cookie"), "\n")
}

func adminClaims(verified bool) map[string]any {
	return map[string]any{
		"sub": "z1", "email": "x@y.com", "email_verified": verified,
		"name": "Ada Admin", "picture": "https://example.com/avatar.png",
	}
}

func TestIssueSession_Admin_Happy(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "x@y.com")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	w := runIssueSession(t, h, "admin.fe3dr.com", adminClaims(true), StateEntry{})

	require.Equal(t, http.StatusFound, w.Code, w.Body.String())
	// Session cookie must be the admin app's own name, not the shared hc_session.
	assert.Contains(t, allCookies(w), "hc_admin_session=sess-blob")
	assert.Equal(t, ProviderName, api.lastReq.Provider)
	assert.Equal(t, "z1", api.lastReq.Subject)
	assert.Equal(t, "internal", api.lastReq.AuthPool)
	assert.Equal(t, "Ada Admin", api.lastReq.Name)
	assert.Equal(t, "https://example.com/avatar.png", api.lastReq.Avatar)
	assert.True(t, api.lastReq.EmailVerified)
}

func TestIssueSession_AdminEmailNotInAllowlist_403(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "allowed@fe3dr.com")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	w := runIssueSession(t, h, "admin.fe3dr.com", adminClaims(true), StateEntry{})

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "email_not_allowed")
	assert.False(t, api.captured, "a denied admin must never be upserted")
}

func TestIssueSession_AdminUnverifiedEmail_Denied(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "x@y.com")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	w := runIssueSession(t, h, "admin.fe3dr.com", adminClaims(false), StateEntry{})

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "email_not_verified")
	assert.False(t, api.captured, "an unverified admin identity must never be upserted")
}

func TestIssueSession_AdminAllowlistUnset_Denied(t *testing.T) {
	// Fail-closed: an unconfigured allowlist must DENY admin login. The k8s
	// secret is mounted optional and the mesh does not strip X-User-* headers,
	// so a missing allowlist would otherwise hand admin to any verified email.
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	w := runIssueSession(t, h, "admin.fe3dr.com", adminClaims(true), StateEntry{})

	require.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "email_not_allowed")
	assert.False(t, api.captured)
}

func TestIssueSession_AdminAllowlist_CaseInsensitive(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", " X@Y.com , other@fe3dr.com ")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	w := runIssueSession(t, h, "admin.fe3dr.com", adminClaims(true), StateEntry{})

	require.Equal(t, http.StatusFound, w.Code, w.Body.String())
}

func TestIssueSession_Customer_NoAllowlistNeeded(t *testing.T) {
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "")
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	claims := map[string]any{"sub": "z2", "email": "cust@example.com", "email_verified": false}
	w := runIssueSession(t, h, "fe3dr.com", claims, StateEntry{MarketingConsent: true})

	require.Equal(t, http.StatusFound, w.Code, w.Body.String())
	assert.Contains(t, allCookies(w), "hc_session=sess-blob")
	assert.Equal(t, "customer", api.lastReq.AuthPool)
	assert.True(t, api.lastReq.MarketingConsent, "DPDP consent captured at login must reach the API")
}

func TestIssueSession_ReturnToOverridesPostLoginURL(t *testing.T) {
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	claims := map[string]any{"sub": "z2", "email": "cust@example.com"}
	w := runIssueSession(t, h, "fe3dr.com", claims, StateEntry{ReturnTo: "/orders/42"})

	require.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/orders/42", w.Header().Get("Location"))
}
