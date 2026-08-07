package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

// temporal_payment.go — start seam + activity implementations for the durable
// payment-resolution flow (workflows/payment_resolution.go).
//
// Gated behind config.PaymentResolutionEnabled (default OFF). With the flag off
// nothing here runs and the two sweeps — order_payment_reconcile_cron (settle a
// captured payment) and stale_order_cron (expire a dead one) — behave exactly as
// they do today. With it on, each order additionally gets a per-order durable
// poll that reaches the SAME decisions sooner, through the SAME idempotent ops.

func paymentResolutionID(orderID uuid.UUID) string {
	return "homechef:payment:" + orderID.String()
}

// paymentResolutionActive reports whether the flow should be driven.
func paymentResolutionActive() bool {
	return temporalRT != nil && config.AppConfig != nil && config.AppConfig.PaymentResolutionEnabled
}

// StartPaymentResolution durably starts the poll for one order's payment, called
// when a gateway payment session is created. Idempotent on the order-keyed
// workflow ID, so a retried session creation never starts a second poll. No-op
// when the flow is disabled.
func StartPaymentResolution(orderID uuid.UUID) {
	if !paymentResolutionActive() {
		return
	}
	if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueuePayments,
		paymentResolutionID(orderID), workflows.PaymentResolutionWorkflow,
		workflows.PaymentResolutionInput{OrderID: orderID}); err != nil {
		// "Already started" is the expected idempotent case; anything else is
		// logged and the sweeps remain the safety net.
		log.Printf("payment resolution: start failed for %s: %v", orderID, err)
	}
}

// SignalPaymentResolved wakes the poll when something else settled the order —
// the HTTP verify leg or the gateway webhook. Best-effort: the poll reaches the
// same answer on its own tick, so a dropped signal costs latency, not
// correctness.
func SignalPaymentResolved(orderID uuid.UUID) {
	if !paymentResolutionActive() {
		return
	}
	if err := temporalRT.Signal(context.Background(), paymentResolutionID(orderID),
		workflows.SignalPaymentResolved, nil); err != nil {
		log.Printf("payment resolution: signal for %s dropped: %v", orderID, err)
	}
}

// ─── Activity implementations ───────────────────────────────────────────────

// ResolveOrderPayment is the workflow's poll: read the order, ask the gateway,
// and settle it if the money is there.
//
// It reuses staleOrderPaymentState — the same tri-state gateway probe the stale
// sweep uses — so the two paths cannot drift on what "in flight" means, which is
// the exact drift that produced the PENDING hole in the first place.
func ResolveOrderPayment(_ context.Context, orderID uuid.UUID) (workflows.PaymentOutcome, error) {
	var order models.Order
	if err := database.DB.Preload("Chef").Preload("Delivery.DeliveryPartner").
		First(&order, "id = ?", orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return workflows.PaymentOutcomeGone, nil
		}
		return workflows.PaymentOutcomeUnknown, fmt.Errorf("payment resolution: load order %s: %w", orderID, err)
	}

	// Already paid — by the client verify, the webhook, or the reconcile cron.
	if order.PaymentStatus == models.PaymentCompleted {
		return workflows.PaymentOutcomeSettled, nil
	}
	// Cancelled, rejected, refunded, or delivered: no longer ours to resolve.
	// Deliberately NOT an error — a customer who cancelled before paying is a
	// normal ending, not a failed workflow.
	if order.PaymentStatus != models.PaymentPending || isTerminalOrderStatus(order.Status) {
		return workflows.PaymentOutcomeGone, nil
	}
	// No gateway session was ever created, so there is nothing to ask about.
	// Expiring this is the stale sweep's long-standing job (it cancels these with
	// zero gateway calls) and it is not urgent enough to duplicate here.
	if order.GatewayOrderID == "" {
		return workflows.PaymentOutcomeInFlight, nil
	}

	state, _, err := staleOrderPaymentState(&order)
	if err != nil {
		return workflows.PaymentOutcomeUnknown, err
	}

	switch state {
	case gatewayCaptured:
		// Settle through the shared core the HTTP verify legs and the reconcile
		// cron use — one implementation of the binding checks and the completion
		// transaction, never a second.
		var ok bool
		var msg string
		switch models.NormalizeProvider(order.PaymentProvider) {
		case models.PaymentProviderCashfree:
			ok, msg, _ = SettleCashfreeOrder(&order)
		default:
			// Stripe, a retired-gateway row, anything unrecognised: not this flow's
			// job, exactly as the reconcile cron treats them.
			return workflows.PaymentOutcomeGone, nil
		}
		if !ok {
			// The gateway said captured but the settle core rejected it (amount
			// or binding mismatch). That is a genuine anomaly, not a dead
			// payment — never cancel on it.
			log.Printf("payment resolution: order %s captured at gateway but settle refused: %s",
				order.OrderNumber, msg)
			return workflows.PaymentOutcomeUnknown, nil
		}
		return workflows.PaymentOutcomeSettled, nil

	case gatewayInFlight:
		return workflows.PaymentOutcomeInFlight, nil

	default:
		return workflows.PaymentOutcomeDead, nil
	}
}

// isTerminalOrderStatus reports whether an order has left the pre-payment world.
func isTerminalOrderStatus(s models.OrderStatus) bool {
	switch s {
	case models.OrderStatusCancelled, models.OrderStatusRejected,
		models.OrderStatusRefunded, models.OrderStatusDelivered:
		return true
	default:
		return false
	}
}

// ExpireUnpaidOrder cancels an order whose payment the gateway confirmed dead,
// releasing the capacity and slot it reserved. Idempotent: the WHERE clause
// re-checks payment_status, so a row the stale sweep already expired is a no-op
// and the capacity is never released twice.
func ExpireUnpaidOrder(_ context.Context, orderID uuid.UUID) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var order models.Order
		if err := tx.Preload("Items").First(&order, "id = ?", orderID).Error; err != nil {
			return err
		}
		if order.PaymentStatus != models.PaymentPending || isTerminalOrderStatus(order.Status) {
			return nil
		}
		res := tx.Model(&models.Order{}).
			Where("id = ? AND payment_status = ?", orderID, models.PaymentPending).
			Updates(map[string]interface{}{
				"status":         models.OrderStatusCancelled,
				"payment_status": models.PaymentFailed,
				"cancel_reason":  "payment not completed",
				"cancelled_at":   time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // someone else expired it between the read and the write
		}
		capDay := CapacityDay(order.CreatedAt)
		for _, it := range order.Items {
			if err := ReleaseCapacity(tx, it.MenuItemID, it.Quantity, capDay); err != nil {
				return err
			}
		}
		if order.DeliverySlot != "" && order.ScheduledFor != nil {
			if err := ReleaseSlot(tx, order.ChefID, order.DeliverySlot, 1, CapacityDay(*order.ScheduledFor)); err != nil {
				return err
			}
		}
		return nil
	})
}

// PublishPaymentStalled raises the ops signal for a payment that has been live
// at the gateway for longer than a customer would ever wait.
//
// Staged on the transactional outbox rather than published directly, so it
// inherits the relay's JetStream dedup (Nats-Msg-Id) and cannot be lost to a
// NATS blip mid-activity — the activity would otherwise be retried and publish
// twice.
func PublishPaymentStalled(_ context.Context, orderID uuid.UUID, waited time.Duration) error {
	var order models.Order
	if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
		return err
	}
	return EnqueueOrderEvent(database.DB, SubjectPaymentStalled, OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Reason:      fmt.Sprintf("payment unresolved at the gateway for %s", waited.Round(time.Minute)),
	})
}
