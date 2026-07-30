package services

import (
	"math"
	"testing"

	"github.com/homechef/api/models"
)

// TestPerDayGrossConservation locks the escrow money-conservation invariant: the
// sum of what each day is worth to the customer (food + its GST + its delivery)
// must equal the advance the customer paid (plan.Total). If it drifts, refunding
// every day would return more or less than was charged.
func TestPerDayGrossConservation(t *testing.T) {
	// Advance as CreateMealPlan would snapshot it: 3 days × ₹100 food, 8% GST,
	// ₹2.99/day delivery → 300 + 24 + 8.97 = 332.97.
	plan := &models.MealPlan{
		Subtotal: 300,
		Tax:      24,
		Total:    332.97,
		Days: []models.MealPlanDay{
			{Price: 100}, {Price: 100}, {Price: 100},
		},
	}

	var sum float64
	for i := range plan.Days {
		g := perDayGross(plan, &plan.Days[i])
		// Each day: 100 food + 8 GST + 2.99 delivery = 110.99.
		if math.Abs(g-110.99) > 0.01 {
			t.Fatalf("per-day gross = %.2f, want 110.99", g)
		}
		sum += g
	}
	if math.Abs(sum-plan.Total) > 0.02 {
		t.Fatalf("sum of per-day gross %.2f != advance %.2f — money not conserved", sum, plan.Total)
	}
}

// TestPerDayGrossUnequalPrices checks conservation holds when days have different
// food prices (GST is apportioned by food share, delivery is flat per day).
func TestPerDayGrossUnequalPrices(t *testing.T) {
	// 120 + 80 food = 200 subtotal; 8% GST = 16; 2 days × 3.00 delivery = 6.
	plan := &models.MealPlan{
		Subtotal: 200,
		Tax:      16,
		Total:    222,
		Days:     []models.MealPlanDay{{Price: 120}, {Price: 80}},
	}
	var sum float64
	for i := range plan.Days {
		sum += perDayGross(plan, &plan.Days[i])
	}
	if math.Abs(sum-plan.Total) > 0.02 {
		t.Fatalf("sum of per-day gross %.2f != advance %.2f", sum, plan.Total)
	}
}

// TestPerDayGrossConservationWithPlatformFee locks the same conservation invariant once
// the customer-facing platform fee is part of the advance. The fee is charged on meal
// plans exactly as on an à la carte order (subtotal × PlatformFeePercent), so it must be
// apportioned back into each day or refunding every day would return LESS than was
// charged and the difference would silently stick to the platform.
func TestPerDayGrossConservationWithPlatformFee(t *testing.T) {
	// 3 days × ₹100 food, 4.99% platform fee = 14.97, 8% GST = 24, ₹2.99/day
	// delivery = 8.97 → 300 + 14.97 + 24 + 8.97 = 347.94.
	plan := &models.MealPlan{
		Subtotal:    300,
		PlatformFee: 14.97,
		Tax:         24,
		Total:       347.94,
		Days:        []models.MealPlanDay{{Price: 100}, {Price: 100}, {Price: 100}},
	}

	var sum float64
	for i := range plan.Days {
		g := perDayGross(plan, &plan.Days[i])
		// Each day: 100 food + 4.99 fee + 8 GST + 2.99 delivery = 115.98.
		if math.Abs(g-115.98) > 0.01 {
			t.Fatalf("per-day gross = %.2f, want 115.98", g)
		}
		sum += g
	}
	if math.Abs(sum-plan.Total) > 0.02 {
		t.Fatalf("sum of per-day gross %.2f != advance %.2f — money not conserved", sum, plan.Total)
	}
}

// TestPlanDeliveryTotalExcludesPlatformFee guards the derivation that has no column
// behind it: delivery is Total − Subtotal − PlatformFee − Tax. If the fee were not
// subtracted it would be misread as delivery — and delivery IS refundable on a
// customer-initiated skip while the platform fee is not, so the platform would refund
// its own fee on every skip.
func TestPlanDeliveryTotalExcludesPlatformFee(t *testing.T) {
	plan := &models.MealPlan{Subtotal: 300, PlatformFee: 14.97, Tax: 24, Total: 347.94}
	if got := planDeliveryTotal(plan); math.Abs(got-8.97) > 0.01 {
		t.Fatalf("plan delivery = %.2f, want 8.97 (fee must not leak into delivery)", got)
	}
}

// TestPerDaySkipRefundExcludesPlatformFee pins the refund policy: a customer-initiated
// skip returns food-minus-commission plus that day's delivery, and never the platform
// fee or GST.
func TestPerDaySkipRefundExcludesPlatformFee(t *testing.T) {
	plan := &models.MealPlan{
		Subtotal:    300,
		PlatformFee: 14.97,
		Tax:         24,
		Total:       347.94,
		Days:        []models.MealPlanDay{{Price: 100}, {Price: 100}, {Price: 100}},
	}
	day := &plan.Days[0]
	// food 100 − 6% commission = 94, + 2.99 delivery = 96.99. No 4.99 fee, no 8 GST.
	got := perDaySkipRefund(plan, day, 0.06)
	if math.Abs(got-96.99) > 0.01 {
		t.Fatalf("skip refund = %.2f, want 96.99 (platform fee and GST are forfeited)", got)
	}
	if got >= perDayGross(plan, day) {
		t.Fatalf("skip refund %.2f must be less than the make-whole gross", got)
	}
}

// TestRefundV3BaseExcludesPlatformFee guards the interaction between the platform fee and
// refund policy v3, which landed independently. mealPlanDayRefundBase is the production
// customer-cancellation base and takes its delivery share from perDayDeliveryRefund — so if
// the platform fee were ever left inside the derived delivery figure, v3 would hand the
// platform's own nonrefundable fee back on every cancellation, at every tier.
func TestRefundV3BaseExcludesPlatformFee(t *testing.T) {
	plan := &models.MealPlan{
		Subtotal:    300,
		PlatformFee: 14.97,
		Tax:         24,
		Total:       347.94,
		Days: []models.MealPlanDay{
			{Price: 100, CommissionRate: 0.06},
			{Price: 100, CommissionRate: 0.06},
			{Price: 100, CommissionRate: 0.06},
		},
	}
	// (100 food − 6 commission) + 8 GST + 2.99 delivery = 104.99. If the 4.99 per-day
	// platform fee leaked into the delivery derivation this reads 109.98 instead.
	got := mealPlanDayRefundBase(plan, &plan.Days[0])
	if math.Abs(got-104.99) > 0.01 {
		t.Fatalf("v3 refund base = %.2f, want 104.99 (platform fee must stay out of delivery)", got)
	}
}

// TestPerDayGrossFoodOnlyFallback: with no snapshotted fees (escrow-off plans),
// a day is worth exactly its food price.
func TestPerDayGrossFoodOnlyFallback(t *testing.T) {
	plan := &models.MealPlan{Subtotal: 0, Tax: 0, Total: 0, Days: []models.MealPlanDay{{Price: 100}}}
	if g := perDayGross(plan, &plan.Days[0]); g != 100 {
		t.Fatalf("food-only fallback = %.2f, want 100", g)
	}
}
