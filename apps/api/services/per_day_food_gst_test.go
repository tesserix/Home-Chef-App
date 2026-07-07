package services

// per_day_food_gst_test.go — #540. perDayFoodGST is the SINGLE proportional food-GST basis
// shared by the chef day-transfer (perDayNetPayout), the customer per-day refund (perDayGross),
// AND the spawned order's reported tax (generateDayOrder) — so reported-TDS == withheld-TDS
// exactly, with no drift from a live-policy-rate recompute.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestPerDayFoodGST_ProportionalBasis(t *testing.T) {
	// Plan: subtotal 1000, tax 180 (18%), one 250-price day → 180 × 250/1000 = 45.
	plan := &models.MealPlan{Subtotal: 1000, Tax: 180, Total: 1240}
	day := &models.MealPlanDay{Price: 250}
	require.InDelta(t, 45.0, perDayFoodGST(plan, day), 1e-9, "food GST apportioned by the day's share of subtotal")

	// The chef day-transfer's gross uses the SAME basis (day.Price + this GST), so the spawned
	// order's reported dayTax = perDayFoodGST matches the withheld basis exactly.
	require.InDelta(t, perDayFoodGST(plan, day), 45.0, 1e-9)
}

func TestPerDayFoodGST_NoSnapshottedSubtotal_ZeroGST(t *testing.T) {
	// A plan with no snapshotted subtotal falls back to food-only (no GST) on BOTH the payout
	// and the reported basis — so they still agree.
	plan := &models.MealPlan{Subtotal: 0, Tax: 180}
	require.Equal(t, 0.0, perDayFoodGST(plan, &models.MealPlanDay{Price: 250}))
}
