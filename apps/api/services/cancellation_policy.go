package services

// cancellation_policy.go — runtime config + the auto-fast-path decision for the
// cancellation-with-arbitration flow (#475/#476). Kept next to the pure
// calculator; ResolveCancellationTiers touches the DB, ClassifyCancellation is
// pure.

import (
	"strconv"

	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ResolveCancellationTiers reads the admin-configurable food-refund percentages
// and the vendor response window from PlatformSettings `cancel.*` keys, falling
// back to the product defaults (90/40/0/0, 15 min). `cancel.platform_fee_refundable`
// is intentionally NOT honored — the platform fee is nonrefundable by policy.
func ResolveCancellationTiers(db *gorm.DB) (CancellationTiers, int) {
	tiers := DefaultCancellationTiers()
	windowMin := 15
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "cancel.%").Find(&settings)
	for _, s := range settings {
		v, err := strconv.Atoi(s.Value)
		if err != nil {
			continue
		}
		switch s.Key {
		case "cancel.refund.not_started_pct":
			tiers.NotStartedPct = clampPct(v)
		case "cancel.refund.materials_pct":
			tiers.MaterialsPct = clampPct(v)
		case "cancel.refund.in_prep_pct":
			tiers.InPrepPct = clampPct(v)
		case "cancel.refund.ready_pct":
			tiers.ReadyPct = clampPct(v)
		case "cancel.vendor_response_minutes":
			if v > 0 {
				windowMin = v
			}
		}
	}
	return tiers, windowMin
}

// CancellationPath is how a cancellation request is handled based on the order's
// fulfilment state.
type CancellationPath string

const (
	// CancelPathFullRefund: the chef isn't engaged (not accepted / rejected), so
	// the customer backs out with a full food refund — the platform fee is still
	// kept. No vendor confirmation needed.
	CancelPathFullRefund CancellationPath = "auto_full_refund"
	// CancelPathVendorReview: the chef has accepted or is preparing → the vendor
	// confirms and picks the refund tier.
	CancelPathVendorReview CancellationPath = "vendor_review"
	// CancelPathNotAllowed: the order is made and/or on its way → not cancellable.
	CancelPathNotAllowed CancellationPath = "not_allowed"
)

// ClassifyCancellation decides the path from the order's current status. Pure.
func ClassifyCancellation(status models.OrderStatus) (CancellationPath, string) {
	switch status {
	case models.OrderStatusPending:
		// Chef hasn't accepted → not engaged → full refund.
		return CancelPathFullRefund, ""
	case models.OrderStatusRejected, models.OrderStatusCancelled:
		// Chef declined (or already cancelled) → full refund.
		return CancelPathFullRefund, ""
	case models.OrderStatusAccepted, models.OrderStatusPreparing:
		// Chef is committed / cooking → the vendor confirms and sets the tier.
		return CancelPathVendorReview, ""
	case models.OrderStatusReady, models.OrderStatusPickedUp,
		models.OrderStatusDelivering, models.OrderStatusDelivered:
		return CancelPathNotAllowed, "This order is already prepared and on its way — please contact support."
	default:
		// Unknown → route to the vendor (safe: no auto-refund).
		return CancelPathVendorReview, ""
	}
}
