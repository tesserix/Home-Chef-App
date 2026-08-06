package services

// #1041: a cancelled plan told the customer a refund happened but never how much, and the one
// place an amount appeared (the wallet ledger) named a bare "75%" against a total that
// percentage does not apply to.

import (
	"testing"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

func disclosurePlan() (*models.MealPlan, []models.MealPlanDay) {
	planID := uuid.New()
	days := []models.MealPlanDay{
		{ID: uuid.New(), MealPlanID: planID, Price: 100, CommissionRate: 0.15, Status: models.MealPlanDayRefunded, RefundStage: models.MPRefundResolved},
		{ID: uuid.New(), MealPlanID: planID, Price: 100, CommissionRate: 0.15, Status: models.MealPlanDayDelivered},
	}
	plan := &models.MealPlan{
		ID: planID, MealPlanNumber: "MP-7b6bf84a",
		Subtotal: 200, Tax: 16, Total: 236, Days: days,
	}
	return plan, days
}

func TestAnnotateMealPlanRefunds_PricesEveryResolvedRefund(t *testing.T) {
	plan, _ := disclosurePlan()
	percent := 75
	plan.Days[0].RefundPercent = &percent

	AnnotateMealPlanRefunds(plan)

	// base = 100 − 15 commission + 8 GST + 10 delivery = 103; 75% = 77.25.
	if plan.Days[0].RefundAmount == nil {
		t.Fatal("refunded day carries no refund amount")
	}
	if got := *plan.Days[0].RefundAmount; got != 77.25 {
		t.Fatalf("refund amount = %.2f, want 77.25", got)
	}
	if plan.Days[1].RefundAmount != nil {
		t.Fatalf("delivered day was priced a refund: %.2f", *plan.Days[1].RefundAmount)
	}
}

// A day the chef priced at 0% is skipped, not refunded — showing "₹0.00 refunded" would be
// less honest than showing nothing.
func TestAnnotateMealPlanRefunds_LeavesAZeroRefundUnpriced(t *testing.T) {
	plan, _ := disclosurePlan()
	percent := 0
	plan.Days[0].RefundPercent = &percent
	plan.Days[0].Status = models.MealPlanDaySkipped

	AnnotateMealPlanRefunds(plan)

	if plan.Days[0].RefundAmount != nil {
		t.Fatalf("skipped day was priced a refund: %.2f", *plan.Days[0].RefundAmount)
	}
}

func TestMealPlanRefundReason_NamesTheBaseThePercentageAppliesTo(t *testing.T) {
	plan, _ := disclosurePlan()

	reason := MealPlanRefundReason(plan, 75)

	want := "Tiffin MP-7b6bf84a — 75% of the meal price less kitchen commission, plus its GST and delivery"
	if reason != want {
		t.Fatalf("reason = %q, want %q", reason, want)
	}
}
