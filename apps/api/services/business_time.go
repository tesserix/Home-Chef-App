package services

// business_time.go — one definition of "what day is it" for the people using
// the app (#local-time).
//
// Timestamps are stored in UTC and that stays right. What was wrong is the
// PRESENTATION: a chef in India closing up at 01:00 IST had that order counted
// against the previous UTC day, so their dashboard, analytics and exports each
// disagreed with the day they had just worked.
//
// The codebase had three different answers to the same question —
// `DATE(created_at)` (UTC), `created_at + interval '330 minutes'` (a magic IST
// offset), and nothing at all — which is why the numbers never lined up. This
// is the single answer. Every day-bucketing query and every day boundary should
// come through here rather than growing a fourth.
//
// One zone, not per-user, and deliberately so: every chef is onboarded with
// PayoutCountry 'IN', prices are INR, and the tax engine is GST/TDS. The env
// var is the seam for the day that stops being true — when the platform takes
// a second country, this becomes a lookup on the chef's own zone and every
// caller already routes through it.

import (
	"os"
	"strings"
	"sync"
	"time"
)

const defaultBusinessTZ = "Asia/Kolkata"

var (
	businessTZOnce sync.Once
	businessTZName string
	businessTZLoc  *time.Location
)

func resolveBusinessTZ() {
	businessTZName = strings.TrimSpace(os.Getenv("BUSINESS_TIMEZONE"))
	if businessTZName == "" {
		businessTZName = defaultBusinessTZ
	}
	loc, err := time.LoadLocation(businessTZName)
	if err != nil {
		// No tzdata in the image, or a typo in the env. Fall back to a fixed
		// +05:30 rather than silently reverting to UTC — reverting is what put
		// every figure half a day out in the first place.
		businessTZName = defaultBusinessTZ
		loc = time.FixedZone("IST", 5*3600+30*60)
	}
	businessTZLoc = loc
}

// BusinessLocation is the zone day boundaries are drawn in. Use it wherever Go
// formats or buckets a date the user will read.
func BusinessLocation() *time.Location {
	businessTZOnce.Do(resolveBusinessTZ)
	return businessTZLoc
}

// BusinessTZName is the IANA name to hand Postgres for `AT TIME ZONE`. Always
// pair it with BusinessLocation on the Go side: bucketing in one zone and
// labelling in another shifts every row by a day and drops the edges.
func BusinessTZName() string {
	businessTZOnce.Do(resolveBusinessTZ)
	return businessTZName
}

// BusinessDay renders t as the calendar day the user experienced it, matching
// what `TO_CHAR(ts AT TIME ZONE BusinessTZName(), 'YYYY-MM-DD')` returns.
func BusinessDay(t time.Time) string {
	return t.In(BusinessLocation()).Format("2006-01-02")
}

// istLoc is the package-local handle on BusinessLocation, for the day/cutoff
// math that reaches for the zone directly. One alias, not one per file — the
// capacity, settlement and availability code each grew their own and that is
// how the same instant ended up on two different days.
var istLoc = BusinessLocation()

// BusinessDayStart is midnight of the business day containing t.
func BusinessDayStart(t time.Time) time.Time {
	loc := BusinessLocation()
	b := t.In(loc)
	return time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, loc)
}

// BusinessDayEnd is the last instant of the business day containing t, for
// queries written with an inclusive upper bound.
func BusinessDayEnd(t time.Time) time.Time {
	return BusinessDayStart(t).AddDate(0, 0, 1).Add(-time.Nanosecond)
}

// BusinessWeekStart is Monday 00:00 of the business week containing t.
//
// Mon–Sun is the week the platform settles on, so this is the boundary every
// chef-facing "this week" figure must share: the dashboard snapshot, the
// earnings breakdown and the weekly statement otherwise roll over on three
// different instants and disagree about the same money (#937).
func BusinessWeekStart(t time.Time) time.Time {
	day := BusinessDayStart(t)
	daysSinceMonday := (int(day.Weekday()) + 6) % 7 // Weekday: Sun=0 … Sat=6
	return day.AddDate(0, 0, -daysSinceMonday)
}

// BusinessMonthStart is the first of the business month containing t.
func BusinessMonthStart(t time.Time) time.Time {
	loc := BusinessLocation()
	b := t.In(loc)
	return time.Date(b.Year(), b.Month(), 1, 0, 0, 0, 0, loc)
}
