package models

// delivery_failure.go — #393. Structured fault handling for a delivery that cannot be
// completed (courier or chef self-delivery). The reporter picks a DeliveryFailureReason
// which SUGGESTS a DeliveryFaultClass, but the money outcome (refund vs release) is only
// executed after an admin CONFIRMS fault in the payout/dispute queue (hybrid model,
// owner-decided 2026-07-06). Ambiguous reasons never auto-resolve.

// DeliveryFailureReason is the structured reason a delivery terminally failed.
type DeliveryFailureReason string

const (
	// Customer-fault: confirmed/correct address, but the customer could not receive it.
	FailureCustomerUnavailable DeliveryFailureReason = "customer_unavailable" // not home / unreachable
	FailureCustomerRefused     DeliveryFailureReason = "customer_refused"      // refused the delivery
	// Platform/driver-fault: the courier could not attempt/complete the delivery.
	FailureDriverNoShow DeliveryFailureReason = "driver_no_show"
	// Chef-fault: food unfit on handover / spoiled in transit.
	FailureFoodDamaged DeliveryFailureReason = "food_damaged"
	// Ambiguous — always route to admin review, never auto-resolve.
	FailureWrongAddress DeliveryFailureReason = "wrong_address"
	FailureOther        DeliveryFailureReason = "other"
)

// ValidDeliveryFailureReason reports whether r is a known reason.
func ValidDeliveryFailureReason(r DeliveryFailureReason) bool {
	switch r {
	case FailureCustomerUnavailable, FailureCustomerRefused, FailureDriverNoShow,
		FailureFoodDamaged, FailureWrongAddress, FailureOther:
		return true
	}
	return false
}

// DeliveryFaultClass is who is at fault for a failed delivery — it drives the money
// policy an admin confirms: customer-fault → no refund + vendor paid; platform/chef →
// full refund + hold reversed; ambiguous → admin decides.
type DeliveryFaultClass string

const (
	FaultCustomer  DeliveryFaultClass = "customer"
	FaultPlatform  DeliveryFaultClass = "platform"
	FaultChef      DeliveryFaultClass = "chef"
	FaultAmbiguous DeliveryFaultClass = "ambiguous"
)

// SuggestedFaultClass maps a reported reason to the fault class it SUGGESTS. Unknown or
// genuinely ambiguous reasons default to FaultAmbiguous so they land in admin review
// rather than auto-executing a money outcome.
func SuggestedFaultClass(r DeliveryFailureReason) DeliveryFaultClass {
	switch r {
	case FailureCustomerUnavailable, FailureCustomerRefused:
		return FaultCustomer
	case FailureDriverNoShow:
		return FaultPlatform
	case FailureFoodDamaged:
		return FaultChef
	default: // wrong_address, other, or unrecognized
		return FaultAmbiguous
	}
}
