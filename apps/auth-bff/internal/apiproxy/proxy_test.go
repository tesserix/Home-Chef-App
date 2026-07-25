package apiproxy

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/homechef/auth-bff/internal/productregistry"
	"github.com/homechef/auth-bff/internal/session"
)

func newTestPayload() *session.Payload {
	now := time.Now()
	return &session.Payload{
		UID: "u1", Email: "chef@example.com", Role: "chef", Pool: "customer",
		IssuedAt: now.Unix(), ExpiresAt: now.Add(time.Hour).Unix(),
	}
}

func TestHandler_BearerToken_Proxies(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := session.NewManager(session.Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: []byte("test-signing-key-32-bytes-pad!!!")})

	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	var capturedAuth, capturedCookie string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedCookie = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/v1/*proxyPath", Handler(&Deps{APIBaseURL: upstream.URL, Sessions: mgr, Signer: signer}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.Header.Set("Authorization", "Bearer "+enc)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: "some-other-cookie-value"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, capturedAuth, "Authorization must not be forwarded upstream")
	assert.Empty(t, capturedCookie, "Cookie must not be forwarded upstream")
}

func TestHandler_SessionCookie_Proxies(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := session.NewManager(session.Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: []byte("test-signing-key-32-bytes-pad!!!")})

	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	var capturedAuth, capturedCookie, capturedUserID string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedCookie = r.Header.Get("Cookie")
		capturedUserID = r.Header.Get(headerproxy.HdrUserID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/v1/*proxyPath", Handler(&Deps{APIBaseURL: upstream.URL, Sessions: mgr, Signer: signer}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"ok":true}`, w.Body.String())
	assert.Equal(t, "u1", capturedUserID, "identity from the decoded cookie session must reach upstream")
	assert.Empty(t, capturedAuth, "Authorization must not be forwarded upstream")
	assert.Empty(t, capturedCookie, "Cookie must not be forwarded upstream")
}

func TestHandler_NoTokenAtAll_Returns401(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := session.NewManager(session.Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: []byte("test-signing-key-32-bytes-pad!!!")})

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream must not be called when there is no session token")
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/v1/*proxyPath", Handler(&Deps{APIBaseURL: upstream.URL, Sessions: mgr, Signer: signer}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_UndecodableToken_ReturnsInvalidSession(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := session.NewManager(session.Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: []byte("test-signing-key-32-bytes-pad!!!")})

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream must not be called when the session token fails to decode")
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/v1/*proxyPath", Handler(&Deps{APIBaseURL: upstream.URL, Sessions: mgr, Signer: signer}))

	// Case 1: garbage Bearer token.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.JSONEq(t, `{"error":"invalid_session"}`, w.Body.String())

	// Case 2: garbage session cookie.
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req2.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: "not-a-real-token"})
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusUnauthorized, w2.Code)
	assert.JSONEq(t, `{"error":"invalid_session"}`, w2.Body.String())
}

func TestHandler_BearerHeaderTakesPrecedenceOverCookie(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := session.NewManager(session.Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: []byte("test-signing-key-32-bytes-pad!!!")})

	bearerPayload := newTestPayload()
	bearerPayload.UID = "bearer-user"
	bearerEnc, err := mgr.Encode(bearerPayload)
	require.NoError(t, err)

	var capturedUserID string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.Header.Get(headerproxy.HdrUserID)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/v1/*proxyPath", Handler(&Deps{APIBaseURL: upstream.URL, Sessions: mgr, Signer: signer}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.Header.Set("Authorization", "Bearer "+bearerEnc)
	// An invalid cookie is present too; it must be ignored since the header wins.
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: "garbage"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "bearer-user", capturedUserID)
}

// newOriginTestFixtures spins up the manager/signer/router trio shared by
// the CSRF-origin tests below, plus an upstream that fails the test if it's
// ever hit — every rejection case in this file must never reach it.
func newOriginTestFixtures(t *testing.T, upstreamCalled *bool) (*session.Manager, *gin.Engine, string) {
	t.Helper()
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := session.NewManager(session.Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: []byte("test-signing-key-32-bytes-pad!!!")})

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*upstreamCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/api/v1/*proxyPath", Handler(&Deps{APIBaseURL: upstream.URL, Sessions: mgr, Signer: signer}))
	return mgr, r, upstream.URL
}

func TestHandler_CookieAuth_MatchingOrigin_Proxies(t *testing.T) {
	var called bool
	mgr, r, _ := newOriginTestFixtures(t, &called)
	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	// httptest.NewRequest defaults req.Host to "example.com" when the
	// target is a bare path; the matching Origin for a same-origin browser
	// call is therefore https://example.com (no X-Forwarded-Proto set, so
	// requestOrigin falls back to its https default).
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "upstream must be called for a matching same-origin request")
}

func TestHandler_CookieAuth_ForeignOrigin_Rejected(t *testing.T) {
	var called bool
	mgr, r, _ := newOriginTestFixtures(t, &called)
	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.JSONEq(t, `{"error":"origin_rejected"}`, w.Body.String())
	assert.False(t, called, "upstream must never be called for a foreign-Origin request")
}

func TestHandler_CookieAuth_NoOriginGET_Allowed(t *testing.T) {
	var called bool
	mgr, r, _ := newOriginTestFixtures(t, &called)
	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	// No Origin header at all: a top-level cross-site GET navigation looks
	// exactly like this. GET is safe, so it must be allowed through.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "upstream must be called for a same-origin-shaped GET with no Origin")
}

func TestHandler_CookieAuth_NoOriginPOST_Rejected(t *testing.T) {
	var called bool
	mgr, r, _ := newOriginTestFixtures(t, &called)
	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	// No Origin header on an unsafe method: real browsers always attach
	// Origin to POST/PUT/PATCH/DELETE, including same-origin ones, so this
	// shape cannot be a legitimate browser call and must be rejected.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chef/onboarding/status", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.JSONEq(t, `{"error":"origin_rejected"}`, w.Body.String())
	assert.False(t, called, "upstream must never be called for an unsafe request with no Origin")
}

func TestHandler_BearerAuth_ForeignOrigin_StillProxies(t *testing.T) {
	var called bool
	mgr, r, _ := newOriginTestFixtures(t, &called)
	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	// Mobile apps have no Origin concept and cannot be tricked into
	// attaching one; the Bearer path must be completely unaffected by the
	// origin check, foreign Origin header or not.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chef/onboarding/status", nil)
	req.Header.Set("Authorization", "Bearer "+enc)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "the Bearer path must never be rejected by the Origin check")
}

func TestHandler_CookieAuth_MatchingOrigin_HonorsForwardedProto(t *testing.T) {
	var called bool
	mgr, r, _ := newOriginTestFixtures(t, &called)
	enc, err := mgr.Encode(newTestPayload())
	require.NoError(t, err)

	// The service always sits behind Istio/Cloudflare, so the real scheme
	// arrives via X-Forwarded-Proto, not TLS on the connection itself. A
	// hardcoded https-only comparison would reject this legitimate
	// same-origin http-fronted request; the check must derive scheme from
	// the header instead.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.AddCookie(&http.Cookie{Name: mgr.CookieName(), Value: enc})
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called)
}

// --- per-app session cookie isolation (SessionCookie / homechef-products.yaml) ---
//
// These tests wire Deps.CookieForHost to the real product registry, exactly
// as cmd/server/main.go does (reg.SessionCookieForHost), and prove the
// cross-app collision this seam closes: each portal's Host must resolve its
// own cookie name, and a cookie minted for a different app must NOT be
// accepted just because it arrived on the right domain.

// newHostCookieFixtures spins up a router wired with the real product
// registry's SessionCookieForHost as Deps.CookieForHost, plus an upstream
// that records the identity headers it received (or fails the test if it's
// hit when it shouldn't be).
func newHostCookieFixtures(t *testing.T) (mgr *session.Manager, r *gin.Engine, capturedUserID *string) {
	t.Helper()
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	mgr, err := session.NewManager(session.Config{EncryptKey: k, MaxAge: time.Hour})
	require.NoError(t, err)
	signer := headerproxy.NewSigner(headerproxy.SignerConfig{Key: []byte("test-signing-key-32-bytes-pad!!!")})

	reg, err := productregistry.Load("../../homechef-products.yaml")
	require.NoError(t, err)

	capturedUserID = new(string)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*capturedUserID = r.Header.Get(headerproxy.HdrUserID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	gin.SetMode(gin.TestMode)
	r = gin.New()
	r.Any("/api/v1/*proxyPath", Handler(&Deps{
		APIBaseURL:    upstream.URL,
		Sessions:      mgr,
		Signer:        signer,
		CookieForHost: reg.SessionCookieForHost,
	}))
	return mgr, r, capturedUserID
}

// hostRequest builds a GET (safe-method, no Origin required) request whose
// Host is exactly host and whose only cookie is {name: value}.
func hostRequest(host, cookieName, cookieValue string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "http://"+host+"/api/v1/chef/onboarding/status", nil)
	req.Host = host
	req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	return req
}

func TestHandler_VendorHost_ReadsVendorCookie_NotDefault(t *testing.T) {
	mgr, r, capturedUserID := newHostCookieFixtures(t)
	p := newTestPayload()
	p.UID = "vendor-user"
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := hostRequest("vendors.fe3dr.com", "hc_vendor_session", enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "vendor-user", *capturedUserID)
}

func TestHandler_WebHost_UsesDefaultSessionCookie(t *testing.T) {
	mgr, r, capturedUserID := newHostCookieFixtures(t)
	p := newTestPayload()
	p.UID = "web-user"
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := hostRequest("fe3dr.com", "hc_session", enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "web-user", *capturedUserID)
}

func TestHandler_AdminHost_UsesAdminCookie(t *testing.T) {
	mgr, r, capturedUserID := newHostCookieFixtures(t)
	p := newTestPayload()
	p.UID = "admin-user"
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := hostRequest("admin.fe3dr.com", "hc_admin_session", enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "admin-user", *capturedUserID)
}

// This is the actual bug: before per-app cookie names were honored, every
// portal shared hc_session on the same .fe3dr.com cookie domain, so a
// customer session cookie was silently accepted (and cross-mixed) on the
// vendor portal. With the fix, a request to vendors.fe3dr.com that only
// carries hc_session (no hc_vendor_session) must be treated as having no
// session at all — hc_session is simply not the cookie name this handler
// looks for on that Host.
func TestHandler_VendorHost_HcSessionCookie_NotAcceptedAsVendorSession(t *testing.T) {
	mgr, r, _ := newHostCookieFixtures(t)
	p := newTestPayload()
	p.UID = "customer-user"
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := hostRequest("vendors.fe3dr.com", "hc_session", enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.JSONEq(t, `{"error":"missing_session"}`, w.Body.String())
}

// The Bearer path must be entirely unaffected by CookieForHost: mobile
// clients send Authorization: Bearer <token> with no cookie, and the Host
// they connect through has no registered app. The Bearer branch in the
// handler never even calls CookieForHost — this proves the resolved cookie
// name plays no role in whether the request is accepted.
func TestHandler_BearerPath_UnaffectedByHostCookieResolver(t *testing.T) {
	mgr, r, capturedUserID := newHostCookieFixtures(t)
	p := newTestPayload()
	p.UID = "mobile-user"
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chef/onboarding/status", nil)
	req.Host = "api.internal.mobile.invalid" // not registered to any app
	req.Header.Set("Authorization", "Bearer "+enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "mobile-user", *capturedUserID)
}

// A Host the registry doesn't recognize must not error — it degrades to the
// Manager's default cookie name (session.Config.CookieName, "hc_session" in
// production) exactly like before this change, so nothing regresses for a
// host that isn't (yet) in homechef-products.yaml.
func TestHandler_UnknownHost_FallsBackToDefaultCookie(t *testing.T) {
	mgr, r, capturedUserID := newHostCookieFixtures(t)
	p := newTestPayload()
	p.UID = "fallback-user"
	enc, err := mgr.Encode(p)
	require.NoError(t, err)

	req := hostRequest("unregistered.fe3dr.com", mgr.CookieName(), enc)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "fallback-user", *capturedUserID)
}
