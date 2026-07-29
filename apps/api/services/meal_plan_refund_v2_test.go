package services

// meal_plan_refund_v2_test.go — pins the v3 refund rule (#834): the refund base is EVERYTHING
// the customer paid for the day — food + GST + delivery — scaled by the chef-agreed percentage.
// This is a deliberate reversal of the v2 rule these tests used to pin (food net of commission,
// GST retained), so the assertions here are the guard against silently drifting back.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// v3Plan is the shared fixture: 2 days, food 320, GST 32, delivery 20 (total 372).
// Per day: 160 food + 16 GST + 10 delivery = 186 gross.
func v3Plan() (*models.MealPlan, *models.MealPlanDay) {
	plan := &models.MealPlan{
		Subtotal: 320, Tax: 32, Total: 372,
		Days: []models.MealPlanDay{
			{Price: 160, CommissionRate: 0.15},
			{Price: 160, CommissionRate: 0.15},
		},
	}
	return plan, &plan.Days[0]
}

func TestMealPlanRefundAmount_V3IncludesGSTAndDelivery(t *testing.T) {
	plan, day := v3Plan()

	// The whole gross comes back at 100% — the platform keeps nothing on a refunded day.
	require.Equal(t, 186.0, MealPlanRefundAmount(plan, day, 100))
	require.Equal(t, 93.0, MealPlanRefundAmount(plan, day, 50))
	require.Equal(t, 0.0, MealPlanRefundAmount(plan, day, 0))

	// The v3 base is strictly LARGER than the v2 one it replaces: v2 deducted the platform
	// commission from the food and kept the GST.
	v2Base := perDaySkipRefund(plan, day, 0.15) // (160 − 24) + 10 = 146
	require.Equal(t, 146.0, v2Base)
	require.Greater(t, MealPlanRefundAmount(plan, day, 100), v2Base,
		"v3 returns the commission and the GST that v2 retained")
}

// The worked example from the issue: ₹100 food + ₹8 GST + ₹5 other = ₹113 paid; cancelled at
// the 75% floor the customer gets ₹84.75 back.
func TestMealPlanRefundAmount_IssueWorkedExample(t *testing.T) {
	plan := &models.MealPlan{
		Subtotal: 100, Tax: 8, Total: 113,
		Days:     []models.MealPlanDay{{Price: 100, CommissionRate: 0.15}},
	}
	day := &plan.Days[0]
	require.Equal(t, 113.0, MealPlanRefundAmount(plan, day, 100))
	require.Equal(t, 84.75, MealPlanRefundAmount(plan, day, 75))
}

// The GST component drives the credit note, so it must track the percentage exactly.
func TestMealPlanRefundGSTComponent(t *testing.T) {
	plan, day := v3Plan()
	require.Equal(t, 16.0, MealPlanRefundGSTComponent(plan, day, 100))
	require.Equal(t, 12.0, MealPlanRefundGSTComponent(plan, day, 75))
	require.Equal(t, 0.0, MealPlanRefundGSTComponent(plan, day, 0))
}

// The base must survive a plan loaded WITHOUT its Days (every refund handler Selects a narrow
// column set and never preloads them). perDayGross divides delivery by len(plan.Days) and so
// collapses to the bare food price there — the exact trap the v3 base avoids.
func TestMealPlanRefundAmount_NoDaysPreloaded(t *testing.T) {
	plan := &models.MealPlan{Subtotal: 320, Tax: 32, Total: 372} // Days deliberately empty
	day := &models.MealPlanDay{Price: 160, CommissionRate: 0.15}

	require.Equal(t, 186.0, MealPlanRefundAmount(plan, day, 100),
		"GST + delivery must not vanish when the plan's Days aren't loaded")
	require.Equal(t, 160.0, perDayGross(plan, day),
		"documents why perDayGross is NOT the v3 base: it silently drops GST + delivery here")
}

// A legacy plan with no snapshotted totals falls back to the bare food price rather than
// refunding a GST it has no record of collecting.
func TestMealPlanRefundAmount_LegacyPlanNoTotals(t *testing.T) {
	plan := &models.MealPlan{}
	day := &models.MealPlanDay{Price: 200}
	require.Equal(t, 200.0, MealPlanRefundAmount(plan, day, 100))
	require.Equal(t, 0.0, MealPlanRefundGSTComponent(plan, day, 100))
}

// Out-of-range percentages are clamped, never extrapolated — a bad caller can't over-refund.
func TestMealPlanRefundAmount_ClampsPercent(t *testing.T) {
	plan, day := v3Plan()
	require.Equal(t, 186.0, MealPlanRefundAmount(plan, day, 150))
	require.Equal(t, 0.0, MealPlanRefundAmount(plan, day, -20))
}

// A pre-v3 day carrying only the old enum still prices correctly.
func TestRefundPercentOf_LegacyEnumFallback(t *testing.T) {
	plan, _ := v3Plan()
	legacy := &models.MealPlanDay{Price: 160, ChefRefundChoice: models.RefundProportionHalf}
	require.Equal(t, 50, legacy.RefundPercentOf())
	require.Equal(t, 93.0, MealPlanRefundAmountForDay(plan, legacy))

	// The v3 column wins whenever it is set.
	p := 75
	legacy.RefundPercent = &p
	require.Equal(t, 75, legacy.RefundPercentOf())
	require.Equal(t, 139.5, MealPlanRefundAmountForDay(plan, legacy))
}

func TestValidRefundProportion(t *testing.T) {
	require.True(t, ValidRefundProportion(models.RefundProportionFull))
	require.True(t, ValidRefundProportion(models.RefundProportionHalf))
	require.True(t, ValidRefundProportion(models.RefundProportionNone))
	require.False(t, ValidRefundProportion("quarter"))
	require.False(t, ValidRefundProportion(""))
}

func TestRefundProportionLabel(t *testing.T) {
	require.Equal(t, models.RefundProportionFull, models.RefundProportionLabel(100))
	require.Equal(t, models.RefundProportionHalf, models.RefundProportionLabel(50))
	require.Equal(t, models.RefundProportionNone, models.RefundProportionLabel(0))
	require.Equal(t, models.RefundProportionPartial, models.RefundProportionLabel(75))
}
