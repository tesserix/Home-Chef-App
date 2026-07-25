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

func Handler(d *Deps) gin.HandlerFunc {
	client := &http.Client{Timeout: 30 * time.Second}
	base := strings.TrimRight(d.APIBaseURL, "/")
	return func(c *gin.Context) {
		// 1. Resolve the session token. Mobile apps send it as a Bearer
		//    header; browser SPAs can't (their session lives in an HttpOnly
		//    cookie JavaScript can never read), so fall back to the session
		//    cookie when no Authorization header is present.
		var token string
		authHeader := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if strings.HasPrefix(authHeader, prefix) {
			token = strings.TrimPrefix(authHeader, prefix)
		} else if cookie, err := c.Request.Cookie(d.Sessions.CookieName()); err == nil {
			token = cookie.Value
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_session"})
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
