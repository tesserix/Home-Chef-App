package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Mgr *Manager
}

func (h *Handler) Register(r gin.IRouter) {
	r.GET("/auth/session", h.session)
	r.POST("/auth/logout", h.logout)
	r.POST("/auth/refresh", h.refresh)
	r.GET("/auth/csrf", h.csrf)
}

func (h *Handler) session(c *gin.Context) {
	p, err := h.read(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"user_id":    p.UID,
		"email":      p.Email,
		"role":       p.Role,
		"pool":       p.Pool,
		"expires_at": p.ExpiresAt,
	})
}

func (h *Handler) logout(c *gin.Context) {
	h.Mgr.Clear(c.Writer)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) refresh(c *gin.Context) {
	p, err := h.read(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	p.ExpiresAt = time.Now().Add(h.Mgr.MaxAge()).Unix()
	enc, err := h.Mgr.Encode(p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encode_failed"})
		return
	}
	h.Mgr.SetCookie(c.Writer, enc)
	c.JSON(http.StatusOK, gin.H{"expires_at": p.ExpiresAt})
}

func (h *Handler) csrf(c *gin.Context) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	tok := hex.EncodeToString(b)
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "hc_csrf",
		Value:    tok,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
	c.JSON(http.StatusOK, gin.H{"csrf_token": tok})
}

// read pulls the session payload from either the session cookie or
// an Authorization: Bearer header (mobile).
func (h *Handler) read(c *gin.Context) (*Payload, error) {
	if ck, err := c.Request.Cookie(h.Mgr.CookieName()); err == nil {
		return h.Mgr.Decode(ck.Value)
	}
	if a := c.GetHeader("Authorization"); strings.HasPrefix(a, "Bearer ") {
		return h.Mgr.Decode(strings.TrimPrefix(a, "Bearer "))
	}
	return nil, errNoCredentials
}

var errNoCredentials = newErr("no credentials")

type sessErr string

func (s sessErr) Error() string { return string(s) }
func newErr(s string) error     { return sessErr(s) }
