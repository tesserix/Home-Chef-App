package services

import (
	"time"

	"github.com/homechef/api/models"
)

// cashfree_inject.go — the narrow exported seam for injecting a Cashfree client,
// so tests in OTHER packages (handlers) can drive the money path against an
// httptest.Server with no live gateway and no GCP Secret Manager. Mirrors
// razorpay_inject.go exactly; production never calls these — it goes through
// GetCashfreeFor's Secret Manager path.

// NewCashfreeTestClient builds a CashfreeClient whose API host is baseURL (an
// httptest.Server) and whose credentials are caller-controlled, so a handler test
// can make order/payment/refund calls resolve against canned responses and
// VerifyCashfreeWebhookMode validate a test-computed HMAC.
//
// mode is stored so the client reports itself honestly, but baseURL takes
// precedence over the mode→host mapping — that is the whole point of the seam.
// fetchedAt is stamped now so GetCashfreeFor serves it within the cache TTL
// rather than re-reading Secret Manager. Test-only.
func NewCashfreeTestClient(baseURL, appID, secretKey, webhookSecret, mode string) *CashfreeClient {
	return &CashfreeClient{
		appID:         appID,
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		mode:          models.NormalizeMode(mode),
		baseURL:       baseURL,
		fetchedAt:     time.Now(),
	}
}

// SetCashfreeClient installs c as the LIVE-slot Cashfree client (nil to clear).
// Test-only. Targets live because handler tests deal in live orders unless they
// say otherwise; a test needing the sandbox slot calls SetCashfreeClientFor.
func SetCashfreeClient(c *CashfreeClient) {
	SetCashfreeClientFor(models.ChefModeLive, c)
}

// CashfreeAmountFromPaise exposes the paise→wire amount type for tests that
// build request fixtures. Not used in production code, where callers pass paise
// into the request structs directly.
func CashfreeAmountFromPaise(paise int) cashfreeAmount { return cashfreeAmount(paise) }
