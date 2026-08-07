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

// Blank and unknown resolve to cashfree — the only INR gateway the platform
// operates. The retired gateway's name is now just another unknown string
// (#1132): no code can reach it, so recognising it bought nothing.
func TestNormalizeProvider_UnknownFallsBackToCashfree(t *testing.T) {
	for _, in := range []string{"", "   ", "razorpy", "legacy", "unknown"} {
		require.Equal(t, PaymentProviderCashfree, NormalizeProvider(in), "input %q", in)
	}
}

func TestNormalizeProvider_RecognisesKnownProvidersCaseInsensitively(t *testing.T) {
	require.Equal(t, PaymentProviderCashfree, NormalizeProvider("cashfree"))
	require.Equal(t, PaymentProviderCashfree, NormalizeProvider("  CashFree "))
	require.Equal(t, PaymentProviderStripe, NormalizeProvider("STRIPE"))
	require.Equal(t, PaymentProviderWallet, NormalizeProvider("Wallet"))
}

// NormalizeProvider coerces, which is right for reading a row and wrong for
// deciding whether to ACT on it: coercion alone would send a retired-gateway
// order into the Cashfree verify leg carrying an id Cashfree never issued. This
// is the predicate that keeps the refusal (#1132).
func TestIsKnownProvider_RefusesARetiredOrUnstampedValue(t *testing.T) {
	require.True(t, IsKnownProvider(PaymentProviderCashfree))
	require.True(t, IsKnownProvider("  CashFree "))
	require.True(t, IsKnownProvider(PaymentProviderStripe))
	require.True(t, IsKnownProvider(PaymentProviderWallet))

	require.False(t, IsKnownProvider("razorpay"), "the retired gateway has no client left to ask")
	require.False(t, IsKnownProvider(""))
	require.False(t, IsKnownProvider("nonsense"))
}

func TestPreferredProvider_IsCashfree(t *testing.T) {
	require.Equal(t, PaymentProviderCashfree, PreferredChefPaymentProvider)
}

// Credit rails are INR-denominated; only Stripe is not. Naming the gateway
// rather than the currency is what would have disabled wallet and loyalty on
// every Cashfree order.
func TestUsesINRPaise_OnlyStripeIsExcluded(t *testing.T) {
	require.True(t, UsesINRPaise(PaymentProviderCashfree))
	require.True(t, UsesINRPaise(PaymentProviderWallet))
	require.False(t, UsesINRPaise(PaymentProviderStripe))
}

func TestIsGatewayProvider_WalletIsTheOnlyNonGateway(t *testing.T) {
	require.True(t, IsGatewayProvider(PaymentProviderCashfree))
	require.True(t, IsGatewayProvider(PaymentProviderStripe))
	require.False(t, IsGatewayProvider(PaymentProviderWallet))
}

// Selection is strict — no coercion — so a typo surfaces instead of being saved
// as a live provider. Wallet is not selectable: it is an outcome, never config.
func TestIsSelectableChefProvider_StrictAndExcludesWallet(t *testing.T) {
	require.True(t, IsSelectableChefProvider("cashfree"))
	require.True(t, IsSelectableChefProvider("stripe"))
	require.False(t, IsSelectableChefProvider("wallet"))
	require.False(t, IsSelectableChefProvider("cashfre"))
	require.False(t, IsSelectableChefProvider(""))
	require.Contains(t, SelectableChefProviders(), PreferredChefPaymentProvider,
		"the preferred provider must actually be selectable")
}

// THE REFUND REFERENCE. Each gateway refunds a different object, and using the
// wrong id is a silent failure: Cashfree refunds an ORDER (no endpoint takes its
// payment id), Stripe a PaymentIntent.
func TestGatewayRefundReference_PerProviderObject(t *testing.T) {
	base := Order{
		GatewayOrderID:        "cf_order",
		GatewayPaymentID:      "cf_pay",
		StripePaymentIntentID: "pi_stripe",
	}

	cf := base
	cf.PaymentProvider = PaymentProviderCashfree
	require.Equal(t, "cf_order", cf.GatewayRefundReference(),
		"Cashfree refunds are order-scoped, so the ORDER id is the reference")

	st := base
	st.PaymentProvider = PaymentProviderStripe
	require.Equal(t, "pi_stripe", st.GatewayRefundReference())

	wallet := base
	wallet.PaymentProvider = PaymentProviderWallet
	require.Empty(t, wallet.GatewayRefundReference(), "a wallet refund is a ledger credit, not a gateway call")

	// Unstamped normalizes to Cashfree, which refunds against the ORDER id.
	unstamped := base
	require.Equal(t, "cf_order", unstamped.GatewayRefundReference())
}

// The 21 orders and 6 meal plans stamped with the retired gateway keep that
// string — restamping them would falsify which rail took the money (#1122). None
// of them carries a gateway order id, so the reference resolves empty and the
// refund path falls to store credit rather than calling Cashfree with an id it
// never issued. This is the guarantee that makes deleting the constant safe.
func TestGatewayRefundReference_ARetiredGatewayRowIsNotRefundableAtCashfree(t *testing.T) {
	legacy := Order{PaymentProvider: "razorpay", GatewayPaymentID: "pay_legacy"}
	require.Empty(t, legacy.GatewayRefundReference())
	require.False(t, legacy.GatewayRefundable(),
		"no id issued by the retired gateway may be presented to Cashfree")
}

// A Cashfree order with only an order id must answer TRUE — answering false is
// what a gateway-naming guard did, and it silently stopped refunding customers.
func TestGatewayRefundable_CashfreeOrderWithOnlyAnOrderIDIsRefundable(t *testing.T) {
	cf := Order{PaymentProvider: PaymentProviderCashfree, GatewayOrderID: "cf-order-1"}
	require.True(t, cf.GatewayRefundable())

	require.False(t, (&Order{}).GatewayRefundable(), "an unpaid order has nothing to refund")
	require.False(t, (*Order)(nil).GatewayRefundable())
}
