package handlers

// meal_plan_cancel_test.go — the customer "cancel before start/approval" rule.
// Pure eligibility unit tests (the handler's DB path mirrors the reject flow's
// status-guarded transition + RefundUndeliveredDays, already covered elsewhere).

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func day(status models.MealPlanDayStatus) models.MealPlanDay {
	return models.MealPlanDay{Status: status}
}

func TestMealPlanCancellableBeforeStart(t *testing.T) {
	cases := []struct {
		name   string
		status models.MealPlanStatus
		days   []models.MealPlanDay
		wantOK bool
	}{
		{"pending request — withdraw", models.MealPlanPendingChef, nil, true},
		{"awaiting customer approval", models.MealPlanAwaitingCustomer, nil, true},
		{"chef modified", models.MealPlanChefModified, nil, true},
		{"confirmed, no day served", models.MealPlanConfirmed,
			[]models.MealPlanDay{day(models.MealPlanDayConfirmed), day(models.MealPlanDayConfirmed)}, true},
		// Started: a day already prepared/delivered → penalty flow, not this one.
		{"confirmed but a day prepared", models.MealPlanConfirmed,
			[]models.MealPlanDay{day(models.MealPlanDayConfirmed), day(models.MealPlanDayPrepared)}, false},
		{"active with a delivered day", models.MealPlanActive,
			[]models.MealPlanDay{day(models.MealPlanDayDelivered)}, false},
		// Terminal states are not cancellable.
		{"already cancelled", models.MealPlanCancelled, nil, false},
		{"active (deliveries in progress)", models.MealPlanActive, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, reason := mealPlanCancellableBeforeStart(tc.status, tc.days)
			require.Equal(t, tc.wantOK, ok, "reason: %q", reason)
			if !tc.wantOK {
				require.NotEmpty(t, reason)
			}
		})
	}
}

// cancelled is NOT a live status, so a cancelled plan must not block a rebooking
// under the duplicate-plan guard (#409) — the two rules must stay consistent.
func TestCancelledIsNotALivePlanStatus(t *testing.T) {
	live := map[models.MealPlanStatus]bool{
		models.MealPlanPendingChef: true, models.MealPlanChefAcceptedFull: true,
		models.MealPlanChefModified: true, models.MealPlanAwaitingCustomer: true,
		models.MealPlanConfirmed: true, models.MealPlanActive: true,
	}
	require.False(t, live[models.MealPlanCancelled], "a cancelled plan frees the customer to rebook")
}
