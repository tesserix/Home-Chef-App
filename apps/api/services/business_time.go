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
