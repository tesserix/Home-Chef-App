package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// mapShadowfaxStatus maps a Shadowfax Unified-API status string to our internal
// DeliveryStatus. We use the on-demand subset of Shadowfax's parcel vocabulary;
// hub-internal statuses (recd_at_fwd_hub, item_manifested, …) we don't surface
// collapse to the nearest customer-meaningful state. Returns ok=false for an
// unrecognised status so callers log-and-ignore rather than mis-transition.
func mapShadowfaxStatus(raw string) (models.DeliveryStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "new", "ofp", "pickup_on_hold", "assigned_for_seller_pickup":
		return models.DeliveryAssigned, true
	case "picked":
		return models.DeliveryPickedUp, true
	case "assigned_for_delivery", "ofd", "recd_at_fwd_hub", "recd_at_fwd_dc":
		return models.DeliveryInTransit, true
	case "delivered":
		return models.DeliveryDelivered, true
	case "cancelled_by_seller", "cancelled_by_customer":
		return models.DeliveryCancelled, true
	case "rts", "rts_d", "rto", "rto_d", "rts_in_process", "rts_ofd":
		return models.DeliveryReturned, true
	case "lost", "on_hold", "nc", "na", "pickup_not_attempted":
		return models.DeliveryFailed, true
	default:
		return "", false
	}
}
