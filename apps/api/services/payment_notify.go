package services

// payment_notify.go — money-movement events (#notify-money).
//
// Charges, failures and refunds previously moved silently: payments.success and
// payments.failed were declared subjects nobody published, and no refund subject
// existed at all, so a customer learned about their own refund only by opening
// the order. These helpers are the publish side; the handlers live in
// notifications.go and are gated on NotifCategoryPayment.
//
// Each is best-effort and never fails its caller — a missing notification must
// not roll back a payment that actually happened.

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// NotifyPaymentSucceeded publishes payments.success for an order that has just
// become paid. Call it at the SINGLE transition (the guarded update's
// RowsAffected > 0 branch), alongside MaybeGrantReward — every gateway's webhook
// and client-verify path shares that shape, and it is what keeps a webhook
// replay from notifying twice.
func NotifyPaymentSucceeded(db *gorm.DB, orderID uuid.UUID) {
	var order models.Order
	if err := db.Select("id, order_number, customer_id, total, refund_amount, payment_method").
		First(&order, "id = ?", orderID).Error; err != nil {
		log.Printf("payment-notify: load order %s: %v", orderID, err)
		return
	}
	publishPaymentEvent(db, SubjectPaymentSuccess, "payment_success", order.CustomerID, map[string]any{
		"type":        "payment_success",
		"orderId":     order.ID.String(),
		"orderNumber": order.OrderNumber,
		"amount":      models.RoundAmount(order.Total),
		"method":      order.PaymentMethod,
	})
}

// NotifyPaymentFailed publishes payments.failed so the customer is told their
// payment did not go through while the order is still retryable, rather than
// discovering it when the kitchen never starts cooking.
func NotifyPaymentFailed(db *gorm.DB, orderID uuid.UUID, reason string) {
	var order models.Order
	if err := db.Select("id, order_number, customer_id, total").
		First(&order, "id = ?", orderID).Error; err != nil {
		log.Printf("payment-notify: load order %s: %v", orderID, err)
		return
	}
	publishPaymentEvent(db, SubjectPaymentFailed, "payment_failed", order.CustomerID, map[string]any{
		"type":        "payment_failed",
		"orderId":     order.ID.String(),
		"orderNumber": order.OrderNumber,
		"amount":      models.RoundAmount(order.Total),
		"reason":      reason,
	})
}

// Refunds are deliberately NOT published here. Every refund path already tells
// the customer — orders.cancelled, orders.voided, orders.cancellation_resolved,
// and the chef's delivery-fee reduction via its own push — so a payments.refunded
// event would be a second notification for the same money. See nats.go.

// publishPaymentEvent stages the event on the transactional outbox so the relay
// delivers it durably — the same path every other notification takes, rather
// than a direct publish that a restart could lose.
func publishPaymentEvent(db *gorm.DB, subject, eventType string, userID uuid.UUID, data map[string]any) {
	if userID == uuid.Nil {
		return
	}
	if err := EnqueueEvent(db, subject, eventType, userID, data); err != nil {
		log.Printf("payment-notify: enqueue %s: %v", subject, err)
		CaptureBackgroundError(err)
	}
}
