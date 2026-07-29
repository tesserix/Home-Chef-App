package services

// meal_plan_refund_v2.go — the refund seam for the meal-plan / group-order refund workflow
// (docs/refund-policy-v3-spec.md), gated behind MEALPLAN_REFUND_FLOW_V2_ENABLED.
//
// REFUND BASE — v3 (#834). GST is now refunded; the platform commission is NOT.
//
//	base = food − platform commission + that day's GST + that day's delivery
//
// For a ₹100 day at a 15% commission with 8% GST and ₹10 delivery, a 100% refund is ₹103:
//
//	food 100 − commission 15 = 85 · + GST 8 · + delivery 10  →  ₹103
//
// This changes exactly ONE thing from v2 (docs/meal-plan-refund-flow-design.md §1, whose
// "FIRM RULE" said GST is never refunded): the tax now goes back. Food-minus-commission plus
// delivery is unchanged — it is the same figure perDaySkipRefund computes for the admin
// approve-skip path, so the two stay aligned apart from the GST term.
//
// There is no separate customer-facing "platform fee" line on a meal plan. The platform's take
// is the commission withheld from the CHEF inside the food price (perDayNetPayout), so
// retaining it means refunding food NET of that commission. The platform therefore keeps its
// margin on a cancelled day, and is out of pocket only the GST and delivery it returns.
//
// Refunding collected GST is why credit notes must be issued alongside
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

// floorPaise truncates to whole paise instead of rounding to nearest.
//
// THIS DIRECTION IS DELIBERATE AND LOAD-BEARING. Each day's refund is computed and rounded
// INDEPENDENTLY, but the exact (unrounded) per-day shares sum to precisely the plan total.
// Round-to-nearest lets several days each round UP, so the plan's refunds can add up to more
// than the customer ever paid — a 13-day plan at ₹49.99 with 5% GST refunds ₹682.37 against
// ₹682.36 charged. Truncating guarantees every day's share is at or below its exact value, so
// the sum can never cross the amount captured. The cost is at most one paise per day, retained
// by the platform; the alternative is minting money.
//
// The epsilon absorbs binary-float representation error (92.59 + 7.41 lands a hair above 100.0)
// so an exact figure is not truncated to a paise below itself.
func floorPaise(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return float64(int64(v*100+1e-9)) / 100
}

// mealPlanDayRefundBase is the v3 refund base for ONE day:
//
//	(food − platform commission) + that day's proportional GST + that day's delivery share
//
// The commission is resolved from the day's FROZEN rate — the rate its held chef transfer was
// sized at — falling back to DefaultCommissionRate for legacy days, exactly as perDaySkipRefund
// and perDayNetPayout do. Using the live rate instead would let a mid-flight rate change move a
// refund away from the money actually held for that day.
//
// GST and delivery are apportioned by the day's share of the plan subtotal rather than by
// dividing by len(plan.Days) the way perDayGross does. Every refund caller loads the plan with
// a narrow Select and no Days preload, where perDayGross silently collapses to the bare food
// price — dropping the very GST and delivery this policy exists to apportion. For equal-priced
// days the two agree exactly.
//
// TWO CEILINGS enforce "never refund more than was paid":
//   - floorPaise, so independently-rounded days can't sum past what was captured;
//   - a hard cap at plan.Total, the amount the Razorpay advance actually captured
//     (VerifyMealPlanAdvance binds the payment to it). The proportional share is only
//     meaningful while the day prices sum to plan.Subtotal, which every legitimate flow
//     maintains — the cap is the backstop for data where they don't, so a single corrupt day
//     price can never draw more out of escrow than went into it.
//
// Falls back to food-minus-commission when the plan carries no snapshotted totals (legacy rows):
// a plan with no recorded tax or delivery must not have either invented for it.
func mealPlanDayRefundBase(plan *models.MealPlan, day *models.MealPlanDay) float64 {
	rate := day.CommissionRate
	if rate <= 0 || rate >= 1 {
		rate = DefaultCommissionRate
	}
	base := day.Price - rate*day.Price
	if plan == nil {
		return floorPaise(base)
	}
	base += perDayFoodGST(plan, day) + perDayDeliveryRefund(plan, day)
	if plan.Total > 0 && base > plan.Total {
		base = plan.Total
	}
	return floorPaise(base)
}

// MealPlanRefundGSTComponent is the GST slice of a refund at the given percentage — the amount
// a credit note must cover so the returned tax is backed out of the filing.
//
// Floored for the same reason the refund is: a note must never claim more output tax reversed
// than was actually returned, or the filing adjustment overshoots in the platform's favour.
func MealPlanRefundGSTComponent(plan *models.MealPlan, day *models.MealPlanDay, percent int) float64 {
	if plan == nil {
		return 0
	}
	return floorPaise(perDayFoodGST(plan, day) * float64(ClampRefundPercent(percent)) / 100)
}

// MealPlanRefundAmount is the customer refund for one day at the agreed percentage, computed
// off the day's refund base (food − commission + GST + delivery). The single amount seam every
// refund path uses.
//
// NOTE: the plan must carry its snapshotted totals (Subtotal/Tax/Total) for the GST and
// delivery terms — callers that Select a subset of columns must include them.
func MealPlanRefundAmount(plan *models.MealPlan, day *models.MealPlanDay, percent int) float64 {
	// floorPaise again on the way out: a percentage of an already-floored base can itself land
	// mid-paise, and rounding that up would put a partial refund above its exact share.
	return floorPaise(mealPlanDayRefundBase(plan, day) * float64(ClampRefundPercent(percent)) / 100)
}

// MealPlanRefundAmountForDay is MealPlanRefundAmount at the percentage already agreed on the
// day (the v3 column, else the pre-v3 enum).
func MealPlanRefundAmountForDay(plan *models.MealPlan, day *models.MealPlanDay) float64 {
	return MealPlanRefundAmount(plan, day, day.RefundPercentOf())
}
