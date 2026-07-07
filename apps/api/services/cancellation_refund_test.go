package services

// cancellation_refund_test.go — the tiered-refund money model (epic #475). The
// central guarantees: platform fee is never refunded, the tier applies to food,
// delivery follows dispatch, and money is conserved exactly across every case.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// An example order in paise: ₹500 food, ₹40 delivery, ₹60 platform fee, ₹30 tax.
const (
	food     = 50000
	delivery = 4000
	fee      = 6000
	tax      = 3000
	grand    = food + delivery + fee + tax
)

func TestComputeCancellationRefund_Conservation(t *testing.T) {
	// Every tier × dispatched flag must conserve money exactly and never refund
	// the platform fee.
	for _, pct := range []int{0, 40, 90, 100} {
		for _, dispatched := range []bool{false, true} {
			r := ComputeCancellationRefund(food, delivery, fee, tax, dispatched, pct)
			require.Equal(t, grand, r.Total+r.VendorKept+r.PlatformKept,
				"conservation failed at pct=%d dispatched=%v", pct, dispatched)
			// The platform fee is always inside PlatformKept.
			require.GreaterOrEqual(t, r.PlatformKept, fee,
				"platform fee must always be kept (pct=%d dispatched=%v)", pct, dispatched)
			require.LessOrEqual(t, r.Total, grand-fee, "customer can never get the platform fee back")
		}
	}
}

func TestComputeCancellationRefund_Tiers(t *testing.T) {
	// not_started 90% (undispatched): 90% of food + all delivery + proportional tax.
	r := ComputeCancellationRefund(food, delivery, fee, tax, false, 90)
	require.Equal(t, 45000, r.FoodRefund)      // 90% of 50000
	require.Equal(t, 4000, r.DeliveryRefund)   // delivery refunded (not dispatched)
	require.Equal(t, food-45000, r.VendorKept) // vendor keeps 10%
	require.Positive(t, r.TaxRefund)

	// in_preparation 0%: no food refund, vendor keeps ALL food. Delivery still
	// refunds (not dispatched), so its proportional share of tax refunds too — the
	// food's tax does not.
	r0 := ComputeCancellationRefund(food, delivery, fee, tax, false, 0)
	require.Equal(t, 0, r0.FoodRefund)
	require.Equal(t, food, r0.VendorKept)
	require.Equal(t, delivery, r0.DeliveryRefund, "delivery still refunds if not dispatched")
	require.Equal(t, tax*delivery/(food+delivery+fee), r0.TaxRefund, "only the refunded delivery's tax refunds")

	// in_preparation 0% AND dispatched: truly nothing refunds — vendor keeps food,
	// platform keeps fee + delivery + all tax.
	rd := ComputeCancellationRefund(food, delivery, fee, tax, true, 0)
	require.Equal(t, 0, rd.Total, "0% + dispatched → no refund at all")
}

func TestComputeCancellationRefund_DispatchedKeepsDelivery(t *testing.T) {
	r := ComputeCancellationRefund(food, delivery, fee, tax, true, 90)
	require.Equal(t, 0, r.DeliveryRefund, "a dispatched order keeps the delivery fee (driver is paid)")
	// Delivery stays inside PlatformKept.
	require.GreaterOrEqual(t, r.PlatformKept, fee+delivery)
}

func TestCancellationTiers_FoodRefundPct(t *testing.T) {
	tiers := DefaultCancellationTiers()
	require.Equal(t, 90, tiers.FoodRefundPct(CancelReasonNotStarted))
	require.Equal(t, 40, tiers.FoodRefundPct(CancelReasonMaterials))
	require.Equal(t, 0, tiers.FoodRefundPct(CancelReasonInPreparation))
	require.Equal(t, 0, tiers.FoodRefundPct(CancelReasonReady))
	// Unknown reason fails safe to 0 (no accidental over-refund).
	require.Equal(t, 0, tiers.FoodRefundPct("bogus"))
	// Percentages clamp.
	require.Equal(t, 100, CancellationTiers{NotStartedPct: 150}.FoodRefundPct(CancelReasonNotStarted))
	require.Equal(t, 0, CancellationTiers{MaterialsPct: -5}.FoodRefundPct(CancelReasonMaterials))
}

func TestComputeCancellationRefund_ZeroOrderNoPanic(t *testing.T) {
	r := ComputeCancellationRefund(0, 0, 0, 0, false, 90)
	require.Equal(t, CancellationRefund{}, r)
}

// #642: CappedAt limits the cancellation refund to the order's still-refundable balance
// (Total − already-refunded) so a cancellation AFTER a prior partial refund can't over-refund,
// while keeping the breakdown's exact-conservation invariant.
func TestCancellationRefund_CappedAt(t *testing.T) {
	full := ComputeCancellationRefund(food, delivery, fee, tax, false, 90)
	grandTotal := full.Total + full.VendorKept + full.PlatformKept

	// No cap when remaining ≥ total (the normal, no-prior-refund case).
	require.Equal(t, full, full.CappedAt(full.Total))
	require.Equal(t, full, full.CappedAt(full.Total+1))

	// Cap bites (a prior refund left less remaining): total == remaining, breakdown sums to it,
	// components scale down, and conservation still holds EXACTLY.
	remaining := full.Total - 1000
	capped := full.CappedAt(remaining)
	require.Equal(t, remaining, capped.Total, "refund is capped at the remaining balance")
	require.Equal(t, capped.Total, capped.FoodRefund+capped.DeliveryRefund+capped.TaxRefund, "breakdown sums to the capped total")
	require.Equal(t, grandTotal, capped.Total+capped.VendorKept+capped.PlatformKept, "conservation holds exactly after the cap")
	require.Less(t, capped.FoodRefund, full.FoodRefund, "components scaled down")

	// Zero remaining → zero refund; the vendor/platform keep everything, conservation intact.
	zero := full.CappedAt(0)
	require.Equal(t, 0, zero.Total)
	require.Equal(t, grandTotal, zero.VendorKept+zero.PlatformKept)

	// Negative remaining is treated as zero (never negative refund).
	require.Equal(t, 0, full.CappedAt(-100).Total)
}
