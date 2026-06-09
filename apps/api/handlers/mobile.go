package handlers

import (
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// MobileHandler serves mobile-app-only endpoints that don't fit any
// domain (chef / orders / etc) — currently just the min-version gate.
type MobileHandler struct{}

type minVersionResponse struct {
	MinVersion    string `json:"minVersion"`
	LatestVersion string `json:"latestVersion"`
	StoreURL      string `json:"storeUrl"`
	Platform      string `json:"platform"`
	App           string `json:"app"`
}

var (
	mobileHandler     *MobileHandler
	mobileHandlerOnce sync.Once
)

// NewMobileHandler returns a singleton MobileHandler.
func NewMobileHandler() *MobileHandler {
	mobileHandlerOnce.Do(func() {
		mobileHandler = &MobileHandler{}
	})
	return mobileHandler
}

// GetMinVersion returns the minimum + latest app version and store URL
// for the requested platform/app. Source of truth is environment vars
// per Wave 1 backend design §3.1 (v1 simplicity — flipping min-version
// requires a pod restart; acceptable since we rev <1x/month).
//
// Env keys: MIN_VERSION_<APP>_<PLATFORM>, LATEST_VERSION_..., STORE_URL_...
// e.g. MIN_VERSION_VENDOR_IOS, STORE_URL_VENDOR_IOS.
//
// GET /api/v1/mobile/min-version?platform=ios|android&app=vendor
func (h *MobileHandler) GetMinVersion(c *gin.Context) {
	platform := c.Query("platform")
	app := c.Query("app")

	if platform != "ios" && platform != "android" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform must be ios or android"})
		return
	}
	if app == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "app is required"})
		return
	}

	resp := minVersionResponse{
		MinVersion:    versionEnv("MIN_VERSION", app, platform, "1.0.0"),
		LatestVersion: versionEnv("LATEST_VERSION", app, platform, "1.0.0"),
		StoreURL:      versionEnv("STORE_URL", app, platform, defaultStoreURL(platform)),
		Platform:      platform,
		App:           app,
	}

	c.JSON(http.StatusOK, resp)
}

// versionEnv resolves an env key of the shape PREFIX_<APP>_<PLATFORM>
// (uppercased), falling back to a default when unset.
func versionEnv(prefix, app, platform, fallback string) string {
	key := prefix + "_" + strings.ToUpper(app) + "_" + strings.ToUpper(platform)
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// defaultStoreURL returns a safe public landing URL by platform when no
// explicit STORE_URL_* env is set. Stops the upgrade gate from showing
// an empty CTA in case of misconfigured prod env.
func defaultStoreURL(platform string) string {
	switch platform {
	case "ios":
		return "https://apps.apple.com/in/"
	case "android":
		return "https://play.google.com/store/"
	default:
		return ""
	}
}
