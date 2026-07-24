package services

import (
	"fmt"
	"time"

	"github.com/homechef/api/models"
)

// meal_plan_day_start.go — the single source of truth for "when does a plan day's
// cooking start (IST)". This anchors the UNIFIED skip/lock boundary (#422): the
// customer may skip a day until its order LOCKS, and the fulfilment cron generates
// (locks) the order exactly `mealPlanLockLead` before this start. So the skip gate
// (handlers/meal_plan.go SkipMealPlanDay) and the order-generation gate
// (generateDueDayOrders) both key off this one function — they can never disagree.

// mealPlanLockLead is how far before a day's cooking start the order is generated
// (locked) — and therefore the deadline by which a customer must have requested a
// skip. One value drives both sides of the boundary.
const mealPlanLockLead = 12 * time.Hour

// ParseClockHHMM parses an "HH:MM" 24-hour string into (hour, minute, ok). ok is
// false for anything malformed or out of range, so callers fall back to a default.
func ParseClockHHMM(s string) (int, int, bool) {
	var hh, mm int
	if n, err := fmt.Sscanf(s, "%d:%d", &hh, &mm); err != nil || n != 2 {
		return 0, 0, false
	}
	if hh < 0 || hh > 23 || mm < 0 || mm > 59 {
		return 0, 0, false
	}
	return hh, mm, true
}

// MealPlanDayStartIST approximates the IST start time of a plan day's cooking. There
// is no exact per-slot start stored, so it uses the chef's ChefSchedule OpenTime for
// that IST weekday (an open row with a valid HH:MM); otherwise a per-slot default —
// lunch 12:00 IST, dinner 19:00 IST. Pure (no DB) so it is unit-tested directly and
// callable from both the handler (skip gate) and the fulfilment cron (lock/gen gate).
func MealPlanDayStartIST(schedules []models.ChefSchedule, day *models.MealPlanDay) time.Time {
	d := day.Date.In(scheduleIST)
	weekday := int(d.Weekday()) // 0=Sunday .. 6=Saturday — matches ChefSchedule.DayOfWeek
	for i := range schedules {
		s := schedules[i]
		if s.DayOfWeek != weekday || s.IsClosed {
			continue
		}
		if hh, mm, ok := ParseClockHHMM(s.OpenTime); ok {
			return time.Date(d.Year(), d.Month(), d.Day(), hh, mm, 0, 0, scheduleIST)
		}
	}
	hh := 12 // lunch default
	if day.Slot == models.MealSlotDinner {
		hh = 19 // dinner default
	}
	return time.Date(d.Year(), d.Month(), d.Day(), hh, 0, 0, 0, scheduleIST)
}
