package services

// temporal_refund.go — start + activity implementations for the durable
// deferred chef-cancel gateway-refund retry flow (workflows.DeferredRefundWorkflow).
// Mirrors temporal_confirm.go's shape: gated behind temporalRT being set (+ a
// deploy-time flag), idempotent on an order-keyed workflow ID. Fires
// IMMEDIATELY when ChefOrderCancelHandler.CancelOrder defers a gateway
// refund (Razorpay unreachable or erroring — see DeferredCancelRefundPrefix
// in deferred_cancel_refund.go), so the customer's refund typically lands
// within seconds/minutes instead of waiting for the next
// RetryDeferredCancelRefunds cron tick (up to ~12 minutes, see
// deferred_cancel_refund.go).
//
// The cron sweep is NOT removed by this — it remains the backstop for the
// cases this workflow can't cover on its own: Temporal down at cancel time,
// or the workflow itself exhausting its 24h retry window.
//
// SAFETY (no double refund): both this workflow's gateway activity and the
// cron's retry call Razorpay with the IDENTICAL stable key,
// RefundFullIdempotencyKey(orderID). Whichever path reaches the gateway
// first "wins"; the other dedups to the same refund at Razorpay's end
// instead of issuing a second one. See deferred_cancel_refund.go's file
// header for the full argument — it applies unchanged here.

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func deferredRefundFlowID(orderID uuid.UUID) string { return "homechef:refund:" + orderID.String() }

// deferredRefundFlowActive reports whether the durable retry flow should be
// driven: Temporal must be up (temporalRT set), and
// DEFERRED_REFUND_FLOW_ENABLED (default true) must not be explicitly
// disabled. The cron backstop runs unconditionally regardless of this gate.
func deferredRefundFlowActive() bool {
	if temporalRT == nil {
		return false
	}
	if config.AppConfig != nil && !config.AppConfig.DeferredRefundFlowEnabled {
		return false
	}
	return true
}

// StartDeferredRefundFlow durably starts the deferred gateway-refund retry
// for one chef cancel. No-op when Temporal is down, the flow is disabled, or
// there is nothing to refund (amountPaise<=0) or no payment to refund against
// (paymentID=="") — RetryDeferredCancelRefunds backstops every one of those
// cases on its own schedule regardless. Idempotent on the order-keyed
// workflow ID: a duplicate start attempt, or the cron healing the order
// first, is harmless — the workflow's gateway activity uses the SAME
// idempotency key either way, so it can never double-refund.
func StartDeferredRefundFlow(orderID uuid.UUID, paymentID string, amountPaise int) {
	if !deferredRefundFlowActive() || amountPaise <= 0 || paymentID == "" {
		return
	}
	in := workflows.DeferredRefundInput{OrderID: orderID, PaymentID: paymentID, AmountPaise: amountPaise}
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueuePayments, deferredRefundFlowID(orderID), workflows.DeferredRefundWorkflow, in); err != nil {
		// An "already started" error is the expected idempotent case; anything
		// else is logged — the cron backstop still retries on its own schedule.
		log.Printf("deferred refund flow: start failed for %s: %v", orderID, err)
	}
}

// ─── Activity implementations (wired onto workflows.* Funcs by the worker) ──

// GatewayRefundForWorkflow issues the deferred gateway refund. Returns an
// error when the gateway is unavailable or refuses the call so the workflow's
// activity retries with backoff; the SAME stable idempotency key the cron
// uses (RefundFullIdempotencyKey) means a retry — here or via the cron — is
// deduped by Razorpay, never a double refund.
func GatewayRefundForWorkflow(_ context.Context, orderID uuid.UUID, paymentID string, amountPaise int) (string, error) {
	rzp := GetRazorpayFor(PaymentModeForOrder(orderID))
	if rzp == nil {
		return "", errors.New("razorpay unavailable")
	}
	refundResp, err := rzp.CreateRefund(paymentID, &RefundRequest{
		Amount: amountPaise,
		Speed:  "normal",
		Notes: map[string]string{
			"order_id":  orderID.String(),
			"reason":    "deferred chef cancel",
			"initiator": "temporal",
		},
		// SAME key CancelOrder / RetryDeferredCancelRefunds use — a lost-response
		// success dedups here instead of double-refunding. #574.
		IdempotencyKey: RefundFullIdempotencyKey(orderID),
	})
	if err != nil {
		return "", err
	}
	return refundResp.ID, nil
}

// PersistDeferredRefundID replaces the deferred-cancel-refund sentinel in
// orders.refund_id with the real gateway refund id once GatewayRefundForWorkflow
// succeeds. Guarded to ONLY replace the sentinel — RowsAffected==0 means a
// concurrent actor (the retry cron, or this workflow's own retried activity
// racing a second run) already healed the row first; a harmless no-op, never
// a double-write. Mirrors retryOneDeferredCancelRefund's guarded UPDATE in
// deferred_cancel_refund.go exactly.
func PersistDeferredRefundID(_ context.Context, orderID uuid.UUID, refundID string) error {
	res := database.DB.Model(&models.Order{}).
		Where("id = ? AND refund_id LIKE ?", orderID, DeferredCancelRefundPrefix+"%").
		Update("refund_id", refundID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil // already healed by the cron or a concurrent run — no-op
	}
	notifyDeferredRefundHealed(orderID)
	return nil
}

// notifyDeferredRefundHealed stages an orders.updated event through the
// existing transactional outbox so the customer's client refreshes now that
// the refund has actually landed. Best-effort: the money-critical write in
// PersistDeferredRefundID already succeeded regardless of this outcome. No
// dedicated "refund landed" subject exists yet (see services/nats.go), so
// this reuses the existing generic order-update subject + EnqueueOrderEvent
// helper rather than hand-rolling new NATS wiring.
func notifyDeferredRefundHealed(orderID uuid.UUID) {
	var order models.Order
	if err := database.DB.Select("id", "order_number", "customer_id", "chef_id", "status", "total").
		First(&order, "id = ?", orderID).Error; err != nil {
		log.Printf("deferred refund flow: order %s healed but event lookup failed: %v", orderID, err)
		return
	}
	if err := EnqueueOrderEvent(database.DB, SubjectOrderUpdated, OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	}); err != nil {
		log.Printf("deferred refund flow: failed to enqueue orders.updated for %s: %v", orderID, err)
	}
}
