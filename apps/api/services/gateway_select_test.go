package services

// gateway_select_test.go — which gateway takes a new charge, and whether a slot
// is usable at all.
//
// The Razorpay fallback these tests used to pin was removed deliberately by
// #1086: Cashfree is now the only gateway an INR charge may be created on, so
// "fall back to razorpay" describes behaviour that no longer exists and the
// assertions asserting it were rewritten rather than kept passing. Selection
// itself is covered by gateway_select_cashfree_only_test.go; what survives here
// is the slot-usability machinery, which still decides whether a checkout is
// about to fail and is still per-mode.
//
// Usability matters because Cashfree separates sandbox from production by
// HOSTNAME: test credentials authenticate against sandbox.cashfree.com and 401
// against api.cashfree.com, so a slot can be fully configured and still not work.

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

// The two slots are independent: a configured TEST slot must not make a LIVE
// checkout read as usable. Getting this wrong would report a real order's
// gateway as ready when it is about to 401.
func TestCashfreeUsableFor_SlotsAreIndependentPerMode(t *testing.T) {
	healthyCashfree(t, models.ChefModeTest)
	withCashfreeClient(t, models.ChefModeLive, nil)

	require.True(t, cashfreeUsableFor(models.ChefModeTest))
	require.False(t, cashfreeUsableFor(models.ChefModeLive),
		"a configured test slot must not stand in for an unconfigured live one")
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

// Whatever this returns is ALSO what gets stamped on the order, so it has to be
// a value NormalizeProvider reads back unchanged — a selection that normalized to
// something else would route the refund at a gateway that never took the money.
// (What each stored value means is pinned in models/payment_provider_test.go.)
func TestSelectCheckoutGateway_ReturnsAValueThatRoundTrips(t *testing.T) {
	for _, configured := range []string{"", "nonsense", models.PaymentProviderRazorpay,
		models.PaymentProviderCashfree, models.PaymentProviderStripe} {
		got := SelectCheckoutGateway(configured, models.ChefModeLive)
		require.Equal(t, got, models.NormalizeProvider(got), "configured %q", configured)
	}
}

// A slot that just failed to create an order reads as unusable until the
// cooldown lapses, so a broken gateway costs ONE checkout a round-trip rather
// than every checkout.
func TestCashfreeUsableFor_BreakerMarksAFailingSlot(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)
	t.Cleanup(func() { cashfreeGatewayBreaker.Delete(models.ChefModeLive) })

	require.True(t, cashfreeUsableFor(models.ChefModeLive))

	NoteCashfreeGatewayFailure(models.ChefModeLive)
	require.False(t, cashfreeUsableFor(models.ChefModeLive),
		"a slot that just failed must be treated as unusable, not re-probed per checkout")

	// The breaker is scoped per mode — a failing live slot must not disable test.
	healthyCashfree(t, models.ChefModeTest)
	require.True(t, cashfreeUsableFor(models.ChefModeTest))
}

// A new chef gets the preferred gateway when it is usable for their mode.
func TestDefaultChefPaymentProvider_PrefersCashfreeWhenConfigured(t *testing.T) {
	healthyCashfree(t, models.ChefModeLive)

	require.Equal(t, models.PaymentProviderCashfree, DefaultChefPaymentProvider(models.ChefModeLive))
	require.Equal(t, models.PreferredChefPaymentProvider, DefaultChefPaymentProvider(models.ChefModeLive))
}

// A configured slot whose credentials do NOT work must read as unusable, so the
// operator sees a named gateway failure instead of a run of unexplained 500s on
// the checkout endpoint.
func TestCashfreeUsableFor_UnhealthySlotIsNotClaimed(t *testing.T) {
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

	require.False(t, cashfreeUsableFor(models.ChefModeLive),
		"a slot with credentials that 401 must not be claimed as usable")
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
