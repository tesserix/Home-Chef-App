package services

// gateway_select_test.go — the Cashfree-preferred / Razorpay-fallback rule.
//
// The fallback exists because Cashfree separates sandbox from production by
// HOSTNAME: a set of test credentials authenticates against sandbox.cashfree.com
// and returns 401 against api.cashfree.com. So a platform that has made Cashfree
// its default while its LIVE slot is still unprovisioned would 503 every real
// checkout. These tests pin that behaviour down, because it is the difference
// between "preferred gateway" and "outage".

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A configured Cashfree slot is used for that mode.
func TestSelectCheckoutGateway_UsesCashfreeWhenConfigured(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "wh", models.ChefModeLive))

	require.Equal(t, models.PaymentProviderCashfree,
		SelectCheckoutGateway(models.PaymentProviderCashfree, models.ChefModeLive))
}

// THE OUTAGE GUARD. A Cashfree chef whose slot has no credentials for this mode
// falls back to Razorpay rather than failing the checkout. Without this, making
// Cashfree the default would take live payments down the moment the live slot
// wasn't provisioned yet.
func TestSelectCheckoutGateway_FallsBackToRazorpayWhenCashfreeUnconfigured(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderRazorpay,
		SelectCheckoutGateway(models.PaymentProviderCashfree, models.ChefModeLive))
}

// The two slots are independent: a configured TEST slot must not make a LIVE
// checkout believe Cashfree is available. Getting this wrong would route a real
// order at a gateway slot that 401s.
func TestSelectCheckoutGateway_SlotsAreIndependentPerMode(t *testing.T) {
	withCashfreeClient(t, models.ChefModeTest,
		NewCashfreeTestClient("", "app_t", "sk_t", "wh_t", models.ChefModeTest))
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderCashfree,
		SelectCheckoutGateway(models.PaymentProviderCashfree, models.ChefModeTest))
	require.Equal(t, models.PaymentProviderRazorpay,
		SelectCheckoutGateway(models.PaymentProviderCashfree, models.ChefModeLive),
		"a configured test slot must not stand in for an unconfigured live one")
}

// Stripe must NOT fall back. A Stripe chef settles in a non-INR currency from a
// Connect country, so degrading to Razorpay would charge the wrong currency
// instead of failing honestly.
func TestSelectCheckoutGateway_StripeNeverFallsBack(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderStripe,
		SelectCheckoutGateway(models.PaymentProviderStripe, models.ChefModeLive))
}

// A blank or unknown provider is an unstamped historical row, which really is
// Razorpay — it must NOT be reinterpreted as the new preferred gateway.
func TestSelectCheckoutGateway_BlankStaysRazorpay(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "wh", models.ChefModeLive))

	require.Equal(t, models.PaymentProviderRazorpay, SelectCheckoutGateway("", models.ChefModeLive))
	require.Equal(t, models.PaymentProviderRazorpay, SelectCheckoutGateway("nonsense", models.ChefModeLive))
}

// A new chef gets the preferred gateway when it is usable for their mode.
func TestDefaultChefPaymentProvider_PrefersCashfreeWhenConfigured(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "wh", models.ChefModeLive))

	require.Equal(t, models.PaymentProviderCashfree, DefaultChefPaymentProvider(models.ChefModeLive))
	require.Equal(t, models.PreferredChefPaymentProvider, DefaultChefPaymentProvider(models.ChefModeLive))
}

// ...and Razorpay when it is not, so a brand-new kitchen is never created unable
// to take a payment. The chef, not the admin, is who would see that failure.
func TestDefaultChefPaymentProvider_FallsBackWhenUnconfigured(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderRazorpay, DefaultChefPaymentProvider(models.ChefModeLive))
}
