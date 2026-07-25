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

	// CookieForHost resolves the request Host to the app-specific session
	// cookie name (e.g. productregistry.Registry.SessionCookieForHost). It is
	// optional: nil, or a host with no match, falls back to Mgr's default
	// cookie name — see Manager.ResolveCookieName. Deliberately typed as a
	// plain func rather than *productregistry.Registry so this package never
	// has to import the registry.
	CookieForHost CookieNameResolver
}

// cookieName resolves the session cookie name for the given request Host.
func (h *Handler) cookieName(host string) string {
	return h.Mgr.ResolveCookieName(h.CookieForHost, host)
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
	h.Mgr.Clear(c.Writer, h.cookieName(c.Request.Host))
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
	h.Mgr.SetCookie(c.Writer, h.cookieName(c.Request.Host), enc)
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
		Secure:   h.Mgr.cfg.Secure, // HTTPS-only in prod (env-driven, matches the session cookie); dev stays false
		SameSite: http.SameSiteStrictMode,
		// HttpOnly deliberately omitted — this is a double-submit CSRF token the client JS must read.
	})
	c.JSON(http.StatusOK, gin.H{"csrf_token": tok})
}

// read pulls the session payload from either the session cookie or
// an Authorization: Bearer header (mobile). The cookie lookup uses the
// per-Host cookie name (see cookieName); the Bearer fallback is unaffected
// by that resolution — mobile requests carry no cookie, so this simply
// falls through to the Authorization header regardless of what the Host
// resolves to.
func (h *Handler) read(c *gin.Context) (*Payload, error) {
	if ck, err := c.Request.Cookie(h.cookieName(c.Request.Host)); err == nil {
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
