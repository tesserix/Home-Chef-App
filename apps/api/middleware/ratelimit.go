package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// rateLimiterEntry tracks one client's bucket plus a last-seen timestamp so
// idle entries can be GC'd to keep the map bounded.
type rateLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiterStore struct {
	mu       sync.Mutex
	clients  map[string]*rateLimiterEntry
	rps      rate.Limit
	burst    int
	gcTTL    time.Duration
	lastSwep time.Time
}

func newRateLimiterStore(rps rate.Limit, burst int) *rateLimiterStore {
	return &rateLimiterStore{
		clients:  make(map[string]*rateLimiterEntry),
		rps:      rps,
		burst:    burst,
		gcTTL:    10 * time.Minute,
		lastSwep: time.Now(),
	}
}

func (s *rateLimiterStore) get(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if now.Sub(s.lastSwep) > s.gcTTL {
		for k, e := range s.clients {
			if now.Sub(e.lastSeen) > s.gcTTL {
				delete(s.clients, k)
			}
		}
		s.lastSwep = now
	}
	e, ok := s.clients[key]
	if !ok {
		e = &rateLimiterEntry{limiter: rate.NewLimiter(s.rps, s.burst)}
		s.clients[key] = e
	}
	e.lastSeen = now
	return e.limiter
}

// clientIP returns the best-effort client IP. Trusts the leftmost X-Forwarded-For
// when present (Cloudflare/Istio sets this) and falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return cf
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// RateLimit returns a per-IP token-bucket middleware. Use for unauthenticated
// surfaces (login, register, password-reset) to throttle credential stuffing
// and email enumeration.
//
//	rps   — sustained requests per second
//	burst — short-burst allowance
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	store := newRateLimiterStore(rate.Limit(rps), burst)
	return func(c *gin.Context) {
		ip := clientIP(c.Request)
		if !store.get(ip).Allow() {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please slow down.",
			})
			return
		}
		c.Next()
	}
}

// RateLimitByUser keys the bucket by authenticated user ID falling back to IP.
// Use for authenticated surfaces where the abuse vector is per-account
// (e.g. order-creation flooding) rather than per-IP.
func RateLimitByUser(rps float64, burst int) gin.HandlerFunc {
	store := newRateLimiterStore(rate.Limit(rps), burst)
	return func(c *gin.Context) {
		key := clientIP(c.Request)
		if uid, ok := c.Get("userID"); ok {
			key = "u:" + toString(uid)
		}
		if !store.get(key).Allow() {
			c.Header("Retry-After", "30")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please slow down.",
			})
			return
		}
		c.Next()
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case interface{ String() string }:
		return x.String()
	}
	return ""
}
