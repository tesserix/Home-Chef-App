package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newVersionRouter(excluded []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(VersionCheck(excluded))
	r.GET("/api/v1/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	r.GET("/health", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

// Stale mobile client (below MIN_VERSION default 1.0.0 is impossible, so we
// raise the floor via env) must get a 426 hard-upgrade wall with a store link.
func TestVersionCheck_StaleClient_426WithStoreLink(t *testing.T) {
	t.Setenv("MIN_VERSION_VENDOR_IOS", "2.0.0")
	t.Setenv("STORE_URL_VENDOR_IOS", "https://apps.apple.com/app/id123")
	r := newVersionRouter(nil)

	req := httptest.NewRequest("GET", "/api/v1/ping", nil)
	req.Header.Set("X-App-Version", "1.5.0")
	req.Header.Set("X-Platform", "ios")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUpgradeRequired, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "upgrade_required", body["error"])
	assert.Equal(t, "2.0.0", body["minVersion"])
	assert.Equal(t, "https://apps.apple.com/app/id123", body["storeUrl"])
}

// When STORE_URL_* is unset, the middleware must still return a usable default
// store link so the mobile "Update now" CTA is never disabled.
func TestVersionCheck_StaleClient_DefaultStoreURLFallback(t *testing.T) {
	t.Setenv("MIN_VERSION_VENDOR_ANDROID", "3.0.0")
	t.Setenv("STORE_URL_VENDOR_ANDROID", "")
	r := newVersionRouter(nil)

	req := httptest.NewRequest("GET", "/api/v1/ping", nil)
	req.Header.Set("X-App-Version", "2.9.9")
	req.Header.Set("X-Platform", "android")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUpgradeRequired, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "https://play.google.com/store/", body["storeUrl"])
}

// A client at or above the minimum passes through untouched.
func TestVersionCheck_CurrentClient_PassesThrough(t *testing.T) {
	t.Setenv("MIN_VERSION_VENDOR_IOS", "2.0.0")
	r := newVersionRouter(nil)

	req := httptest.NewRequest("GET", "/api/v1/ping", nil)
	req.Header.Set("X-App-Version", "2.0.0")
	req.Header.Set("X-Platform", "ios")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// And a strictly-newer client also passes.
	req2 := httptest.NewRequest("GET", "/api/v1/ping", nil)
	req2.Header.Set("X-App-Version", "2.4.1+99")
	req2.Header.Set("X-Platform", "ios")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
}

// Web / admin / curl traffic carries neither header and is never gated.
func TestVersionCheck_NoMobileHeaders_NoOp(t *testing.T) {
	t.Setenv("MIN_VERSION_VENDOR_IOS", "9.9.9")
	r := newVersionRouter(nil)

	req := httptest.NewRequest("GET", "/api/v1/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

// An unrecognized platform is not gated (defensive: only ios/android count).
func TestVersionCheck_UnknownPlatform_NoOp(t *testing.T) {
	t.Setenv("MIN_VERSION_VENDOR_IOS", "9.9.9")
	r := newVersionRouter(nil)

	req := httptest.NewRequest("GET", "/api/v1/ping", nil)
	req.Header.Set("X-App-Version", "1.0.0")
	req.Header.Set("X-Platform", "windows-phone")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

// Excluded path prefixes bypass the check even for a stale client.
func TestVersionCheck_ExcludedPath_Bypasses(t *testing.T) {
	t.Setenv("MIN_VERSION_VENDOR_IOS", "9.9.9")
	r := newVersionRouter([]string{"/health"})

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("X-App-Version", "1.0.0")
	req.Header.Set("X-Platform", "ios")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSemverLess(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "1.0.1", true},
		{"1.0.1", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"1.2.0", "1.10.0", true}, // numeric, not lexical
		{"v1.0.0", "1.0.1", true}, // leading v stripped
		{"1.0.3+12", "1.0.3", false},
		{"1.0.3+12", "1.0.4", true}, // build suffix ignored, patch lower
		{"2.0.0", "10.0.0", true},   // major numeric
		{"1.0", "1.0.1", true},      // missing patch -> 0
		{"abc", "1.0.0", true},      // non-numeric -> 0.0.0
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.want, semverLess(tc.a, tc.b), "semverLess(%q,%q)", tc.a, tc.b)
	}
}

func TestParseSemver(t *testing.T) {
	assert.Equal(t, [3]int{1, 2, 3}, parseSemver("1.2.3"))
	assert.Equal(t, [3]int{1, 2, 3}, parseSemver("v1.2.3"))
	assert.Equal(t, [3]int{1, 2, 3}, parseSemver("1.2.3+45"))
	assert.Equal(t, [3]int{1, 2, 0}, parseSemver("1.2"))
	assert.Equal(t, [3]int{0, 0, 0}, parseSemver(""))
	assert.Equal(t, [3]int{1, 0, 0}, parseSemver("1.x.y")) // non-numeric segments -> 0
}

func TestMinVersionFromEnv(t *testing.T) {
	t.Setenv("MIN_VERSION_VENDOR_IOS", "4.5.6")
	assert.Equal(t, "4.5.6", minVersionFromEnv("vendor", "ios"))
	// Unset platform falls back to the permissive default.
	assert.Equal(t, "1.0.0", minVersionFromEnv("vendor", "macos"))
}
