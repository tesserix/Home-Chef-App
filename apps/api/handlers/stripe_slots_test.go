package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStripeSlotAliases(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"sandbox", "test"}, {"test", "test"}, {"prod", "live"}, {"production", "live"}, {"live", "live"}, {"", "live"}} {
		got, ok := stripeCredentialSlot(tc.input)
		require.True(t, ok)
		require.Equal(t, tc.want, got)
	}
	_, ok := stripeCredentialSlot("typo")
	require.False(t, ok)
}

func TestStripeAdminRejectsInvalidKeysBeforeSecretWrites(t *testing.T) {
	t.Setenv("FE3DR_SECRET_WRITES_PAUSED", "false")
	for _, body := range []string{
		`{"mode":"invalid","keyId":"mk_fixture"}`,
		`{"mode":"test","secretKey":"sk_live_fixture"}`,
		`{"mode":"test","publishableKey":"mk_not_a_publishable_key"}`,
		`{"mode":"live","keyId":"sk_test_not_an_id"}`,
		`{"mode":"live","secretKey":"sk_test_fixture","publishableKey":"pk_live_wrong"}`,
	} {
		r := gin.New()
		r.PUT("/keys", (&AdminHandler{}).UpdateStripeGatewayKeys)
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/keys", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		require.NotContains(t, w.Body.String(), "sk_")
	}
}
