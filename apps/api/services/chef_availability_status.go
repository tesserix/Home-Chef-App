package services

// chef_availability_status.go — the customer-facing REAL-TIME open/closed status of a kitchen.
//
// The card in the customer app used to show "Open" straight off ChefProfile.AcceptingOrders, but a
// customer could still be rejected at checkout because ordering is ALSO gated by the chef's daily
// cutoff (IsPastDailyClose), the platform operating hours (IsPlatformOpen), and — for auto-schedule
// kitchens — today's ChefSchedule window (which the 5-minute cron only reconciles into
// AcceptingOrders with a lag). This computes ONE authoritative status that mirrors exactly the gates
// the order path enforces, plus the next open/close time so the UI can show "Opening soon · N min" /
// "Closing soon · N min" pills (Uber-Eats style) within a 30-minute window. Reuses IsPastDailyClose,
// ParseCutoff, IsPlatformOpen, and the schedule-minute helpers — no duplicated time math.

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// Availability statuses (the `status` field). The app switches its pill/dot on these.
const (
	AvailOpen        = "open"         // orderable, next close (if any) > soon-window away
	AvailClosingSoon = "closing_soon" // orderable, closes within availSoonWindowMin
	AvailOpeningSoon = "opening_soon" // not orderable, opens within availSoonWindowMin
	AvailPaused      = "paused"       // chef took a timed pause, resuming later than the soon-window
	AvailClosed      = "closed"       // not orderable, no imminent open today
)

// availSoonWindowMin is how close (minutes) to an open/close transition counts as "soon" — the
// window that lights up the "opening soon"/"closing soon" pill. 30 min, matching the ask.
const availSoonWindowMin = 30

// ChefAvailability is an alias for the response DTO (defined in models to avoid an import cycle with
// ChefProfileResponse). The decision logic here builds and returns it.
type ChefAvailability = models.ChefAvailability

// availInputs are the resolved inputs to the pure decision, all clock values already reduced to IST
// minutes-since-midnight (or -1 when absent/invalid). Keeping the core pure over primitives makes
// every branch unit-testable with no DB or wall clock.
type availInputs struct {
	nowMin          int
	acceptingOrders bool
	pausedUntilMin  int // -1 when not paused (or the pause already elapsed)
	scheduleGates   bool // auto-schedule ON and today has an open (non-closed) schedule row
	schedOpenMin    int  // -1 when absent/invalid
	schedCloseMin   int  // -1 when absent/invalid
	dailyCloseMin   int  // effective à-la-carte close = later of the set cutoffs; -1 when cutoffs off/none
	pastDailyClose  bool
	platformOpen    bool
	platformOpenMin int // today's platform open; -1 when unconfigured / all-day
	platformCloseMin int // today's platform close; -1 when unconfigured / all-day
}

// decideAvailability is the pure core: given resolved inputs it returns the status + label. No DB,
// no time.Now — every value comes in through availInputs so this is fully table-testable.
func decideAvailability(in availInputs) ChefAvailability {
	paused := in.pausedUntilMin >= 0
	// For auto-schedule kitchens the live schedule window is authoritative (closes the cron lag);
	// for everyone else the schedule doesn't gate ordering, so withinSchedule is vacuously true.
	withinSchedule := !in.scheduleGates ||
		(in.schedOpenMin >= 0 && in.schedCloseMin > in.schedOpenMin &&
			in.nowMin >= in.schedOpenMin && in.nowMin < in.schedCloseMin)

	orderable := in.platformOpen && in.acceptingOrders && !paused && withinSchedule && !in.pastDailyClose

	if orderable {
		// Earliest upcoming close among the schedule close, the daily cutoff, and the platform close.
		next := -1
		cands := []int{}
		if in.scheduleGates {
			cands = append(cands, in.schedCloseMin)
		}
		cands = append(cands, in.dailyCloseMin, in.platformCloseMin)
		for _, c := range cands {
			if c > in.nowMin && (next < 0 || c < next) {
				next = c
			}
		}
		if next >= 0 && next-in.nowMin <= availSoonWindowMin {
			mins := next - in.nowMin
			return ChefAvailability{Orderable: true, Status: AvailClosingSoon, MinutesToChange: mins, Label: soonLabel("Closing soon", mins)}
		}
		return ChefAvailability{Orderable: true, Status: AvailOpen, Label: "Open"}
	}

	// Not orderable — a timed pause has its own known resume time.
	if paused {
		mins := in.pausedUntilMin - in.nowMin
		if mins >= 0 && mins <= availSoonWindowMin {
			return ChefAvailability{Status: AvailOpeningSoon, MinutesToChange: mins, Label: soonLabel("Opening soon", mins)}
		}
		return ChefAvailability{Status: AvailPaused, Label: "Back at " + hhmmLabel(in.pausedUntilMin)}
	}

	// Otherwise the next open today is the schedule open (auto-schedule kitchens before their open)
	// or the platform open — whichever is still ahead. A manual-off kitchen or a past-cutoff day has
	// no known re-open today, so it's simply "Closed".
	next := -1
	openCands := []int{}
	if in.scheduleGates {
		openCands = append(openCands, in.schedOpenMin)
	}
	openCands = append(openCands, in.platformOpenMin)
	for _, o := range openCands {
		if o > in.nowMin && (next < 0 || o < next) {
			next = o
		}
	}
	if next >= 0 {
		mins := next - in.nowMin
		if mins <= availSoonWindowMin {
			return ChefAvailability{Status: AvailOpeningSoon, MinutesToChange: mins, Label: soonLabel("Opening soon", mins)}
		}
		return ChefAvailability{Status: AvailClosed, Label: "Opens at " + hhmmLabel(next)}
	}
	return ChefAvailability{Status: AvailClosed, Label: "Closed"}
}

// soonLabel formats "<verb> · N min" (or hours when ≥60, though the soon window is 30).
func soonLabel(verb string, mins int) string {
	if mins <= 0 {
		return verb
	}
	return fmt.Sprintf("%s · %d min", verb, mins)
}

// hhmmLabel renders IST minutes-since-midnight as a 12-hour clock label, e.g. 1080 → "6:00 pm".
func hhmmLabel(min int) string {
	if min < 0 {
		return ""
	}
	h, m := (min/60)%24, min%60
	ampm := "am"
	h12 := h
	if h >= 12 {
		ampm = "pm"
	}
	if h12 == 0 {
		h12 = 12
	} else if h12 > 12 {
		h12 -= 12
	}
	return fmt.Sprintf("%d:%02d %s", h12, m, ampm)
}

// istMinutes reduces a wall-clock time to IST minutes-since-midnight.
func istMinutes(now time.Time) int {
	t := now.In(capacityIST)
	return t.Hour()*60 + t.Minute()
}

// pausedUntilMinutes maps a chef's PausedUntil to IST minutes-since-midnight, or -1 when it's unset
// or already elapsed. A pause that spills past midnight (rare — max 60 min) is clamped to end-of-day
// so it still reads as "later today".
func pausedUntilMinutes(pausedUntil *time.Time, now time.Time) int {
	if pausedUntil == nil || !pausedUntil.After(now) {
		return -1
	}
	pu := pausedUntil.In(capacityIST)
	nowIST := now.In(capacityIST)
	if pu.YearDay() != nowIST.YearDay() || pu.Year() != nowIST.Year() {
		return 24*60 - 1
	}
	return pu.Hour()*60 + pu.Minute()
}

// laterCutoffMinutes returns the effective à-la-carte daily close in IST minutes: the LATER of the
// set lunch/dinner cutoffs (IsPastDailyClose only closes the day once BOTH set cutoffs have passed),
// or -1 when cutoffs are disabled / none are set.
func laterCutoffMinutes(cap *models.ChefCapacitySettings) int {
	if cap == nil || !cap.CutoffEnabled {
		return -1
	}
	lh, lm, lok := ParseCutoff(cap.LunchCutoff)
	dh, dm, dok := ParseCutoff(cap.DinnerCutoff)
	later := -1
	if lok {
		later = lh*60 + lm
	}
	if dok && dh*60+dm > later {
		later = dh*60 + dm
	}
	return later
}

// ComputeChefAvailability resolves the DB/clock inputs for one chef and returns its real-time
// availability. todaySched is the chef's ChefSchedule row for today (nil if none), cap is the chef's
// capacity/cutoff settings (nil = none), and now is the reference time. Platform hours are read from
// the cached policy. The list + detail handlers batch-load todaySched/cap and call this per chef.
func ComputeChefAvailability(chef *models.ChefProfile, todaySched *models.ChefSchedule, cap *models.ChefCapacitySettings, now time.Time) ChefAvailability {
	in := availInputs{
		nowMin:          istMinutes(now),
		acceptingOrders: chef.AcceptingOrders,
		pausedUntilMin:  pausedUntilMinutes(chef.PausedUntil, now),
		schedOpenMin:    -1,
		schedCloseMin:   -1,
		dailyCloseMin:   laterCutoffMinutes(cap),
		pastDailyClose:  IsPastDailyClose(cap, now),
	}
	if chef.AutoScheduleEnabled && todaySched != nil && !todaySched.IsClosed {
		in.scheduleGates = true
		in.schedOpenMin = scheduleMinutes(todaySched.OpenTime)
		in.schedCloseMin = scheduleMinutes(todaySched.CloseTime)
	}
	in.platformOpen, in.platformOpenMin, in.platformCloseMin = platformHoursToday(now)
	return decideAvailability(in)
}

// ChefAvailabilityBatch computes the real-time availability for a page of chefs in two queries
// (today's schedule rows + capacity settings), keyed by chef id — no N+1. Platform hours are read
// once from the cached policy inside ComputeChefAvailability. Used by the chef list handler; the
// detail handler calls ComputeChefAvailability directly for its single chef.
func ChefAvailabilityBatch(chefs []models.ChefProfile, now time.Time) map[uuid.UUID]models.ChefAvailability {
	out := make(map[uuid.UUID]models.ChefAvailability, len(chefs))
	if len(chefs) == 0 {
		return out
	}
	ids := make([]uuid.UUID, len(chefs))
	for i := range chefs {
		ids[i] = chefs[i].ID
	}
	weekday := int(now.In(capacityIST).Weekday()) // 0=Sun..6=Sat, matches ChefSchedule.DayOfWeek

	schedByChef := make(map[uuid.UUID]models.ChefSchedule)
	var scheds []models.ChefSchedule
	database.DB.Where("chef_id IN ? AND day_of_week = ?", ids, weekday).Find(&scheds)
	for i := range scheds {
		schedByChef[scheds[i].ChefID] = scheds[i]
	}
	capByChef := make(map[uuid.UUID]models.ChefCapacitySettings)
	var caps []models.ChefCapacitySettings
	database.DB.Where("chef_id IN ?", ids).Find(&caps)
	for i := range caps {
		capByChef[caps[i].ChefID] = caps[i]
	}

	for i := range chefs {
		ch := &chefs[i]
		var sched *models.ChefSchedule
		if s, ok := schedByChef[ch.ID]; ok {
			sched = &s
		}
		var capSet *models.ChefCapacitySettings
		if cs, ok := capByChef[ch.ID]; ok {
			capSet = &cs
		}
		out[ch.ID] = ComputeChefAvailability(ch, sched, capSet, now)
	}
	return out
}

// ChefAvailabilityOne computes availability for a single chef given all their schedule rows (the
// detail handler already loads the full week). Picks today's IST row and loads the chef's capacity
// settings, then defers to ComputeChefAvailability.
func ChefAvailabilityOne(chef *models.ChefProfile, schedules []models.ChefSchedule, now time.Time) models.ChefAvailability {
	weekday := int(now.In(capacityIST).Weekday())
	var today *models.ChefSchedule
	for i := range schedules {
		if schedules[i].DayOfWeek == weekday {
			today = &schedules[i]
			break
		}
	}
	return ComputeChefAvailability(chef, today, GetChefCapacitySettings(chef.ID), now)
}

// platformHoursToday returns (openNow, todayOpenMin, todayCloseMin) for the platform policy. Open/
// close minutes are -1 when unconfigured or today isn't an operating day (in which case the policy's
// open/close boundaries don't help the pill). openNow reuses IsPlatformOpen so the gate stays in one
// place.
func platformHoursToday(now time.Time) (bool, int, int) {
	open, _ := IsPlatformOpen()
	p := GetPlatformPolicy()
	if p.OpeningTime == "" || p.ClosingTime == "" {
		return open, -1, -1
	}
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return open, -1, -1
	}
	local := now.In(loc)
	if len(p.OperatingDays) > 0 {
		today := int(local.Weekday())
		found := false
		for _, d := range p.OperatingDays {
			if d == today {
				found = true
				break
			}
		}
		if !found {
			return open, -1, -1
		}
	}
	oh, om, ook := ParseCutoff(p.OpeningTime)
	ch, cm, cok := ParseCutoff(p.ClosingTime)
	openMin, closeMin := -1, -1
	if ook {
		openMin = oh*60 + om
	}
	if cok {
		closeMin = ch*60 + cm
	}
	return open, openMin, closeMin
}
