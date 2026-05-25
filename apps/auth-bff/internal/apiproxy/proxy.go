// Package apiproxy forwards mobile/SPA API requests to the upstream Go API.
//
// Mobile apps hold a BFF session token (the AES-GCM-encrypted Payload from
// session.Manager.Encode) and send it as `Authorization: Bearer <token>`.
// The API itself only accepts HMAC-signed requests from the BFF — there is
// no Bearer auth path on the API. This handler bridges the two:
//
//	mobile  ── Bearer session_token ──▶  BFF /api/v1/*  ── HMAC + X-User-* ──▶  API /api/v1/*
//
// On success the upstream response is streamed back unchanged.
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
		// 1. Extract Bearer token.
		authHeader := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_bearer"})
			return
		}
		token := strings.TrimPrefix(authHeader, prefix)

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

		// 5. Mirror client headers except Authorization (replaced by HMAC)
		//    and hop-by-hop headers (must not be forwarded).
		for k, vs := range c.Request.Header {
			lower := strings.ToLower(k)
			if lower == "authorization" {
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
