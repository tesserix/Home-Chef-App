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
//	browser ── hc_session cookie     ──▶  BFF /api/v1/*  ── HMAC + X-User-* ──▶  API /api/v1/*
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

// requestOrigin reconstructs the scheme://host this request actually
// arrived at, as seen from outside the cluster. The service is always
// behind Istio/Cloudflare, so c.Request.TLS is nil and the scheme has to be
// read off X-Forwarded-Proto (set by the edge, comma-joined if there were
// multiple hops — the first value is the one the client used). Absent that
// header, default to https: every real deployment of this service sits
// behind TLS-terminating infra, so a missing header means an edge that
// forgot to set it, not a plaintext deployment. c.Request.Host already
// carries whatever Host the client sent (or the ingress rewrote it to),
// which is what a browser's Origin host will match for a same-origin call.
func requestOrigin(r *http.Request) string {
	scheme := "https"
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" {
		if i := strings.IndexByte(xfp, ','); i >= 0 {
			xfp = xfp[:i]
		}
		scheme = strings.TrimSpace(xfp)
	}
	return scheme + "://" + r.Host
}

// checkOrigin enforces that a cookie-authenticated request actually
// originated from this same BFF's own origin. It must only be called for
// requests authenticated via the session cookie — Bearer-authenticated
// requests (mobile) have no meaningful Origin and must not go through this.
//
// scheme://host, not host alone, is what's compared: Origin is a full
// origin, and each SPA (vendors.fe3dr.com, fe3dr.com, ...) is always served
// over https at a fixed host, so a legitimate call's Origin is always
// exactly this request's own reconstructed scheme://host — there's no
// legitimate case where the scheme differs but the host matches. Comparing
// host alone would still stop the cross-site attack (the attacker can't
// forge our host into their page's Origin), but it would also silently
// accept a downgraded-scheme replay if one ever reached this service, which
// buys nothing and costs a real check. The one thing this deliberately does
// NOT do is special-case www.fe3dr.com vs fe3dr.com: origin is derived from
// the request's own Host on every call, never hardcoded, so both domains
// compare correctly against themselves without any extra normalization —
// hardcoding a single expected host is exactly the trap to avoid here.
func checkOrigin(c *gin.Context) bool {
	origin := c.GetHeader("Origin")
	if origin != "" {
		return origin == requestOrigin(c.Request)
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
		} else if cookie, err := c.Request.Cookie(d.Sessions.CookieName()); err == nil {
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
