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

// TestPerDayGrossFoodOnlyFallback: with no snapshotted fees (escrow-off plans),
// a day is worth exactly its food price.
func TestPerDayGrossFoodOnlyFallback(t *testing.T) {
	plan := &models.MealPlan{Subtotal: 0, Tax: 0, Total: 0, Days: []models.MealPlanDay{{Price: 100}}}
	if g := perDayGross(plan, &plan.Days[0]); g != 100 {
		t.Fatalf("food-only fallback = %.2f, want 100", g)
	}
}
