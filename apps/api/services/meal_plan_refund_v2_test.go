package services

// meal_plan_refund_v2_test.go — pins the refund rule: the v2 refund is (food − commission) PLUS the
// day's delivery fee, scaled by the chef's Full/Half/None decision. Per the 2026-07 policy the
// delivery fee IS refunded but GST and the platform commission are NOT, so it stays below the legacy
// make-whole gross (which also returns GST).

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestMealPlanRefundAmount_IncludesDeliveryExcludesGSTAndFee(t *testing.T) {
	plan := &models.MealPlan{
		Subtotal: 320, Tax: 32, Total: 372, // 2 days: food 320, GST 32, delivery 20
		Days: []models.MealPlanDay{
			{Price: 160, CommissionRate: 0.15},
			{Price: 160, CommissionRate: 0.15},
		},
	}
	day := &plan.Days[0]

	// Full = (food − commission) + the day's delivery = (160 − 24) + 10 = 146. Delivery IS
	// refunded (the day's food-share of the plan's 20 delivery); GST + commission are NOT.
	require.Equal(t, 146.0, MealPlanRefundAmount(plan, day, models.RefundProportionFull))
	require.Equal(t, 73.0, MealPlanRefundAmount(plan, day, models.RefundProportionHalf))
	require.Equal(t, 0.0, MealPlanRefundAmount(plan, day, models.RefundProportionNone))

	// Still excludes GST: the legacy make-whole gross (food + GST + delivery) stays strictly
	// larger than a FULL v2 refund, because gross also returns the GST the v2 refund omits.
	gross := perDayGross(plan, day) // 160 + 16 GST + 10 delivery = 186
	require.Equal(t, 186.0, gross)
	require.Greater(t, gross, MealPlanRefundAmount(plan, day, models.RefundProportionFull),
		"gross includes GST; the v2 refund must not")
}

// Legacy days with no frozen commission rate fall back to DefaultCommissionRate (not 0),
// so an unset rate never yields a full-food (commission-included) refund.
func TestMealPlanRefundAmount_LegacyRateFallback(t *testing.T) {
	plan := &models.MealPlan{Subtotal: 200, Days: []models.MealPlanDay{{Price: 200, CommissionRate: 0}}}
	day := &plan.Days[0]
	full := MealPlanRefundAmount(plan, day, models.RefundProportionFull)
	require.Less(t, full, 200.0, "unset rate must still deduct the default commission, not refund full food")
	require.Equal(t, perDaySkipRefund(plan, day, DefaultCommissionRate), full)
}

func TestValidRefundProportion(t *testing.T) {
	require.True(t, ValidRefundProportion(models.RefundProportionFull))
	require.True(t, ValidRefundProportion(models.RefundProportionHalf))
	require.True(t, ValidRefundProportion(models.RefundProportionNone))
	require.False(t, ValidRefundProportion("quarter"))
	require.False(t, ValidRefundProportion(""))
}
