package services

// gateway_select_cashfree_only_test.go — #1086 Phase 5, step 2. Cashfree is the
// only gateway a new INR charge may be created on. The Razorpay fallback these
// tests replace was load-bearing while the live Cashfree slot was unprovisioned;
// with the merchant account live and zero paid Razorpay orders in production it
// is now the one path that could still mint a Razorpay order.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The whole point of the phase: no input to the checkout selector produces
// razorpay any more, including the stored value the existing estate carries.
func TestSelectCheckoutGateway_NeverSelectsRazorpay(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)

	for _, configured := range []string{"", models.PaymentProviderRazorpay, models.PaymentProviderCashfree, "nonsense"} {
		require.Equal(t, models.PaymentProviderCashfree,
			SelectCheckoutGateway(configured, models.ChefModeLive), "configured=%q", configured)
	}
}

// An unusable slot no longer degrades onto a gateway we are retiring. The
// checkout fails at the Cashfree call instead — loudly, at the point of failure,
// rather than by quietly taking the money somewhere else.
func TestSelectCheckoutGateway_StaysOnCashfreeWhenTheSlotIsUnusable(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderCashfree,
		SelectCheckoutGateway(models.PaymentProviderCashfree, models.ChefModeLive))
}

// A failing slot must not reroute new charges either — the breaker still exists
// for the PA disclosure, but it has no second gateway to send anyone to.
func TestSelectCheckoutGateway_BreakerDoesNotRerouteToRazorpay(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)
	t.Cleanup(func() { cashfreeGatewayBreaker.Delete(models.ChefModeLive) })

	NoteCashfreeGatewayFailure(models.ChefModeLive)

	require.Equal(t, models.PaymentProviderCashfree,
		SelectCheckoutGateway("", models.ChefModeLive))
}

// Stripe is untouched — it is a different currency and a different country, not
// a fallback for India.
func TestSelectCheckoutGateway_StripeSurvivesTheRazorpayRemoval(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderStripe,
		SelectCheckoutGateway(models.PaymentProviderStripe, models.ChefModeLive))
}

// A new kitchen is created on the only gateway that can charge for it, whatever
// the slot's current health says.
func TestDefaultChefPaymentProvider_IsAlwaysCashfree(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderCashfree, DefaultChefPaymentProvider(models.ChefModeLive))
}

// Razorpay can no longer be configured on a chef — the strict validator is what
// every admin and chef-facing provider switch goes through.
func TestSelectableChefProviders_ExcludesRazorpay(t *testing.T) {
	require.NotContains(t, models.SelectableChefProviders(), models.PaymentProviderRazorpay)
	require.False(t, models.IsSelectableChefProvider(models.PaymentProviderRazorpay))
	require.True(t, models.IsSelectableChefProvider(models.PaymentProviderCashfree))
	require.True(t, models.IsSelectableChefProvider(models.PaymentProviderStripe))
}

// Stored razorpay rows keep their meaning: they are a factual record of the
// gateway that took the money, and a refund still has to be routed there.
func TestNormalizeProvider_StillReadsAStoredRazorpayRow(t *testing.T) {
	require.Equal(t, models.PaymentProviderRazorpay,
		models.NormalizeProvider(models.PaymentProviderRazorpay))
}
