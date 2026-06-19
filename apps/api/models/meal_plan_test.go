package models

import "testing"

func TestMealPlanAcceptedTotal(t *testing.T) {
	p := MealPlan{Days: []MealPlanDay{
		{Price: 120, Status: MealPlanDayConfirmed},
		{Price: 120, Status: MealPlanDayDelivered},
		{Price: 120, Status: MealPlanDayDeclined},  // chef cherry-picked out → refunded, excluded
		{Price: 120, Status: MealPlanDaySkipped},   // customer skipped → refunded, excluded
		{Price: 120, Status: MealPlanDayRequested}, // still in scope
	}}
	if got := p.AcceptedTotal(); got != 360 {
		t.Fatalf("AcceptedTotal = %v, want 360 (confirmed+delivered+requested)", got)
	}
}
