package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecretMutationHandlersPauseBeforeParsingOrSideEffects(t *testing.T) {
	t.Setenv("APP_SECRET_WRITES_PAUSED", "true")
	gin.SetMode(gin.TestMode)
	for name, handler := range map[string]gin.HandlerFunc{
		"vendor payout":      (&ChefHandler{}).SavePayoutDetails,
		"driver payout":      (&DriverOnboardingHandler{}).DriverOnboardingPayout,
		"cashfree checkout":  (&AdminHandler{}).UpdateCashfreeGatewayKeys,
		"stripe checkout":    (&AdminHandler{}).UpdateStripeGatewayKeys,
		"cashfree payouts":   (&AdminPayoutRailHandler{}).UpdateCashfreePayoutKeys,
		"settlement account": (&AdminPayoutRailHandler{}).SetPlatformSettlementAccount,
		"sandbox bank":       (&AdminPayoutRailHandler{}).SeedChefTestBankAccount,
	} {
		t.Run(name, func(t *testing.T) {
			router := gin.New()
			router.POST("/mutation", handler)
			result := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/mutation", strings.NewReader("invalid json"))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(result, request)
			if result.Code != http.StatusServiceUnavailable {
				t.Fatalf("status %d, want 503", result.Code)
			}
			if result.Header().Get("Retry-After") == "" {
				t.Fatal("missing retry guidance")
			}
			if !strings.Contains(result.Body.String(), "SECRET_WRITES_PAUSED") {
				t.Fatal("missing pause error code")
			}
		})
	}
}
