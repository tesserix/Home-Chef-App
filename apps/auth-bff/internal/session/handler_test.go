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
	req.AddCookie(&http.Cookie{Name: "hc_csrf", Value: "refresh-token"})
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("X-CSRF-Token", "refresh-token")
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

// --- CookieForHost (per-app session cookie isolation) ---
//
// Handler.CookieForHost lets a caller (in production, main.go wires in
// productregistry.Registry.SessionCookieForHost) resolve the session cookie
// name per request Host, so each portal on the shared .fe3dr.com cookie
// domain gets its own cookie instead of colliding on the single hc_session
// name. This package deliberately takes a plain func(string) string rather
// than importing productregistry, so these tests use a small stand-in map
// instead of loading the real registry — the resolver's own correctness is
// covered by productregistry's tests.
func vendorAdminResolver(host string) string {
	switch host {
	case "vendors.fe3dr.com":
		return "hc_vendor_session"
	case "admin.fe3dr.com":
		return "hc_admin_session"
	default:
		return "" // unmatched → caller falls back to the default cookie name
	}
}

func TestHandler_Session_HostSpecificCookie_Accepted(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	h := &Handler{Mgr: mgr, CookieForHost: vendorAdminResolver}
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "vendor-user", Email: "chef@example.com", Role: "chef", Pool: "business",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.Host = "vendors.fe3dr.com"
	req.AddCookie(&http.Cookie{Name: "hc_vendor_session", Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "vendor-user", body["user_id"])
}

// This is the bug this change closes: a request to vendors.fe3dr.com that
// only carries the default hc_session cookie (e.g. a stale customer session
// on the shared .fe3dr.com domain) must NOT be accepted as a vendor session
// — the handler looks for hc_vendor_session on that host and hc_session
// simply isn't it.
func TestHandler_Session_DefaultCookieOnVendorHost_NotAccepted(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	h := &Handler{Mgr: mgr, CookieForHost: vendorAdminResolver}
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "customer-user", Email: "a@b.com", Role: "customer", Pool: "customer",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.Host = "vendors.fe3dr.com"
	req.AddCookie(&http.Cookie{Name: "hc_session", Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// A Host the resolver doesn't recognize falls back to the Manager's default
// cookie name rather than erroring — e.g. fe3dr.com itself, or any host not
// covered by the stand-in resolver.
func TestHandler_Session_UnmatchedHost_FallsBackToDefaultCookie(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	h := &Handler{Mgr: mgr, CookieForHost: vendorAdminResolver}
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "web-user", Email: "a@b.com", Role: "customer", Pool: "customer",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.Host = "fe3dr.com"
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// Mobile's Bearer path must be unaffected by CookieForHost being wired at
// all — there is no cookie and no Host-based app, and the handler falls
// through to the Authorization header exactly as before.
func TestHandler_Session_Bearer_UnaffectedByCookieForHost(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	h := &Handler{Mgr: mgr, CookieForHost: vendorAdminResolver}
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "u-mobile", Email: "m@b.com", Role: "driver", Pool: "business",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, _ := mgr.Encode(p)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	req.Host = "some-mobile-gateway.invalid" // unmatched by the resolver, no cookie at all
	req.Header.Set("Authorization", "Bearer "+enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"user_id":"u-mobile"`)
}

func TestHandler_Logout_ClearsHostSpecificCookie(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	h := &Handler{Mgr: mgr, CookieForHost: vendorAdminResolver}
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.Host = "admin.fe3dr.com"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	setCookie := w.Header().Get("Set-Cookie")
	assert.Contains(t, setCookie, "hc_admin_session=")
	assert.NotContains(t, setCookie, "hc_session=")
}

func TestHandler_Refresh_SetsHostSpecificCookie(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	h := &Handler{Mgr: mgr, CookieForHost: vendorAdminResolver}
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "vendor-user", Email: "chef@example.com", Role: "chef", Pool: "business",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(5 * time.Minute).Unix()}
	enc, _ := mgr.Encode(p)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.Host = "vendors.fe3dr.com"
	req.AddCookie(&http.Cookie{Name: "hc_vendor_session", Value: enc})
	req.AddCookie(&http.Cookie{Name: "hc_csrf", Value: "refresh-token"})
	req.Header.Set("Origin", "https://vendors.fe3dr.com")
	req.Header.Set("X-CSRF-Token", "refresh-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Header().Get("Set-Cookie"), "hc_vendor_session=")
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
	assert.Equal(t, tok, body["csrfToken"], "browser clients consume the camelCase field")

	cookie := w.Header().Get("Set-Cookie")
	assert.Contains(t, cookie, "hc_csrf="+tok)
	assert.Contains(t, cookie, "SameSite=Strict")
}

func TestHandler_Refresh_CookieSessionRequiresCSRFToken(t *testing.T) {
	h, mgr := newTestHandler(t)
	r := gin.New()
	h.Register(r)

	p := &Payload{UID: "u1", Email: "a@b.com", Role: "customer", Pool: "customer",
		IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()}
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	req.AddCookie(&http.Cookie{Name: "hc_csrf", Value: "cookie-token"})
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("X-CSRF-Token", "wrong-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.JSONEq(t, `{"error":"csrf_rejected"}`, w.Body.String())
}

func TestHandler_Logout_CookieSessionRejectsForeignOrigin(t *testing.T) {
	h, mgr := newTestHandler(t)
	r := gin.New()
	h.Register(r)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: "ambient-session"})
	req.AddCookie(&http.Cookie{Name: "hc_csrf", Value: "csrf-token"})
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("X-CSRF-Token", "csrf-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.JSONEq(t, `{"error":"origin_rejected"}`, w.Body.String())
}

// #671: the CSRF cookie's Secure flag must follow the manager's env-driven config — HTTPS-only in
// prod, off in dev — mirroring the session cookie. (HttpOnly stays off: a double-submit token the
// client JS must read.)
func TestHandler_CSRF_CookieSecureFollowsConfig(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := NewManager(Config{EncryptKey: k, MaxAge: time.Hour, Secure: true})
	require.NoError(t, err)
	prod := &Handler{Mgr: mgr}
	r := gin.New()
	prod.Register(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/csrf", nil))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Set-Cookie"), "Secure", "prod (Secure config) → CSRF cookie is HTTPS-only")

	// dev default (Secure:false) → no Secure attribute so it still works over http.
	dev, _ := newTestHandler(t)
	rDev := gin.New()
	dev.Register(rDev)
	wDev := httptest.NewRecorder()
	rDev.ServeHTTP(wDev, httptest.NewRequest(http.MethodGet, "/auth/csrf", nil))
	assert.NotContains(t, wDev.Header().Get("Set-Cookie"), "Secure", "dev → not Secure")
}
