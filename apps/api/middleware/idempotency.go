package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/services"
)

// IdempotencyConfig configures the Idempotency-Key middleware.
//
// IncludedPathPrefixes — only POST/PUT/PATCH requests whose path
//
//	starts with one of these are deduped. Lets
//	us scope to the mutations that actually
//	need it (orders / payments / docs / menu)
//	rather than every mutation in the API.
//
// ResponseTTL          — how long to cache a non-5xx response.
//
//	24h matches the Wave 1 backend design.
//
// PendingTTL           — how long the "in flight" marker survives
//
//	if the handler crashes. Should be longer
//	than any reasonable single-request timeout.
type IdempotencyConfig struct {
	IncludedPathPrefixes []string
	ResponseTTL          time.Duration
	PendingTTL           time.Duration
}

// Idempotency returns middleware that dedups POST/PUT/PATCH requests
// carrying an Idempotency-Key header. Opt-in: requests without the
// header pass through untouched (web + admin + delivery portals don't
// need this; we only flip it on for mobile flows that retry on flaky
// networks).
//
// Cache key includes the user ID, idempotency key, method, path, and
// a body hash so a replay with a different body is treated as a
// fresh request (defends against accidental key reuse across calls).
//
// Fail-open behavior:
//   - Redis unreachable → request runs uncached.
//   - 5xx response → marker cleared so a retry can re-execute.
//   - 4xx response IS cached (clients shouldn't retry validation
//     errors expecting different results).
func Idempotency(cfg IdempotencyConfig) gin.HandlerFunc {
	if cfg.ResponseTTL == 0 {
		cfg.ResponseTTL = 24 * time.Hour
	}
	if cfg.PendingTTL == 0 {
		cfg.PendingTTL = 60 * time.Second
	}
	redisClient := services.GetRedisClient()

	return func(c *gin.Context) {
		if !isMutation(c.Request.Method) {
			c.Next()
			return
		}
		if !matchesAnyPrefix(c.Request.URL.Path, cfg.IncludedPathPrefixes) {
			c.Next()
			return
		}
		idemKey := c.GetHeader("Idempotency-Key")
		if idemKey == "" {
			c.Next()
			return
		}
		if !redisClient.IsConnected() {
			c.Next()
			return
		}

		// Read the body once; replay into the handler via NopCloser
		// so c.ShouldBindJSON / c.GetRawData inside handlers still see it.
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		userID := ""
		if uid, ok := c.Get("userID"); ok {
			userID = toString(uid)
		}

		cacheKey := idempotencyCacheKey(userID, idemKey, c.Request.Method, c.Request.URL.Path, body)
		pendingKey := cacheKey + ":pending"

		ctx, cancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
		defer cancel()

		// Cached response replay path.
		if cached := loadCachedResponse(ctx, redisClient, cacheKey); cached != nil {
			c.Header("Idempotent-Replayed", "true")
			writeCachedResponse(c, cached)
			c.Abort()
			return
		}

		// Try to claim the in-flight slot. If another pod / request
		// is mid-flight on the same key we 409 — the client should
		// poll or retry shortly. This is rare for human-driven UX
		// but matters under push-notification storms.
		acquired, err := redisClient.SetNX(ctx, pendingKey, "1", cfg.PendingTTL)
		if err != nil || !acquired {
			if err != nil {
				// Redis hiccup — fail open.
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"error": "Request with this Idempotency-Key is already in flight",
			})
			return
		}

		// Wrap the writer so we can capture the response for caching.
		capture := &idempotencyCapture{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = capture

		c.Next()

		// Cache the response unless it's a 5xx (those are transient).
		// Release the pending marker either way so the client can retry.
		storeCtx, storeCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer storeCancel()
		if capture.status < 500 {
			payload := cachedResponse{
				Status:      capture.status,
				ContentType: capture.Header().Get("Content-Type"),
				Body:        capture.body.Bytes(),
			}
			if data, err := json.Marshal(payload); err == nil {
				_ = redisClient.Set(storeCtx, cacheKey, string(data), cfg.ResponseTTL)
			}
		}
		_ = redisClient.Del(storeCtx, pendingKey)
	}
}

func isMutation(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	}
	return false
}

func matchesAnyPrefix(path string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func idempotencyCacheKey(userID, idemKey, method, path string, body []byte) string {
	sum := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(sum[:8]) // 16 hex chars — collision-safe enough for keyed dedup
	if userID == "" {
		userID = "anon"
	}
	return fmt.Sprintf("idem:%s:%s:%s:%s:%s", userID, idemKey, method, path, bodyHash)
}

type cachedResponse struct {
	Status      int    `json:"status"`
	ContentType string `json:"contentType"`
	Body        []byte `json:"body"`
}

func loadCachedResponse(ctx context.Context, r *services.RedisClient, key string) *cachedResponse {
	raw, err := r.Get(ctx, key)
	if err != nil || raw == "" {
		return nil
	}
	var resp cachedResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil
	}
	return &resp
}

func writeCachedResponse(c *gin.Context, resp *cachedResponse) {
	if resp.ContentType != "" {
		c.Header("Content-Type", resp.ContentType)
	}
	c.Writer.WriteHeader(resp.Status)
	_, _ = c.Writer.Write(resp.Body)
}

// idempotencyCapture wraps gin.ResponseWriter to tee Write/WriteHeader
// calls into a buffer so we can cache the full response. Status
// defaults to 200 if the handler never calls WriteHeader explicitly
// (matches Go's stdlib behavior).
type idempotencyCapture struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (w *idempotencyCapture) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *idempotencyCapture) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *idempotencyCapture) WriteString(s string) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}
