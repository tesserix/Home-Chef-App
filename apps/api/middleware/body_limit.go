package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit bounds request bodies before parsers or upload handlers can buffer
// them. MaxBytesReader also protects requests with no Content-Length header.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request_too_large"})
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
