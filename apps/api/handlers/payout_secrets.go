package handlers

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

func storePayoutSecrets(c *gin.Context, owner string, write func(context.Context, string, string, string) error, fields map[string]string) bool {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	names := make([]string, 0, len(fields))
	for field := range fields {
		names = append(names, field)
	}
	sort.Strings(names)
	for _, field := range names {
		if fields[field] == "" {
			continue
		}
		if err := write(ctx, owner, field, fields[field]); err != nil {
			c.Header("Retry-After", "5")
			c.JSON(http.StatusServiceUnavailable, gin.H{"code": "SECRET_WRITE_FAILED", "error": "Payment details were not fully saved. Retry the complete request."})
			return false
		}
	}
	return true
}
