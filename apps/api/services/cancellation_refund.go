package services

// cancellation_refund.go — the pure money model for tiered cancellation refunds
// (epic #475). A customer cancels; the vendor picks a REASON that selects a
// food-refund tier. Invariants:
//   - the PLATFORM FEE (and its share of tax) is ALWAYS kept — never refundable.
//   - the tier % applies to the FOOD (vendor) portion only.
//   - tax refunds in proportion to the refunded pre-tax amount.
//   - delivery refunds only if no driver has been dispatched.
//   - money is conserved EXACTLY: grandTotal == Total + VendorKept + PlatformKept.
// This function is deliberately pure (no DB, no gateway) so the money logic can
// be exhaustively unit-tested before anything moves real money.

// CancellationReason selects the refund tier. The vendor chooses it when
// confirming a cancellation.
type CancellationReason string

const (
	CancelReasonNotStarted    CancellationReason = "not_started"
	CancelReasonMaterials     CancellationReason = "materials_purchased"
	CancelReasonInPreparation CancellationReason = "in_preparation"
	CancelReasonReady         CancellationReason = "ready"
)

// CancellationTiers holds the admin-configurable food-refund percentages per
// reason (resolved from PlatformSettings at runtime; these are the defaults).
type CancellationTiers struct {
	NotStartedPct int
	MaterialsPct  int
	InPrepPct     int
	ReadyPct      int
}

// DefaultCancellationTiers — the defaults from the product spec (90 / 40 / 0 / 0).
func DefaultCancellationTiers() CancellationTiers {
	return CancellationTiers{NotStartedPct: 90, MaterialsPct: 40, InPrepPct: 0, ReadyPct: 0}
}

func clampPct(p int) int {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// FoodRefundPct returns the food-refund percentage for a reason. An unknown
// reason yields 0 (fail safe — no accidental over-refund).
func (t CancellationTiers) FoodRefundPct(reason CancellationReason) int {
	switch reason {
	case CancelReasonNotStarted:
		return clampPct(t.NotStartedPct)
	case CancelReasonMaterials:
		return clampPct(t.MaterialsPct)
	case CancelReasonInPreparation:
		return clampPct(t.InPrepPct)
	case CancelReasonReady:
		return clampPct(t.ReadyPct)
	default:
		return 0
	}
}

// CancellationRefund is the money breakdown of a cancellation (all in paise).
type CancellationRefund struct {
	FoodRefund     int // refunded from the food subtotal
	DeliveryRefund int // refunded from delivery (0 when a driver is dispatched)
	TaxRefund      int // refunded from tax, proportional to the refunded pre-tax base
	Total          int // what the customer gets back
	VendorKept     int // the vendor keeps (food not refunded — their sunk cost)
	PlatformKept   int // the platform keeps (fee + its tax + any kept delivery)
}

// CappedAt limits the customer refund to remainingPaise — the order's STILL-refundable
// balance (Total − already-refunded). The tier model computes off the ORIGINAL order amounts,
// oblivious to a prior partial refund on a different channel (e.g. a customer-issue
// RefundIssueToWallet, which credits the wallet + bumps order.refund_amount without touching the
// line items), so an early-stage cancellation could otherwise return more than what's still owed
// → cumulative refund past the order total (#642). When the cap bites, the food/delivery/tax
// refund components scale down proportionally so the breakdown still sums to the capped total and
// conservation holds exactly (grand == Total + VendorKept + PlatformKept — VendorKept/PlatformKept
// absorb the difference via the remainder). A negative remaining yields a zero refund. Pure.
func (r CancellationRefund) CappedAt(remainingPaise int) CancellationRefund {
	if remainingPaise < 0 {
		remainingPaise = 0
	}
	if r.Total <= remainingPaise {
		return r // still within the remaining balance — no cap needed
	}
	grand := r.Total + r.VendorKept + r.PlatformKept // the original order grand total (conserved)
	foodPaise := r.FoodRefund + r.VendorKept         // original food subtotal (VendorKept = food − foodRefund)
	food, delivery, tax := 0, 0, 0
	if r.Total > 0 {
		// Scale food + delivery to the capped total; give tax the residual so the three sum
		// EXACTLY to remainingPaise (no integer-division drift).
		food = r.FoodRefund * remainingPaise / r.Total
		delivery = r.DeliveryRefund * remainingPaise / r.Total
		tax = remainingPaise - food - delivery
		if tax < 0 {
			tax = 0
		}
	}
	total := food + delivery + tax
	vendorKept := foodPaise - food
	return CancellationRefund{
		FoodRefund:     food,
		DeliveryRefund: delivery,
		TaxRefund:      tax,
		Total:          total,
		VendorKept:     vendorKept,
		PlatformKept:   grand - total - vendorKept, // remainder keeps conservation exact
	}
}

// ComputeCancellationRefund computes the refund for a cancellation. foodRefundPct
// comes from the vendor's chosen tier (see CancellationTiers.FoodRefundPct).
func ComputeCancellationRefund(foodPaise, deliveryPaise, platformFeePaise, taxPaise int, dispatched bool, foodRefundPct int) CancellationRefund {
	foodRefundPct = clampPct(foodRefundPct)

	foodRefund := foodPaise * foodRefundPct / 100
	deliveryRefund := 0
	if !dispatched {
		deliveryRefund = deliveryPaise
	}
	// Tax refunds in proportion to the refunded pre-tax amount over the total
	// pre-tax base — the platform fee's share of tax is therefore always kept.
	preTaxTotal := foodPaise + deliveryPaise + platformFeePaise
	taxRefund := 0
	if preTaxTotal > 0 {
		taxRefund = taxPaise * (foodRefund + deliveryRefund) / preTaxTotal
	}

	total := foodRefund + deliveryRefund + taxRefund
	grand := foodPaise + deliveryPaise + platformFeePaise + taxPaise
	vendorKept := foodPaise - foodRefund
	return CancellationRefund{
		FoodRefund:     foodRefund,
		DeliveryRefund: deliveryRefund,
		TaxRefund:      taxRefund,
		Total:          total,
		VendorKept:     vendorKept,
		// Everything not refunded to the customer and not kept by the vendor —
		// computed as the remainder so conservation holds exactly (no rounding drift).
		PlatformKept: grand - total - vendorKept,
	}
}
