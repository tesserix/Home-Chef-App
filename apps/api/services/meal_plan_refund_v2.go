package services

// meal_plan_refund_v2.go — the refund seam for the v2 meal-plan / group-order refund workflow
// (docs/meal-plan-refund-flow-design.md), gated behind MEALPLAN_REFUND_FLOW_V2_ENABLED.
//
// FIRM RULE (2026-07): the refund base is the food value net of the platform commission PLUS the
// day's delivery fee (perDaySkipRefund). It EXCLUDES GST and the platform commission; the delivery
// fee for the cancelled/skipped day IS refunded. The >12h auto-approve and a chef "accept full"
// refund 100% of that base; "half" refunds 50%; "none" refunds nothing (the chef started prep and
// is paid in full). This unifies every meal-plan refund path onto one base — unlike the legacy cancel
// path (perDayGross), which also returned the GST.

import (
	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// MealPlanRefundFlowV2Active reports whether the v2 refund workflow is switched on.
func MealPlanRefundFlowV2Active() bool {
	return config.AppConfig != nil && config.AppConfig.MealPlanRefundFlowV2Enabled
}

// ValidRefundProportion reports whether p is one of the three allowed decisions.
func ValidRefundProportion(p models.RefundProportion) bool {
	return p == models.RefundProportionFull || p == models.RefundProportionHalf || p == models.RefundProportionNone
}

// refundProportionFactor maps a proportion to its multiplier; anything unknown → 0 (safe).
func refundProportionFactor(p models.RefundProportion) float64 {
	switch p {
	case models.RefundProportionFull:
		return 1.0
	case models.RefundProportionHalf:
		return 0.5
	default:
		return 0.0
	}
}

// MealPlanRefundAmount is the customer refund for one day at the given proportion, computed off the
// base (food − commission + the day's delivery via perDaySkipRefund). It NEVER includes GST or the
// platform commission, but DOES include the day's delivery fee. This is the single amount seam every
// v2 refund path uses. NOTE: the plan must carry its snapshotted totals (Subtotal/Tax/Total) for the
// delivery term — callers that Select a subset must include those columns.
func MealPlanRefundAmount(plan *models.MealPlan, day *models.MealPlanDay, p models.RefundProportion) float64 {
	// Basis: the day's frozen commission rate (what the held transfer was sized at);
	// perDaySkipRefund falls back to DefaultCommissionRate for legacy days (rate 0).
	base := perDaySkipRefund(plan, day, day.CommissionRate) // (food − commission) + delivery (no GST)
	return Round2(base * refundProportionFactor(p))
}
