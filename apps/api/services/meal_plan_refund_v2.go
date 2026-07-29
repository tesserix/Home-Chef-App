package services

// meal_plan_refund_v2.go — the refund seam for the meal-plan / group-order refund workflow
// (docs/refund-policy-v3-spec.md), gated behind MEALPLAN_REFUND_FLOW_V2_ENABLED.
//
// REFUND BASE — v3 (#834), a deliberate REVERSAL of the v2 rule it replaces.
//
// The base is now EVERYTHING THE CUSTOMER PAID for that day: food + GST + delivery. The v2
// rule (and docs/meal-plan-refund-flow-design.md §1, whose "FIRM RULE" this supersedes)
// refunded food net of the platform commission and kept the GST; a cancelled day now returns
// the whole gross and the platform keeps nothing on it:
//
//	₹100 food + ₹8 GST + ₹5 delivery = ₹113 paid. Cancelled at the 75% floor → ₹84.75 back.
//
// There is no separate customer-facing "platform fee" line on a meal plan — the platform's
// take is the commission withheld from the CHEF inside the food price (perDayNetPayout), so
// refunding 100% of the food price is exactly what "the platform fee is refunded too" means
// here. Refunding collected GST is why credit notes must be issued alongside
// (services/credit_note.go): the tax genuinely goes back, so the filing must be adjusted.
//
// The proportion is a BOUNDED PERCENTAGE (0–100) the chef chooses within a lead-time floor,
// not the old {full,half,none} enum — see meal_plan_refund_tiers.go.

import (
	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// MealPlanRefundFlowV2Active reports whether the refund workflow is switched on.
func MealPlanRefundFlowV2Active() bool {
	return config.AppConfig != nil && config.AppConfig.MealPlanRefundFlowV2Enabled
}

// ValidRefundProportion reports whether p is one of the recognised legacy labels. Retained
// for the pre-v3 request shape the vendor apps still send ({"choice":"full"}) — a new client
// sends a percentage, which is validated against the day's floor instead.
func ValidRefundProportion(p models.RefundProportion) bool {
	return p == models.RefundProportionFull || p == models.RefundProportionHalf || p == models.RefundProportionNone
}

// ClampRefundPercent bounds a percentage into 0–100.
func ClampRefundPercent(p int) int {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

// mealPlanDayGrossPaid is the v3 refund base: the FULL amount the customer paid for one day —
// food + that day's proportional GST + that day's delivery share.
//
// It deliberately apportions delivery by the day's share of the plan subtotal (the basis
// perDayFoodGST uses) rather than dividing by len(plan.Days) the way perDayGross does. Every
// refund caller loads the plan with a narrow Select and no Days preload, where perDayGross
// silently collapses to the bare food price — dropping the very GST and delivery this policy
// exists to return. For equal-priced days the two agree exactly.
//
// Falls back to the bare food price when the plan carries no snapshotted totals (legacy rows).
func mealPlanDayGrossPaid(plan *models.MealPlan, day *models.MealPlanDay) float64 {
	if plan == nil {
		return Round2(day.Price)
	}
	return Round2(day.Price + perDayFoodGST(plan, day) + perDayDeliveryRefund(plan, day))
}

// MealPlanRefundGSTComponent is the GST slice of a refund at the given percentage — the amount
// a credit note must cover so the returned tax is backed out of the filing.
func MealPlanRefundGSTComponent(plan *models.MealPlan, day *models.MealPlanDay, percent int) float64 {
	if plan == nil {
		return 0
	}
	return Round2(perDayFoodGST(plan, day) * float64(ClampRefundPercent(percent)) / 100)
}

// MealPlanRefundAmount is the customer refund for one day at the agreed percentage, computed
// off the full gross the customer paid (food + GST + delivery). The single amount seam every
// refund path uses.
//
// NOTE: the plan must carry its snapshotted totals (Subtotal/Tax/Total) for the GST and
// delivery terms — callers that Select a subset of columns must include them.
func MealPlanRefundAmount(plan *models.MealPlan, day *models.MealPlanDay, percent int) float64 {
	return Round2(mealPlanDayGrossPaid(plan, day) * float64(ClampRefundPercent(percent)) / 100)
}

// MealPlanRefundAmountForDay is MealPlanRefundAmount at the percentage already agreed on the
// day (the v3 column, else the pre-v3 enum).
func MealPlanRefundAmountForDay(plan *models.MealPlan, day *models.MealPlanDay) float64 {
	return MealPlanRefundAmount(plan, day, day.RefundPercentOf())
}
