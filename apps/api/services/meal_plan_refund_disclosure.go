package services

// meal_plan_refund_disclosure.go — what the customer is told about a refund (#1041).
//
// The refund base is food − commission + GST + delivery, so the agreed percentage never
// applies to the plan total the customer paid. Stating the percentage alone left a gap the
// customer had no way to reconcile; both seams here name the base the percentage acts on.

import (
	"fmt"

	"github.com/homechef/api/models"
)

// MealPlanRefundReason is the ledger line for one day's refund.
func MealPlanRefundReason(plan *models.MealPlan, percent int) string {
	return fmt.Sprintf(
		"Tiffin %s — %d%% of the meal price less kitchen commission, plus its GST and delivery",
		plan.MealPlanNumber, ClampRefundPercent(percent),
	)
}

// AnnotateMealPlanRefunds prices the refund on every day that actually received one, for the
// customer-facing plan response. Days that were skipped, forfeited, or never refunded are left
// unpriced so the client shows nothing rather than a misleading ₹0.00.
func AnnotateMealPlanRefunds(plan *models.MealPlan) {
	if plan == nil {
		return
	}
	for i := range plan.Days {
		day := &plan.Days[i]
		day.RefundAmount = nil
		if day.Status != models.MealPlanDayRefunded {
			continue
		}
		amount := MealPlanRefundAmountForDay(plan, day)
		if amount <= 0 {
			continue
		}
		day.RefundAmount = &amount
	}
}
