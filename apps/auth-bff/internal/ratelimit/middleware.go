package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type Limiter struct {
	mu      sync.Mutex
	clients map[string]*entry
	rps     rate.Limit
	burst   int
	ttl     time.Duration
}

type entry struct {
	lim     *rate.Limiter
	lastHit time.Time
}

// New creates a limiter that allows up to `rps` requests per second per key,
// with `burst` instantaneous capacity. Idle entries older than `ttl` are
// reaped lazily on each request.
func New(rps float64, burst int) *Limiter {
	return &Limiter{
		clients: map[string]*entry{},
		rps:     rate.Limit(rps),
		burst:   burst,
		ttl:     10 * time.Minute,
	}
}

// Middleware keys by client IP.
func (l *Limiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate_limited"})
			return
		}
		c.Next()
	}
}

// Allow exposes the per-key check for testing.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	e, ok := l.clients[key]
	if !ok {
		e = &entry{lim: rate.NewLimiter(l.rps, l.burst)}
		l.clients[key] = e
	}
	e.lastHit = now
	// Reap idle entries occasionally (every call is fine for low cardinality)
	for k, v := range l.clients {
		if now.Sub(v.lastHit) > l.ttl {
			delete(l.clients, k)
		}
	}
	return e.lim.Allow()
}
