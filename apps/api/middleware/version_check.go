package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// VersionCheck enforces a minimum mobile-app version per platform.
//
// Behavior:
//   - If the request carries neither X-App-Version nor X-Platform, the
//     middleware is a no-op (web / admin / curl traffic is not gated).
//   - If both headers are present and the version is below the
//     configured minimum, returns 426 Upgrade Required with a JSON body
//     describing the required version + store URL so the mobile client
//     can render an actionable wall.
//   - Excluded paths bypass the check entirely (typically /health,
//     /metrics, the min-version endpoint itself, webhooks).
//
// Min version source of truth: env vars MIN_VERSION_<APP>_<PLATFORM>
// (defaults to "1.0.0" when unset, which lets every released version
// through and effectively disables the wall). Pair with /api/v1/mobile/
// min-version so the client and middleware read the same source.
//
// `app` is currently hard-coded to "vendor" because this API only
// serves the vendor app on mobile; extend if/when customer + delivery
// apps come online.
func VersionCheck(excludedPathPrefixes []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		for _, p := range excludedPathPrefixes {
			if strings.HasPrefix(path, p) {
				c.Next()
				return
			}
		}

		appVersion := c.GetHeader("X-App-Version")
		platform := strings.ToLower(c.GetHeader("X-Platform"))
		if appVersion == "" || platform == "" {
			c.Next()
			return
		}
		if platform != "ios" && platform != "android" {
			c.Next()
			return
		}

		min := minVersionFromEnv("vendor", platform)
		if semverLess(appVersion, min) {
			storeURL := os.Getenv("STORE_URL_VENDOR_" + strings.ToUpper(platform))
			if storeURL == "" {
				// Mirror handlers/mobile.go's defaultStoreURL so the
				// upgrade wall CTA always has a valid target, even
				// before ops sets the STORE_URL_* env. Without this
				// the 426 body has storeUrl="", which renders a
				// disabled "Update now" button on the mobile wall.
				switch platform {
				case "ios":
					storeURL = "https://apps.apple.com/in/"
				case "android":
					storeURL = "https://play.google.com/store/"
				}
			}
			c.AbortWithStatusJSON(http.StatusUpgradeRequired, gin.H{
				"error":      "upgrade_required",
				"message":    "App version " + appVersion + " is below required " + min + ". Please update.",
				"minVersion": min,
				"storeUrl":   storeURL,
			})
			return
		}
		c.Next()
	}
}

func minVersionFromEnv(app, platform string) string {
	key := "MIN_VERSION_" + strings.ToUpper(app) + "_" + strings.ToUpper(platform)
	if v := os.Getenv(key); v != "" {
		return v
	}
	return "1.0.0"
}

// semverLess returns true when a < b in dotted-number semver. Strips a
// leading "v" and an optional "+build" suffix (mobile sends versions of
// the shape "1.0.3+12"). Non-numeric segments are treated as 0.
func semverLess(a, b string) bool {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "+-"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	out := [3]int{0, 0, 0}
	for i := 0; i < 3 && i < len(parts); i++ {
		out[i] = atoiSafe(parts[i])
	}
	return out
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}
