package services

// order_payment_settle.go — the single implementation of "an order's payment is
// captured, so make the order paid" (#872 step 2). Moved out of `handlers`
// (#872 step 2, Task 1) so it can be called from three places instead of two:
// the Cashfree/Razorpay HTTP verify legs (handlers) AND the order-payment
// reconcile cron (order_payment_reconcile_cron.go, this package) — which needed
// this core and could not previously reach it, since `services` cannot import
// `handlers`.
//
// This file grows across the extraction: Task 1 moves the guarded completion
// transaction (CompletionBlockedStatuses, NotifyChefNewOrderTx,
// CompleteOrderPaymentTx, CompleteRazorpayOrderTx, CompleteCashfreeOrderTx).
// Task 2 adds the wallet-settlement cluster (OrderSettlements,
// ApplyChefRecoveryDeduction, DebitOrderWallet, SettleWalletTopUps,
// SettleOrderWallet). Task 3 adds the two gateway-settle cores
// (SettleCashfreeOrder, SettleRazorpayOrderFromPayment).

import (
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// NotifyChefNewOrderTx stages the actionable "new order" push to the chef
// within a payment-completion transaction. Orders are created pre-payment, so
// the chef is only notified once money is captured (previously this fired in
// CreateOrder, pushing unpaid/abandoned orders to the kitchen — see
// handlers/orders.go). Callers MUST guard the call on the pending→completed
// transition so a webhook arriving after the client verify (or vice-versa)
// doesn't double-notify; the OrderEvent mirrors the one CreateOrder used.
func NotifyChefNewOrderTx(tx *gorm.DB, order *models.Order) error {
	return EnqueueOrderEvent(tx, SubjectChefNewOrder, OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	})
}

// CompletionBlockedStatuses are the payment states a success/capture/verify must NOT
// re-complete from: already `completed` (idempotent — no duplicate side effects) and
// `refunded` (a refund is terminal; re-stamping it `completed` would silently re-enable
// the chef payout on money already returned to the customer — #563). `failed` is
// deliberately NOT here, so a retry-after-decline can still complete the order.
var CompletionBlockedStatuses = []models.PaymentStatus{models.PaymentCompleted, models.PaymentRefunded}

// CompleteOrderPaymentTx flips an order pending→completed exactly ONCE and, only on
// that single transition, notifies the chef and emits order.paid — the provider-generic
// core shared by the Razorpay / Stripe / wallet / cron verify+settle paths (#395 item 2,
// #555, #872 step 2). Every one of them used to read `wasUnpaid` from the in-memory
// order and then update unconditionally + emit order.paid unconditionally, so a webhook
// or re-verify that completed the order underneath left a duplicate chef "new order"
// push AND a duplicate order.paid event. The guarded UPDATE (WHERE payment_status <>
// 'completed') makes exactly one racing path perform the transition; the chef notify +
// event fire only on RowsAffected>0. `updates` are the provider-specific columns to
// stamp alongside payment_status=completed; `event` is the order.paid payload. Returns
// whether THIS call performed the transition.
func CompleteOrderPaymentTx(tx *gorm.DB, order *models.Order, updates, event map[string]interface{}) (bool, error) {
	cols := map[string]interface{}{"payment_status": models.PaymentCompleted}
	for k, v := range updates {
		cols[k] = v
	}
	res := tx.Model(&models.Order{}).
		Where("id = ? AND payment_status NOT IN ?", order.ID, CompletionBlockedStatuses).
		Updates(cols)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil // a concurrent webhook/verify already completed it — no re-fire
	}
	// Keep the in-memory order consistent for any post-tx idempotent steps.
	order.PaymentStatus = models.PaymentCompleted
	if err := NotifyChefNewOrderTx(tx, order); err != nil {
		return false, err
	}
	if err := EnqueueEvent(tx, "orders.paid", "order.paid", order.CustomerID, event); err != nil {
		return false, err
	}
	return true, nil
}

// CompleteRazorpayOrderTx is the Razorpay-specific wrapper over CompleteOrderPaymentTx
// (#395). The caller's post-tx steps (referral reward, saga, wallet DEBIT) are
// idempotently keyed; the wallet TOP-UP transfer is NOT (see #395 follow-up) — but this
// helper does not invoke it, and on a raced (RowsAffected==0) verify the top-up
// recomputes the same deterministic split, so this does not add a double-transfer path.
func CompleteRazorpayOrderTx(tx *gorm.DB, order *models.Order, method, paymentID string, amountPaise int) (bool, error) {
	return CompleteOrderPaymentTx(tx, order,
		map[string]interface{}{"payment_method": method, "razorpay_payment_id": paymentID},
		map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       FromPaise(amountPaise),
			"method":       method,
			"provider":     "razorpay",
		})
}

// CompleteCashfreeOrderTx is the Cashfree wrapper over the provider-generic
// CompleteOrderPaymentTx, mirroring CompleteRazorpayOrderTx. The guarded UPDATE
// inside CompleteOrderPaymentTx is what makes exactly one of a racing
// verify/webhook/cron trio perform the pending→completed transition, so the chef push
// and order.paid event fire once.
func CompleteCashfreeOrderTx(tx *gorm.DB, order *models.Order, method, cfPaymentID string, amountPaise int) (bool, error) {
	return CompleteOrderPaymentTx(tx, order,
		map[string]interface{}{
			"payment_method":              method,
			models.GatewayPaymentIDColumn: cfPaymentID,
		},
		map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       FromPaise(amountPaise),
			"method":       method,
			"provider":     models.PaymentProviderCashfree,
		})
}
