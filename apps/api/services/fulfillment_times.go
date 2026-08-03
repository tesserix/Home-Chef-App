package services

// fulfillment_times.go — realistic "preferred delivery/pickup time" suggestions
// for the home-tiffin handshake (#709).
//
// THE PROBLEM THIS FIXES
//
// The checkout used to propose slots as "now + 1h, every 30 min for 5 hours".
// That is the wrong MODEL for a home kitchen: order at 6am and it suggested
// 9am — a time no home cook serves. Suggestions must come from the CHEF's own
// meal service, not a stopwatch started at checkout, and they must read as real
// meals (lunch / dinner), not arbitrary half-hours.
//
// WHERE THE TIMES COME FROM (the chef's own configuration, then sensible defaults)
//
//  1. The kitchen's open hours for that weekday (ChefSchedule) ARE the offer —
//     a chef open 10:00–22:00 is bookable across all of it. A day the kitchen is
//     closed produces no times.
//  2. A chef who configured explicit service windows
//     (ChefCapacitySettings.LunchSlot*/DinnerSlot*) gets exactly those instead —
//     that narrowing is their own deliberate choice.
//  3. Only when neither is configured do the platform default meal windows apply.
//  4. Never earlier than now + prep headroom (from the chef's PrepTime), and it
//     rolls forward across days until enough real slots are found.
//
// Meal names LABEL a slot, they never gate it — platform windows carving holes
// out of an open kitchen's day is what made checkout times look arbitrary.
//
// The customer still defaults to "as soon as ready" (the chef decides); these are
// the concrete alternatives they can propose instead.

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/homechef/api/models"
)

// SuggestedFulfillmentTime is one proposable delivery/pickup time.
type SuggestedFulfillmentTime struct {
	At    time.Time `json:"at"`    // exact instant (RFC3339, +05:30)
	Label string    `json:"label"` // clock label, e.g. "1:00 pm"
	Day   string    `json:"day"`   // "Today" / "Tomorrow" / "Sat"
	Meal  string    `json:"meal"`  // "Breakfast" / "Lunch" / "Snacks" / "Dinner"
}

const (
	defaultPrepMinutes         = 45
	fulfillmentHorizonDays     = 7
	defaultFulfillmentSlotSpan = 30 * time.Minute

	// FulfillmentTimesPageSize must cover a full open day at half-hour steps —
	// a 12-hour kitchen is 24 slots — or late slots fall off and the customer
	// cannot book dinner from a kitchen that is plainly open for it.
	FulfillmentTimesPageSize = 32
)

type mealWindow struct {
	meal       string
	start, end string // "HH:MM" IST
}

// defaultMealWindows are the platform's canonical home-kitchen meal times, used
// when the chef hasn't configured their own. Bounded further by the chef's open
// hours, so an unconfigured early-only or evening-only kitchen still shows sanely.
func defaultMealWindows() []mealWindow {
	return []mealWindow{
		{"Breakfast", "08:00", "10:00"},
		{"Lunch", "12:00", "14:30"},
		{"Snacks", "16:30", "18:00"},
		{"Dinner", "19:00", "21:30"},
	}
}

// chefConfiguredWindows returns only the service windows the chef set themselves,
// or nil when they set none — in which case their open hours are the offer.
func chefConfiguredWindows(cap *models.ChefCapacitySettings) []mealWindow {
	if cap == nil {
		return nil
	}
	var ws []mealWindow
	if cap.LunchSlotStart != "" && cap.LunchSlotEnd != "" {
		ws = append(ws, mealWindow{"Lunch", cap.LunchSlotStart, cap.LunchSlotEnd})
	}
	if cap.DinnerSlotStart != "" && cap.DinnerSlotEnd != "" {
		ws = append(ws, mealWindow{"Dinner", cap.DinnerSlotStart, cap.DinnerSlotEnd})
	}
	return ws
}

// mealLabelFor names a slot by time of day. The bands are contiguous so every
// time inside a kitchen's open hours gets a label and the UI groups cleanly.
func mealLabelFor(t time.Time) string {
	switch h := t.In(istLoc).Hour(); {
	case h < 11:
		return "Breakfast"
	case h < 16:
		return "Lunch"
	case h < 19:
		return "Snacks"
	default:
		return "Dinner"
	}
}

var prepNumberRe = regexp.MustCompile(`\d+`)

// ParsePrepMinutes turns a chef PrepTime string ("30-45 min", "45 min",
// "1 hour") into minutes, taking the UPPER number so headroom is generous.
// Defaults to defaultPrepMinutes when unparseable.
func ParsePrepMinutes(prep string) int {
	best := 0
	for _, n := range prepNumberRe.FindAllString(prep, -1) {
		var v int
		if _, err := fmt.Sscanf(n, "%d", &v); err == nil && v > best {
			best = v
		}
	}
	if best <= 0 {
		return defaultPrepMinutes
	}
	if strings.Contains(strings.ToLower(prep), "hour") && best <= 12 {
		best *= 60
	}
	return best
}

// BuildSuggestedFulfillmentTimes returns up to `limit` realistic proposable
// times, reading the chef's meal windows + open hours and never earlier than
// now + prep headroom. Pure (takes `now`) so it is deterministically testable.
func BuildSuggestedFulfillmentTimes(
	cap *models.ChefCapacitySettings,
	schedules []models.ChefSchedule,
	prepMinutes int,
	now time.Time,
	limit int,
) []SuggestedFulfillmentTime {
	if limit <= 0 {
		limit = 12
	}
	if prepMinutes <= 0 {
		prepMinutes = defaultPrepMinutes
	}
	configured := chefConfiguredWindows(cap)

	byWeekday := make(map[int]models.ChefSchedule, len(schedules))
	for _, s := range schedules {
		byWeekday[s.DayOfWeek] = s
	}

	earliest := now.In(istLoc).Add(time.Duration(prepMinutes) * time.Minute)
	seen := make(map[int64]bool)
	out := make([]SuggestedFulfillmentTime, 0, limit)

	for dayOffset := 0; dayOffset < fulfillmentHorizonDays && len(out) < limit; dayOffset++ {
		day := CapacityDay(now).AddDate(0, 0, dayOffset)
		weekday := int(day.In(istLoc).Weekday())

		// Kitchen open-hours bound for the weekday, when the chef configured one.
		var openT, closeT time.Time
		haveSchedule := false
		if s, ok := byWeekday[weekday]; ok {
			if s.IsClosed {
				continue // kitchen shut this weekday — no times.
			}
			o, ok1 := atIST(day, s.OpenTime)
			cl, ok2 := atIST(day, s.CloseTime)
			if ok1 && ok2 && cl.After(o) {
				openT, closeT, haveSchedule = o, cl, true
			}
		}

		// The kitchen's open hours are the offer unless the chef narrowed them
		// themselves; platform defaults only cover a chef with no schedule at all.
		type slotWindow struct {
			meal       string // "" — label each slot from its own time of day
			start, end time.Time
		}
		var dayWindows []slotWindow
		switch {
		case len(configured) > 0:
			for _, w := range configured {
				ws, ok1 := atIST(day, w.start)
				we, ok2 := atIST(day, w.end)
				if ok1 && ok2 && we.After(ws) {
					dayWindows = append(dayWindows, slotWindow{w.meal, ws, we})
				}
			}
		case haveSchedule:
			dayWindows = []slotWindow{{"", openT, closeT}}
		default:
			for _, w := range defaultMealWindows() {
				ws, ok1 := atIST(day, w.start)
				we, ok2 := atIST(day, w.end)
				if ok1 && ok2 && we.After(ws) {
					dayWindows = append(dayWindows, slotWindow{w.meal, ws, we})
				}
			}
		}

		for _, w := range dayWindows {
			ws, we := w.start, w.end
			if haveSchedule { // never offer a time the kitchen is shut.
				if ws.Before(openT) {
					ws = openT
				}
				if we.After(closeT) {
					we = closeT
				}
				if !we.After(ws) {
					continue
				}
			}
			for t := ceilToHalfHour(ws); t.Before(we) && len(out) < limit; t = t.Add(defaultFulfillmentSlotSpan) {
				if t.Before(earliest) {
					continue
				}
				if seen[t.Unix()] {
					continue
				}
				seen[t.Unix()] = true
				meal := w.meal
				if meal == "" {
					meal = mealLabelFor(t)
				}
				out = append(out, SuggestedFulfillmentTime{
					At:    t,
					Label: t.In(istLoc).Format("3:04 pm"),
					Day:   fulfillmentDayLabel(day, now),
					Meal:  meal,
				})
			}
		}
	}
	return out
}

// ceilToHalfHour rounds a time UP to the next :00 or :30 (in IST).
func ceilToHalfHour(t time.Time) time.Time {
	t = t.In(istLoc).Truncate(time.Minute)
	switch m := t.Minute(); {
	case m == 0 || m == 30:
		return t
	case m < 30:
		return t.Add(time.Duration(30-m) * time.Minute)
	default:
		return t.Add(time.Duration(60-m) * time.Minute)
	}
}

// fulfillmentDayLabel is "Today" / "Tomorrow" / a short weekday name.
func fulfillmentDayLabel(day, now time.Time) string {
	d := CapacityDay(day)
	today := CapacityDay(now)
	switch {
	case d.Equal(today):
		return "Today"
	case d.Equal(today.AddDate(0, 0, 1)):
		return "Tomorrow"
	default:
		return day.In(istLoc).Format("Mon")
	}
}
