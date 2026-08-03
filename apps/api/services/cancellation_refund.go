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

// CancellationOrder is the order money the tier model operates on, all in paise.
//
// A struct rather than positional ints deliberately: every field is a rupee
// amount of the same type, so a transposed argument moves money silently and
// compiles cleanly. #962 was exactly that class of defect.
type CancellationOrder struct {
	// FoodPaise is the order subtotal at LIST price, before any promo discount.
	FoodPaise        int
	DeliveryPaise    int
	PlatformFeePaise int
	TaxPaise         int
	// DiscountPaise is the promo the customer did NOT pay. Checkout computes it on
	// the subtotal (handlers/orders.go validateAndCalculateDiscount), and the order
	// total is `subtotal + delivery + fee + tax + tip − discount`, so it belongs on
	// the food base and nowhere else.
	DiscountPaise int
	// Dispatched — a driver is already carrying the order, so the delivery fee is
	// non-refundable (the driver is paid regardless).
	Dispatched bool
	// The tax FROZEN on the order per supply. Refunding tax requires knowing which
	// supply carried it: once the platform fee sits at 18% and the food at 5%,
	// value stops being a proxy for tax and apportioning by value hands back tax
	// belonging to a fee that was kept — money that cannot be credit-noted, because
	// that supply did happen. All zero on an order priced before the split, which
	// falls back to the proportional model that was correct for a single rate.
	TaxFoodPaise     int
	TaxDeliveryPaise int
	TaxServicePaise  int
}

// hasSupplyTax reports whether this order carries the per-supply snapshot.
func (o CancellationOrder) hasSupplyTax() bool {
	return o.TaxFoodPaise != 0 || o.TaxDeliveryPaise != 0 || o.TaxServicePaise != 0
}

// EffectiveFoodPaise is the food the customer actually paid for: list price less
// the promo they did not pay. Floored at 0 — a discount larger than the subtotal
// must never produce a negative food base (it would invert the whole breakdown).
func (o CancellationOrder) EffectiveFoodPaise() int {
	food := o.FoodPaise - o.DiscountPaise
	if food < 0 {
		return 0
	}
	return food
}

// GrandPaise is what the customer was actually charged, and therefore the figure
// conservation is measured against: it equals orders.total exactly.
func (o CancellationOrder) GrandPaise() int {
	return o.EffectiveFoodPaise() + o.DeliveryPaise + o.PlatformFeePaise + o.TaxPaise
}

// ComputeCancellationRefund computes the refund for a cancellation. foodRefundPct
// comes from the vendor's chosen tier (see CancellationTiers.FoodRefundPct).
//
// #962: the tier is applied to the food the customer PAID (subtotal − discount),
// not the list subtotal. Computing against the list price made the model's grand
// total exceed orders.total on any discounted order; CappedAt then scaled the
// refund down and attributed the phantom gap to the vendor, inventing a retained
// share out of a promo nobody paid. Production carried a 100%-refunded order
// whose snapshot still claimed the chef kept ₹40.53 — a figure shown to the
// customer, and (until the #947 solvency cap) very nearly paid to a chef.
//
// KNOWN GAP — the customer tip is not modelled here. orders.total includes it,
// so a cancelled order does not return the customer's tip and the money sits
// outside this breakdown entirely. It is not silently absorbed into PlatformKept
// (grand excludes it too), it is simply unhandled. Tracked in #964, alongside the
// larger defect that tips never reach the chef at all; no cancelled order has
// ever carried one. Fixing it needs a tip column on the snapshot, so it is
// deliberately out of scope here rather than half-done.
func ComputeCancellationRefund(o CancellationOrder, foodRefundPct int) CancellationRefund {
	foodRefundPct = clampPct(foodRefundPct)

	foodPaise := o.EffectiveFoodPaise()
	foodRefund := foodPaise * foodRefundPct / 100
	deliveryRefund := 0
	if !o.Dispatched {
		deliveryRefund = o.DeliveryPaise
	}
	taxRefund := refundableTax(o, foodPaise, foodRefund, deliveryRefund)

	total := foodRefund + deliveryRefund + taxRefund
	vendorKept := foodPaise - foodRefund
	return CancellationRefund{
		FoodRefund:     foodRefund,
		DeliveryRefund: deliveryRefund,
		TaxRefund:      taxRefund,
		Total:          total,
		VendorKept:     vendorKept,
		// Everything not refunded to the customer and not kept by the vendor —
		// computed as the remainder so conservation holds exactly (no rounding drift).
		PlatformKept: o.GrandPaise() - total - vendorKept,
	}
}

// refundableTax is the tax attributable to the part of the order being given
// back — and ONLY that part.
//
// With the per-supply snapshot it is exact: the food's own tax scaled by how much
// food is refunded, plus the delivery's own tax when the delivery is refunded.
// The platform fee's tax is never touched here, because the fee is kept and that
// supply happened.
//
// Without the snapshot (an order priced before tax was split) the old
// proportional model stands: it is correct whenever every supply shared one rate,
// which is exactly the condition under which those orders were charged.
func refundableTax(o CancellationOrder, foodPaise, foodRefund, deliveryRefund int) int {
	if o.hasSupplyTax() {
		tax := 0
		if foodPaise > 0 {
			tax += o.TaxFoodPaise * foodRefund / foodPaise
		}
		if deliveryRefund > 0 && o.DeliveryPaise > 0 {
			tax += o.TaxDeliveryPaise * deliveryRefund / o.DeliveryPaise
		}
		return tax
	}
	preTaxTotal := foodPaise + o.DeliveryPaise + o.PlatformFeePaise
	if preTaxTotal <= 0 {
		return 0
	}
	return o.TaxPaise * (foodRefund + deliveryRefund) / preTaxTotal
}
