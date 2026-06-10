package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/logger"
)

// quietPaths are health/readiness/metrics probes that fire constantly and add
// no signal to the access log.
var quietPaths = map[string]bool{
	"/health":  true,
	"/ready":   true,
	"/metrics": true,
}

// RequestLogger emits one structured (JSON) access-log line per request,
// carrying the correlation id from context plus method, route, status, and
// latency. Replaces gin's default text logger so prod logs are machine-parsable
// in Cloud Logging. Probe paths are skipped to keep the log readable.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		if quietPaths[path] {
			return
		}

		l := logger.FromContext(c.Request.Context())
		args := []any{
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			l.Error("http_request", append(args, "errors", c.Errors.String())...)
			return
		}
		l.Info("http_request", args...)
	}
}
