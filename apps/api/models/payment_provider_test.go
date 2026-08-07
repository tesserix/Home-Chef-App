package models

// payment_provider_test.go — the provider vocabulary.
//
// These predicates replaced ~40 inline string comparisons, so each one is now a
// single point of failure for a whole class of money routing. The tests below pin
// the two asymmetries that a "tidy this up" refactor would most plausibly break.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Blank and unknown resolve to cashfree since #1086 — the only INR gateway the
// platform operates. The historical-row argument for razorpay is spent: every
// provider column in production is explicitly stamped, and razorpay is now a
// recognised value in its own right (below) rather than the catch-all.
func TestNormalizeProvider_UnknownFallsBackToCashfree(t *testing.T) {
	for _, in := range []string{"", "   ", "razorpy", "RAZORPAY!", "unknown"} {
		require.Equal(t, PaymentProviderCashfree, NormalizeProvider(in), "input %q", in)
	}
}

// The load-bearing half of that flip: an EXPLICIT razorpay must still come back
// as razorpay. Letting it fall through to the new default would re-label 21 live
// historical orders as Cashfree and route their refunds at a gateway that never
// took the money.
func TestNormalizeProvider_ExplicitRazorpayIsPreserved(t *testing.T) {
	for _, in := range []string{"razorpay", "RAZORPAY", "  Razorpay "} {
		require.Equal(t, PaymentProviderRazorpay, NormalizeProvider(in), "input %q", in)
	}
}

func TestNormalizeProvider_RecognisesKnownProvidersCaseInsensitively(t *testing.T) {
	require.Equal(t, PaymentProviderCashfree, NormalizeProvider("cashfree"))
	require.Equal(t, PaymentProviderCashfree, NormalizeProvider("  CashFree "))
	require.Equal(t, PaymentProviderStripe, NormalizeProvider("STRIPE"))
	require.Equal(t, PaymentProviderWallet, NormalizeProvider("Wallet"))
	require.Equal(t, PaymentProviderRazorpay, NormalizeProvider("Razorpay"))
}

func TestPreferredProvider_IsCashfree(t *testing.T) {
	require.Equal(t, PaymentProviderCashfree, PreferredChefPaymentProvider)
}

// Credit rails are INR-denominated: Cashfree takes them exactly as Razorpay does,
// only Stripe does not. The old `!EqualFold(provider,"stripe")` spelling happened
// to be right; naming Razorpay instead would have disabled wallet and loyalty on
// every Cashfree order.
func TestUsesINRPaise_OnlyStripeIsExcluded(t *testing.T) {
	require.True(t, UsesINRPaise(PaymentProviderRazorpay))
	require.True(t, UsesINRPaise(PaymentProviderCashfree))
	require.True(t, UsesINRPaise(PaymentProviderWallet))
	require.False(t, UsesINRPaise(PaymentProviderStripe))
}

func TestIsGatewayProvider_WalletIsTheOnlyNonGateway(t *testing.T) {
	require.True(t, IsGatewayProvider(PaymentProviderRazorpay))
	require.True(t, IsGatewayProvider(PaymentProviderCashfree))
	require.True(t, IsGatewayProvider(PaymentProviderStripe))
	require.False(t, IsGatewayProvider(PaymentProviderWallet))
}

// Selection is strict — no coercion — so a typo surfaces instead of being saved
// as razorpay. Wallet is not selectable: it is an outcome, never a configuration.
func TestIsSelectableChefProvider_StrictAndExcludesWallet(t *testing.T) {
	require.True(t, IsSelectableChefProvider("cashfree"))
	require.True(t, IsSelectableChefProvider("stripe"))
	// razorpay was removed from the selectable set by #1086 — it stays READABLE
	// off a historical row but nothing may be newly configured onto it.
	require.False(t, IsSelectableChefProvider("razorpay"))
	require.False(t, IsSelectableChefProvider("wallet"))
	require.False(t, IsSelectableChefProvider("razorpy"))
	require.False(t, IsSelectableChefProvider(""))
	require.Contains(t, SelectableChefProviders(), PreferredChefPaymentProvider,
		"the preferred provider must actually be selectable")
}

// THE REFUND REFERENCE. Each gateway refunds a different object, and using the
// wrong id is a silent failure: Cashfree refunds an ORDER (no endpoint takes its
// payment id), Razorpay refunds a PAYMENT, Stripe a PaymentIntent.
func TestGatewayRefundReference_PerProviderObject(t *testing.T) {
	base := Order{
		RazorpayOrderID:       "cf_or_rzp_order",
		RazorpayPaymentID:     "pay_rzp",
		StripePaymentIntentID: "pi_stripe",
	}

	rzp := base
	rzp.PaymentProvider = PaymentProviderRazorpay
	require.Equal(t, "pay_rzp", rzp.GatewayRefundReference())

	cf := base
	cf.PaymentProvider = PaymentProviderCashfree
	require.Equal(t, "cf_or_rzp_order", cf.GatewayRefundReference(),
		"Cashfree refunds are order-scoped, so the ORDER id is the reference")

	st := base
	st.PaymentProvider = PaymentProviderStripe
	require.Equal(t, "pi_stripe", st.GatewayRefundReference())

	wallet := base
	wallet.PaymentProvider = PaymentProviderWallet
	require.Empty(t, wallet.GatewayRefundReference(), "a wallet refund is a ledger credit, not a gateway call")

	// Unstamped normalizes to Cashfree, which refunds against the ORDER id.
	unstamped := base
	require.Equal(t, "cf_or_rzp_order", unstamped.GatewayRefundReference())
}

// GatewayRefundable is what the five former `!= "razorpay"` guards now ask. A
// Cashfree order with only an order id must answer TRUE — answering false is what
// would have silently stopped refunding those customers.
func TestGatewayRefundable_CashfreeOrderWithOnlyAnOrderIDIsRefundable(t *testing.T) {
	cf := Order{PaymentProvider: PaymentProviderCashfree, RazorpayOrderID: "cf-order-1"}
	require.True(t, cf.GatewayRefundable())

	// A Razorpay order needs the PAYMENT id — an order id alone is not refundable,
	// because the money may never have been captured.
	rzpOrderOnly := Order{PaymentProvider: PaymentProviderRazorpay, RazorpayOrderID: "order_x"}
	require.False(t, rzpOrderOnly.GatewayRefundable())

	rzpPaid := Order{PaymentProvider: PaymentProviderRazorpay, RazorpayPaymentID: "pay_x"}
	require.True(t, rzpPaid.GatewayRefundable())

	require.False(t, (&Order{}).GatewayRefundable(), "an unpaid order has nothing to refund")
	require.False(t, (*Order)(nil).GatewayRefundable())
}
