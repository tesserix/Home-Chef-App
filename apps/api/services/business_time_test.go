package services

import (
	"testing"
	"time"
)

// ist builds an instant in the business zone, so a case reads as the wall clock
// the chef actually saw.
func businessAt(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, BusinessLocation())
}

// TestBusinessDayStart_LateEveningStaysOnItsOwnDay covers the original defect:
// an order taken at 01:00 IST is 19:30 UTC the PREVIOUS day, so UTC-based
// bucketing credited it to the day before the chef worked it.
func TestBusinessDayStart_LateEveningStaysOnItsOwnDay(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
		want time.Time
	}{
		{"just after IST midnight", businessAt(2026, time.August, 3, 0, 15), businessAt(2026, time.August, 3, 0, 0)},
		{"IST morning", businessAt(2026, time.August, 3, 9, 0), businessAt(2026, time.August, 3, 0, 0)},
		{"last minute of the IST day", businessAt(2026, time.August, 3, 23, 59), businessAt(2026, time.August, 3, 0, 0)},
		// 18:00 UTC on the 2nd is 23:30 IST on the 2nd — still the 2nd.
		{"UTC evening before the IST rollover", time.Date(2026, time.August, 2, 18, 0, 0, 0, time.UTC), businessAt(2026, time.August, 2, 0, 0)},
		// 19:00 UTC on the 2nd is 00:30 IST on the 3rd — already the 3rd.
		{"UTC evening after the IST rollover", time.Date(2026, time.August, 2, 19, 0, 0, 0, time.UTC), businessAt(2026, time.August, 3, 0, 0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BusinessDayStart(tc.at); !got.Equal(tc.want) {
				t.Errorf("BusinessDayStart(%v) = %v, want %v", tc.at, got, tc.want)
			}
		})
	}
}

// TestBusinessDayEnd_IsInclusiveUpperBound checks the end bound closes the same
// day it opens and admits the last instant of it — queries use `<= end`.
func TestBusinessDayEnd_IsInclusiveUpperBound(t *testing.T) {
	at := businessAt(2026, time.August, 3, 14, 30)
	start, end := BusinessDayStart(at), BusinessDayEnd(at)

	if got, want := end.Sub(start), 24*time.Hour-time.Nanosecond; got != want {
		t.Errorf("day span = %v, want %v", got, want)
	}
	lastInstant := businessAt(2026, time.August, 3, 23, 59).Add(59*time.Second + 999999999*time.Nanosecond)
	if end.Before(lastInstant) {
		t.Errorf("BusinessDayEnd(%v) = %v excludes %v", at, end, lastInstant)
	}
	if !end.Before(BusinessDayStart(at).AddDate(0, 0, 1)) {
		t.Errorf("BusinessDayEnd(%v) = %v leaks into the next day", at, end)
	}
}

// TestBusinessWeekStart_LandsOnMondayMidnight walks a full week, including the
// Sunday→Monday rollover where the rolling-7-day window used to disagree.
func TestBusinessWeekStart_LandsOnMondayMidnight(t *testing.T) {
	monday := businessAt(2026, time.August, 3, 0, 0) // 2026-08-03 is a Monday
	if monday.Weekday() != time.Monday {
		t.Fatalf("fixture is not a Monday: %v", monday.Weekday())
	}

	for offset := range 7 {
		day := monday.AddDate(0, 0, offset)
		for _, hour := range []int{0, 12, 23} {
			at := day.Add(time.Duration(hour) * time.Hour)
			if got := BusinessWeekStart(at); !got.Equal(monday) {
				t.Errorf("BusinessWeekStart(%v, %v) = %v, want %v", at.Weekday(), at, got, monday)
			}
		}
	}

	// The Sunday 23:59 → Monday 00:00 step must move the window forward exactly
	// one week, never a fraction of a day.
	sundayLate := monday.AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute)
	nextMonday := monday.AddDate(0, 0, 7)
	if got := BusinessWeekStart(sundayLate); !got.Equal(monday) {
		t.Errorf("late Sunday = %v, want the week still open at %v", got, monday)
	}
	if got := BusinessWeekStart(nextMonday); !got.Equal(nextMonday) {
		t.Errorf("Monday midnight = %v, want the new week %v", got, nextMonday)
	}
}

// TestBusinessWeekStart_MatchesSettlement is the alignment guarantee itself: the
// week the chef's dashboard and earnings screen open is exactly the week the
// settlement statement most recently closed.
func TestBusinessWeekStart_MatchesSettlement(t *testing.T) {
	for _, at := range []time.Time{
		businessAt(2026, time.August, 3, 0, 0),                 // Monday, first instant
		businessAt(2026, time.August, 5, 13, 45),               // midweek
		businessAt(2026, time.August, 9, 23, 30),               // Sunday, last hour
		time.Date(2026, time.August, 2, 19, 0, 0, 0, time.UTC), // just past the IST rollover
	} {
		start := BusinessWeekStart(at)
		_, closedEnd := MostRecentClosedWeek(at)
		if !start.Equal(closedEnd) {
			t.Errorf("at %v: week opens %v but settlement closed at %v", at, start, closedEnd)
		}
	}
}

// TestMostRecentClosedWeek_IsAWholeMonSunWeek guards the settlement window after
// it was rewritten onto the shared helper.
func TestMostRecentClosedWeek_IsAWholeMonSunWeek(t *testing.T) {
	start, end := MostRecentClosedWeek(businessAt(2026, time.August, 5, 13, 45))

	if got, want := end.Sub(start), 7*24*time.Hour; got != want {
		t.Errorf("closed week span = %v, want %v", got, want)
	}
	loc := BusinessLocation()
	if got := start.In(loc).Weekday(); got != time.Monday {
		t.Errorf("closed week starts on %v, want Monday", got)
	}
	if got := end.In(loc).Weekday(); got != time.Monday {
		t.Errorf("closed week ends on %v, want the following Monday (exclusive)", got)
	}
	// The closed week must be entirely in the past — never the week in progress.
	if !end.Before(businessAt(2026, time.August, 5, 13, 45)) {
		t.Errorf("closed week end %v is not before now", end)
	}
}

// TestBusinessMonthStart_LandsOnTheFirst covers the month bound, including the
// UTC instant that belongs to the next IST month.
func TestBusinessMonthStart_LandsOnTheFirst(t *testing.T) {
	cases := []struct {
		name string
		at   time.Time
		want time.Time
	}{
		{"midmonth", businessAt(2026, time.August, 17, 9, 0), businessAt(2026, time.August, 1, 0, 0)},
		{"first of the month", businessAt(2026, time.August, 1, 0, 0), businessAt(2026, time.August, 1, 0, 0)},
		{"last night of the month", businessAt(2026, time.July, 31, 23, 45), businessAt(2026, time.July, 1, 0, 0)},
		// 19:00 UTC on 31 Jul is 00:30 IST on 1 Aug — an August order.
		{"UTC last evening rolls into the next IST month", time.Date(2026, time.July, 31, 19, 0, 0, 0, time.UTC), businessAt(2026, time.August, 1, 0, 0)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BusinessMonthStart(tc.at); !got.Equal(tc.want) {
				t.Errorf("BusinessMonthStart(%v) = %v, want %v", tc.at, got, tc.want)
			}
		})
	}
}

// TestCapacityDay_DelegatesToBusinessDayStart keeps the capacity calendar and
// the business day as one definition — they were separate zones, which is how
// the same order landed on two different days.
func TestCapacityDay_DelegatesToBusinessDayStart(t *testing.T) {
	for _, at := range []time.Time{
		businessAt(2026, time.August, 3, 0, 1),
		businessAt(2026, time.August, 3, 23, 59),
		time.Date(2026, time.August, 2, 19, 0, 0, 0, time.UTC),
		time.Now(),
	} {
		if got, want := CapacityDay(at), BusinessDayStart(at); !got.Equal(want) {
			t.Errorf("CapacityDay(%v) = %v, want %v", at, got, want)
		}
	}
}
