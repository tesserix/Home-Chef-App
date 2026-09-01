package autologin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/auth-bff/internal/apiclient"
	"github.com/homechef/auth-bff/internal/zitadel"
)

func postAutoLogin(t *testing.T, deps *Deps, body, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	NewHandler(deps).Register(r)
	req := httptest.NewRequest("POST", "/auth/auto-login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func deviceDeps(t *testing.T) (*Deps, *fakeAPI) {
	t.Helper()
	api := &fakeAPI{resp: &apiclient.UpsertUserResponse{UserID: "u1"}}
	deps := newDeps(t,
		&fakeVerifier{tok: &zitadel.VerifiedToken{
			Subject: "z1", Email: "x@y.com", Claims: map[string]any{},
		}},
		api,
		&fakeSessions{encoded: "sess"},
	)
	return deps, api
}

func TestAutoLogin_ForwardsTheDeviceDescription(t *testing.T) {
	deps, api := deviceDeps(t)

	w := postAutoLogin(t, deps, `{"id_token":"valid.test.token","pool":"customer",
		"device_id":"device-a","platform":"ios","device_label":"iPhone 17","app_version":"1.4.0"}`, "49.207.1.1:5555")

	require.Equal(t, 200, w.Code)
	require.True(t, api.captured)
	assert.Equal(t, "device-a", api.lastReq.DeviceID)
	assert.Equal(t, "ios", api.lastReq.Platform)
	assert.Equal(t, "iPhone 17", api.lastReq.DeviceLabel)
	assert.Equal(t, "1.4.0", api.lastReq.AppVersion)
}

// The IP is the one thing here the client must not get to choose: it drives the
// location shown in a security email, so it comes from the connection.
func TestAutoLogin_TakesTheIPFromTheConnectionNotTheBody(t *testing.T) {
	deps, api := deviceDeps(t)

	w := postAutoLogin(t, deps, `{"id_token":"valid.test.token","pool":"customer",
		"device_id":"device-a","ip":"8.8.8.8"}`, "49.207.1.1:5555")

	require.Equal(t, 200, w.Code)
	assert.Equal(t, "49.207.1.1", api.lastReq.IP)
}

// Apps built before the device change send no device fields; they must still
// sign in.
func TestAutoLogin_WorksWithoutAnyDeviceFields(t *testing.T) {
	deps, api := deviceDeps(t)

	w := postAutoLogin(t, deps,
		`{"id_token":"valid.test.token","pool":"customer"}`, "49.207.1.1:5555")

	require.Equal(t, 200, w.Code)
	assert.Empty(t, api.lastReq.DeviceID)
}
