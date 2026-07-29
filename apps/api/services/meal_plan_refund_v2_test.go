package services

// meal_plan_refund_v2_test.go — pins the v3 refund rule (#834):
//
//	base = food − platform commission + that day's GST + that day's delivery
//
// v3 changes exactly ONE term from v2: the GST now comes back. The platform KEEPS its
// commission on a cancelled day, so the base sits below the gross the customer paid — these
// assertions are the guard against it drifting up onto the gross or back down to v2.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// v3Plan is the shared fixture: 2 days, food 320, GST 32, delivery 20 (total 372).
// Per day at a frozen 15% commission:
//
//	food 160 − commission 24 = 136 · + GST 16 · + delivery 10  →  base 162
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

func TestMealPlanRefundAmount_V3RefundsGSTButRetainsCommission(t *testing.T) {
	plan, day := v3Plan()

	require.Equal(t, 162.0, MealPlanRefundAmount(plan, day, 100))
	require.Equal(t, 81.0, MealPlanRefundAmount(plan, day, 50))
	require.Equal(t, 0.0, MealPlanRefundAmount(plan, day, 0))

	// Above v2 by EXACTLY the GST — the one term v3 changed.
	v2Base := perDaySkipRefund(plan, day, 0.15) // (160 − 24) + 10 = 146
	require.Equal(t, 146.0, v2Base)
	require.Equal(t, Round2(v2Base+perDayFoodGST(plan, day)), MealPlanRefundAmount(plan, day, 100))

	// Below the gross the customer paid by EXACTLY the retained commission.
	require.Equal(t, 186.0, perDayGross(plan, day)) // 160 + 16 + 10
	require.Equal(t, 24.0, Round2(perDayGross(plan, day)-MealPlanRefundAmount(plan, day, 100)),
		"the platform keeps its 15% commission on a cancelled day")
}

// The worked example behind the decision: a ₹100 day with 8% GST and ₹10 delivery refunds ₹103.
func TestMealPlanRefundAmount_DecisionWorkedExample(t *testing.T) {
	plan := &models.MealPlan{
		Subtotal: 100, Tax: 8, Total: 118,
		Days:     []models.MealPlanDay{{Price: 100, CommissionRate: 0.15}},
	}
	day := &plan.Days[0]
	require.Equal(t, 103.0, MealPlanRefundAmount(plan, day, 100), "85 food-net + 8 GST + 10 delivery")
	require.Equal(t, 77.25, MealPlanRefundAmount(plan, day, 75))
	require.Equal(t, 118.0, plan.Total, "the customer paid 118; the platform retains its 15")
}

// The GST component drives the credit note, so it must track the percentage exactly. It is
// unaffected by the commission — the tax was charged on the full food price.
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

	require.Equal(t, 162.0, MealPlanRefundAmount(plan, day, 100),
		"GST + delivery must not vanish when the plan's Days aren't loaded")
	require.Equal(t, 160.0, perDayGross(plan, day),
		"documents why perDayGross is NOT the v3 base: it silently drops GST + delivery here")
}

// A legacy plan with no snapshotted totals refunds food net of the DEFAULT commission and
// nothing else — it must not invent a GST or delivery it has no record of collecting.
func TestMealPlanRefundAmount_LegacyPlanNoTotals(t *testing.T) {
	plan := &models.MealPlan{}
	day := &models.MealPlanDay{Price: 200}
	require.Equal(t, 188.0, MealPlanRefundAmount(plan, day, 100), "200 − 6% default commission")
	require.Equal(t, 0.0, MealPlanRefundGSTComponent(plan, day, 100))
}

// A day with no frozen rate falls back to the default rather than refunding the full food.
func TestMealPlanRefundAmount_LegacyRateFallback(t *testing.T) {
	plan := &models.MealPlan{Subtotal: 200, Days: []models.MealPlanDay{{Price: 200}}}
	day := &plan.Days[0]
	require.Less(t, MealPlanRefundAmount(plan, day, 100), 200.0,
		"an unset rate must still deduct the default commission, not refund the whole food price")
}

// Out-of-range percentages are clamped, never extrapolated — a bad caller can't over-refund.
func TestMealPlanRefundAmount_ClampsPercent(t *testing.T) {
	plan, day := v3Plan()
	require.Equal(t, 162.0, MealPlanRefundAmount(plan, day, 150))
	require.Equal(t, 0.0, MealPlanRefundAmount(plan, day, -20))
}

// A pre-v3 day carrying only the old enum still prices correctly.
func TestRefundPercentOf_LegacyEnumFallback(t *testing.T) {
	plan, _ := v3Plan()
	legacy := &models.MealPlanDay{Price: 160, CommissionRate: 0.15, ChefRefundChoice: models.RefundProportionHalf}
	require.Equal(t, 50, legacy.RefundPercentOf())
	require.Equal(t, 81.0, MealPlanRefundAmountForDay(plan, legacy))

	// The v3 column wins whenever it is set.
	p := 75
	legacy.RefundPercent = &p
	require.Equal(t, 75, legacy.RefundPercentOf())
	require.Equal(t, 121.5, MealPlanRefundAmountForDay(plan, legacy))
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
