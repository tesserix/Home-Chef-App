// Package handlers — support_chat.go: SupportChatHandler proxies the mobile
// support-chat routes (customer→Tesserix support) to the shared otto service
// in support-platform. The web storefront reaches otto through the homechef-web
// nginx proxy; mobile authenticates with a BFF bearer token, so this is the
// server-side equivalent — it resolves the verified customer identity from the
// JWT-hydrated user and pins the homechef tenant/store scope, letting otto skip
// the anonymous OTP step.
//
// otto's session cookie is HttpOnly, so a native app can't read it. On create,
// otto mints the cookie; this handler lifts the token out of the Set-Cookie
// header into the JSON body as "session_token" (and echoes it as an
// X-Otto-Session response header). The app stores it and sends it back as
// X-Otto-Session on every later call, which otto accepts in lieu of the cookie.
//
// Real-time delivery (WebSocket) bypasses this handler: the app dials otto
// directly via the fe3dr.com Istio route using the short-lived ticket minted by
// /ws-ticket. Only REST + ws-ticket flow through here.
package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/config"
	"github.com/homechef/api/middleware"
)

const (
	// Pinned per HomeChef — mirrors apps/web/nginx.conf so mobile-created
	// threads share the web tenant/store and surface in the platform inbox.
	ottoTenantID = "homechef"
	ottoStoreID  = "default"

	ottoStorefrontBase = "/api/v1/storefront/otto"
	ottoSessionCookie  = "otto_session"

	// Chat payloads are tiny; cap both the request and the relayed body so a
	// runaway upstream can't balloon API memory.
	ottoMaxBody = 1 << 20 // 1 MiB
	ottoTimeout = 15 * time.Second
)

// SupportChatHandler proxies /api/v1/support/chat/* to otto's storefront surface.
type SupportChatHandler struct {
	baseURL      string
	secret       string
	wsPublicBase string
	hc           *http.Client
}

// NewSupportChatHandler builds the handler from config. When OTTO_URL or
// OTTO_INTERNAL_AUTH is empty the returned handler is DARK: every route answers
// 503 (support_unavailable). This is the intended default until the feature is
// wired via the homechef-api chart env.
func NewSupportChatHandler() *SupportChatHandler {
	return newSupportChatHandler(
		config.AppConfig.OTTOURL,
		config.AppConfig.OTTOInternalAuth,
		config.AppConfig.OTTOWSPublicBase,
	)
}

// newSupportChatHandler is the testable constructor.
func newSupportChatHandler(baseURL, secret, wsPublicBase string) *SupportChatHandler {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	secret = strings.TrimSpace(secret)
	if baseURL == "" || secret == "" {
		return &SupportChatHandler{} // dark
	}
	return &SupportChatHandler{
		baseURL:      baseURL,
		secret:       secret,
		wsPublicBase: strings.TrimRight(strings.TrimSpace(wsPublicBase), "/"),
		hc:           &http.Client{Timeout: ottoTimeout},
	}
}

func (h *SupportChatHandler) enabled() bool {
	return h.baseURL != "" && h.secret != "" && h.hc != nil
}

// Register mounts the support-chat routes on g. The group MUST already have
// BFFAuth applied so the caller's identity is on the context. The explicit
// route list IS the path allowlist — no other otto subpath is reachable.
func (h *SupportChatHandler) Register(g *gin.RouterGroup) {
	g.POST("/conversations", h.proxy(http.MethodPost, staticPath("/conversations")))
	g.GET("/resume", h.proxy(http.MethodGet, staticPath("/resume")))
	g.GET("/conversations/:id", h.proxy(http.MethodGet, convPath("")))
	g.GET("/conversations/:id/messages", h.proxy(http.MethodGet, convPath("/messages")))
	g.POST("/conversations/:id/messages", h.proxy(http.MethodPost, convPath("/messages")))
	g.POST("/conversations/:id/close", h.proxy(http.MethodPost, convPath("/close")))
	g.GET("/conversations/:id/queue", h.proxy(http.MethodGet, convPath("/queue")))
	g.POST("/conversations/:id/feedback", h.proxy(http.MethodPost, convPath("/feedback")))
	g.POST("/conversations/:id/ws-ticket", h.wsTicket())
}

type pathFunc func(c *gin.Context) string

func staticPath(p string) pathFunc { return func(*gin.Context) string { return p } }

func convPath(suffix string) pathFunc {
	return func(c *gin.Context) string {
		return "/conversations/" + url.PathEscape(c.Param("id")) + suffix
	}
}

// ottoResponse relays otto's reply verbatim (minus 5xx bodies — see relay).
type ottoResponse struct {
	Status       int
	Body         []byte
	ContentType  string
	SessionToken string // lifted from a Set-Cookie header, non-empty on create
}

// identity resolves the verified caller otto attributes the thread to.
func (h *SupportChatHandler) identity(c *gin.Context) (userID, email, name string) {
	if u, ok := middleware.GetUser(c); ok && u != nil {
		userID = u.ID.String()
		email = u.Email
		name = strings.TrimSpace(u.FirstName + " " + u.LastName)
		return userID, email, name
	}
	// Fallback to the raw header-derived context values (BFFAuth seeds these).
	userID = c.GetString(middleware.CtxUserID)
	email = c.GetString(middleware.CtxUserEmail)
	return userID, email, name
}

// forward proxies one request to otto's storefront surface, injecting scope,
// identity, and the internal secret. Non-transport failures are relayed as data.
func (h *SupportChatHandler) forward(c *gin.Context, method, path string) (*ottoResponse, bool) {
	if !h.enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "support_unavailable"})
		return nil, false
	}

	userID, email, name := h.identity(c)
	if userID == "" {
		// The group is BFFAuth-protected, so this should be unreachable; deny
		// defensively rather than open an anonymous (OTP-required) otto thread.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return nil, false
	}

	var body []byte
	if method != http.MethodGet && c.Request.Body != nil {
		body, _ = io.ReadAll(io.LimitReader(c.Request.Body, ottoMaxBody))
	}

	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), method, h.baseURL+ottoStorefrontBase+path, rdr)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "support_unreachable"})
		return nil, false
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Internal-Auth", h.secret)
	req.Header.Set("X-Tenant-Id", ottoTenantID)
	req.Header.Set("X-Store-Id", ottoStoreID)
	req.Header.Set("X-User-Id", userID)
	if email != "" {
		req.Header.Set("X-User-Email", email)
	}
	if name != "" {
		req.Header.Set("X-User-Name", name)
	}
	if sess := c.GetHeader("X-Otto-Session"); sess != "" {
		req.Header.Set("X-Otto-Session", sess)
	}

	resp, err := h.hc.Do(req)
	if err != nil {
		log.Printf("support_chat: otto forward failed path=%s err=%v", path, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "support_unreachable"})
		return nil, false
	}
	defer resp.Body.Close()

	rbody, _ := io.ReadAll(io.LimitReader(resp.Body, ottoMaxBody))
	out := &ottoResponse{
		Status:      resp.StatusCode,
		Body:        rbody,
		ContentType: resp.Header.Get("Content-Type"),
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == ottoSessionCookie && ck.Value != "" {
			out.SessionToken = ck.Value
			break
		}
	}
	return out, true
}

func (h *SupportChatHandler) proxy(method string, path pathFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, ok := h.forward(c, method, path(c))
		if !ok {
			return
		}
		h.relay(c, res)
	}
}

// wsTicket mints an otto WebSocket ticket and augments the response with the
// public ws_url the app dials directly (real-time bypasses this proxy).
func (h *SupportChatHandler) wsTicket() gin.HandlerFunc {
	return func(c *gin.Context) {
		res, ok := h.forward(c, http.MethodPost, convPath("/ws-ticket")(c))
		if !ok {
			return
		}
		if res.Status != http.StatusOK {
			h.relay(c, res)
			return
		}
		var minted struct {
			Ticket string `json:"ticket"`
		}
		if err := json.Unmarshal(res.Body, &minted); err != nil || minted.Ticket == "" {
			c.JSON(http.StatusBadGateway, gin.H{"error": "ticket_mint_failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"ticket": minted.Ticket,
			"ws_url": h.wsURL(c, c.Param("id")),
		})
	}
}

// relay writes otto's response back, suppressing 5xx bodies (never leak otto
// internals) and surfacing a freshly minted session token into the body + header.
func (h *SupportChatHandler) relay(c *gin.Context, res *ottoResponse) {
	if res.Status >= 500 {
		c.JSON(http.StatusBadGateway, gin.H{"error": "support_unreachable"})
		return
	}
	ct := res.ContentType
	if ct == "" {
		ct = "application/json"
	}
	body := res.Body
	if res.SessionToken != "" {
		if merged, ok := mergeSessionToken(res.Body, res.SessionToken); ok {
			body = merged
		}
		c.Header("X-Otto-Session", res.SessionToken)
	}
	c.Data(res.Status, ct, body)
}

// wsURL builds the public WebSocket URL, preferring the configured base and
// otherwise deriving it from the forwarded host.
func (h *SupportChatHandler) wsURL(c *gin.Context, convID string) string {
	base := h.wsPublicBase
	if base == "" {
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}
		scheme := "wss"
		if c.GetHeader("X-Forwarded-Proto") == "http" {
			scheme = "ws"
		}
		base = scheme + "://" + host
	}
	return base + ottoStorefrontBase + "/conversations/" + url.PathEscape(convID) + "/ws"
}

// mergeSessionToken injects a top-level "session_token" into a JSON object body.
// Returns false when the body isn't a JSON object.
func mergeSessionToken(body []byte, token string) ([]byte, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, false
	}
	tok, err := json.Marshal(token)
	if err != nil {
		return nil, false
	}
	obj["session_token"] = tok
	out, err := json.Marshal(obj)
	if err != nil {
		return nil, false
	}
	return out, true
}
