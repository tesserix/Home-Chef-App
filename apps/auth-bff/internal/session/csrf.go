package session

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"net/url"
	"strings"
)

const (
	CSRFCookieName = "hc_csrf"
	CSRFHeaderName = "X-CSRF-Token"
)

// SameOrigin reports whether Origin identifies the request's own host.
// Cookie-authenticated unsafe requests must always have an Origin header.
func SameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// ValidCSRFToken performs a constant-time double-submit comparison between
// the JavaScript-readable CSRF cookie and the explicit request header.
func ValidCSRFToken(r *http.Request) bool {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	header := r.Header.Get(CSRFHeaderName)
	if header == "" {
		return false
	}
	cookieHash := sha256.Sum256([]byte(cookie.Value))
	headerHash := sha256.Sum256([]byte(header))
	return subtle.ConstantTimeCompare(cookieHash[:], headerHash[:]) == 1
}
