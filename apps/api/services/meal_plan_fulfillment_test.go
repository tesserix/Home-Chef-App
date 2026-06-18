package services

// Unit tests for the tiffin fulfilment + escrow helpers (#197/#194). Pure
// logic only — DB-backed generation/release is exercised in handler/E2E tests.

import (
	"testing"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

func TestAllDaysTerminal(t *testing.T) {
	if allDaysTerminal(&models.MealPlan{}) {
		t.Fatal("a plan with no days is not terminal")
	}
	mixed := &models.MealPlan{Days: []models.MealPlanDay{
		{Status: models.MealPlanDayDelivered},
		{Status: models.MealPlanDayConfirmed}, // still live
	}}
	if allDaysTerminal(mixed) {
		t.Fatal("a confirmed day means the plan is not yet complete")
	}
	done := &models.MealPlan{Days: []models.MealPlanDay{
		{Status: models.MealPlanDayDelivered},
		{Status: models.MealPlanDaySkipped},
		{Status: models.MealPlanDayDeclined},
		{Status: models.MealPlanDayRefunded},
		{Status: models.MealPlanDayCancelled},
	}}
	if !allDaysTerminal(done) {
		t.Fatal("all days terminal → plan should complete")
	}
}

func TestIsPayableDayStatus(t *testing.T) {
	for _, s := range []models.MealPlanDayStatus{
		models.MealPlanDayAccepted, models.MealPlanDayConfirmed,
		models.MealPlanDayPrepared, models.MealPlanDayDelivered,
	} {
		if !isPayableDayStatus(s) {
			t.Fatalf("%s should be payable (chef payout held)", s)
		}
	}
	for _, s := range []models.MealPlanDayStatus{
		models.MealPlanDayRequested, models.MealPlanDayDeclined,
		models.MealPlanDaySkipped, models.MealPlanDayCancelled, models.MealPlanDayRefunded,
	} {
		if isPayableDayStatus(s) {
			t.Fatalf("%s must NOT hold a chef payout", s)
		}
	}
}

func TestMealPlanEscrowActive(t *testing.T) {
	saved := config.AppConfig
	defer func() { config.AppConfig = saved }()

	config.AppConfig = nil
	if MealPlanEscrowActive() {
		t.Fatal("escrow must be inactive when config is nil (fail safe)")
	}
	config.AppConfig = &config.Config{MealPlanEscrowEnabled: false}
	if MealPlanEscrowActive() {
		t.Fatal("escrow must be inactive when the flag is false")
	}
	config.AppConfig = &config.Config{MealPlanEscrowEnabled: true}
	if !MealPlanEscrowActive() {
		t.Fatal("escrow must be active when the flag is true")
	}
}

func TestDayRefundKey_Idempotent(t *testing.T) {
	id := uuid.New()
	want := "mealplan-refund:" + id.String()
	if got := dayRefundKey(id); got != want {
		t.Fatalf("dayRefundKey = %q, want %q (stable per-day key for ledger dedup)", got, want)
	}
}
