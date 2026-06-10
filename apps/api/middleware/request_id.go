package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/logger"
)

// RequestIDHeader is the inbound/outbound correlation-ID header. Frontends and
// the BFF may set it to stitch a client action to its server requests; if
// absent we mint one.
const RequestIDHeader = "X-Request-ID"

// RequestIDKey is the gin.Context key handlers can read the id from.
const RequestIDKey = "request_id"

// RequestID extracts X-Request-ID (or mints a UUID), stamps it onto the gin
// context, the request's context (for logger.FromContext), and the response
// header. Runs first so every downstream log line, span, and audit row shares
// the same correlation id.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(RequestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(RequestIDKey, id)
		c.Request = c.Request.WithContext(
			logger.ContextWithCorrelationID(c.Request.Context(), id),
		)
		c.Header(RequestIDHeader, id)
		c.Next()
	}
}
