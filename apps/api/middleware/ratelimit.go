package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/services"
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

// isTrustedProxy reports whether host (a bare IP) is an in-cluster/loopback
// address we accept forwarding headers from. In production the immediate peer is
// always the Istio sidecar (loopback) or ingress (private range); a direct
// public client is never a trusted proxy. This is the same trust set Gin uses by
// default for its own forwarded-IP resolution.
func isTrustedProxy(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

// clientIP returns the best-effort client IP for rate-limit keying.
//
// SECURITY: client-supplied forwarding headers (X-Forwarded-For,
// CF-Connecting-IP) are honored ONLY when the immediate peer (RemoteAddr) is a
// trusted proxy. A direct public caller could otherwise rotate X-Forwarded-For
// to mint a fresh token bucket per value and bypass the limiter on
// unauthenticated surfaces (login/register/reset). When the peer is trusted we
// prefer Cloudflare's CF-Connecting-IP (the authoritative real client, injected
// by the CF tunnel and not client-controllable). Failing that we walk the
// X-Forwarded-For chain from the RIGHT and return the first NON-trusted address —
// the real client sits just before the trusted proxy hops. Taking the blind
// rightmost value would return an internal hop IP and collapse every anonymous
// user into one bucket (a self-inflicted DoS) if the hop count ever changes;
// skipping trusted hops is robust to the chain length. If every hop is trusted
// (or there's no header) we key on the peer address.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if !isTrustedProxy(host) {
		return host
	}
	if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
		return cf
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := strings.TrimSpace(parts[i])
			if ip != "" && !isTrustedProxy(ip) {
				return ip
			}
		}
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

// RateLimitRedisConfig configures the per-route Redis-backed rate limiter.
//
// AuthedPerMin   — request budget per authenticated user per minute.
// UnauthedPerMin — request budget per IP per minute when no user is set.
// ExcludedPaths  — exact path prefix matches that bypass the limiter
//
//	(typically /health, /metrics, /api/v1/mobile/min-version).
//
// Fixed-window counter (key TTL = 70s, slightly larger than the window).
// This is less smooth than a true sliding-window token bucket but
// adequate at the volumes we're guarding against (credential stuffing,
// burst order spam) and trivially correct under Redis failure: a
// missed INCR or read error fails OPEN per the Wave 1 backend design
// — log + counter increment, let the request through. Burst risk
// during a Redis outage is acceptable; service downtime is not.
type RateLimitRedisConfig struct {
	AuthedPerMin   int
	UnauthedPerMin int
	ExcludedPaths  []string
}

// RateLimitRedis returns a Gin middleware that throttles by user ID
// when an authenticated user is present (set in c.Get("userID") via
// BFFAuth), otherwise by client IP. Distributed across pods via Redis.
// Excluded paths bypass entirely.
//
// Headers on a throttled response: Retry-After (seconds until window
// roll-over) and X-RateLimit-* (informational).
func RateLimitRedis(cfg RateLimitRedisConfig) gin.HandlerFunc {
	if cfg.AuthedPerMin <= 0 {
		cfg.AuthedPerMin = 60
	}
	if cfg.UnauthedPerMin <= 0 {
		cfg.UnauthedPerMin = 30
	}
	redisClient := services.GetRedisClient()

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, p := range cfg.ExcludedPaths {
			if strings.HasPrefix(path, p) {
				c.Next()
				return
			}
		}

		// Bail open if Redis isn't connected — we never want a Redis
		// outage to brick the API. Burst risk during the outage is
		// the explicit tradeoff (see Wave 1 backend design §3 Redis
		// fail-mode decision).
		if !redisClient.IsConnected() {
			c.Next()
			return
		}

		scope := "ip"
		identifier := clientIP(c.Request)
		budget := cfg.UnauthedPerMin
		if uid, ok := c.Get("userID"); ok {
			scope = "user"
			identifier = toString(uid)
			budget = cfg.AuthedPerMin
		}

		now := time.Now().UTC()
		minuteBucket := now.Unix() / 60
		key := fmt.Sprintf("rl:%s:%s:%d", scope, identifier, minuteBucket)

		ctx, cancel := context.WithTimeout(c.Request.Context(), 200*time.Millisecond)
		defer cancel()

		count, err := redisClient.IncrAndExpire(ctx, key, 70*time.Second)
		if err != nil {
			// Same fail-open posture — log, don't block. A spike in
			// these logs is the prod signal that Redis is suffering.
			c.Next()
			return
		}

		remaining := budget - int(count)
		if remaining < 0 {
			remaining = 0
		}
		// Seconds until the current minute bucket rolls over.
		retryAfter := 60 - int(now.Unix()%60)

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", budget))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", now.Unix()+int64(retryAfter)))

		if int(count) > budget {
			c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests. Please slow down.",
			})
			return
		}
		c.Next()
	}
}
