package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover BFFIdentify, which resolves identity WITHOUT enforcing it
// so that RateLimitRedis (registered on the /api/v1 group, ahead of the
// per-subgroup BFFAuth) can key per-user budgets by user. The bug it fixes:
// the limiter ran before any auth, so c.Get("userID") was never set and every
// authenticated caller was silently demoted to the per-IP budget.
//
// The security-critical invariant is that BFFIdentify must never let a request
// SKIP enforcement — it only ever marks resolution as done after a successful
// verification.

func testKey() []byte { return []byte("identify-test-key") }

// identifyChain mirrors production wiring: BFFIdentify on the group, BFFAuth on
// the protected subgroup. Returns what the final handler observed.
func identifyChain(t *testing.T, cfg BFFAuthConfig, req *http.Request) (status int, sawUserID string, handlerRan bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(BFFIdentify(cfg))
	// Stand-in for the rate limiter: records the identity visible at the point
	// the limiter would key its bucket.
	v1.Use(func(c *gin.Context) {
		if uid, ok := c.Get("userID"); ok {
			sawUserID = toString(uid)
		}
		c.Next()
	})
	protected := v1.Group("/protected")
	protected.Use(BFFAuth(cfg))
	protected.GET("", func(c *gin.Context) {
		handlerRan = true
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code, sawUserID, handlerRan
}

// The core regression: a signed request must be visible as a USER to
// middleware registered before BFFAuth.
func TestBFFIdentify_SignedRequest_VisibleToEarlierMiddleware(t *testing.T) {
	uid := uuid.New().String()
	id := BFFIdentity{UserID: uid, Email: "chef@fe3dr.com", Role: "chef", Pool: "customer"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	attachSigned(req, nil, testKey(), id, time.Now().Unix())

	status, sawUserID, ran := identifyChain(t, BFFAuthConfig{HMACKey: testKey()}, req)

	assert.Equal(t, http.StatusOK, status)
	assert.True(t, ran, "handler should run for a validly signed request")
	assert.Equal(t, uid, sawUserID,
		"rate limiter must see the user id; empty means it would key by IP and re-introduce the bug")
}

// An anonymous caller stays anonymous and is still rejected by BFFAuth.
func TestBFFIdentify_NoCredentials_AnonymousAndStillEnforced(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)

	status, sawUserID, ran := identifyChain(t, BFFAuthConfig{HMACKey: testKey()}, req)

	assert.Equal(t, http.StatusUnauthorized, status, "BFFAuth must still enforce")
	assert.Empty(t, sawUserID)
	assert.False(t, ran)
}

// A forged signature must NOT be identified and must NOT skip enforcement.
// This is the key security property: if BFFIdentify marked resolution done on
// failure, BFFAuth would trust it and let the request through.
func TestBFFIdentify_ForgedSignature_NotIdentified_And401(t *testing.T) {
	id := BFFIdentity{UserID: uuid.New().String(), Email: "attacker@evil.com", Role: "admin", Pool: "internal"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	attachSigned(req, nil, []byte("wrong-key"), id, time.Now().Unix())

	status, sawUserID, ran := identifyChain(t, BFFAuthConfig{HMACKey: testKey()}, req)

	assert.Equal(t, http.StatusUnauthorized, status, "forged signature must be rejected")
	assert.Empty(t, sawUserID, "forged identity must never reach the limiter")
	assert.False(t, ran, "handler must not run")
}

// A stale timestamp is a replay attempt: not identified, still 401.
func TestBFFIdentify_StaleTimestamp_NotIdentified_And401(t *testing.T) {
	id := BFFIdentity{UserID: uuid.New().String(), Email: "chef@fe3dr.com", Role: "chef", Pool: "customer"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	attachSigned(req, nil, testKey(), id, time.Now().Add(-10*time.Minute).Unix())

	status, sawUserID, ran := identifyChain(t, BFFAuthConfig{HMACKey: testKey()}, req)

	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Empty(t, sawUserID)
	assert.False(t, ran)
}

// BFFIdentify must not consume the body: the POST body has to survive intact
// for BFFAuth's re-hash AND for the handler to bind.
func TestBFFIdentify_PostBody_SurvivesForHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uid := uuid.New().String()
	id := BFFIdentity{UserID: uid, Email: "chef@fe3dr.com", Role: "chef", Pool: "customer"}
	body := []byte(`{"item":"dal","qty":1}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/protected", bytesReader(body))
	attachSigned(req, body, testKey(), id, time.Now().Unix())

	var got string
	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(BFFIdentify(BFFAuthConfig{HMACKey: testKey()}))
	protected := v1.Group("/protected")
	protected.Use(BFFAuth(BFFAuthConfig{HMACKey: testKey()}))
	protected.POST("", func(c *gin.Context) {
		b, _ := c.GetRawData()
		got = string(b)
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "signature over the body must still verify after BFFIdentify read it")
	assert.Equal(t, string(body), got, "handler must still see the full body")
}

// Reused resolution must not weaken the signature check: BFFAuth trusting the
// marker is only safe because the marker implies a successful verify.
func TestBFFIdentify_TamperedRole_NotIdentified_And401(t *testing.T) {
	id := BFFIdentity{UserID: uuid.New().String(), Email: "chef@fe3dr.com", Role: "chef", Pool: "customer"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	attachSigned(req, nil, testKey(), id, time.Now().Unix())
	// Swap the role AFTER signing — the MAC binds identity, so this must fail.
	req.Header.Set(HdrUserRole, "admin")

	status, sawUserID, ran := identifyChain(t, BFFAuthConfig{HMACKey: testKey()}, req)

	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Empty(t, sawUserID)
	assert.False(t, ran)
}

// BFFAuth must remain self-sufficient on groups where BFFIdentify never ran
// (wsGroup, /internal — both attach bffAuth directly, outside /api/v1).
func TestBFFAuth_WithoutIdentify_StillResolves(t *testing.T) {
	gin.SetMode(gin.TestMode)
	uid := uuid.New().String()
	id := BFFIdentity{UserID: uid, Email: "chef@fe3dr.com", Role: "chef", Pool: "customer"}
	req := httptest.NewRequest(http.MethodGet, "/internal/thing", nil)
	attachSigned(req, nil, testKey(), id, time.Now().Unix())

	var sawUserID string
	r := gin.New()
	g := r.Group("/internal")
	g.Use(BFFAuth(BFFAuthConfig{HMACKey: testKey()}))
	g.GET("/thing", func(c *gin.Context) {
		if v, ok := c.Get("userID"); ok {
			sawUserID = toString(v)
		}
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, uid, sawUserID, "BFFAuth must still resolve identity on its own")
}

// The resolved marker is a Gin context key, not a header — a client must not be
// able to set it and skip verification.
func TestBFFIdentify_ClientCannotForgeResolvedMarker(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	// Try every plausible header spelling of the internal marker.
	req.Header.Set("bff_identity_resolved", "true")
	req.Header.Set("X-Bff-Identity-Resolved", "true")
	req.Header.Set("Bff-Identity-Resolved", "true")

	status, sawUserID, ran := identifyChain(t, BFFAuthConfig{HMACKey: testKey()}, req)

	assert.Equal(t, http.StatusUnauthorized, status, "header must not bypass enforcement")
	assert.Empty(t, sawUserID)
	assert.False(t, ran)
}

func TestHasBFFCredentials(t *testing.T) {
	cases := []struct {
		name string
		set  func(*http.Request)
		want bool
	}{
		{"none", func(*http.Request) {}, false},
		{"signature", func(r *http.Request) { r.Header.Set(HdrSignature, "abc") }, true},
		{"bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer tok") }, true},
		{"basic auth is not a BFF credential", func(r *http.Request) { r.Header.Set("Authorization", "Basic xyz") }, false},
		{"empty bearer prefix only", func(r *http.Request) { r.Header.Set("Authorization", "Bearer ") }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/x", nil)
			tc.set(r)
			assert.Equal(t, tc.want, hasBFFCredentials(r))
		})
	}
}

// Guards the intent of the fix at the config level: budgets must stay above
// observed organic traffic (~40 req/min browsing + checkout). If someone tunes
// these back down toward organic load, real checkouts start 429ing again.
func TestRateLimitRedisConfig_DefaultsAreAboveOrganicTraffic(t *testing.T) {
	const observedOrganicPeakPerMin = 40
	cfg := RateLimitRedisConfig{AuthedPerMin: 300, UnauthedPerMin: 120}
	assert.Greater(t, cfg.AuthedPerMin, observedOrganicPeakPerMin*2,
		"authed budget must leave real headroom over organic traffic")
	assert.Greater(t, cfg.UnauthedPerMin, observedOrganicPeakPerMin,
		"unauthed budget is per-IP and must tolerate NAT/CGNAT sharing")
}

func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
