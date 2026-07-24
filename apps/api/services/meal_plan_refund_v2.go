package services

// meal_plan_refund_v2.go — the refund seam for the v2 meal-plan / group-order refund workflow
// (docs/meal-plan-refund-flow-design.md), gated behind MEALPLAN_REFUND_FLOW_V2_ENABLED.
//
// FIRM RULE: the refund base EXCLUDES the platform fee, GST, and delivery. The customer is
// refunded only the food value net of the platform commission (perDaySkipRefund); the platform
// ALWAYS keeps its fee + GST + delivery. The >12h auto-approve and a chef "accept full" refund
// 100% of that base; "half" refunds 50%; "none" refunds nothing (the chef started prep and is
// paid in full). This unifies every meal-plan refund path onto one fee/GST-excluded base — unlike
// the legacy cancel path (perDayGross), which wrongly included GST + delivery.

import (
	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// MealPlanRefundFlowV2Active reports whether the v2 refund workflow is switched on.
func MealPlanRefundFlowV2Active() bool {
	return config.AppConfig != nil && config.AppConfig.MealPlanRefundFlowV2Enabled
}

// RefundProportion is how much of a day's fee/GST-excluded food base is refunded: the >12h
// auto path and a chef "accept full" refund 100%; "half" refunds 50%; "none" refunds nothing.
type RefundProportion string

const (
	RefundProportionFull RefundProportion = "full"
	RefundProportionHalf RefundProportion = "half"
	RefundProportionNone RefundProportion = "none"
)

// ValidRefundProportion reports whether p is one of the three allowed decisions.
func ValidRefundProportion(p RefundProportion) bool {
	return p == RefundProportionFull || p == RefundProportionHalf || p == RefundProportionNone
}

// refundProportionFactor maps a proportion to its multiplier; anything unknown → 0 (safe).
func refundProportionFactor(p RefundProportion) float64 {
	switch p {
	case RefundProportionFull:
		return 1.0
	case RefundProportionHalf:
		return 0.5
	default:
		return 0.0
	}
}

// MealPlanRefundAmount is the customer refund for one day at the given proportion, computed off
// the fee/GST-EXCLUDED base (food − commission via perDaySkipRefund). It NEVER includes the
// platform fee, GST, or delivery. This is the single amount seam every v2 refund path uses.
func MealPlanRefundAmount(plan *models.MealPlan, day *models.MealPlanDay, p RefundProportion) float64 {
	// Basis: the day's frozen commission rate (what the held transfer was sized at);
	// perDaySkipRefund falls back to DefaultCommissionRate for legacy days (rate 0).
	base := perDaySkipRefund(plan, day, day.CommissionRate) // food − commission (no GST, no delivery)
	return Round2(base * refundProportionFactor(p))
}
