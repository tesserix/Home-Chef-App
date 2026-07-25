# Phase 5 — HomeChef mobile customers chat with Tesserix support (otto)

**Date:** 2026-07-26
**Spec:** `slm-support-platform/docs/superpowers/specs/2026-07-25-otto-platform-inbox-design.md` §"Phase 5"
**Repos touched:** `Home-Chef-App` (apps/api + apps/mobile-customer + packages/mobile-shared), `tesserix-k8s` (homechef-api chart)
**Status:** Ready to execute

---

## Goal

Give HomeChef `mobile-customer` users a dedicated "Chat with support" screen (no floating
widget) that opens a conversation in otto's `homechef` tenant, so it lands in the platform
inbox where Tesserix admins answer (Phases 1-3, already shipped). Mirror mark8ly's proven
mobile support kit.

Three deliverables:
- **A.** A small authenticated proxy group in the HomeChef Go API (`/api/v1/support/chat/*`)
  that pins `X-Tenant-Id: homechef` + `X-Store-Id: default`, injects `X-Internal-Auth`, and
  forwards the JWT-verified user identity to otto's storefront surface (skips OTP).
- **B.** A ported RN support-chat kit in `packages/mobile-shared/src/support/` + a
  `mobile-customer` screen reached from the Profile hub.
- **C.** homechef-api chart env wiring (`OTTO_URL` + `OTTO_INTERNAL_AUTH` + `OTTO_WS_PUBLIC_BASE`).

---

## Key discoveries (verified during planning — do not re-derive)

1. **WS already routed — no Istio change needed.** `tesserix-k8s/manifests/homechef-istio/virtualservice.yaml`
   (the `homechef-web` VirtualService for `fe3dr.com` / `www.fe3dr.com`) **already** carries the
   storefront-otto WS route (lines 96-108):
   `regex: ^/api/v1/storefront/otto/conversations/[^/]+/ws$` → `support-platform-otto.support-platform.svc.cluster.local:8089`,
   `timeout: 3600s`, `Host` header rewritten to the otto service. So a mobile WS to
   `wss://fe3dr.com/api/v1/storefront/otto/conversations/:id/ws?ticket=…` works with existing
   infra. **Deliverable C needs NO VirtualService change.**
2. **WS auth = ticket, not session.** otto `storefront_handler.go` `websocket()` reads only
   `?ticket=`, validates `audience==customer`, `ConversationID` match, and that the ticket's
   session token still owns the conversation. The WS group runs with **no** CustomerContext
   middleware, so the direct-to-otto WS needs **no** `X-Internal-Auth`/`X-Tenant-Id` headers on
   the upgrade — the ticket carries tenant/store/session scope. **Ship WS→poll v1** (SSE layer
   skipped; polling is the guaranteed fallback so B is shippable even if WS ever misbehaves).
3. **Public host is `fe3dr.com`, not `api.fe3dr.com`.** The mobile REST base is
   `EXPO_PUBLIC_API_URL = https://fe3dr.com/api` (prod, `apps/mobile-customer/eas.json`), and the
   web widget builds its WS on `window.location.host` = `fe3dr.com`. So `OTTO_WS_PUBLIC_BASE = wss://fe3dr.com`.
4. **otto scope contract = `homechef`/`default`.** `apps/web/nginx.conf` (lines 49-61) pins
   `X-Tenant-Id: homechef`, `X-Store-Id: default`, `X-Internal-Auth: ${OTTO_INTERNAL_AUTH}`,
   upstream `${OTTO_URL}/api/v1/storefront/otto/*`. The proxy MUST pin the same store id so
   mobile-created threads share the web tenant/store and surface in the platform inbox.
5. **Session token is cookieless via body/header.** otto mints an HttpOnly `otto_session` cookie
   on create; the RN app can't read it, so the proxy lifts it out of `Set-Cookie` into the JSON
   body as `session_token` **and** an `X-Otto-Session` response header. The client stores it and
   echoes `X-Otto-Session` on later calls (otto's `RequireCustomerSession` accepts the header in
   lieu of the cookie). Verified against mark8ly `ottobridge/bridge.go` + `support_test.go`.
6. **API identity headers derived from the JWT-hydrated user.** HomeChef routes run behind
   `middleware.BFFAuth` (mobile uses the `Authorization: Bearer` fallback validated against
   auth-bff `/auth/session`). Handlers read `middleware.GetUser(c) → *models.User`
   (`ID uuid.UUID`, `Email`, `FirstName`, `LastName`). Map: `X-User-Id = user.ID.String()`,
   `X-User-Email = user.Email`, `X-User-Name = FirstName+" "+LastName`. Non-empty `X-User-Id`
   makes otto skip OTP.
7. **Reasons match otto's homechef whitelist.** otto `internal/conversation/model.go` `TenantReasons["homechef"]`
   = `{order_tracking, delivery_issue, refund, chef_question, account_issue, other, general_question}`,
   `general_question` in `NoStatus`, `NeedsDOB` empty. Reuse the exact set from
   `apps/web/src/features/support/OttoChat.tsx` `HOMECHEF_REASONS`. No DOB field ever shown.
8. **Zero new npm deps.** The mark8ly kit imports only `zod` (v4.4.3 present — verified
   `.passthrough()`/`.default()`/`.optional().default()` work at runtime), `expo-secure-store`
   (present), `react`, `react-native` (present). The `homechef-otto` ExternalSecret already exists
   (`external-secrets/prod/homechef/externalsecret.yaml`, secret `homechef-otto` key
   `INTERNAL_AUTH_SECRET` ← GCP SM `prod-support-platform-otto-internal-auth`) and is already
   consumed by `homechef-web`. No new secret, no lockfile change.
9. **Shared package + subpath export.** Kit lands in the existing workspace package
   `@homechef/mobile-shared` (`packages/mobile-shared`). Its `package.json` `exports` map is
   explicit (no wildcard) and Metro runs with `disableHierarchicalLookup: true`, so a new
   `"./support"` export entry is **required** for runtime resolution (TS resolves via the
   tsconfig `@homechef/mobile-shared/*` wildcard either way). Adding an `exports` subpath is not
   an npm dependency and needs no lockfile regen.

---

## Global constraints

- **Branch (Home-Chef-App):** `feat/otto-support-chat` off latest `origin/main`. Fetch + merge
  `origin/main` at start AND re-check before each commit (concurrent sessions are possible).
- **tesserix-k8s:** work on `main`, **do NOT push** (leave commits/instructions for the user to
  push; ArgoCD deploys). The user applies infra.
- **Git identity (both repos, before any commit):**
  `git config user.name "sam123ben"` / `git config user.email "samyak.rout@gmail.com"`.
- **No AI references** anywhere (commits, PR, code, comments).
- **NEVER create SQL files** in Home-Chef-App (this feature touches no DB).
- **ZERO new npm dependencies** — adding one is out of scope (lockfile regen is blocked locally).
- **Concurrent-session safety:** before every commit run `git fetch origin && git status` and
  `git log --oneline -1 origin/main`; rebase/merge if `origin/main` advanced; re-run verifies.
- **Ship path:** mobile ships in the next **batched** TestFlight/EAS build (commit+push each fix
  to the branch → PR → merge to main; do NOT cut a build here). API ships via push→CI→Kargo/ArgoCD.
- **Verify (must pass before "done"):**
  - Go: `cd apps/api && go build ./... && go vet ./... && go test ./...`
    (fast targeted: `go test ./handlers/ -run SupportChat -v`).
  - TS app: `cd <repo root> && npx tsc --noEmit -p apps/mobile-customer/tsconfig.json`.
  - TS kit (pure-logic tests): `cd packages/mobile-shared && npx vitest run src/__tests__/support`.
  - Infra: `cd tesserix-k8s && helm template charts/apps/homechef-api -f charts/apps/homechef-api/values.yaml -f charts/apps/homechef-api/values-prod.yaml | grep -A3 OTTO`.
- **Build policy:** Claude uses language tooling only (`go build/vet/test`, `tsc`, `vitest`,
  `helm template`). NEVER run container builds / image pushes / deploys / `kubectl apply`.

---

## Task 1 — HomeChef Go API otto proxy (deliverable A)

Files:
- **new** `apps/api/handlers/support_chat.go`
- **new** `apps/api/handlers/support_chat_test.go`
- **edit** `apps/api/config/config.go` (3 fields + 3 env reads)
- **edit** `apps/api/routes/routes.go` (instantiate + register group)
- **edit** `apps/api/.env.example` (document the 3 vars)

### 1.1 `apps/api/config/config.go`

Add three fields to the `Config` struct (place after the `MongoDBName` block, before
`// NATS`):

```go
	// Otto support-chat (Phase 5) — the mobile support-chat proxy
	// (handlers/support_chat.go) forwards to the shared otto storefront surface.
	// OTTOURL is otto's in-cluster address; OTTOInternalAuth is the shared
	// X-Internal-Auth secret. BOTH empty ⇒ the proxy returns 503 (feature dark).
	// OTTOWSPublicBase is the public WebSocket origin the mobile app dials
	// directly (e.g. "wss://fe3dr.com"); empty ⇒ derived per-request from the
	// forwarded host.
	OTTOURL          string
	OTTOInternalAuth string
	OTTOWSPublicBase string
```

Add to the `AppConfig = &Config{ ... }` literal (place near the `MongoURI`/`MongoDBName`
lines):

```go
		// Otto support-chat (Phase 5)
		OTTOURL:          getEnv("OTTO_URL", ""),
		OTTOInternalAuth: getEnv("OTTO_INTERNAL_AUTH", ""),
		OTTOWSPublicBase: getEnv("OTTO_WS_PUBLIC_BASE", ""),
```

### 1.2 `apps/api/handlers/support_chat.go` (new — complete file)

```go
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
```

### 1.3 `apps/api/routes/routes.go`

**(a)** Instantiate alongside the other handlers (near line 196, next to
`supportHandler := handlers.NewSupportHandler()`):

```go
	supportChatHandler := handlers.NewSupportChatHandler()
```

**(b)** Register a new group right after the existing `support := v1.Group("/support")` block
(the ticket routes, ~line 728). Add:

```go
		// Otto support-chat proxy (Phase 5) — customer↔Tesserix support via the
		// shared otto service. Distinct from the /support/tickets group above:
		// this is a live-chat surface, forwarded to otto's storefront API. The
		// explicit route list inside Register IS the path allowlist. A per-user
		// limiter keeps the write paths (create/reply/ws-ticket) in check; the
		// poll GETs (resume/messages ~every 5s) sit well under it.
		supportChat := v1.Group("/support/chat")
		supportChat.Use(bffAuth(bffKey, bffWindow), middleware.RateLimitByUser(2, 5))
		supportChatHandler.Register(supportChat)
```

> Note: `/support/tickets/*` and `/support/chat/*` do not collide in gin's route tree
> (no shared wildcard segment). `RateLimitByUser(2, 5)` = 2 rps sustained / 5 burst per user.

### 1.4 `apps/api/.env.example`

Append after the feature-flags block:

```bash
# Otto support-chat (Phase 5). The mobile support-chat proxy forwards to the
# shared otto service in support-platform. BOTH empty ⇒ /api/v1/support/chat/*
# returns 503 (feature dark). In prod these come from the homechef-api chart:
# OTTO_URL from values, OTTO_INTERNAL_AUTH from the homechef-otto ExternalSecret.
OTTO_URL=
OTTO_INTERNAL_AUTH=
# Public WebSocket origin the mobile app dials directly (bypasses this proxy).
# Empty ⇒ derived from the forwarded host. Prod: wss://fe3dr.com
OTTO_WS_PUBLIC_BASE=
```

### 1.5 `apps/api/handlers/support_chat_test.go` (new — complete file)

```go
package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

// newSupportChatRig wires a gin engine whose routes are guarded by a stub
// middleware that mimics BFFAuth (sets the hydrated *models.User on the
// context), pointed at a fake otto server. Returns the engine + the fake's
// closer.
func newSupportChatRig(t *testing.T, base, secret, wsBase string, otto http.HandlerFunc) (*gin.Engine, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	var srv *httptest.Server
	if otto != nil {
		srv = httptest.NewServer(otto)
		if base == "" {
			base = srv.URL
		}
	}

	h := newSupportChatHandler(base, secret, wsBase)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user", &models.User{
			ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Email:     "buyer@example.com",
			FirstName: "Sam",
			LastName:  "Rout",
		})
		c.Next()
	})
	h.Register(r.Group("/support/chat"))

	closer := func() {
		if srv != nil {
			srv.Close()
		}
	}
	return r, closer
}

func TestSupportChat_Create_ForwardsScopeIdentityAndMergesSessionToken(t *testing.T) {
	var sawTenant, sawStore, sawUser, sawEmail, sawName, sawBody string
	r, closeFn := newSupportChatRig(t, "", "secret", "wss://api.test", func(w http.ResponseWriter, req *http.Request) {
		sawTenant = req.Header.Get("X-Tenant-Id")
		sawStore = req.Header.Get("X-Store-Id")
		sawUser = req.Header.Get("X-User-Id")
		sawEmail = req.Header.Get("X-User-Email")
		sawName = req.Header.Get("X-User-Name")
		b, _ := io.ReadAll(req.Body)
		sawBody = string(b)
		if !strings.HasSuffix(req.URL.Path, "/api/v1/storefront/otto/conversations") {
			t.Errorf("otto path = %q", req.URL.Path)
		}
		http.SetCookie(w, &http.Cookie{Name: "otto_session", Value: "sess-xyz", Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"conversation":{"id":"c1","case_id":"CS-1"},"first_message":{"id":"m1"}}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	body := `{"message":"hi","reason":"order_tracking","status_info":"late"}`
	req, _ := http.NewRequest(http.MethodPost, "/support/chat/conversations", strings.NewReader(body))
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if sawTenant != "homechef" || sawStore != "default" {
		t.Errorf("scope: tenant=%q store=%q", sawTenant, sawStore)
	}
	if sawUser != "11111111-1111-1111-1111-111111111111" {
		t.Errorf("X-User-Id = %q (skips OTP)", sawUser)
	}
	if sawEmail != "buyer@example.com" || sawName != "Sam Rout" {
		t.Errorf("identity: email=%q name=%q", sawEmail, sawName)
	}
	if !strings.Contains(sawBody, `"reason":"order_tracking"`) {
		t.Errorf("body not forwarded verbatim: %q", sawBody)
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["session_token"] != "sess-xyz" {
		t.Errorf("session_token = %v, want sess-xyz", out["session_token"])
	}
	if _, ok := out["conversation"]; !ok {
		t.Errorf("conversation missing from relayed body")
	}
	if rec.Header().Get("X-Otto-Session") != "sess-xyz" {
		t.Errorf("X-Otto-Session header not set")
	}
}

func TestSupportChat_PostMessage_ForwardsSessionHeaderAndId(t *testing.T) {
	var sawSession, sawPath string
	r, closeFn := newSupportChatRig(t, "", "secret", "", func(w http.ResponseWriter, req *http.Request) {
		sawSession = req.Header.Get("X-Otto-Session")
		sawPath = req.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":{"id":"m2"}}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/support/chat/conversations/c1/messages", strings.NewReader(`{"body":"hello"}`))
	req.Header.Set("X-Otto-Session", "sess-xyz")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if sawSession != "sess-xyz" {
		t.Errorf("X-Otto-Session forwarded = %q", sawSession)
	}
	if !strings.HasSuffix(sawPath, "/api/v1/storefront/otto/conversations/c1/messages") {
		t.Errorf("path = %q", sawPath)
	}
}

func TestSupportChat_WsTicket_AugmentsWithWsURL(t *testing.T) {
	r, closeFn := newSupportChatRig(t, "", "secret", "wss://fe3dr.com", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ticket":"tkt-123"}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/support/chat/conversations/c1/ws-ticket", nil)
	req.Header.Set("X-Otto-Session", "sess-xyz")
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var out struct {
		Ticket string `json:"ticket"`
		WsURL  string `json:"ws_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Ticket != "tkt-123" {
		t.Errorf("ticket = %q", out.Ticket)
	}
	want := "wss://fe3dr.com/api/v1/storefront/otto/conversations/c1/ws"
	if out.WsURL != want {
		t.Errorf("ws_url = %q, want %q", out.WsURL, want)
	}
}

func TestSupportChat_5xxSuppressed(t *testing.T) {
	r, closeFn := newSupportChatRig(t, "", "secret", "", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"otto_internal_secret_leak"}`))
	})
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/support/chat/resume", nil)
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "otto_internal_secret_leak") {
		t.Errorf("5xx body leaked: %s", rec.Body.String())
	}
}

func TestSupportChat_Dark_Returns503(t *testing.T) {
	// Empty base/secret ⇒ dark handler; no fake otto needed.
	r, closeFn := newSupportChatRig(t, "", "", "", nil)
	defer closeFn()

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/support/chat/resume", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}
```

> Verify the `models.User` field set (`ID`, `Email`, `FirstName`, `LastName`) matches
> `apps/api/models/user.go` — confirmed during planning. `uuid` is already a direct dep.

### Task 1 verify

```bash
cd apps/api
go build ./...        # expect: no output (success)
go vet ./...          # expect: no output
go test ./handlers/ -run SupportChat -v   # expect: 6 PASS lines, ok
go test ./...         # expect: ok across packages (no new failures)
```

### Task 1 commit

```bash
cd <Home-Chef-App root>
git fetch origin && git status && git log --oneline -1 origin/main   # concurrency check
git add apps/api/handlers/support_chat.go apps/api/handlers/support_chat_test.go \
        apps/api/config/config.go apps/api/routes/routes.go apps/api/.env.example
git commit -m "feat(support): proxy mobile support-chat to otto storefront (homechef tenant)"
```

---

## Task 2 — RN support-chat kit port (deliverable B, shared package)

Port only what's needed from `mark8ly/packages/mobile-shared/support/*`, adapted to
`@homechef/mobile-shared`. **SSE layer skipped** (WS→poll only). Zero new deps.

New files under `packages/mobile-shared/src/support/`:
`types.ts`, `events.ts`, `outbox.ts`, `storage.ts`, `client.ts`, `useSupportChat.ts`,
`SupportChatView.tsx`, `index.ts`.
Plus a pure-logic test: `packages/mobile-shared/src/__tests__/support/support-logic.test.ts`.
Plus one edit: `packages/mobile-shared/package.json` (`exports` subpath).

### 2.1 `packages/mobile-shared/package.json` — add subpath export

In the `"exports"` object, add (keeps alphabetical-ish order, after `"./screens"`):

```json
    "./support": "./src/support/index.ts",
```

(No dependency change → no lockfile regen. `zod`, `expo-secure-store`, `react`,
`react-native` are already present/peer.)

### 2.2 `src/support/types.ts` (verbatim port — complete file)

```ts
// Shared support-chat types + Zod schemas for the mobile support kit. Shapes
// mirror otto's storefront responses relayed by the HomeChef API proxy;
// schemas are lenient (passthrough) on the conversation so a future otto field
// addition doesn't break parsing.
import { z } from "zod";

export const SupportSenderTypeSchema = z.enum(["customer", "staff", "system"]);
export type SupportSenderType = z.infer<typeof SupportSenderTypeSchema>;

export const SupportStatusSchema = z.enum(["pending", "active", "closed"]);
export type SupportStatus = z.infer<typeof SupportStatusSchema>;

export const SupportMessageSchema = z.object({
  id: z.string(),
  conversation_id: z.string().optional(),
  sender_type: SupportSenderTypeSchema,
  sender_name: z.string().optional().default(""),
  body: z.string(),
  created_at: z.string(),
});
export type SupportMessage = z.infer<typeof SupportMessageSchema>;

export const SupportConversationSchema = z
  .object({
    id: z.string(),
    case_id: z.string().optional().default(""),
    status: SupportStatusSchema,
    subject: z.string().optional().default(""),
    created_at: z.string().optional(),
    closed_at: z.string().optional(),
  })
  .passthrough();
export type SupportConversation = z.infer<typeof SupportConversationSchema>;

export const QueueStateSchema = z.object({
  status: SupportStatusSchema,
  position: z.number().default(0),
  total_pending: z.number().default(0),
  estimated_wait_seconds: z.number().default(0),
  all_busy: z.boolean().default(false),
});
export type QueueState = z.infer<typeof QueueStateSchema>;

export const WsTicketSchema = z.object({
  ticket: z.string(),
  ws_url: z.string(),
});
export type WsTicket = z.infer<typeof WsTicketSchema>;

/** An intake reason offered before the first message is sent. */
export interface IntakeReason {
  value: string;
  label: string;
  /** When false, the free-text "what's going on" field is optional. */
  requiresStatus?: boolean;
}

/** Input for opening a new support conversation. */
export interface CreateConversationInput {
  message: string;
  reason: string;
  statusInfo: string;
  subject?: string;
  name?: string;
  email?: string;
  /** Required by otto only for order/account reasons — never used by HomeChef. */
  dob?: string;
}

/** Post-case survey payload. */
export interface FeedbackInput {
  call_rating: number;
  query_resolved: boolean;
  staff_rating: number;
  comments?: string;
}

/** Maps the friendly input onto otto's create wire shape. */
export function toCreateBody(input: CreateConversationInput): Record<string, unknown> {
  return {
    message: input.message,
    reason: input.reason,
    status_info: input.statusInfo,
    subject: input.subject ?? "",
    name: input.name ?? "",
    email: input.email ?? "",
    dob: input.dob ?? "",
  };
}
```

### 2.3 `src/support/events.ts` (verbatim port — complete file)

```ts
// WebSocket event parsing + the pure message-merge reducer. Kept free of React
// Native imports so it is unit-testable under node; the RN socket lifecycle
// lives in useSupportChat.
import { SupportMessageSchema, type SupportMessage } from "./types";

// OttoEvent is the normalised form of an otto WebSocket envelope
// ({type, room, payload}). Unknown/garbage frames collapse to "unknown".
export type OttoEvent =
  | { kind: "message"; message: SupportMessage }
  | { kind: "conversation_updated"; conversationId?: string }
  | { kind: "conversation_closed" }
  | { kind: "unknown" };

const UNKNOWN: OttoEvent = { kind: "unknown" };

/** Parses a raw WebSocket frame into a typed OttoEvent. */
export function parseOttoEvent(raw: string): OttoEvent {
  let env: unknown;
  try {
    env = JSON.parse(raw);
  } catch {
    return UNKNOWN;
  }
  if (!env || typeof env !== "object") return UNKNOWN;
  const e = env as { type?: unknown; payload?: unknown };
  const payload = (e.payload ?? {}) as Record<string, unknown>;

  switch (e.type) {
    case "otto.message.created": {
      const parsed = SupportMessageSchema.safeParse(payload.message);
      return parsed.success ? { kind: "message", message: parsed.data } : UNKNOWN;
    }
    case "otto.conversation.closed":
      return { kind: "conversation_closed" };
    case "otto.conversation.updated": {
      const conv = payload.conversation as { id?: string } | undefined;
      const conversationId =
        (typeof payload.conversation_id === "string" ? payload.conversation_id : undefined) ??
        conv?.id;
      return { kind: "conversation_updated", conversationId };
    }
    default:
      return UNKNOWN;
  }
}

/**
 * Appends a message, de-duplicating by id (the sender sees their own message
 * optimistically AND echoed back over the socket) and keeping the list ordered
 * by created_at.
 */
export function mergeMessage(
  messages: SupportMessage[],
  incoming: SupportMessage,
): SupportMessage[] {
  if (messages.some((m) => m.id === incoming.id)) return messages;
  const next = [...messages, incoming];
  next.sort((a, b) => (a.created_at < b.created_at ? -1 : a.created_at > b.created_at ? 1 : 0));
  return next;
}

/** Merges a batch (initial history / resume) into existing messages. */
export function mergeMessages(
  messages: SupportMessage[],
  batch: SupportMessage[],
): SupportMessage[] {
  return batch.reduce(mergeMessage, messages);
}
```

### 2.4 `src/support/outbox.ts` (verbatim port — complete file)

```ts
// Durable send-outbox for the support chat. Guarantees a customer's message is
// never lost mid-way: persisted to device storage the instant the user hits
// send, shown optimistically, re-sent with backoff until otto confirms it —
// surviving network drops, backgrounding, and cold starts.
//
// Delivery is at-least-once (otto has no client idempotency key), so we retry
// only on network/5xx errors, never on a 4xx — duplicates are rare, a lost
// message impossible. Pure logic here is unit-tested; the RN wiring lives in
// useSupportChat.
import type { SupportMessage } from "./types";

// KVStorage is the minimal async key/value contract (expo-secure-store
// satisfies it). Injected by the app so this module stays platform-agnostic.
export interface KVStorage {
  getItem(key: string): Promise<string | null>;
  setItem(key: string, value: string): Promise<void>;
  removeItem(key: string): Promise<void>;
}

export type OutboxStatus = "queued" | "sending" | "failed";

export interface OutboxItem {
  clientMsgId: string;
  conversationId: string;
  body: string;
  createdAt: string;
  attempts: number;
  status: OutboxStatus;
}

const KEY_PREFIX = "otto_outbox:";

export function outboxKey(conversationId: string): string {
  return `${KEY_PREFIX}${conversationId}`;
}

/** Capped exponential backoff for re-send attempts. */
export function backoffMs(attempts: number): number {
  return Math.min(30_000, 1_000 * 2 ** Math.max(0, attempts - 1));
}

/** Renders a queued/sending/failed outbox item as an optimistic message. */
export function outboxItemToMessage(item: OutboxItem): SupportMessage & { pending: true; failed: boolean } {
  return {
    id: item.clientMsgId,
    conversation_id: item.conversationId,
    sender_type: "customer",
    sender_name: "",
    body: item.body,
    created_at: item.createdAt,
    pending: true,
    failed: item.status === "failed",
  };
}

export function addItem(items: OutboxItem[], item: OutboxItem): OutboxItem[] {
  if (items.some((i) => i.clientMsgId === item.clientMsgId)) return items;
  return [...items, item];
}

export function removeItem(items: OutboxItem[], clientMsgId: string): OutboxItem[] {
  return items.filter((i) => i.clientMsgId !== clientMsgId);
}

export function markItem(items: OutboxItem[], clientMsgId: string, patch: Partial<OutboxItem>): OutboxItem[] {
  return items.map((i) => (i.clientMsgId === clientMsgId ? { ...i, ...patch } : i));
}

export async function loadOutbox(storage: KVStorage, conversationId: string): Promise<OutboxItem[]> {
  try {
    const raw = await storage.getItem(outboxKey(conversationId));
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as OutboxItem[]) : [];
  } catch {
    return [];
  }
}

export async function saveOutbox(storage: KVStorage, conversationId: string, items: OutboxItem[]): Promise<void> {
  if (items.length === 0) {
    await storage.removeItem(outboxKey(conversationId));
    return;
  }
  await storage.setItem(outboxKey(conversationId), JSON.stringify(items));
}

/** Whether a failed send should be retried. 4xx (except 429) is terminal. */
export function isRetryable(status: number | null): boolean {
  if (status === null) return true; // transport error → retry
  if (status === 429) return true;
  return status < 400 || status >= 500;
}
```

### 2.5 `src/support/storage.ts` (verbatim port — complete file)

```ts
// secureStoreKV — a KVStorage adapter over expo-secure-store, used to persist
// the support send-outbox so queued messages survive a cold start. SecureStore
// keys allow only [A-Za-z0-9._-], so keys are sanitised at the boundary (the
// outbox uses a ":" separator that's valid for MMKV/AsyncStorage but not
// SecureStore).
//
// Note: SecureStore caps values at ~2KB on iOS. A typical outbox (a few short
// queued messages) is well under that.
import * as SecureStore from "expo-secure-store";

import type { KVStorage } from "./outbox";

const safeKey = (k: string): string => k.replace(/[^A-Za-z0-9._-]/g, "_");

export const secureStoreKV: KVStorage = {
  getItem: (key) => SecureStore.getItemAsync(safeKey(key)),
  setItem: (key, value) => SecureStore.setItemAsync(safeKey(key), value),
  removeItem: (key) => SecureStore.deleteItemAsync(safeKey(key)),
};
```

### 2.6 `src/support/client.ts` (adapted — SSE builder removed; complete file)

Only change vs mark8ly: `buildSseUrl` is dropped (SSE is not part of v1). Everything
else — bearer auth with single-flight refresh, `X-Otto-Session` carry, `session_token`
capture, envelope unwrapping — is identical.

```ts
// createSupportClient — the shared REST client for the mobile support proxy.
// Constructed with a base URL + basePath + token getters; the only
// support-specific behaviour lives here, once:
//   - sends the bearer session token (single-flight refresh on 401)
//   - carries the opaque otto session as X-Otto-Session on every call and
//     captures the fresh token the proxy returns in the create body (the app
//     can't read otto's HttpOnly cookie), optionally persisting it
//   - unwraps otto's { conversation } / { messages } / { message } envelopes
import { z } from "zod";

import {
  CreateConversationInput,
  FeedbackInput,
  QueueStateSchema,
  SupportConversationSchema,
  SupportMessageSchema,
  WsTicketSchema,
  toCreateBody,
  type QueueState,
  type SupportConversation,
  type SupportMessage,
  type WsTicket,
} from "./types";

export class SupportError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message);
    this.name = "SupportError";
  }
}

export interface SupportClientConfig {
  /** API origin, e.g. "https://fe3dr.com/api". */
  baseUrl: string;
  /** Path prefix for this surface, e.g. "/v1/support/chat". */
  basePath: string;
  /** Returns the cached bearer session token, or null when signed out. */
  getToken: () => Promise<string | null>;
  /** Mints a fresh bearer token; called once on a 401 before giving up. */
  refreshToken?: () => Promise<string | null>;
  /** Called when a (refreshed) token is still rejected with 401. */
  onUnauthorized?: () => void | Promise<void>;
  /** Loads a persisted otto session token on first use (cross-launch resume). */
  loadSessionToken?: () => Promise<string | null> | string | null;
  /** Persists the otto session token when otto mints a new one. */
  saveSessionToken?: (token: string) => Promise<void> | void;
}

export interface ResumeResult {
  conversation: SupportConversation;
  messages: SupportMessage[];
}

export function createSupportClient(config: SupportClientConfig) {
  let sessionToken: string | null = null;
  let sessionLoaded = false;

  async function loadSession(): Promise<string | null> {
    if (!sessionLoaded) {
      sessionLoaded = true;
      if (config.loadSessionToken) {
        sessionToken = (await config.loadSessionToken()) ?? null;
      }
    }
    return sessionToken;
  }

  async function rememberSession(token: string): Promise<void> {
    sessionToken = token;
    if (config.saveSessionToken) await config.saveSessionToken(token);
  }

  // Single-flight token refresh — parallel 401s share one refresh call.
  let inflightRefresh: Promise<string | null> | null = null;
  function refresh(): Promise<string | null> {
    if (!config.refreshToken) return Promise.resolve(null);
    if (!inflightRefresh) {
      inflightRefresh = config.refreshToken().finally(() => {
        inflightRefresh = null;
      });
    }
    return inflightRefresh;
  }

  async function send(token: string | null, method: string, path: string, body?: unknown): Promise<Response> {
    const session = await loadSession();
    const headers: Record<string, string> = { Accept: "application/json" };
    if (token) headers["Authorization"] = `Bearer ${token}`;
    if (session) headers["X-Otto-Session"] = session;
    let payload: string | undefined;
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      payload = JSON.stringify(body);
    }
    return fetch(`${config.baseUrl}${config.basePath}${path}`, { method, headers, body: payload });
  }

  async function request(method: string, path: string, body?: unknown): Promise<any> {
    let token = await config.getToken();
    let res = await send(token, method, path, body);

    if (res.status === 401 && token) {
      const refreshed = await refresh();
      if (refreshed) {
        token = refreshed;
        res = await send(refreshed, method, path, body);
      }
      if (res.status === 401) {
        await config.onUnauthorized?.();
        throw new SupportError(401, "unauthorized", "Session expired");
      }
    }

    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: "error", message: res.statusText }));
      throw new SupportError(res.status, err.error ?? "error", err.message ?? res.statusText);
    }

    if (res.status === 204) return undefined;
    const data = await res.json();
    // otto mints the session cookie on create; the proxy surfaces it in the
    // body as session_token. Capture + persist it for later calls.
    if (data && typeof data === "object" && typeof data.session_token === "string") {
      await rememberSession(data.session_token);
    }
    return data;
  }

  return {
    /** Opens a new conversation and returns it plus the first message. */
    createConversation: async (
      input: CreateConversationInput,
    ): Promise<{ conversation: SupportConversation; firstMessage: SupportMessage }> => {
      const data = await request("POST", "/conversations", toCreateBody(input));
      return {
        conversation: SupportConversationSchema.parse(data.conversation),
        firstMessage: SupportMessageSchema.parse(data.first_message),
      };
    },

    /** Returns the customer's most recent open thread, or null. */
    resume: async (): Promise<ResumeResult | null> => {
      const data = await request("GET", "/resume");
      if (!data || data.conversation == null) return null;
      return {
        conversation: SupportConversationSchema.parse(data.conversation),
        messages: z.array(SupportMessageSchema).parse(data.messages ?? []),
      };
    },

    getConversation: async (id: string): Promise<SupportConversation> => {
      const data = await request("GET", `/conversations/${encodeURIComponent(id)}`);
      return SupportConversationSchema.parse(data.conversation);
    },

    listMessages: async (id: string): Promise<SupportMessage[]> => {
      const data = await request("GET", `/conversations/${encodeURIComponent(id)}/messages`);
      return z.array(SupportMessageSchema).parse(data.messages ?? []);
    },

    postMessage: async (id: string, body: string): Promise<SupportMessage> => {
      const data = await request("POST", `/conversations/${encodeURIComponent(id)}/messages`, { body });
      return SupportMessageSchema.parse(data.message);
    },

    close: async (id: string): Promise<SupportConversation> => {
      const data = await request("POST", `/conversations/${encodeURIComponent(id)}/close`);
      return SupportConversationSchema.parse(data.conversation);
    },

    submitFeedback: async (id: string, feedback: FeedbackInput): Promise<SupportConversation> => {
      const data = await request("POST", `/conversations/${encodeURIComponent(id)}/feedback`, feedback);
      return SupportConversationSchema.parse(data.conversation);
    },

    getQueue: async (id: string): Promise<QueueState> => {
      const data = await request("GET", `/conversations/${encodeURIComponent(id)}/queue`);
      return QueueStateSchema.parse(data);
    },

    getWsTicket: async (id: string): Promise<WsTicket> => {
      const data = await request("POST", `/conversations/${encodeURIComponent(id)}/ws-ticket`);
      return WsTicketSchema.parse(data);
    },

    /** Builds the full WebSocket URL (origin + path + ?ticket=). */
    buildWsUrl: (ticket: WsTicket): string =>
      `${ticket.ws_url}?ticket=${encodeURIComponent(ticket.ticket)}`,

    /** The currently held otto session token, if any. */
    currentSessionToken: (): string | null => sessionToken,
  };
}

export type SupportClient = ReturnType<typeof createSupportClient>;
```

### 2.7 `src/support/useSupportChat.ts` (adapted — WS→poll only, SSE removed; complete file)

Changes vs mark8ly: the SSE transport is removed (no `openSSE`/`sse.ts`, no
`preferSSERef`/`sseRef`/`wsFailuresRef`). The realtime layer is WebSocket with reconnect,
and the polling fallback (unchanged) recovers any missed frame while the socket is down.
Outbox + app-state reconnect unchanged.

```ts
// useSupportChat — the React Native hook that drives a live support thread with
// durability guarantees:
//   - resume-or-create, load history
//   - real-time delivery over the otto WebSocket (reconnect + app-state)
//   - a DURABLE SEND OUTBOX: outgoing messages are persisted to device storage
//     and re-sent with backoff until otto confirms them
//   - a POLLING FALLBACK: while the socket is down, inbound messages are
//     recovered by polling REST, so a missed live frame is never permanent
//
// Pure logic (events, outbox, request shaping) lives in sibling modules and is
// unit-tested; this file is the RN-bound glue.
import { useCallback, useEffect, useRef, useState } from "react";
import { AppState, type AppStateStatus } from "react-native";

import { SupportError, type SupportClient } from "./client";
import { mergeMessage, mergeMessages, parseOttoEvent } from "./events";
import {
  addItem,
  backoffMs,
  isRetryable,
  loadOutbox,
  markItem,
  outboxItemToMessage,
  removeItem,
  saveOutbox,
  type KVStorage,
  type OutboxItem,
} from "./outbox";
import type { CreateConversationInput, SupportConversation, SupportMessage } from "./types";

export type SupportChatStatus = "idle" | "loading" | "connecting" | "ready" | "closed" | "error";

export type DisplayMessage = SupportMessage & { pending?: boolean; failed?: boolean };

export interface UseSupportChatOptions {
  client: SupportClient;
  /** Resume the customer's open thread on mount. Default true. */
  autoResume?: boolean;
  /** Persistent storage for the send outbox (SecureStore). Omit for in-memory. */
  storage?: KVStorage;
  /** Poll interval (ms) used as the fallback while the socket is down. */
  pollIntervalMs?: number;
}

export interface UseSupportChat {
  conversation: SupportConversation | null;
  messages: DisplayMessage[];
  status: SupportChatStatus;
  connected: boolean;
  /** True while any queued message is still unconfirmed. */
  hasPending: boolean;
  error: string | null;
  startConversation: (input: CreateConversationInput) => Promise<void>;
  sendMessage: (body: string) => Promise<void>;
  retryFailed: () => void;
  closeConversation: () => Promise<void>;
  refresh: () => Promise<void>;
}

const MAX_BACKOFF_MS = 15_000;
const DEFAULT_POLL_MS = 5_000;

function newClientMsgId(): string {
  return `cmid-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function nowIso(): string {
  return new Date().toISOString();
}

function displayMessages(server: SupportMessage[], outbox: OutboxItem[]): DisplayMessage[] {
  const pending = outbox.map(outboxItemToMessage) as DisplayMessage[];
  return mergeMessages(server, pending as SupportMessage[]) as DisplayMessage[];
}

export function useSupportChat({
  client,
  autoResume = true,
  storage,
  pollIntervalMs = DEFAULT_POLL_MS,
}: UseSupportChatOptions): UseSupportChat {
  const [conversation, setConversation] = useState<SupportConversation | null>(null);
  const [serverMessages, setServerMessages] = useState<SupportMessage[]>([]);
  const [outbox, setOutbox] = useState<OutboxItem[]>([]);
  const [status, setStatus] = useState<SupportChatStatus>("idle");
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const socketRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const retryRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptsRef = useRef(0);
  const closedByUsRef = useRef(false);
  const convIdRef = useRef<string | null>(null);
  const connectedRef = useRef(false);
  const outboxRef = useRef<OutboxItem[]>([]);
  const drainingRef = useRef(false);

  const setConnectedBoth = (v: boolean) => {
    connectedRef.current = v;
    setConnected(v);
  };

  // ── Outbox helpers ────────────────────────────────────────────────
  const persistOutbox = useCallback(
    (next: OutboxItem[]) => {
      outboxRef.current = next;
      setOutbox(next);
      if (storage && convIdRef.current) void saveOutbox(storage, convIdRef.current, next);
    },
    [storage],
  );

  const scheduleDrain = (delay: number) => {
    if (retryRef.current) clearTimeout(retryRef.current);
    retryRef.current = setTimeout(() => void drainOutbox(), delay);
  };

  const drainOutbox = useCallback(async () => {
    if (drainingRef.current) return;
    drainingRef.current = true;
    try {
      // eslint-disable-next-line no-constant-condition
      while (true) {
        const next = outboxRef.current.find((i) => i.status === "queued");
        if (!next) break;
        persistOutbox(markItem(outboxRef.current, next.clientMsgId, { status: "sending" }));
        try {
          const serverMsg = await client.postMessage(next.conversationId, next.body);
          setServerMessages((prev) => mergeMessage(prev, serverMsg));
          persistOutbox(removeItem(outboxRef.current, next.clientMsgId));
        } catch (e) {
          const httpStatus = e instanceof SupportError ? e.status : null;
          const attempts = next.attempts + 1;
          if (isRetryable(httpStatus)) {
            persistOutbox(markItem(outboxRef.current, next.clientMsgId, { status: "queued", attempts }));
            scheduleDrain(backoffMs(attempts));
            break;
          }
          persistOutbox(markItem(outboxRef.current, next.clientMsgId, { status: "failed", attempts }));
        }
      }
    } finally {
      drainingRef.current = false;
    }
  }, [client, persistOutbox]);

  const sendMessage = useCallback(
    async (body: string) => {
      const id = convIdRef.current;
      const trimmed = body.trim();
      if (!id || !trimmed) return;
      const item: OutboxItem = {
        clientMsgId: newClientMsgId(),
        conversationId: id,
        body: trimmed,
        createdAt: nowIso(),
        attempts: 0,
        status: "queued",
      };
      persistOutbox(addItem(outboxRef.current, item));
      void drainOutbox();
    },
    [persistOutbox, drainOutbox],
  );

  const retryFailed = useCallback(() => {
    const requeued = outboxRef.current.map((i) =>
      i.status === "failed" ? { ...i, status: "queued" as const, attempts: 0 } : i,
    );
    persistOutbox(requeued);
    void drainOutbox();
  }, [persistOutbox, drainOutbox]);

  // ── Realtime (WebSocket → polling) ────────────────────────────────
  const clearReconnect = () => {
    if (reconnectRef.current) {
      clearTimeout(reconnectRef.current);
      reconnectRef.current = null;
    }
  };

  const teardownSocket = useCallback(() => {
    closedByUsRef.current = true;
    clearReconnect();
    if (socketRef.current) {
      socketRef.current.close();
      socketRef.current = null;
    }
    setConnectedBoth(false);
  }, []);

  const connect = useCallback(
    async (conversationId: string) => {
      closedByUsRef.current = false;

      const reconnect = () => {
        const attempt = (attemptsRef.current += 1);
        const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** (attempt - 1));
        clearReconnect();
        reconnectRef.current = setTimeout(() => {
          if (convIdRef.current) void connect(convIdRef.current);
        }, delay);
      };
      const onOpen = () => {
        attemptsRef.current = 0;
        setConnectedBoth(true);
        setStatus("ready");
        // Reconcile anything missed while offline + flush the outbox.
        void refresh();
        void drainOutbox();
      };
      const onFrame = (raw: string) => {
        const event = parseOttoEvent(raw);
        if (event.kind === "message") {
          setServerMessages((prev) => mergeMessage(prev, event.message));
        } else if (event.kind === "conversation_closed") {
          setStatus("closed");
          setConversation((prev) => (prev ? { ...prev, status: "closed" } : prev));
          teardownSocket();
        }
      };

      let ticket;
      try {
        ticket = await client.getWsTicket(conversationId);
      } catch {
        // Ticket mint failed (e.g. no Istio WS route, transient) — polling
        // fallback keeps the thread live; retry the socket with backoff.
        reconnect();
        return;
      }
      if (closedByUsRef.current) return;

      const ws = new WebSocket(client.buildWsUrl(ticket));
      socketRef.current = ws;
      ws.onopen = onOpen;
      ws.onmessage = (ev) => {
        const data = (ev as { data?: unknown }).data;
        onFrame(typeof data === "string" ? data : "");
      };
      ws.onerror = () => setConnectedBoth(false);
      ws.onclose = () => {
        setConnectedBoth(false);
        socketRef.current = null;
        if (closedByUsRef.current) return;
        reconnect();
      };
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [client, teardownSocket, drainOutbox],
  );

  const refresh = useCallback(async () => {
    if (!convIdRef.current) return;
    try {
      const msgs = await client.listMessages(convIdRef.current);
      setServerMessages((prev) => mergeMessages(prev, msgs));
    } catch (e) {
      setError(e instanceof Error ? e.message : "refresh failed");
    }
  }, [client]);

  const adoptConversation = useCallback(
    async (conv: SupportConversation, initial: SupportMessage[]) => {
      setConversation(conv);
      setServerMessages((prev) => mergeMessages(prev, initial));
      convIdRef.current = conv.id;
      if (storage) {
        const persisted = await loadOutbox(storage, conv.id);
        if (persisted.length) {
          const requeued = persisted.map((i) => (i.status === "sending" ? { ...i, status: "queued" as const } : i));
          outboxRef.current = requeued;
          setOutbox(requeued);
        }
      }
      if (conv.status === "closed") {
        setStatus("closed");
        teardownSocket();
        return;
      }
      setStatus("connecting");
      await connect(conv.id);
      void drainOutbox();
    },
    [connect, teardownSocket, storage, drainOutbox],
  );

  const startConversation = useCallback(
    async (input: CreateConversationInput) => {
      setStatus("loading");
      setError(null);
      try {
        const { conversation: conv, firstMessage } = await client.createConversation(input);
        await adoptConversation(conv, [firstMessage]);
      } catch (e) {
        setStatus("error");
        setError(e instanceof Error ? e.message : "could not start chat");
        throw e;
      }
    },
    [client, adoptConversation],
  );

  const closeConversation = useCallback(async () => {
    const id = convIdRef.current;
    if (!id) return;
    const conv = await client.close(id);
    setConversation(conv);
    setStatus("closed");
    teardownSocket();
  }, [client, teardownSocket]);

  // Resume on mount.
  useEffect(() => {
    let cancelled = false;
    if (!autoResume) {
      setStatus("idle");
      return;
    }
    setStatus("loading");
    client
      .resume()
      .then((res) => {
        if (cancelled) return;
        if (res) void adoptConversation(res.conversation, res.messages);
        else setStatus("idle");
      })
      .catch((e) => {
        if (cancelled) return;
        setStatus("idle");
        setError(e instanceof Error ? e.message : null);
      });
    return () => {
      cancelled = true;
      teardownSocket();
      if (retryRef.current) clearTimeout(retryRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Foreground reconnect.
  useEffect(() => {
    const onChange = (next: AppStateStatus) => {
      if (next === "active" && convIdRef.current && !socketRef.current) {
        if (!conversation || conversation.status !== "closed") void connect(convIdRef.current);
      }
    };
    const sub = AppState.addEventListener("change", onChange);
    return () => sub.remove();
  }, [connect, conversation]);

  // Polling fallback — while the socket is down, recover inbound messages over
  // REST so a dropped live frame is never permanent.
  useEffect(() => {
    const open = !!conversation && conversation.status !== "closed";
    if (!open) return;
    const timer = setInterval(() => {
      if (!connectedRef.current && convIdRef.current) void refresh();
    }, pollIntervalMs);
    return () => clearInterval(timer);
  }, [conversation, pollIntervalMs, refresh]);

  return {
    conversation,
    messages: displayMessages(serverMessages, outbox),
    status,
    connected,
    hasPending: outbox.length > 0,
    error,
    startConversation,
    sendMessage,
    retryFailed,
    closeConversation,
    refresh,
  };
}
```

### 2.8 `src/support/SupportChatView.tsx` (verbatim port — complete file)

Port the mark8ly `SupportChatView.tsx` **verbatim** (it is fully palette-driven and imports
only `react` + `react-native` + sibling `./useSupportChat` and `./types` — no app-specific
deps). Reproduce it exactly as in
`mark8ly/packages/mobile-shared/support/SupportChatView.tsx` (the `SupportPalette` interface,
`SupportChatViewProps`, `SupportChatView`, `ThreadView`, `StatusBar`, `MessageBubble`,
`IntakeForm`, and the `StyleSheet`). No edits required — the HomeChef screen supplies the
persimmon/coral palette via props.

> During execution, copy that file byte-for-byte into `src/support/SupportChatView.tsx`.

### 2.9 `src/support/index.ts` (barrel — complete file)

```ts
// Barrel for the shared mobile support-chat module (WS→poll; no SSE).
export * from "./types";
export * from "./events";
export * from "./outbox";
export * from "./client";
export * from "./useSupportChat";
export * from "./SupportChatView";
export { secureStoreKV } from "./storage";
```

### 2.10 `src/__tests__/support/support-logic.test.ts` (new — pure-logic vitest)

```ts
import { describe, expect, it } from "vitest";

import { mergeMessage, mergeMessages, parseOttoEvent } from "../../support/events";
import { backoffMs, isRetryable } from "../../support/outbox";
import type { SupportMessage } from "../../support/types";

const msg = (id: string, at: string): SupportMessage => ({
  id,
  sender_type: "customer",
  sender_name: "",
  body: id,
  created_at: at,
});

describe("parseOttoEvent", () => {
  it("parses a message.created envelope", () => {
    const ev = parseOttoEvent(
      JSON.stringify({ type: "otto.message.created", payload: { message: msg("m1", "2026-01-01T00:00:00Z") } }),
    );
    expect(ev.kind).toBe("message");
  });
  it("maps conversation.closed", () => {
    expect(parseOttoEvent(JSON.stringify({ type: "otto.conversation.closed" })).kind).toBe("conversation_closed");
  });
  it("collapses garbage to unknown", () => {
    expect(parseOttoEvent("not json").kind).toBe("unknown");
    expect(parseOttoEvent(JSON.stringify({ type: "nope" })).kind).toBe("unknown");
  });
});

describe("mergeMessage", () => {
  it("dedupes by id and sorts by created_at", () => {
    let out = mergeMessages([], [msg("b", "2026-01-02T00:00:00Z"), msg("a", "2026-01-01T00:00:00Z")]);
    out = mergeMessage(out, msg("a", "2026-01-01T00:00:00Z")); // dup
    expect(out.map((m) => m.id)).toEqual(["a", "b"]);
  });
});

describe("outbox retry policy", () => {
  it("retries transport + 5xx + 429, not other 4xx", () => {
    expect(isRetryable(null)).toBe(true);
    expect(isRetryable(500)).toBe(true);
    expect(isRetryable(429)).toBe(true);
    expect(isRetryable(400)).toBe(false);
    expect(isRetryable(422)).toBe(false);
  });
  it("caps backoff at 30s", () => {
    expect(backoffMs(1)).toBe(1000);
    expect(backoffMs(99)).toBe(30_000);
  });
});
```

### Task 2 verify

```bash
cd <Home-Chef-App root>
npx tsc --noEmit -p apps/mobile-customer/tsconfig.json   # resolves @homechef/mobile-shared/support via tsconfig paths
cd packages/mobile-shared && npx vitest run src/__tests__/support   # expect: all pass
```

If `tsc` flags `.passthrough()` under this zod build (it did NOT in the runtime check —
verified `4.4.3` accepts it), replace `.passthrough()` with `z.looseObject({...})` in
`types.ts` (`SupportConversationSchema = z.looseObject({ ... })` — same lenient behaviour, no
new dep). Only apply if `tsc` actually errors.

### Task 2 commit

```bash
cd <Home-Chef-App root>
git fetch origin && git status && git log --oneline -1 origin/main
git add packages/mobile-shared/src/support packages/mobile-shared/src/__tests__/support \
        packages/mobile-shared/package.json
git commit -m "feat(support): port WS+poll support-chat kit into mobile-shared"
```

---

## Task 3 — mobile-customer support screen + Help entry (deliverable B, app)

Files:
- **new** `apps/mobile-customer/app/support-chat.tsx`
- **edit** `apps/mobile-customer/app/(tabs)/profile.tsx` (add one nav row)

### 3.1 `apps/mobile-customer/app/support-chat.tsx` (new — complete file)

```tsx
// Chat with Tesserix support (Phase 5). Renders the shared SupportChatView
// wired to the HomeChef API support-chat proxy, which bridges to otto's
// homechef tenant. A Tesserix admin answers from the platform inbox; the
// conversation carries the signed-in customer's identity so they skip OTP.
import { useMemo } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useRouter } from "expo-router";
import { ChevronLeft } from "lucide-react-native";
import * as SecureStore from "expo-secure-store";

import {
  createSupportClient,
  secureStoreKV,
  SupportChatView,
  useSupportChat,
  type IntakeReason,
  type SupportPalette,
} from "@homechef/mobile-shared/support";
import { customerColors } from "@homechef/mobile-shared/theme";
import { refreshSession } from "@homechef/mobile-shared/auth";
import { useAuthStore } from "../store/auth-store";

// Matches otto's TenantReasons["homechef"] whitelist + the web widget
// (apps/web/src/features/support/OttoChat.tsx). No DOB — HomeChef identifies
// orders by the signed-in account. general_question skips the summary field.
const HOMECHEF_REASONS: IntakeReason[] = [
  { value: "general_question", label: "Ask a quick question", requiresStatus: false },
  { value: "order_tracking", label: "Order tracking / ETA" },
  { value: "delivery_issue", label: "Delivery problem" },
  { value: "refund", label: "Refund request" },
  { value: "chef_question", label: "Question about a chef or dish" },
  { value: "account_issue", label: "Account / login issue" },
  { value: "other", label: "Something else" },
];

const SESSION_KEY = "otto_homechef_support_session";

export default function SupportChatScreen() {
  const router = useRouter();
  const user = useAuthStore((s) => s.user);

  const client = useMemo(
    () =>
      createSupportClient({
        baseUrl: process.env.EXPO_PUBLIC_API_URL!,
        basePath: "/v1/support/chat",
        getToken: async () => useAuthStore.getState().accessToken,
        refreshToken: refreshSession,
        onUnauthorized: () => {
          // Refresh already failed inside the client; tear the session down so
          // the layout auth guard routes back to login.
          useAuthStore.getState().logout();
        },
        loadSessionToken: () => SecureStore.getItemAsync(SESSION_KEY),
        saveSessionToken: (t) => SecureStore.setItemAsync(SESSION_KEY, t),
      }),
    [],
  );

  const chat = useSupportChat({ client, storage: secureStoreKV });

  // Outgoing bubbles are neutral charcoal-on-soft, not coral — every message
  // the customer sends would otherwise repeat the accent. Coral stays reserved
  // for the single Send/submit action (`primary`).
  const palette = useMemo<SupportPalette>(
    () => ({
      background: customerColors.canvas,
      surface: customerColors.surface.soft,
      bubbleOwn: customerColors.charcoal.DEFAULT,
      textOnOwn: customerColors.canvas,
      text: customerColors.charcoal.DEFAULT,
      textSecondary: customerColors.charcoal.soft,
      border: customerColors.hairline,
      primary: customerColors.coral.DEFAULT,
      onPrimary: customerColors.canvas,
      danger: customerColors.destructive.DEFAULT,
    }),
    [],
  );

  const defaults = useMemo(
    () => ({
      name: [user?.firstName, user?.lastName].filter(Boolean).join(" ") || undefined,
      email: user?.email ?? undefined,
    }),
    [user?.firstName, user?.lastName, user?.email],
  );

  return (
    <SafeAreaView style={styles.fill} edges={["top", "left", "right"]}>
      <View style={styles.header}>
        <Pressable
          onPress={() => router.back()}
          accessibilityRole="button"
          accessibilityLabel="Go back"
          hitSlop={8}
          style={styles.back}
        >
          <ChevronLeft size={24} color={customerColors.charcoal.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Chat with support</Text>
        <View style={styles.back} />
      </View>
      <View style={styles.fill}>
        <SupportChatView
          chat={chat}
          palette={palette}
          reasons={HOMECHEF_REASONS}
          defaults={defaults}
          introTitle="How can we help?"
          introSubtitle="Message the Fe3dr support team — we'll get back to you here."
          composerPlaceholder="Type a message…"
          statusPlaceholder="e.g. Order #ORD-2041 stuck on 'preparing'"
        />
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1, backgroundColor: customerColors.canvas },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: 8,
    paddingVertical: 8,
    borderBottomWidth: StyleSheet.hairlineWidth,
    borderBottomColor: customerColors.hairline,
  },
  back: { width: 40, height: 40, alignItems: "center", justifyContent: "center" },
  title: { fontSize: 17, fontWeight: "600", color: customerColors.charcoal.DEFAULT },
});
```

> `refreshSession` is exported from `@homechef/mobile-shared/auth` (verified in
> `src/auth/bff-session.ts` → re-exported via `src/auth/index.ts`). `useAuthStore.getState().user`
> carries `firstName`/`lastName`/`email` (`User` type). The app Stack runs
> `headerShown: false`, so the screen ships its own header (expo-router auto-registers
> `app/support-chat.tsx` as `/support-chat`).

### 3.2 `apps/mobile-customer/app/(tabs)/profile.tsx` — add the entry

Add `LifeBuoy` to the lucide import block (top of file):

```tsx
  LifeBuoy,
```

Then add a "Help & support" nav row in the Privacy & Legal section — insert it right
**before** the existing "Legal" row (after the "Download my data" row + its divider):

```tsx
        <NavRowDivider />
        <NavRow
          icon={<LifeBuoy size={18} color={customerColors.charcoal.soft} />}
          label="Help & support"
          onPress={() => router.push('/support-chat')}
        />
```

> Placement: it sits between "Download my data" and "Legal". `LifeBuoy` exists in
> `lucide-react-native`; if `tsc` cannot resolve it, fall back to `HelpCircle` (also present).

### Task 3 verify

```bash
cd <Home-Chef-App root>
npx tsc --noEmit -p apps/mobile-customer/tsconfig.json   # expect: no errors
```

Optional simulator smoke (per the mobile local-run recipe) once API + infra are live:
Profile → "Help & support" → intake form → send → reply arrives (WS or ≤5s poll).

### Task 3 commit

```bash
cd <Home-Chef-App root>
git fetch origin && git status && git log --oneline -1 origin/main
git add apps/mobile-customer/app/support-chat.tsx apps/mobile-customer/app/\(tabs\)/profile.tsx
git commit -m "feat(support): add Chat with support screen to mobile-customer profile"
```

---

## Task 4 — Infra: homechef-api chart env (deliverable C)

**No Istio VirtualService change** (the storefront-otto WS route already exists on the
fe3dr.com host — discovery #1). **No new ExternalSecret** (`homechef-otto` already exists —
discovery #8). Only wire three env vars into homechef-api.

Repo: `tesserix-k8s` (work on `main`, **do not push** — leave for the user).

### 4.1 `charts/apps/homechef-api/values.yaml` — add to the `env:` map

Under the existing `env:` block (after `MONGODB_DB`, near the top of the map):

```yaml
  # Otto support-chat (Phase 5) — the mobile support-chat proxy forwards to the
  # shared otto service in support-platform. OTTO_INTERNAL_AUTH is injected
  # separately from the homechef-otto ExternalSecret (see _helpers.tpl).
  OTTO_URL: "http://support-platform-otto.support-platform.svc.cluster.local:8089"
```

### 4.2 `charts/apps/homechef-api/values-prod.yaml` — add the public WS base

Under the prod `env:` block (after `BFF_SESSION_URL`):

```yaml
  # Public WebSocket origin the mobile app dials directly for real-time chat.
  # Matches the storefront-otto WS route on the fe3dr.com VirtualService.
  OTTO_WS_PUBLIC_BASE: "wss://fe3dr.com"
```

### 4.3 `charts/apps/homechef-api/templates/_helpers.tpl` — secret-backed env

In the `homechef-api.containerEnv` define, add a secret-backed block (place it after the
`JWT_REFRESH_SECRET` block, following the existing `optional: true` pattern used by the
MongoDB/OpenExchangeRates blocks):

```yaml
- name: OTTO_INTERNAL_AUTH
  valueFrom:
    secretKeyRef:
      name: homechef-otto
      key: INTERNAL_AUTH_SECRET
      optional: true
```

> The `homechef-otto` Secret is materialised by
> `external-secrets/prod/homechef/externalsecret.yaml` (key `INTERNAL_AUTH_SECRET` ←
> GCP SM `prod-support-platform-otto-internal-auth`) and is already consumed by
> `homechef-web`. `optional: true` keeps the API booting if the secret is briefly
> absent (proxy stays dark → 503, never crashes). The helper is shared by the API
> Deployment and the Temporal worker; the worker harmlessly gets the same env.

### Task 4 verify (helm template only — no apply/deploy)

```bash
cd tesserix-k8s
helm template charts/apps/homechef-api \
  -f charts/apps/homechef-api/values.yaml \
  -f charts/apps/homechef-api/values-prod.yaml | grep -B1 -A6 'OTTO'
# expect to see:
#   - name: OTTO_URL            value: "http://support-platform-otto...:8089"
#   - name: OTTO_WS_PUBLIC_BASE value: "wss://fe3dr.com"
#   - name: OTTO_INTERNAL_AUTH  valueFrom: secretKeyRef: name: homechef-otto key: INTERNAL_AUTH_SECRET
```

### Task 4 commit (do NOT push; user pushes)

```bash
cd tesserix-k8s
git config user.name "sam123ben" && git config user.email "samyak.rout@gmail.com"
git fetch origin && git status && git log --oneline -1 origin/main
git add charts/apps/homechef-api/values.yaml charts/apps/homechef-api/values-prod.yaml \
        charts/apps/homechef-api/templates/_helpers.tpl
git commit -m "feat(homechef-api): wire OTTO_URL + OTTO_INTERNAL_AUTH for mobile support-chat"
# STOP — leave for the user to push (ArgoCD deploys).
```

---

## Rollout order

1. **API** (Task 1) — merge to `main` → CI builds image → Kargo promotes → ArgoCD rolls out.
   Until Task 4's env lands, `OTTO_URL`/`OTTO_INTERNAL_AUTH` are empty → `/api/v1/support/chat/*`
   returns 503 (dark, safe).
2. **Infra** (Task 4) — user pushes tesserix-k8s → ArgoCD syncs homechef-api env → proxy live.
3. **Mobile** (Tasks 2-3) — merge to `main`; ships in the next **batched** EAS/TestFlight build.
   Older installed builds simply don't show the row; nothing breaks.

Each step is independently shippable; the screen shows an empty/idle state until the API + env
are live, and WS silently degrades to polling if the socket ever fails.

---

## Self-review

**Coverage A-C:**
- **A (Go proxy):** ✅ new group `/api/v1/support/chat/*` behind `bffAuth` + `RateLimitByUser`;
  pins `X-Tenant-Id: homechef` / `X-Store-Id: default` / `X-Internal-Auth`; forwards
  `X-User-Id/Email/Name` from the JWT-hydrated user (skips OTP); explicit route allowlist;
  1 MiB body cap; 15s timeout; 5xx suppression; `otto_session` cookie→`session_token` body +
  `X-Otto-Session` header passthrough; `ws-ticket` augmented with public `ws_url`; empty
  config → 503 (dark); `config.go` env; `.env.example`; table tests covering
  scope/identity/session-merge/path/ws_url/5xx-suppression/503-dark.
- **B (RN kit + screen):** ✅ kit in `packages/mobile-shared/src/support/` (types, events,
  outbox, storage, client, WS→poll hook, view, barrel) + `"./support"` export; screen
  `app/support-chat.tsx` wired to `EXPO_PUBLIC_API_URL` + `/v1/support/chat`, bearer via
  `useAuthStore`, refresh via `refreshSession`, SecureStore session persistence, coral palette,
  exact HOMECHEF_REASONS; Profile "Help & support" entry; **no floating widget**; pure-logic
  vitest.
- **C (infra):** ✅ `OTTO_URL` (values) + `OTTO_WS_PUBLIC_BASE` (values-prod) + secret-backed
  `OTTO_INTERNAL_AUTH` (helper). No Istio change needed (WS route pre-exists); no new secret.

**Placeholders:** none — every file is complete except `SupportChatView.tsx`, which is an
explicit verbatim copy of a named existing file (byte-for-byte; no edits) to avoid duplicating
~410 unchanged lines here.

**Type consistency:**
- Go: `middleware.GetUser → *models.User{ID uuid.UUID, Email, FirstName, LastName}` ✅;
  `middleware.CtxUserID/CtxUserEmail`, `middleware.RateLimitByUser(int,int)`, `bffAuth(bffKey,bffWindow)`
  all exist ✅; module `github.com/homechef/api`, go 1.26.1 ✅; `uuid` + `gin` already deps ✅.
- TS: `EXPO_PUBLIC_API_URL = https://fe3dr.com/api` so `basePath = "/v1/support/chat"` yields
  `/api/v1/support/chat/*` (matches the Go `v1 := r.Group("/api/v1")` + `/support/chat`) ✅;
  `WsTicket.ws_url` supplied by the proxy, consumed by `buildWsUrl` ✅; `SupportPalette` fields
  populated from `customerColors` ✅; zod `.passthrough()`/`.default()` verified on 4.4.3 ✅;
  tsconfig `@homechef/mobile-shared/*` wildcard + new `exports` subpath both resolve `./support` ✅.
- Contracts cross-checked against otto `storefront_handler.go` (create/resume/get/messages/
  close/queue/feedback/ws-ticket paths + `X-Otto-Session` header auth) and mark8ly
  `ottobridge/bridge.go` + `support_test.go` (session-token merge, ws_url augmentation) ✅.

**Risks / fallbacks (all with a documented remedy in-plan):**
- zod `.passthrough()` under `tsc` → swap to `z.looseObject` (Task 2 verify note).
- `LifeBuoy` icon absent → `HelpCircle` (Task 3.2 note).
- WS never connects (route/proxy hiccup) → polling fallback guarantees delivery; ws-ticket
  failure is caught and retried (hook `connect` catch).
- Empty otto config → 503 dark; `optional: true` secret keeps the pod booting.
