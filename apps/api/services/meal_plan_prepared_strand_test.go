package services

import (
	"testing"

	"github.com/homechef/api/models"
)

// A day the chef cooked ahead of the order lock must still be treated as needing
// an order.
//
// Regression: generateDueDayOrders and sweepStuckDays both keyed on
// status == confirmed. Orders generate mealPlanLockLead (12h) before cook-start,
// so a lunch locks at midnight while "Tomorrow's prep" invites the chef to mark
// it prepared the previous afternoon. A day marked prepared first therefore never
// got an order — so MarkMealPlanDayDelivered (which looks the day up BY order_id)
// could never fire, the stuck-day sweep skipped it too, and `prepared` is not
// terminal, so completeFinishedPlans never ran. The customer's escrow for that day
// had no path out: not delivered, not refunded, not released. Cooking early must
// never void a day.
func TestDayAwaitingOrder_IncludesPrepared(t *testing.T) {
	for _, s := range []models.MealPlanDayStatus{
		models.MealPlanDayConfirmed,
		models.MealPlanDayPrepared,
	} {
		if !dayAwaitingOrder(s) {
			t.Errorf("status %q still needs an order generated, but was skipped", s)
		}
	}
}

// Terminal and pre-confirmation states must NOT be picked up: generating an order
// for a refunded or declined day would re-park money that was already returned,
// and a `requested` day has not been paid for at all.
func TestDayAwaitingOrder_ExcludesTerminalAndUnpaid(t *testing.T) {
	for _, s := range []models.MealPlanDayStatus{
		models.MealPlanDayRequested,
		models.MealPlanDayAccepted,
		models.MealPlanDayDeclined,
		models.MealPlanDayDelivered,
		models.MealPlanDaySkipped,
		models.MealPlanDaySkipRequested,
		models.MealPlanDayCancelled,
		models.MealPlanDayRefunded,
		models.MealPlanDayFailed,
	} {
		if dayAwaitingOrder(s) {
			t.Errorf("status %q must not be treated as awaiting an order", s)
		}
	}
}
