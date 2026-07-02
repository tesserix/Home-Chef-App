package autologin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{ d *Deps }

func NewHandler(d *Deps) *Handler { return &Handler{d: d} }

func (h *Handler) Register(r gin.IRouter) {
	r.POST("/auth/auto-login", h.post)
}

func (h *Handler) post(c *gin.Context) {
	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_body"})
		return
	}
	resp, err := h.d.AutoLogin(c.Request.Context(), req)
	switch {
	case errors.Is(err, ErrTenantNotAllowed):
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant_not_allowed"})
	case errors.Is(err, ErrEmailNotAllowed):
		c.JSON(http.StatusForbidden, gin.H{"error": "email_not_allowed"})
	case errors.Is(err, ErrTokenInvalid):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
	case err != nil:
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream_error"})
	default:
		c.JSON(http.StatusOK, resp)
	}
}

// PostHandler returns the raw HandlerFunc for callers (e.g. main.go) that
// want to mount the endpoint on a custom Gin group (rate-limited, etc.)
// rather than via Register's default routing.
func (h *Handler) PostHandler() gin.HandlerFunc { return h.post }
