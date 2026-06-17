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
// recorder. Optionally sets an authenticated userID into the gin context.
func fire(r http.Handler, path, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("X-Forwarded-For", ip)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- clientIP (pure) ---

func TestClientIP_PrefersForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")
	assert.Equal(t, "203.0.113.5", clientIP(req))
}

func TestClientIP_SingleForwardedFor(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	assert.Equal(t, "203.0.113.9", clientIP(req))
}

func TestClientIP_CloudflareHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("CF-Connecting-IP", "198.51.100.7")
	assert.Equal(t, "198.51.100.7", clientIP(req))
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
