package handlers

// cancellation_refund_destination_test.go — a cancellation refund must go back to
// the customer's ORIGINAL payment method.
//
// The customer used to pick wallet-vs-original and the picker defaulted to
// "wallet", so store credit became the normal outcome of a cancellation. That is
// money the customer cannot use: WalletCheckoutEnabled (WALLET_CHECKOUT_ENABLED,
// #141) is off in production, so wallet credit can't be applied at checkout, and
// nothing is ever refunded at the gateway — Razorpay showed 0 refunds while the
// app told the customer they'd been "refunded to your wallet".
//
// These pin the rule: a Razorpay-paid order always refunds to the gateway, and
// wallet survives only as the fallback where there is no gateway payment to
// refund against (ExecuteCancellationRefund hard-errors on "original" there).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestResolveRefundDestination_RazorpayPaid_GoesToOriginal(t *testing.T) {
	order := &models.Order{
		ID:                uuid.New(),
		PaymentProvider:   "razorpay",
		RazorpayPaymentID: "pay_TEQWrxHzKeC1gA",
	}

	require.Equal(t, "original", resolveRefundDestination(order),
		"a gateway-paid order must refund to the card, not to unspendable store credit")
}

func TestResolveRefundDestination_ProviderCaseInsensitive(t *testing.T) {
	order := &models.Order{
		ID:                uuid.New(),
		PaymentProvider:   "Razorpay",
		RazorpayPaymentID: "pay_TEQGrsIslhLEFI",
	}

	require.Equal(t, "original", resolveRefundDestination(order),
		"provider casing must not silently divert a refund to the wallet")
}

// No gateway payment to refund against → wallet is the only possible destination.
// ExecuteCancellationRefund errors on "original" here, so this keeps that
// unreachable rather than failing the refund outright.
func TestResolveRefundDestination_NoGatewayPayment_FallsBackToWallet(t *testing.T) {
	order := &models.Order{ID: uuid.New(), PaymentProvider: "wallet"}

	require.Equal(t, "wallet", resolveRefundDestination(order))
}

func TestResolveRefundDestination_RazorpayProviderButNoPaymentID_FallsBackToWallet(t *testing.T) {
	// Provider says razorpay but nothing was captured — refunding "original"
	// would hard-error in ExecuteCancellationRefund.
	order := &models.Order{ID: uuid.New(), PaymentProvider: "razorpay"}

	require.Equal(t, "wallet", resolveRefundDestination(order))
}

func TestResolveRefundDestination_EmptyProvider_FallsBackToWallet(t *testing.T) {
	order := &models.Order{ID: uuid.New()}

	require.Equal(t, "wallet", resolveRefundDestination(order))
}

// The regression that bit us: a customer-supplied destination must not be able to
// divert a gateway-paid refund into the wallet. The handler no longer reads the
// field, so this asserts the resolver ignores everything except the order.
func TestResolveRefundDestination_IgnoresClientPreference(t *testing.T) {
	order := &models.Order{
		ID:                uuid.New(),
		PaymentProvider:   "razorpay",
		RazorpayPaymentID: "pay_TEQBJGrRLOPJKy",
	}

	// Whatever an older client sends, the destination is derived from the order.
	require.Equal(t, "original", resolveRefundDestination(order),
		"a stale client sending refundDestination=wallet must not trap the customer's money")
}
