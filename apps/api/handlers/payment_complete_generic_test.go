package handlers

// payment_complete_generic_test.go — #555. verifyStripePayment and settleFullWalletOrder
// used to run an UNCONDITIONAL completion UPDATE + an UNCONDITIONAL order.paid emit, so a
// re-verify or a verify/webhook race double-emitted order.paid (and double-pushed the chef
// "new order") — the same defect #553 fixed for Razorpay. Both now route through the
// provider-generic completeOrderPaymentTx, whose guarded UPDATE (WHERE payment_status <>
// 'completed') fires the notify + event on exactly one transition. These tests exercise
// that shared core with the Stripe and wallet payloads — no live gateway needed.

import (
	"testing"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func stripeEvent(order *models.Order) map[string]interface{} {
	return map[string]interface{}{
		"order_id": order.ID.String(), "order_number": order.OrderNumber,
		"amount": 500.0, "method": "card", "provider": "stripe", "currency": "INR",
	}
}

func walletEvent(order *models.Order) map[string]interface{} {
	return map[string]interface{}{
		"order_id": order.ID.String(), "order_number": order.OrderNumber,
		"amount": order.Total, "method": "wallet", "provider": "wallet",
	}
}

// Stripe: the first completion flips the order + emits order.paid and the chef push once.
func TestCompleteOrderPaymentTx_Stripe_FirstCompletionEmitsOnce(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "", "")
	order := loadPayOrder(t, db, orderID)

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = completeOrderPaymentTx(tx, order, map[string]interface{}{"payment_method": "card"}, stripeEvent(order))
		return err
	}))
	require.True(t, ok)
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"), "one order.paid")
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder), "one chef push")
}

// Stripe: a re-verify (or verify/webhook race) on an already-completed order is a no-op —
// the #555 fix. Previously this second call emitted a duplicate order.paid.
func TestCompleteOrderPaymentTx_Stripe_ReVerifyDoesNotDoubleEmit(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "", "")

	fresh := loadPayOrder(t, db, orderID)
	stale := loadPayOrder(t, db, orderID) // still pending in memory — the racing verify's snapshot

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := completeOrderPaymentTx(tx, fresh, map[string]interface{}{"payment_method": "card"}, stripeEvent(fresh))
		return err
	}))

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = completeOrderPaymentTx(tx, stale, map[string]interface{}{"payment_method": "card"}, stripeEvent(stale))
		return err
	}))
	require.False(t, ok, "second completion performs no transition")
	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"), "still exactly one order.paid")
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder), "still exactly one chef push")
}

// #563: a REFUNDED order must never be re-completed — that would silently re-enable the
// chef payout on money already returned. The guard excludes refunded, so it's a no-op.
func TestCompleteOrderPaymentTx_RefundedOrderNotReCompleted(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "refunded", 500, "rzp_ref", "")
	order := loadPayOrder(t, db, orderID)

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = completeOrderPaymentTx(tx, order, map[string]interface{}{"payment_method": "card"}, stripeEvent(order))
		return err
	}))
	require.False(t, ok, "a refunded order is not re-completed")
	require.Equal(t, string(models.PaymentRefunded), paymentStatusOf(t, db, orderID))
	require.Equal(t, int64(0), countOutbox(t, db, "orders.paid"), "no order.paid re-emit on a refunded order")
}

// #563: a FAILED order (a prior card decline) MUST still complete on retry — `failed` is
// intentionally NOT in the blocked set. Retry-after-decline is preserved.
func TestCompleteOrderPaymentTx_FailedOrderCompletesOnRetry(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "failed", 500, "rzp_retry", "")
	order := loadPayOrder(t, db, orderID)

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = completeOrderPaymentTx(tx, order, map[string]interface{}{"payment_method": "card"}, stripeEvent(order))
		return err
	}))
	require.True(t, ok, "a failed order completes on retry")
	require.Equal(t, string(models.PaymentCompleted), paymentStatusOf(t, db, orderID))
	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"))
}

// Wallet (full store-credit order): a retried settle stamps the wallet columns once and
// never double-emits order.paid.
func TestCompleteOrderPaymentTx_Wallet_RetryDoesNotDoubleEmit(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 300, "", "")

	fresh := loadPayOrder(t, db, orderID)
	stale := loadPayOrder(t, db, orderID)
	walletUpdates := map[string]interface{}{"payment_method": "wallet", "payment_provider": "wallet", "wallet_applied": 300.0}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := completeOrderPaymentTx(tx, fresh, walletUpdates, walletEvent(fresh))
		return err
	}))

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = completeOrderPaymentTx(tx, stale, walletUpdates, walletEvent(stale))
		return err
	}))
	require.False(t, ok)
	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"), "still exactly one order.paid")
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder), "still exactly one chef push")
}
