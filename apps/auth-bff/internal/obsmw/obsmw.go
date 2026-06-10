// Package obsmw holds lightweight observability middleware for auth-bff:
// request-id propagation and trace-id echo, mirroring the API's correlation
// surface so a login can be followed across both services.
package obsmw

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// RequestIDHeader is the inbound/outbound correlation-ID header.
const RequestIDHeader = "X-Request-ID"

// RequestID extracts X-Request-ID (or mints a UUID) and echoes it on the
// response so a client action can be stitched to its auth-bff request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}

// TraceContext echoes the active OpenTelemetry span's trace id as X-Trace-ID.
// Must run AFTER the otelgin middleware that establishes the span.
func TraceContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		sc := trace.SpanContextFromContext(c.Request.Context())
		if sc.HasTraceID() {
			c.Header("X-Trace-ID", sc.TraceID().String())
		}
		c.Next()
	}
}
