package oidc

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// browserDeviceCookie holds the id that lets the API tell one browser from
// another (#1164). It identifies an install, never a session, so it outlives
// sign-out and carries no user data.
const browserDeviceCookie = "hc_device"

const browserDeviceMaxAge = int(365 * 24 * time.Hour / time.Second)

// The cookie is client-writable, so only our own shape is trusted.
var browserDeviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// browserDeviceID returns this browser's device id, minting and setting one if
// the cookie is absent or does not look like ours.
func browserDeviceID(c *gin.Context) string {
	if ck, err := c.Request.Cookie(browserDeviceCookie); err == nil && browserDeviceIDPattern.MatchString(ck.Value) {
		return ck.Value
	}
	id := "web-" + uuid.NewString()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     browserDeviceCookie,
		Value:    id,
		Path:     "/",
		MaxAge:   browserDeviceMaxAge,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})
	return id
}

// browserLabel reduces a User-Agent to a browser name a person recognises in
// a security email. Deliberately coarse — a full UA string is noise there.
func browserLabel(ua string) string {
	switch {
	case ua == "":
		return "Browser"
	case strings.Contains(ua, "Edg/"):
		return "Microsoft Edge"
	case strings.Contains(ua, "OPR/"):
		return "Opera"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	case strings.Contains(ua, "Chrome/"):
		return "Chrome"
	case strings.Contains(ua, "Safari/"):
		return "Safari"
	default:
		return "Browser"
	}
}
