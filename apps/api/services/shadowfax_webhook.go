package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// parseShadowfaxCallback extracts the client order id + mapped delivery status
// from a Shadowfax Push Callback body. Pure + testable: returns ok=false for an
// unparseable body, a missing order_id, or an unrecognised status — the caller
// then acks (200) and ignores rather than mis-transitioning the order.
//
// Shadowfax sends `order_id` = the `client_order_id` we set at create time (our
// order number), NOT the AWB — so resolution keys off the order number.
func parseShadowfaxCallback(payload []byte) (clientOrderID string, status models.DeliveryStatus, ok bool) {
	var cb struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(payload, &cb); err != nil {
		return "", "", false
	}
	if cb.OrderID == "" {
		return "", "", false
	}
	st, mapped := mapShadowfaxStatus(cb.Status)
	if !mapped {
		return cb.OrderID, "", false
	}
	return cb.OrderID, st, true
}

// handleShadowfaxWebhook applies a Shadowfax Push Callback to the matching
// delivery + order. It resolves the delivery via the order number (Shadowfax's
// order_id), maps the status, stamps timestamps, and propagates a terminal
// `delivered` to the order so the customer timeline + chef screen move. Unknown
// order/status are acked (return nil) so Shadowfax stops retrying.
func (s *ProviderService) handleShadowfaxWebhook(provider *models.DeliveryProvider, payload []byte) error {
	clientOrderID, status, ok := parseShadowfaxCallback(payload)
	if !ok {
		log.Printf("shadowfax webhook: ignoring unparseable/unknown-status callback (order=%q)", clientOrderID)
		return nil
	}

	var order models.Order
	if err := database.DB.Where("order_number = ?", clientOrderID).First(&order).Error; err != nil {
		log.Printf("shadowfax webhook: order %q not found — acking", clientOrderID)
		return nil
	}
	var delivery models.Delivery
	if err := database.DB.Where("order_id = ?", order.ID).First(&delivery).Error; err != nil {
		log.Printf("shadowfax webhook: delivery for order %q not found — acking", clientOrderID)
		return nil
	}

	// Idempotent: a repeat of the same status is a no-op write (no harm).
	updates := map[string]interface{}{"status": status, "provider_status": ""}
	now := time.Now()
	switch status {
	case models.DeliveryPickedUp:
		updates["picked_up_at"] = now
	case models.DeliveryDelivered:
		updates["delivered_at"] = now
	case models.DeliveryCancelled:
		updates["cancelled_at"] = now
	}
	if err := database.DB.Model(&delivery).Updates(updates).Error; err != nil {
		return fmt.Errorf("shadowfax webhook: update delivery: %w", err)
	}

	if status == models.DeliveryDelivered {
		database.DB.Model(&models.Order{}).Where("id = ?", order.ID).
			Update("status", models.OrderStatusDelivered)
		// Mirror the generic 3PL path: release escrow/meal-plan/group payouts.
		MarkMealPlanDayDelivered(order.ID)
		MarkGroupOrderDelivered(order.ID)
		_ = ReleaseOrderPayouts(order.ID)
	}

	log.Printf("shadowfax webhook: order=%s delivery=%s -> %s", clientOrderID, delivery.ID, status)
	return nil
}
