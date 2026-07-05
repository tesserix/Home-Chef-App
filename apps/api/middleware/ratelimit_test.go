package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func okHandler(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }

// fire sends a GET to path from a fixed X-Forwarded-For IP and returns the
// recorder. RemoteAddr is set to a trusted in-cluster peer (the Istio sidecar)
// so the forwarding header is honored — mirroring production, where every
// request reaches the app through the mesh.
func fire(r http.Handler, path, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	req.RemoteAddr = "10.0.0.1:41000"
	req.Header.Set("X-Forwarded-For", ip)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- clientIP (pure) ---

// A direct public caller (untrusted peer) cannot use X-Forwarded-For to change
// its rate-limit key — the header is ignored and the peer address is used.
func TestClientIP_UntrustedPeer_IgnoresForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.50:5000" // public → untrusted
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	assert.Equal(t, "203.0.113.50", clientIP(req))
}

// The bypass this closes: rotating X-Forwarded-For from the same untrusted peer
// must resolve to the SAME key, so the attacker can't mint a fresh bucket per
// value.
func TestClientIP_ForgedForwardedFor_CannotBypass(t *testing.T) {
	mk := func(xff string) *http.Request {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "198.51.100.9:6000" // public → untrusted
		req.Header.Set("X-Forwarded-For", xff)
		return req
	}
	assert.Equal(t, clientIP(mk("1.1.1.1")), clientIP(mk("2.2.2.2")))
	assert.Equal(t, "198.51.100.9", clientIP(mk("9.9.9.9")))
}

// Behind a trusted proxy, Cloudflare's CF-Connecting-IP is authoritative.
func TestClientIP_TrustedPeer_PrefersCloudflare(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080" // sidecar → trusted
	req.Header.Set("CF-Connecting-IP", "198.51.100.7")
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.5")
	assert.Equal(t, "198.51.100.7", clientIP(req))
}

// Behind a trusted proxy with no Cloudflare header, take the rightmost XFF hop
// (appended by the closest trusted proxy) — never the client-controllable
// leftmost entry.
func TestClientIP_TrustedPeer_RightmostForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.2:9000" // private → trusted
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.5")
	assert.Equal(t, "203.0.113.5", clientIP(req))
}

// The real multi-hop chain is `client, cf-edge, gateway` where the tail hops are
// internal. clientIP must skip the trusted (private) tail and return the real
// client — never the internal hop IP, which would collapse all users into one
// bucket.
func TestClientIP_TrustedPeer_SkipsInternalTail(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080" // sidecar → trusted
	// client(public), cf-edge(public), gateway(private)
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 198.51.100.2, 10.0.0.9")
	assert.Equal(t, "198.51.100.2", clientIP(req))
}

// If every forwarded hop is a trusted/internal IP (no public client in the
// chain), fall back to the peer address rather than keying on an internal hop.
func TestClientIP_TrustedPeer_AllHopsInternal_FallsBack(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	assert.Equal(t, "127.0.0.1", clientIP(req))
}

func TestClientIP_FallbackRemoteAddr(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.4:54321"
	assert.Equal(t, "192.0.2.4", clientIP(req))
}

func TestToString(t *testing.T) {
	assert.Equal(t, "abc", toString("abc"))
	assert.Equal(t, "", toString(123)) // unsupported types -> ""
	assert.Equal(t, "5", toString(stringer{}))
}

type stringer struct{}

func (stringer) String() string { return "5" }

// --- in-memory RateLimit (per-IP token bucket) ---

func TestRateLimit_ThrottlesAfterBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// rps 0 so the bucket never refills within the test; burst 3.
	r.GET("/login", RateLimit(0, 3), okHandler)

	ip := "203.0.113.10"
	for i := 0; i < 3; i++ {
		require.Equalf(t, http.StatusOK, fire(r, "/login", ip).Code, "request %d", i+1)
	}
	w := fire(r, "/login", ip)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "60", w.Header().Get("Retry-After"))
}

func TestRateLimit_IsolatesByIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/login", RateLimit(0, 1), okHandler)

	require.Equal(t, http.StatusOK, fire(r, "/login", "203.0.113.20").Code)
	require.Equal(t, http.StatusTooManyRequests, fire(r, "/login", "203.0.113.20").Code)
	// A different IP has its own bucket.
	require.Equal(t, http.StatusOK, fire(r, "/login", "203.0.113.21").Code)
}

func TestRateLimitByUser_KeysByUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/orders", func(c *gin.Context) { c.Set("userID", "user-42") }, RateLimitByUser(0, 1), okHandler)

	// Same user from two different IPs shares one bucket.
	require.Equal(t, http.StatusOK, fire(r, "/orders", "203.0.113.30").Code)
	w := fire(r, "/orders", "203.0.113.31")
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "30", w.Header().Get("Retry-After"))
}

// --- Redis-backed RateLimitRedis (miniredis) ---

func TestRateLimitRedis_UnauthThrottlesAfterBudget(t *testing.T) {
	startMiniredis(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/x", RateLimitRedis(RateLimitRedisConfig{UnauthedPerMin: 3}), okHandler)

	ip := "203.0.113.40"
	for i := 0; i < 3; i++ {
		w := fire(r, "/api/v1/x", ip)
		require.Equalf(t, http.StatusOK, w.Code, "request %d", i+1)
		assert.Equal(t, "3", w.Header().Get("X-RateLimit-Limit"))
		// remaining counts down: 2, 1, 0
		rem, _ := strconv.Atoi(w.Header().Get("X-RateLimit-Remaining"))
		assert.Equal(t, 3-(i+1), rem)
	}
	w := fire(r, "/api/v1/x", ip)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	assert.Equal(t, "0", w.Header().Get("X-RateLimit-Remaining"))
}

func TestRateLimitRedis_AuthedUsesAuthedBudget(t *testing.T) {
	startMiniredis(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/x",
		func(c *gin.Context) { c.Set("userID", "user-99") },
		RateLimitRedis(RateLimitRedisConfig{AuthedPerMin: 2, UnauthedPerMin: 100}),
		okHandler,
	)

	require.Equal(t, http.StatusOK, fire(r, "/api/v1/x", "203.0.113.50").Code)
	w2 := fire(r, "/api/v1/x", "203.0.113.50")
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "2", w2.Header().Get("X-RateLimit-Limit")) // authed budget, not unauth 100
	require.Equal(t, http.StatusTooManyRequests, fire(r, "/api/v1/x", "203.0.113.50").Code)
}

func TestRateLimitRedis_ExcludedPathBypasses(t *testing.T) {
	startMiniredis(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", RateLimitRedis(RateLimitRedisConfig{UnauthedPerMin: 1, ExcludedPaths: []string{"/health"}}), okHandler)

	ip := "203.0.113.60"
	for i := 0; i < 5; i++ {
		require.Equal(t, http.StatusOK, fire(r, "/health", ip).Code)
	}
}

func TestRateLimitRedis_DefaultBudgets(t *testing.T) {
	startMiniredis(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Zero values -> defaults: unauth 30/min.
	r.GET("/api/v1/x", RateLimitRedis(RateLimitRedisConfig{}), okHandler)

	w := fire(r, "/api/v1/x", "203.0.113.70")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "30", w.Header().Get("X-RateLimit-Limit"))
}

// Fail-open: when Redis is not connected the limiter must let every request
// through (a Redis outage must never brick the API).
func TestRateLimitRedis_FailOpen_WhenRedisDown(t *testing.T) {
	clearRedisClient(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/x", RateLimitRedis(RateLimitRedisConfig{UnauthedPerMin: 1}), okHandler)

	ip := "203.0.113.80"
	for i := 0; i < 10; i++ {
		w := fire(r, "/api/v1/x", ip)
		require.Equalf(t, http.StatusOK, w.Code, "request %d should pass while Redis is down", i+1)
		// No rate-limit headers are emitted on the fail-open path.
		assert.Empty(t, w.Header().Get("X-RateLimit-Limit"))
	}
}

// Window roll-over: after the minute bucket advances, the counter resets and
// a previously-throttled client is allowed again.
func TestRateLimitRedis_WindowRollover(t *testing.T) {
	mr := startMiniredis(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/x", RateLimitRedis(RateLimitRedisConfig{UnauthedPerMin: 1}), okHandler)

	ip := "203.0.113.90"
	require.Equal(t, http.StatusOK, fire(r, "/api/v1/x", ip).Code)
	require.Equal(t, http.StatusTooManyRequests, fire(r, "/api/v1/x", ip).Code)

	// Advance miniredis past the 70s key TTL so the fixed-window bucket key
	// expires; the next request lands in a fresh bucket and is allowed.
	mr.FastForward(71 * time.Second)
	require.Equal(t, http.StatusOK, fire(r, "/api/v1/x", ip).Code)
}
