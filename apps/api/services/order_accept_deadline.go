package services

// order_accept_deadline.go — when does a chef run out of time to accept? (#694)
//
// THE RULE
//
// An order stays the chef's to accept until THEIR KITCHEN CLOSES for the meal it
// was placed for, on the date it was placed for. Order today's lunch and the
// chef's lunch service ends at 14:00 → they have until 14:00 today. Order
// tomorrow's dinner and their dinner ends at 22:00 → they have until 22:00
// tomorrow.
//
// WHAT THIS REPLACES, AND WHY IT MATTERED
//
// The first cut of the sweep — and temporal/workflows/order.go's
// chefAcceptTimeout — both used "30 minutes after payment". That is not a
// slightly-wrong number, it is the wrong MODEL: it would have voided and
// refunded every ADVANCE order half an hour after it was placed. A customer
// booking tomorrow's dinner would have been refunded before the chef ever
// plausibly looked. The deadline has to come from the chef's own schedule, not
// from a stopwatch started at checkout.
//
// WHERE THE TIME COMES FROM (the chef's own configuration, in order)
//
//  1. The slot's END for that meal (ChefCapacitySettings.LunchSlotEnd /
//     DinnerSlotEnd) — the chef's stated "my lunch service runs until 14:00".
//     This is the closest thing to "the kitchen is closed for this meal".
//  2. The kitchen's CLOSE for that weekday (ChefSchedule.CloseTime) — a chef who
//     hasn't configured slot windows still says when they shut.
//  3. A platform default per meal, so an unconfigured chef still has a deadline
//     and the customer's money is never captured forever.
//
// Deliberately NOT used: LunchCutoff / DinnerCutoff. Those are ORDER cutoffs —
// when the chef stops ACCEPTING NEW orders (e.g. "no lunch orders after 11:00").
// Voiding against them would kill an order placed at 10:55 at 11:00, five minutes
// later, while the chef is still cooking that very lunch. A cutoff is a door
// closing on new business, not a kitchen closing.

import (
	"fmt"
	"strings"
	"time"

	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// Platform fallbacks for a chef who has configured neither slot windows nor
// opening hours. Late enough to be generous to the chef — the cost of voiding
// too early (killing an order they would have cooked) is far worse than voiding
// an hour late.
const (
	defaultLunchCloseIST  = "15:00"
	defaultDinnerCloseIST = "22:00"
)

// AcceptDeadline is when the chef's chance to accept an order expires, plus how
// it was worked out — the source is carried so a void is explainable after the
// fact ("we closed it at your stated dinner end of 22:00"), not a bare timestamp.
type AcceptDeadline struct {
	At time.Time
	// Slot is the meal the order was placed for: "lunch" or "dinner".
	Slot string
	// Source is where At came from: "slot_end", "kitchen_close" or "platform_default".
	Source string
}

// inferSlot works out which meal an order belongs to.
//
// DeliverySlot is authoritative when the customer picked a scheduled window. An
// ASAP order has none, so it is inferred from the hour it is FOR: anything at or
// before 16:00 IST is lunch, later is dinner. The boundary is late on purpose —
// misfiling a 15:00 order as dinner gives the chef longer, which is the harmless
// direction.
func inferSlot(order *models.Order) string {
	if s := strings.ToLower(strings.TrimSpace(order.DeliverySlot)); s == "lunch" || s == "dinner" {
		return s
	}
	at := order.CreatedAt
	if order.ScheduledFor != nil {
		at = *order.ScheduledFor
	}
	if at.In(capacityIST).Hour() <= 16 {
		return string(models.MealSlotLunch)
	}
	return string(models.MealSlotDinner)
}

// serviceDay is the IST calendar day the order is FOR — the scheduled day when
// there is one, else the day it was placed. Using the placed day for an advance
// order is exactly the bug this file exists to prevent.
func serviceDay(order *models.Order) time.Time {
	if order.ScheduledFor != nil {
		return CapacityDay(*order.ScheduledFor)
	}
	return CapacityDay(order.CreatedAt)
}

// atIST puts an "HH:MM" IST wall-clock time onto a calendar day.
func atIST(day time.Time, hhmm string) (time.Time, bool) {
	hhmm = strings.TrimSpace(hhmm)
	if hhmm == "" {
		return time.Time{}, false
	}
	var h, m int
	if _, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m); err != nil {
		return time.Time{}, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return time.Time{}, false
	}
	d := day.In(capacityIST)
	return time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, capacityIST), true
}

// ResolveAcceptDeadline works out when `order` stops being acceptable, reading
// the chef's own configuration. It never fails: an unconfigured chef falls back
// to the platform default, because "no deadline" means the customer's money sits
// captured forever, which is the hole this closes.
func ResolveAcceptDeadline(db *gorm.DB, order *models.Order) AcceptDeadline {
	slot := inferSlot(order)
	day := serviceDay(order)

	// 1. The chef's stated end of THIS meal's service.
	var cap models.ChefCapacitySettings
	if err := db.Where("chef_id = ?", order.ChefID).First(&cap).Error; err == nil {
		end := cap.LunchSlotEnd
		if slot == string(models.MealSlotDinner) {
			end = cap.DinnerSlotEnd
		}
		if at, ok := atIST(day, end); ok {
			return AcceptDeadline{At: at, Slot: slot, Source: "slot_end"}
		}
	}

	// 2. When the kitchen closes on that weekday.
	var sched models.ChefSchedule
	if err := db.Where("chef_id = ? AND day_of_week = ?", order.ChefID, int(day.In(capacityIST).Weekday())).
		First(&sched).Error; err == nil && !sched.IsClosed {
		if at, ok := atIST(day, sched.CloseTime); ok {
			return AcceptDeadline{At: at, Slot: slot, Source: "kitchen_close"}
		}
	}

	// 3. Platform default — a deadline always exists.
	def := defaultLunchCloseIST
	if slot == string(models.MealSlotDinner) {
		def = defaultDinnerCloseIST
	}
	at, _ := atIST(day, def)
	return AcceptDeadline{At: at, Slot: slot, Source: "platform_default"}
}

// Reminder schedule (#694).
const (
	// acceptReminderLeadIn — reminders start this long before the deadline.
	acceptReminderLeadIn = 2 * time.Hour
	// acceptReminderEvery — and repeat at this cadence until the deadline.
	acceptReminderEvery = 30 * time.Minute
)

// AcceptReminderTimes returns the moments a chef should be nudged about an
// unaccepted order: from two hours before the deadline, every 30 minutes, up to
// but NOT including the deadline itself.
//
// The deadline is excluded deliberately — at the deadline the order is voided,
// and a "your order is waiting" push landing in the same second as "your order
// was cancelled" is worse than silence.
//
// Times already in the past are skipped rather than fired late: an order placed
// 40 minutes before close should get the reminders that are still ahead of it,
// not an instant burst of the ones it missed.
func AcceptReminderTimes(deadline, now time.Time) []time.Time {
	var out []time.Time
	for t := deadline.Add(-acceptReminderLeadIn); t.Before(deadline); t = t.Add(acceptReminderEvery) {
		if t.After(now) {
			out = append(out, t)
		}
	}
	return out
}
