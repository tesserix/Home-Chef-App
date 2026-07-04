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
