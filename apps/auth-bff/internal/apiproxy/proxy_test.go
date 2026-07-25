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
