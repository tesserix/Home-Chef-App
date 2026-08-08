package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// #1164 finding 3. The login challenge is keyed by this string, so what it
// returns decides whether two handsets on one account get their own code or
// fight over a single one.

func deviceCtx(headers map[string]string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/mfa/challenge", nil)
	for k, v := range headers {
		c.Request.Header.Set(k, v)
	}
	return c
}

func TestChallengeDevice_DistinguishesTwoInstallsOfOneApp(t *testing.T) {
	a := challengeDevice(deviceCtx(map[string]string{"X-Client-App": "customer", "X-Device-Id": "install-a"}))
	b := challengeDevice(deviceCtx(map[string]string{"X-Client-App": "customer", "X-Device-Id": "install-b"}))

	require.NotEmpty(t, a)
	require.NotEqual(t, a, b)
}

func TestChallengeDevice_SeparatesTheSameInstallAcrossApps(t *testing.T) {
	customer := challengeDevice(deviceCtx(map[string]string{"X-Client-App": "customer", "X-Device-Id": "same"}))
	vendor := challengeDevice(deviceCtx(map[string]string{"X-Client-App": "vendor", "X-Device-Id": "same"}))

	require.NotEqual(t, customer, vendor)
}

// A client that predates the header must land in the account-wide bucket rather
// than a device bucket it can never address again.
func TestChallengeDevice_EmptyWithoutTheHeader(t *testing.T) {
	require.Empty(t, challengeDevice(deviceCtx(map[string]string{"X-Client-App": "customer"})))
	require.Empty(t, challengeDevice(deviceCtx(map[string]string{"X-Device-Id": "   "})))
}
