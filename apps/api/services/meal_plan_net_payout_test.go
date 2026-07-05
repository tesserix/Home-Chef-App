package services

// meal_plan_net_payout_test.go — #518. The chef's held meal-plan-day transfer must
// be NET (food + per-day food-GST − commission − TDS), mirroring the order path's
// ComputeOrderEarnings, NOT the gross food price. Delivery is the driver's money and
// is excluded from the chef's gross.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestPerDayNetPayout_IncludesFoodGSTNetsCommissionAndTDS(t *testing.T) {
	// plan: 5 days, food subtotal 1000, GST 50 (5%), delivery 50 (total 1100).
	plan := &models.MealPlan{Subtotal: 1000, Tax: 50, Total: 1100, Days: make([]models.MealPlanDay, 5)}
	day := &models.MealPlanDay{Price: 200}

	// dayFoodGST = 50 * (200/1000)      = 10.00
	// gross      = 200 + 10             = 210.00   (delivery excluded — driver's)
	// commission = 0.06 * 200           = 12.00    (on food subtotal only)
	// tds        = 0.01 * 210           = 2.10
	// net        = 210 − 12 − 2.10      = 195.90
	require.Equal(t, 195.90, perDayNetPayout(plan, day, 0.06))
}

func TestPerDayNetPayout_ZeroSubtotalFallsBackToFoodOnly(t *testing.T) {
	// Defensive: a plan with no snapshotted subtotal ⇒ no proportional GST, gross = price.
	plan := &models.MealPlan{Subtotal: 0, Tax: 0, Total: 0, Days: make([]models.MealPlanDay, 1)}
	day := &models.MealPlanDay{Price: 200}
	// gross=200; commission=12; tds=2.00; net=186.00
	require.Equal(t, 186.00, perDayNetPayout(plan, day, 0.06))
}

func TestPerDayNetPayout_IsStrictlyLessThanGross(t *testing.T) {
	// The whole point of #518: the chef is no longer paid the gross food price.
	plan := &models.MealPlan{Subtotal: 900, Tax: 45, Total: 1000, Days: make([]models.MealPlanDay, 3)}
	day := &models.MealPlanDay{Price: 300}
	net := perDayNetPayout(plan, day, 0.06)
	require.Less(t, net, day.Price, "net must be below the gross food price (commission+TDS deducted)")
}
