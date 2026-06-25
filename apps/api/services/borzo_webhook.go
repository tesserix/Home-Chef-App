package services

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// borzoCallback is the subset of a Borzo order-change notification we consume.
// Borzo wraps the order either at the top level or under an "order" key depending
// on the notification; parseBorzoCallback tolerates both. Courier lat/lng feed
// the customer's live-tracking map. NOTE: exact field nesting is confirmed against
// a real sandbox callback during live validation; the resolver keys off order_id.
type borzoCallback struct {
	OrderID  json.Number `json:"order_id"`
	Status   string      `json:"status"`
	TrackURL string      `json:"track_url"`
	Courier  struct {
		Latitude  json.Number `json:"latitude"`
		Longitude json.Number `json:"longitude"`
		Phone     string      `json:"phone"`
		Name      string      `json:"name"`
	} `json:"courier"`
}

// parseBorzoCallback extracts the order id + mapped status + courier position
// from a Borzo callback body, tolerating a top-level or "order"-nested shape.
// Returns ok=false for an unparseable body / missing order id / unknown status.
func parseBorzoCallback(payload []byte) (orderID string, status models.DeliveryStatus, cb borzoCallback, ok bool) {
	// Try top-level first.
	_ = json.Unmarshal(payload, &cb)
	if cb.OrderID.String() == "" || cb.OrderID.String() == "0" {
		// Fall back to an "order"-nested shape.
		var wrap struct {
			Order borzoCallback `json:"order"`
		}
		if err := json.Unmarshal(payload, &wrap); err == nil && wrap.Order.OrderID.String() != "" {
			cb = wrap.Order
		}
	}
	id := cb.OrderID.String()
	if id == "" || id == "0" {
		return "", "", cb, false
	}
	st, mapped := mapBorzoStatus(cb.Status)
	if !mapped {
		return id, "", cb, false
	}
	return id, st, cb, true
}

// handleBorzoWebhook applies a Borzo order-change notification to the matching
// delivery. Resolves by external_delivery_id (= Borzo's order_id), maps the
// status, stamps timestamps, stores the live courier position (for the customer
// map), and propagates a terminal `delivered` to the order. Unknown order/status
// are acked (return nil) so Borzo stops retrying.
func (s *ProviderService) handleBorzoWebhook(provider *models.DeliveryProvider, payload []byte) error {
	orderID, status, cb, ok := parseBorzoCallback(payload)
	if !ok {
		log.Printf("borzo webhook: ignoring unparseable/unknown-status callback (order=%q)", orderID)
		return nil
	}

	var delivery models.Delivery
	if err := database.DB.Where("external_delivery_id = ? AND provider_id = ?", orderID, provider.ID).
		First(&delivery).Error; err != nil {
		log.Printf("borzo webhook: delivery for order_id %q not found — acking", orderID)
		return nil
	}

	updates := map[string]interface{}{"status": status, "provider_status": cb.Status}
	now := time.Now()
	switch status {
	case models.DeliveryPickedUp:
		updates["picked_up_at"] = now
	case models.DeliveryDelivered:
		updates["delivered_at"] = now
	case models.DeliveryCancelled:
		updates["cancelled_at"] = now
	}
	if lat, err := strconv.ParseFloat(cb.Courier.Latitude.String(), 64); err == nil && lat != 0 {
		updates["rider_latitude"] = lat
	}
	if lng, err := strconv.ParseFloat(cb.Courier.Longitude.String(), 64); err == nil && lng != 0 {
		updates["rider_longitude"] = lng
	}
	if cb.Courier.Phone != "" {
		updates["rider_phone"] = cb.Courier.Phone
	}
	if cb.Courier.Name != "" {
		updates["rider_name"] = cb.Courier.Name
	}
	if cb.TrackURL != "" {
		updates["external_tracking_url"] = cb.TrackURL
	}
	if err := database.DB.Model(&delivery).Updates(updates).Error; err != nil {
		return fmt.Errorf("borzo webhook: update delivery: %w", err)
	}

	if status == models.DeliveryDelivered && delivery.OrderID != uuid.Nil {
		database.DB.Model(&models.Order{}).Where("id = ?", delivery.OrderID).
			Update("status", models.OrderStatusDelivered)
		MarkMealPlanDayDelivered(delivery.OrderID)
		MarkGroupOrderDelivered(delivery.OrderID)
		_ = ReleaseOrderPayouts(delivery.OrderID)
	}

	log.Printf("borzo webhook: order_id=%s delivery=%s -> %s", orderID, delivery.ID, status)
	return nil
}
