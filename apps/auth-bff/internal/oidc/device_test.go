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
	"github.com/homechef/auth-bff/internal/gip"
)

func exchangeWithDeviceCookie(t *testing.T, cookie string) (*httptest.ResponseRecorder, *fakeAPI) {
	t.Helper()
	t.Setenv("HOMECHEF_ADMIN_ALLOWED_EMAILS", "x@y.com")
	ver := &fakeVerifier{tok: &gip.VerifiedToken{
		UID: "g1", Email: "x@y.com", TenantID: "HomeChef-Internal-gyofe", Provider: "password",
		Claims: map[string]any{
			"sub": "g1", "email": "x@y.com",
			"firebase": map[string]any{"sign_in_provider": "password", "tenant": "HomeChef-Internal-gyofe"},
		},
	}}
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	r := gin.New()
	newHandlers(t, ver, api).Register(r)

	req := httptest.NewRequest("POST", "http://admin.fe3dr.com/auth/exchange", strings.NewReader(`{"id_token":"valid"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "49.207.1.1:4444"
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh) Safari/605.1")
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: browserDeviceCookie, Value: cookie})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, api
}

func TestExchange_IssuesABrowserDeviceIDAndForwardsIt(t *testing.T) {
	w, api := exchangeWithDeviceCookie(t, "")

	require.Equal(t, http.StatusOK, w.Code)
	require.NotEmpty(t, api.lastReq.DeviceID)
	assert.Equal(t, "web", api.lastReq.Platform)
	assert.Equal(t, "49.207.1.1", api.lastReq.IP)
	assert.Contains(t, w.Header().Get("Set-Cookie"), browserDeviceCookie+"=")
}

// A returning browser must reuse its own id, or every visit would look like a
// new device and mail the user.
func TestExchange_ReusesAnExistingBrowserDeviceID(t *testing.T) {
	w, api := exchangeWithDeviceCookie(t, "known-browser")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "known-browser", api.lastReq.DeviceID)
}

// The cookie is attacker-writable, so a value that isn't one of ours is
// discarded rather than stored as a device id.
func TestExchange_RejectsAMalformedDeviceCookie(t *testing.T) {
	_, api := exchangeWithDeviceCookie(t, "<script>"+strings.Repeat("x", 300))

	require.NotContains(t, api.lastReq.DeviceID, "<script>")
	require.LessOrEqual(t, len(api.lastReq.DeviceID), 64)
}
