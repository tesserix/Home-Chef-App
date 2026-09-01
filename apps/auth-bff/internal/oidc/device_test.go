package oidc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/auth-bff/internal/apiclient"
)

func issueWithDeviceCookie(t *testing.T, cookie string) (*httptest.ResponseRecorder, *fakeAPI) {
	t.Helper()
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "x@y.com")
	gin.SetMode(gin.TestMode)
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	h := newHandlers(t, api)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "http://admin.fe3dr.com/auth/callback", nil)
	req.RemoteAddr = "49.207.1.1:4444"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh) Safari/605.1")
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: browserDeviceCookie, Value: cookie})
	}
	c.Request = req
	app, err := h.Registry.ResolveByHost("admin.fe3dr.com")
	require.NoError(t, err)
	h.issueSession(c, app, adminClaims(true), StateEntry{})
	return w, api
}

func TestIssueSession_IssuesABrowserDeviceIDAndForwardsIt(t *testing.T) {
	w, api := issueWithDeviceCookie(t, "")

	require.Equal(t, http.StatusFound, w.Code)
	require.NotEmpty(t, api.lastReq.DeviceID)
	assert.Equal(t, "web", api.lastReq.Platform)
	assert.Equal(t, "49.207.1.1", api.lastReq.IP)
	assert.Contains(t, w.Header().Get("Set-Cookie"), browserDeviceCookie+"=")
}

// A returning browser must reuse its own id, or every visit would look like a
// new device and mail the user.
func TestIssueSession_ReusesAnExistingBrowserDeviceID(t *testing.T) {
	w, api := issueWithDeviceCookie(t, "known-browser")

	require.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "known-browser", api.lastReq.DeviceID)
}

// The cookie is attacker-writable, so a value that isn't one of ours is
// discarded rather than stored as a device id.
func TestIssueSession_RejectsAMalformedDeviceCookie(t *testing.T) {
	_, api := issueWithDeviceCookie(t, "<script>"+strings.Repeat("x", 300))

	require.NotContains(t, api.lastReq.DeviceID, "<script>")
	require.LessOrEqual(t, len(api.lastReq.DeviceID), 64)
}
