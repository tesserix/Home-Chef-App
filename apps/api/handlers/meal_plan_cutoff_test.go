package handlers

// meal_plan_cutoff_test.go — unit tests for the tiffin booking-date helpers:
// dates are interpreted as IST midnight (not UTC) so the 12h lead-time cutoff is
// correct regardless of the server clock's zone (containers run UTC).

import (
	"testing"
	"time"
)

func TestParsePlanDate_IsISTMidnight(t *testing.T) {
	d, err := parsePlanDate("2026-06-19")
	if err != nil {
		t.Fatalf("parsePlanDate error: %v", err)
	}
	// IST is UTC+5:30 with no DST.
	_, offset := d.Zone()
	if offset != 5*3600+30*60 {
		t.Fatalf("zone offset = %d, want 19800 (IST)", offset)
	}
	// IST midnight 2026-06-19 == 2026-06-18T18:30:00Z.
	if got := d.UTC(); !got.Equal(time.Date(2026, 6, 18, 18, 30, 0, 0, time.UTC)) {
		t.Fatalf("IST midnight in UTC = %v, want 2026-06-18T18:30:00Z", got)
	}
}

func TestParsePlanDate_Invalid(t *testing.T) {
	if _, err := parsePlanDate("19-06-2026"); err == nil {
		t.Fatal("expected error for non-YYYY-MM-DD date")
	}
}

func TestDayBeforeCutoff(t *testing.T) {
	// "now" = 2026-06-18 18:00 IST (12:30 UTC). Cutoff = now + 12h = 2026-06-19 06:00 IST.
	now := time.Date(2026, 6, 18, 12, 30, 0, 0, time.UTC) // 18:00 IST

	// Tomorrow (2026-06-19 00:00 IST) is BEFORE the 06:00 IST cutoff → too soon.
	before, err := dayBeforeCutoff("2026-06-19", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !before {
		t.Fatal("2026-06-19 booked at 18:00 IST should be past the 12h cutoff")
	}

	// Day after (2026-06-20 00:00 IST) is comfortably after the cutoff → bookable.
	before, err = dayBeforeCutoff("2026-06-20", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if before {
		t.Fatal("2026-06-20 should be bookable (well past the cutoff)")
	}
}

func TestDayBeforeCutoff_MorningBookingAllowsNextDay(t *testing.T) {
	// Booking early morning 2026-06-18 06:00 IST (00:30 UTC); cutoff = 18:00 IST.
	now := time.Date(2026, 6, 18, 0, 30, 0, 0, time.UTC) // 06:00 IST
	// Tomorrow 2026-06-19 00:00 IST is well after the 18:00-today cutoff → bookable.
	before, err := dayBeforeCutoff("2026-06-19", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if before {
		t.Fatal("2026-06-19 booked at 06:00 IST should be bookable")
	}
}
