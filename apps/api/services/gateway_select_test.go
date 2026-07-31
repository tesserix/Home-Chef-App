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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// healthyCashfree installs a slot whose health check PASSES.
//
// Selection now probes before claiming a gateway (see cashfreeUsableFor), so a
// client pointed at the real host would fail the probe and every test would read
// as "unusable". The stub returns 404 on the sentinel beneficiary — which is what
// a correctly authenticated Cashfree replies, and therefore what HealthCheck
// treats as healthy.
func healthyCashfree(t *testing.T, mode string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"order_not_found","message":"no such order"}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { cashfreeHealth.Delete(mode); cashfreeGatewayBreaker.Delete(mode) })
	cashfreeHealth.Delete(mode)
	cashfreeGatewayBreaker.Delete(mode)
	withCashfreeClient(t, mode, NewCashfreeTestClient(srv.URL, "app", "sk", "wh", mode))
}

// A configured Cashfree slot is used for that mode.
func TestSelectCheckoutGateway_UsesCashfreeWhenConfigured(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)

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
	healthyCashfree(t, models.ChefModeTest)
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

// Cashfree is the platform default for EVERY INR kitchen, including ones whose
// stored provider is razorpay or blank.
//
// That stored value is not a choice the chef made — it is what every row held
// before there was an alternative — so honouring it would strand the entire
// existing estate on the old gateway. Overriding it is safe because the ORDER,
// not the chef, is authoritative afterwards: the gateway that takes the payment
// is stamped on the order, and refunds and reconciliation read that.
func TestSelectCheckoutGateway_PrefersCashfreeForExistingRazorpayChefs(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)

	require.Equal(t, models.PaymentProviderCashfree, SelectCheckoutGateway("", models.ChefModeLive))
	require.Equal(t, models.PaymentProviderCashfree,
		SelectCheckoutGateway(models.PaymentProviderRazorpay, models.ChefModeLive))
	require.Equal(t, models.PaymentProviderCashfree, SelectCheckoutGateway("nonsense", models.ChefModeLive))
}

// The stored-value MEANING is untouched by that preference. NormalizeProvider
// still reads a blank row as razorpay, which is what keeps historical orders'
// refunds routed to the gateway that actually took their money.
func TestSelectCheckoutGateway_DoesNotChangeStoredProviderMeaning(t *testing.T) {
	require.Equal(t, models.PaymentProviderRazorpay, models.NormalizeProvider(""))
	require.Equal(t, models.PaymentProviderRazorpay, models.NormalizeProvider("nonsense"))
}

// A slot that just failed to create an order is skipped until the cooldown
// lapses, so a broken gateway costs ONE checkout a round-trip rather than every
// checkout — which matters now that Cashfree is tried first for everyone.
func TestSelectCheckoutGateway_BreakerSkipsAFailingSlot(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)
	t.Cleanup(func() { cashfreeGatewayBreaker.Delete(models.ChefModeLive) })

	require.Equal(t, models.PaymentProviderCashfree, SelectCheckoutGateway("", models.ChefModeLive))

	NoteCashfreeGatewayFailure(models.ChefModeLive)
	require.Equal(t, models.PaymentProviderRazorpay, SelectCheckoutGateway("", models.ChefModeLive),
		"a slot that just failed must be skipped, not retried per checkout")

	// The breaker is scoped per mode — a failing live slot must not disable test.
	healthyCashfree(t, models.ChefModeTest)
	require.Equal(t, models.PaymentProviderCashfree, SelectCheckoutGateway("", models.ChefModeTest))
}

// A new chef gets the preferred gateway when it is usable for their mode.
func TestDefaultChefPaymentProvider_PrefersCashfreeWhenConfigured(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)

	require.Equal(t, models.PaymentProviderCashfree, DefaultChefPaymentProvider(models.ChefModeLive))
	require.Equal(t, models.PreferredChefPaymentProvider, DefaultChefPaymentProvider(models.ChefModeLive))
}

// ...and Razorpay when it is not, so a brand-new kitchen is never created unable
// to take a payment. The chef, not the admin, is who would see that failure.
func TestDefaultChefPaymentProvider_FallsBackWhenUnconfigured(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.Equal(t, models.PaymentProviderRazorpay, DefaultChefPaymentProvider(models.ChefModeLive))
}

// A configured slot whose credentials do NOT work must read as unusable.
//
// This is the case that was previously wrong and customer-visible: the delivery
// quote names the payment aggregator in the checkout's RBI PA disclosure, so a
// slot that is present-but-401ing would have had the page claim Cashfree while
// the charge silently fell back to Razorpay — naming the wrong processor on a
// regulatory disclosure.
func TestSelectCheckoutGateway_UnhealthySlotIsNotClaimed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"authentication_failed","message":"authentication Failed"}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		cashfreeHealth.Delete(models.ChefModeLive)
		cashfreeGatewayBreaker.Delete(models.ChefModeLive)
	})
	cashfreeHealth.Delete(models.ChefModeLive)
	cashfreeGatewayBreaker.Delete(models.ChefModeLive)
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient(srv.URL, "app", "sk", "wh", models.ChefModeLive))

	require.Equal(t, models.PaymentProviderRazorpay,
		SelectCheckoutGateway("", models.ChefModeLive),
		"a slot with credentials that 401 must not be claimed as the gateway")
}

// The probe result is cached, so selection costs at most one request per mode
// per cooldown rather than one per checkout.
func TestSelectCheckoutGateway_HealthProbeIsCached(t *testing.T) {
	var probes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probes++
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		cashfreeHealth.Delete(models.ChefModeLive)
		cashfreeGatewayBreaker.Delete(models.ChefModeLive)
	})
	cashfreeHealth.Delete(models.ChefModeLive)
	cashfreeGatewayBreaker.Delete(models.ChefModeLive)
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient(srv.URL, "app", "sk", "wh", models.ChefModeLive))

	for i := 0; i < 5; i++ {
		require.Equal(t, models.PaymentProviderCashfree, SelectCheckoutGateway("", models.ChefModeLive))
	}
	require.Equal(t, 1, probes, "five checkouts must share one health probe")
}
