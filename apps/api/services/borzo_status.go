package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// mapBorzoStatus maps a Borzo Business-API order status to our internal
// DeliveryStatus. Borzo's order-level enum is coarse (new/available/active/
// delayed/completed/canceled); finer per-point delivery statuses (picked-up vs
// en route) come through callbacks and can be added once validated live.
// Returns ok=false for an unrecognised status so callers log-and-ignore.
func mapBorzoStatus(raw string) (models.DeliveryStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "new", "available":
		return models.DeliveryAssigned, true
	case "active", "delayed":
		return models.DeliveryInTransit, true
	case "completed":
		return models.DeliveryDelivered, true
	case "canceled", "cancelled":
		return models.DeliveryCancelled, true
	default:
		return "", false
	}
}
