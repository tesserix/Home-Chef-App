package services

// meal_plan_refund_v2_test.go — pins the FIRM rule: the v2 refund base excludes the platform
// fee, GST, and delivery. It is food − commission, scaled by the chef's Full/Half/None decision,
// and always strictly less than the legacy make-whole gross (which wrongly included GST+delivery).

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestMealPlanRefundAmount_ExcludesFeeGSTDelivery(t *testing.T) {
	plan := &models.MealPlan{
		Subtotal: 320, Tax: 32, Total: 372, // 2 days: food 320, GST 32, delivery 20
		Days: []models.MealPlanDay{
			{Price: 160, CommissionRate: 0.15},
			{Price: 160, CommissionRate: 0.15},
		},
	}
	day := &plan.Days[0]

	// Full = food − commission = 160 − 0.15×160 = 136 (no GST, no delivery).
	require.Equal(t, 136.0, MealPlanRefundAmount(plan, day, models.RefundProportionFull))
	require.Equal(t, 68.0, MealPlanRefundAmount(plan, day, models.RefundProportionHalf))
	require.Equal(t, 0.0, MealPlanRefundAmount(plan, day, models.RefundProportionNone))

	// Proof it excludes GST + delivery: the legacy make-whole gross (food + GST + delivery)
	// is strictly larger than even a FULL v2 refund.
	gross := perDayGross(plan, day) // 160 + 16 GST + 10 delivery = 186
	require.Equal(t, 186.0, gross)
	require.Greater(t, gross, MealPlanRefundAmount(plan, day, models.RefundProportionFull),
		"legacy gross includes GST+delivery; the v2 refund must never")
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
