package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/homechef/api/logger"
	"go.opentelemetry.io/otel/trace"
)

// TraceContext bridges the active OpenTelemetry span into the logging/audit
// world: it copies the current trace id into the request context (so
// logger.FromContext stamps trace_id on every line) and echoes it back as
// X-Trace-ID for client-side correlation. Must run AFTER the otelgin
// middleware, which establishes the span.
func TraceContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		sc := trace.SpanContextFromContext(c.Request.Context())
		if sc.HasTraceID() {
			tid := sc.TraceID().String()
			c.Request = c.Request.WithContext(
				logger.ContextWithTraceID(c.Request.Context(), tid),
			)
			c.Header("X-Trace-ID", tid)
		}
		c.Next()
	}
}
