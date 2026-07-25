// Package apiproxy forwards mobile/SPA API requests to the upstream Go API.
//
// Mobile apps hold a BFF session token (the AES-GCM-encrypted Payload from
// session.Manager.Encode) and send it as `Authorization: Bearer <token>`.
// Browser SPAs hold the exact same token, but as the value of the HttpOnly
// session cookie (session.Manager.SetCookie) — JavaScript in the browser
// can never read it, so it can't be placed in an Authorization header. This
// handler accepts the token from either source, preferring the header:
//
//	mobile  ── Bearer session_token ──▶  BFF /api/v1/*  ── HMAC + X-User-* ──▶  API /api/v1/*
//	browser ── per-app session cookie ─▶  BFF /api/v1/*  ── HMAC + X-User-* ──▶  API /api/v1/*
//
// The cookie name is resolved per request Host via Deps.CookieForHost (each
// portal owns its own cookie — see productregistry.Registry.SessionCookieForHost
// and homechef-products.yaml's sessionCookie field) so that, e.g., a
// vendors.fe3dr.com session and a fe3dr.com session never collide on the
// shared .fe3dr.com cookie domain. The Bearer path never consults it.
//
// The API itself only accepts HMAC-signed requests from the BFF — there is
// no Bearer auth path on the API. On success the upstream response is
// streamed back unchanged.
//
// # CSRF
//
// Unlike the Bearer path, the cookie path is ambiently authenticated:
// browsers attach cookies automatically, including on cross-site requests.
// The cookie is SameSite=Lax (internal/session/cookie.go), which already
// blocks cross-site POST/PUT/PATCH/DELETE, but a top-level cross-site GET
// navigation, or a browser that ignores SameSite, still carries it. So
// cookie-authenticated requests additionally get an Origin check here; see
// checkOrigin. Bearer-authenticated requests skip it entirely — mobile
// clients don't send Origin and can't be tricked into attaching a header.
package apiproxy

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/auth-bff/internal/headerproxy"
	"github.com/homechef/auth-bff/internal/session"
)

type Deps struct {
	APIBaseURL string
	Sessions   *session.Manager
	Signer     *headerproxy.Signer

	// CookieForHost resolves the request Host to the app-specific session
	// cookie name (e.g. productregistry.Registry.SessionCookieForHost). It
	// is only consulted on the cookie fallback path below — the Bearer path
	// never touches it, so mobile requests (Authorization header, no Host-
	// based app) are completely unaffected whether this is nil or not.
	// Optional: nil, or an unmatched host, falls back to Sessions' default
	// cookie name.
	CookieForHost session.CookieNameResolver
}

// hopByHop headers must not be forwarded by an intermediary (RFC 7230 §6.1).
var hopByHop = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

// safeMethods never carry side effects, so browsers are allowed to send them
// cross-site without an Origin header — top-level navigations are the
// classic example. Every unsafe method, by contrast, always gets an Origin
// header from a real browser (including on same-origin requests), so an
// unsafe request with no Origin at all cannot be a legitimate browser call.
var safeMethods = map[string]struct{}{
	http.MethodGet:     {},
	http.MethodHead:    {},
	http.MethodOptions: {},
}

// originHost pulls the host out of an Origin header value
// ("https://fe3dr.com" -> "fe3dr.com"), returning "" if it isn't a usable
// absolute origin. Origin is always scheme://host[:port] or the literal
// "null"; it never carries a path.
func originHost(origin string) string {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return ""
	}
	return u.Host
}

// checkOrigin enforces that a cookie-authenticated request actually
// originated from this same BFF's own origin. It must only be called for
// requests authenticated via the session cookie — Bearer-authenticated
// requests (mobile) have no meaningful Origin and must not go through this.
//
// The comparison is host-only, deliberately. An earlier version rebuilt
// scheme://host from X-Forwarded-Proto and compared full origins; that
// rejected every real cookie-authenticated POST in production
// (403 origin_rejected on placing an order). Cloudflare terminates TLS and
// the tunnel reaches Istio over plaintext, so the header arriving here says
// http while the browser's Origin says https — the two could never match.
// It went unnoticed at first because same-origin GETs send no Origin header
// at all and so never reached the comparison.
//
// Host alone is what actually carries the CSRF guarantee: an attacker's page
// cannot put our host in its Origin. The scheme adds nothing here — this
// service is only ever reachable over HTTPS from outside the cluster.
//
// This deliberately does NOT special-case www.fe3dr.com vs fe3dr.com: the
// expected host is read from the request itself on every call, never
// hardcoded, so each domain compares correctly against itself.
func checkOrigin(c *gin.Context) bool {
	if origin := c.GetHeader("Origin"); origin != "" {
		return originHost(origin) == c.Request.Host
	}
	// No Origin at all. Fine for a safe method (e.g. a top-level GET
	// navigation, which browsers don't attach Origin to); anything else
	// with no Origin is not a shape a real browser produces.
	_, safe := safeMethods[c.Request.Method]
	return safe
}

func Handler(d *Deps) gin.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}
	base := strings.TrimRight(d.APIBaseURL, "/")
	return func(c *gin.Context) {
		// 1. Resolve the session token. Mobile apps send it as a Bearer
		//    header; browser SPAs can't (their session lives in an HttpOnly
		//    cookie JavaScript can never read), so fall back to the session
		//    cookie when no Authorization header is present.
		var token string
		var viaCookie bool
		if bearer, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); ok {
			token = bearer
		} else if cookie, err := c.Request.Cookie(d.Sessions.ResolveCookieName(d.CookieForHost, c.Request.Host)); err == nil {
			token = cookie.Value
			viaCookie = true
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_session"})
			return
		}

		// 1b. CSRF guard: only the cookie is ambiently attached by the
		//     browser, so only the cookie path needs an Origin check.
		//     Bearer requests (mobile) are exempt — see package doc.
		if viaCookie && !checkOrigin(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "origin_rejected"})
			return
		}

		// 2. Resolve session → identity. Decode also enforces expiry.
		p, err := d.Sessions.Decode(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_session"})
			return
		}

		// 3. Buffer body so we can re-hash it for the HMAC signature.
		var body []byte
		if c.Request.Body != nil {
			body, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "body_read_failed"})
				return
			}
		}

		// 4. Build the upstream request, preserving method, full path, and
		//    query string so the API sees exactly what the client asked for.
		outURL := base + c.Request.URL.Path
		if c.Request.URL.RawQuery != "" {
			outURL += "?" + c.Request.URL.RawQuery
		}
		outReq, err := http.NewRequestWithContext(
			c.Request.Context(),
			c.Request.Method,
			outURL,
			bytes.NewReader(body),
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "build_request_failed"})
			return
		}

		// 5. Mirror client headers except Authorization (replaced by HMAC),
		//    Cookie (carries the session secret the API has no use for and
		//    must never see — it only trusts the HMAC signature the BFF
		//    adds), and hop-by-hop headers (must not be forwarded).
		for k, vs := range c.Request.Header {
			lower := strings.ToLower(k)
			if lower == "authorization" || lower == "cookie" {
				continue
			}
			if _, hop := hopByHop[lower]; hop {
				continue
			}
			for _, v := range vs {
				outReq.Header.Add(k, v)
			}
		}

		// 6. HMAC-sign the upstream request with the session identity.
		if err := d.Signer.Sign(outReq, body, headerproxy.Identity{
			UserID: p.UID,
			Email:  p.Email,
			Role:   p.Role,
			Pool:   p.Pool,
		}); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "sign_failed"})
			return
		}

		// 7. Execute and stream the response back to the client.
		resp, err := client.Do(outReq)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "upstream_unreachable"})
			return
		}
		defer resp.Body.Close()

		for k, vs := range resp.Header {
			lower := strings.ToLower(k)
			if _, hop := hopByHop[lower]; hop {
				continue
			}
			for _, v := range vs {
				c.Writer.Header().Add(k, v)
			}
		}
		c.Writer.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}
