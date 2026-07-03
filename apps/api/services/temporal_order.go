package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

// temporal_order.go — start + signal forwarding for the Order lifecycle Saga
// (#122), plus the idempotent service ops its activities wrap. Everything here is
// gated behind config.OrderSagaEnabled (default OFF) so prod behaviour is
// unchanged until ops validates the saga on the cluster; the synchronous HTTP
// handlers stay authoritative meanwhile. Because the activities are idempotent,
// enabling the saga never double-acts alongside any residual handler logic.

func orderSagaID(orderID uuid.UUID) string { return "homechef:order:" + orderID.String() }

// sagaActive reports whether the saga should be driven (Temporal up + flag on).
func sagaActive() bool {
	return temporalRT != nil && config.AppConfig != nil && config.AppConfig.OrderSagaEnabled
}

// StartOrderSaga durably starts the post-payment order lifecycle. Idempotent on
// the order-keyed workflow ID, so a re-trigger (e.g. a retried payment webhook)
// never starts a second saga. No-op when the saga is disabled.
func StartOrderSaga(orderID uuid.UUID) {
	if !sagaActive() {
		return
	}
	id := orderSagaID(orderID)
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueueOrders, id, workflows.OrderSagaWorkflow, workflows.OrderSagaInput{OrderID: orderID}); err != nil {
		// An "already started" error is the expected idempotent case; anything
		// else is logged (the synchronous handlers remain the safety net).
		log.Printf("order saga: start failed for %s: %v", orderID, err)
	}
}

// signalOrderSaga forwards a lifecycle signal to a running saga. Best-effort: if
// the saga isn't running (flag flipped on after the order was paid) the signal
// is dropped with a log — the synchronous handler already did the work.
func signalOrderSaga(orderID uuid.UUID, signal string, arg any) {
	if !sagaActive() {
		return
	}
	if err := temporalRT.Signal(context.Background(), orderSagaID(orderID), signal, arg); err != nil {
		log.Printf("order saga: signal %s for %s dropped: %v", signal, orderID, err)
	}
}

// SignalOrderChefDecision forwards the chef's accept/reject.
func SignalOrderChefDecision(orderID uuid.UUID, accepted bool, reason string) {
	signalOrderSaga(orderID, workflows.SignalChefDecision, workflows.ChefDecisionSignal{Accepted: accepted, Reason: reason})
}

// SignalOrderReady forwards the chef marking the order ready for pickup.
func SignalOrderReady(orderID uuid.UUID) { signalOrderSaga(orderID, workflows.SignalOrderReady, nil) }

// SignalOrderDelivered forwards the delivery completion.
func SignalOrderDelivered(orderID uuid.UUID) {
	signalOrderSaga(orderID, workflows.SignalOrderDelivered, nil)
}

// SignalOrderCancelled forwards a cancellation so the saga compensates (refund).
func SignalOrderCancelled(orderID uuid.UUID, reason string) {
	signalOrderSaga(orderID, workflows.SignalOrderCancelled, workflows.OrderCancelSignal{Reason: reason})
}

// ─── Activity implementations (wired onto the workflows.* Funcs by the worker) ──

// NotifyChefNewOrder publishes the new-order notification to the chef via the
// transactional outbox. Mirrors what CreateOrder enqueues; safe to call from the
// retried saga activity.
func NotifyChefNewOrder(_ context.Context, orderID uuid.UUID) error {
	var order models.Order
	if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
		return fmt.Errorf("notify chef: load order %s: %w", orderID, err)
	}
	return EnqueueOrderEvent(database.DB, SubjectChefNewOrder, OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	})
}

// SettleOrderPayouts parks a delivered order's payout in a customer-confirmation
// hold (#387) — the settle-on-delivery saga step no longer releases funds
// directly; the customer confirming (or #388, gated by
// ORDER_PAYOUT_AUTO_RELEASE_ENABLED) drives the real release off release_eligible.
// Durable + retried (the returned error retries the activity on a transient DB
// failure) and idempotent (the hold update only fires from the pre-delivery state).
func SettleOrderPayouts(_ context.Context, orderID uuid.UUID) error {
	return SetOrderHoldAwaitingConfirmation(database.DB, orderID)
}

// CompensateOrderRefund is the saga's compensation — refund the customer to
// wallet store credit and mark the order refunded. Idempotent: the RefundedAt
// guard + the wallet idempotency key make a retry a no-op. (Gateway refund to
// the original method is upgraded in #123; wallet credit is the existing,
// fully-idempotent refund path used elsewhere.)
func CompensateOrderRefund(_ context.Context, orderID uuid.UUID, reason string) error {
	var order models.Order
	if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
		return fmt.Errorf("refund: load order %s: %w", orderID, err)
	}
	if order.RefundedAt != nil {
		return nil // already refunded
	}
	// Claw back the chef/rider Route split first (#123) — the platform must not
	// pay out an order it is refunding. No-op unless live payout movement is on;
	// guarded by RefundedAt above so it only runs once per order.
	if err := ReverseOrderPayouts(orderID); err != nil {
		return fmt.Errorf("refund: reverse payouts for %s: %w", orderID, err)
	}
	if order.Total > 0 {
		if _, err := CreditWallet(database.DB, order.CustomerID, order.Total,
			models.WalletSourceRefund, &order.ID,
			fmt.Sprintf("Order %s refunded: %s", order.OrderNumber, reason),
			"saga-refund:"+order.ID.String(), nil); err != nil {
			return fmt.Errorf("refund: credit wallet for %s: %w", orderID, err)
		}
	}
	now := time.Now()
	if err := database.DB.Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]any{
		"status":         models.OrderStatusRefunded,
		"payment_status": models.PaymentRefunded,
		"refunded_at":    &now,
		"refund_amount":  order.Total,
		"refund_reason":  reason,
	}).Error; err != nil {
		return err
	}
	// Cross-guard the payout hold (#457): flip the hold to withheld/reversed so the
	// admin queue and any release can never pay out this refunded order. Best-effort
	// — never fail the refund on a hold-drive error. NOTE: the ReverseOrderPayouts
	// above already clawed the Route split; the helper's released→reversed branch may
	// call ReverseOrderPayouts a SECOND time. That is safe because the second gateway
	// reverse is GATEWAY-REJECTED and logged non-fatal (order_payout.go:108 —
	// "already reversed errors on Razorpay; logged, not fatal"), NOT because
	// stampPayoutSettled skips it.
	if hErr := WithholdOrReverseOrderHoldForRefund(database.DB, orderID, "saga refund: "+reason); hErr != nil {
		log.Printf("payout cross-guard failed for saga refund order %s: %v", orderID, hErr)
	}
	return nil
}
