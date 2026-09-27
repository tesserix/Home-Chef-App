package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/services"
)

func secretMutationPaused(c *gin.Context) bool {
	if !services.SecretWritesPaused() {
		return false
	}
	c.Header("Retry-After", "60")
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
		"error": "Payment-detail and gateway-credential changes are temporarily unavailable. Please try again shortly.",
		"code":  "SECRET_WRITES_PAUSED",
	})
	return true
}
