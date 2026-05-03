package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/homechef/api/config"
)

// SecurityHeaders applies a defense-in-depth set of HTTP response headers.
// CSP is intentionally narrow and applies to API responses only — frontends
// set their own CSP at the SPA level.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(self), microphone=(), camera=(), payment=(self)")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-site")
		if config.IsProduction() {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		}
		c.Next()
	}
}
