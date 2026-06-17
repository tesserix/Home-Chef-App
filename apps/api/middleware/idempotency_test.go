package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- pure helpers ---

func TestIsMutation(t *testing.T) {
	assert.True(t, isMutation(http.MethodPost))
	assert.True(t, isMutation(http.MethodPut))
	assert.True(t, isMutation(http.MethodPatch))
	assert.False(t, isMutation(http.MethodGet))
	assert.False(t, isMutation(http.MethodDelete))
	assert.False(t, isMutation(http.MethodOptions))
}

func TestMatchesAnyPrefix(t *testing.T) {
	prefixes := []string{"/api/v1/orders", "/api/v1/payments"}
	assert.True(t, matchesAnyPrefix("/api/v1/orders/123", prefixes))
	assert.True(t, matchesAnyPrefix("/api/v1/payments", prefixes))
	assert.False(t, matchesAnyPrefix("/api/v1/chefs", prefixes))
	assert.False(t, matchesAnyPrefix("/api/v1/orders", nil))
}

func TestIdempotencyCacheKey_StableAndBodySensitive(t *testing.T) {
	k1 := idempotencyCacheKey("user-1", "key-abc", "POST", "/api/v1/orders", []byte(`{"a":1}`))
	k2 := idempotencyCacheKey("user-1", "key-abc", "POST", "/api/v1/orders", []byte(`{"a":1}`))
	assert.Equal(t, k1, k2, "same inputs must produce same key")

	// Different body -> different key (defends against accidental key reuse).
	kBody := idempotencyCacheKey("user-1", "key-abc", "POST", "/api/v1/orders", []byte(`{"a":2}`))
	assert.NotEqual(t, k1, kBody)

	// Different user -> different key.
	kUser := idempotencyCacheKey("user-2", "key-abc", "POST", "/api/v1/orders", []byte(`{"a":1}`))
	assert.NotEqual(t, k1, kUser)

	// Anonymous user is namespaced under "anon".
	kAnon := idempotencyCacheKey("", "key-abc", "POST", "/api/v1/orders", []byte(`{"a":1}`))
	assert.True(t, strings.HasPrefix(kAnon, "idem:anon:"))
}

// --- middleware behavior (miniredis) ---

func newIdemRouter(t *testing.T, counter *int64) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := IdempotencyConfig{
		IncludedPathPrefixes: []string{"/api/v1/orders"},
		ResponseTTL:          24 * time.Hour,
		PendingTTL:           60 * time.Second,
	}
	r.POST("/api/v1/orders", Idempotency(cfg), func(c *gin.Context) {
		n := atomic.AddInt64(counter, 1)
		c.JSON(http.StatusCreated, gin.H{"order": "created", "run": n})
	})
	return r
}

func postIdem(r http.Handler, body, key string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/v1/orders", strings.NewReader(body))
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Same Idempotency-Key on a mutation returns the cached result and the handler
// only runs once (no double-write / double-charge).
func TestIdempotency_ReplaysCachedResponse_NoDoubleWrite(t *testing.T) {
	startMiniredis(t)
	var runs int64
	r := newIdemRouter(t, &runs)

	first := postIdem(r, `{"amount":100}`, "idem-key-1")
	require.Equal(t, http.StatusCreated, first.Code)
	assert.Empty(t, first.Header().Get("Idempotent-Replayed"))
	assert.Contains(t, first.Body.String(), `"run":1`)

	second := postIdem(r, `{"amount":100}`, "idem-key-1")
	require.Equal(t, http.StatusCreated, second.Code)
	assert.Equal(t, "true", second.Header().Get("Idempotent-Replayed"))
	// Replayed body is byte-identical to the first response (still run 1).
	assert.Equal(t, first.Body.String(), second.Body.String())

	assert.Equal(t, int64(1), atomic.LoadInt64(&runs), "handler must execute exactly once")
}

// A different body with the same key is treated as a fresh request (the body
// hash is part of the cache key).
func TestIdempotency_DifferentBody_NotReplayed(t *testing.T) {
	startMiniredis(t)
	var runs int64
	r := newIdemRouter(t, &runs)

	require.Equal(t, http.StatusCreated, postIdem(r, `{"amount":100}`, "idem-key-2").Code)
	w := postIdem(r, `{"amount":200}`, "idem-key-2")
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Empty(t, w.Header().Get("Idempotent-Replayed"))
	assert.Equal(t, int64(2), atomic.LoadInt64(&runs))
}

// Requests without the header pass through untouched (opt-in).
func TestIdempotency_NoHeader_PassesThrough(t *testing.T) {
	startMiniredis(t)
	var runs int64
	r := newIdemRouter(t, &runs)

	require.Equal(t, http.StatusCreated, postIdem(r, `{"x":1}`, "").Code)
	require.Equal(t, http.StatusCreated, postIdem(r, `{"x":1}`, "").Code)
	assert.Equal(t, int64(2), atomic.LoadInt64(&runs), "no dedup without a key")
}

// Non-included paths are not deduped even with a key present.
func TestIdempotency_NonIncludedPath_PassesThrough(t *testing.T) {
	startMiniredis(t)
	var runs int64
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/chefs", Idempotency(IdempotencyConfig{IncludedPathPrefixes: []string{"/api/v1/orders"}}),
		func(c *gin.Context) { atomic.AddInt64(&runs, 1); c.Status(http.StatusCreated) })

	post := func() {
		req := httptest.NewRequest("POST", "/api/v1/chefs", strings.NewReader(`{}`))
		req.Header.Set("Idempotency-Key", "k")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}
	post()
	post()
	assert.Equal(t, int64(2), atomic.LoadInt64(&runs))
}

// GET (non-mutation) is never deduped.
func TestIdempotency_GetNotDeduped(t *testing.T) {
	startMiniredis(t)
	var runs int64
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/orders", Idempotency(IdempotencyConfig{IncludedPathPrefixes: []string{"/api/v1/orders"}}),
		func(c *gin.Context) { atomic.AddInt64(&runs, 1); c.Status(http.StatusOK) })

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/api/v1/orders", nil)
		req.Header.Set("Idempotency-Key", "k")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}
	assert.Equal(t, int64(2), atomic.LoadInt64(&runs))
}

// 4xx responses ARE cached (clients shouldn't retry validation errors).
func TestIdempotency_4xxIsCached(t *testing.T) {
	startMiniredis(t)
	var runs int64
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/orders", Idempotency(IdempotencyConfig{IncludedPathPrefixes: []string{"/api/v1/orders"}}),
		func(c *gin.Context) { atomic.AddInt64(&runs, 1); c.JSON(http.StatusBadRequest, gin.H{"error": "bad"}) })

	require.Equal(t, http.StatusBadRequest, postIdem(r, `{}`, "k4xx").Code)
	w := postIdem(r, `{}`, "k4xx")
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "true", w.Header().Get("Idempotent-Replayed"))
	assert.Equal(t, int64(1), atomic.LoadInt64(&runs), "4xx cached, handler runs once")
}

// 5xx responses are NOT cached and the pending marker is cleared so a retry
// re-executes the handler.
func TestIdempotency_5xxNotCached_RetryReexecutes(t *testing.T) {
	startMiniredis(t)
	var runs int64
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/orders", Idempotency(IdempotencyConfig{IncludedPathPrefixes: []string{"/api/v1/orders"}}),
		func(c *gin.Context) {
			n := atomic.AddInt64(&runs, 1)
			if n == 1 {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "transient"})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"ok": true})
		})

	require.Equal(t, http.StatusInternalServerError, postIdem(r, `{}`, "k5xx").Code)
	w := postIdem(r, `{}`, "k5xx")
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Empty(t, w.Header().Get("Idempotent-Replayed"))
	assert.Equal(t, int64(2), atomic.LoadInt64(&runs), "5xx not cached, retry runs handler again")
}

// 24h TTL: after the response TTL elapses the cached entry is gone and the
// handler executes again on the same key.
func TestIdempotency_ResponseTTLExpiry(t *testing.T) {
	mr := startMiniredis(t)
	var runs int64
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/orders", Idempotency(IdempotencyConfig{
		IncludedPathPrefixes: []string{"/api/v1/orders"},
		ResponseTTL:          24 * time.Hour,
	}), func(c *gin.Context) { atomic.AddInt64(&runs, 1); c.JSON(http.StatusCreated, gin.H{"ok": true}) })

	require.Equal(t, http.StatusCreated, postIdem(r, `{}`, "kttl").Code)
	require.Equal(t, "true", postIdem(r, `{}`, "kttl").Header().Get("Idempotent-Replayed"))

	mr.FastForward(25 * time.Hour) // past the 24h response TTL
	w := postIdem(r, `{}`, "kttl")
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Empty(t, w.Header().Get("Idempotent-Replayed"))
	assert.Equal(t, int64(2), atomic.LoadInt64(&runs))
}

// Fail-open: with Redis down the middleware runs the handler uncached.
func TestIdempotency_FailOpen_WhenRedisDown(t *testing.T) {
	clearRedisClient(t)
	var runs int64
	r := newIdemRouter(t, &runs)

	require.Equal(t, http.StatusCreated, postIdem(r, `{}`, "k").Code)
	w := postIdem(r, `{}`, "k")
	require.Equal(t, http.StatusCreated, w.Code)
	assert.Empty(t, w.Header().Get("Idempotent-Replayed"))
	assert.Equal(t, int64(2), atomic.LoadInt64(&runs), "no dedup while Redis is down")
}

// In-flight collision: a second request arriving while the pending marker is
// still set (no cached response yet) gets a 409.
func TestIdempotency_InFlightCollision_409(t *testing.T) {
	mr := startMiniredis(t)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/orders", Idempotency(IdempotencyConfig{
		IncludedPathPrefixes: []string{"/api/v1/orders"},
		PendingTTL:           60 * time.Second,
	}), func(c *gin.Context) { c.JSON(http.StatusCreated, gin.H{"ok": true}) })

	// Pre-seed the pending marker exactly as the middleware would, simulating
	// another pod/request mid-flight on the same key (anon user, body "{}").
	body := []byte(`{}`)
	pendingKey := idempotencyCacheKey("", "kflight", "POST", "/api/v1/orders", body) + ":pending"
	mr.Set(pendingKey, "1")

	w := postIdem(r, `{}`, "kflight")
	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "already in flight")
}
