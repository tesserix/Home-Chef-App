package services

// Unit tests for the chef capacity & cutoff helpers (#48). The atomic
// reserve/release are DB-backed (exercised in handler/integration tests); here
// we cover the pure cutoff + remaining logic.

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

func istAt(h, m int) time.Time {
	// A fixed IST instant on 2026-06-20 at h:m, expressed in UTC.
	return time.Date(2026, 6, 20, h, m, 0, 0, capacityIST)
}

func TestParseCutoff(t *testing.T) {
	cases := []struct {
		in   string
		h, m int
		ok   bool
	}{
		{"10:00", 10, 0, true},
		{"16:30", 16, 30, true},
		{"00:00", 0, 0, true},
		{"23:59", 23, 59, true},
		{"", 0, 0, false},
		{"9:00", 0, 0, false},  // not zero-padded / wrong length
		{"24:00", 0, 0, false}, // hour out of range
		{"10:60", 0, 0, false}, // minute out of range
		{"1a:00", 0, 0, false}, // non-digit
		{"10-00", 0, 0, false}, // wrong separator
	}
	for _, c := range cases {
		h, m, ok := ParseCutoff(c.in)
		if ok != c.ok || (ok && (h != c.h || m != c.m)) {
			t.Fatalf("ParseCutoff(%q) = (%d,%d,%v), want (%d,%d,%v)", c.in, h, m, ok, c.h, c.m, c.ok)
		}
	}
}

func TestIsPastCutoff(t *testing.T) {
	if !IsPastCutoff("10:00", istAt(10, 1)) {
		t.Fatal("10:01 should be past a 10:00 cutoff")
	}
	if !IsPastCutoff("10:00", istAt(10, 0)) {
		t.Fatal("exactly 10:00 should count as past (at/after)")
	}
	if IsPastCutoff("10:00", istAt(9, 59)) {
		t.Fatal("09:59 should NOT be past a 10:00 cutoff")
	}
	if IsPastCutoff("", istAt(23, 0)) {
		t.Fatal("a blank cutoff is never past")
	}
}

func TestIsPastDailyClose(t *testing.T) {
	on := func(l, d string) *models.ChefCapacitySettings {
		return &models.ChefCapacitySettings{CutoffEnabled: true, LunchCutoff: l, DinnerCutoff: d}
	}
	// Disabled → never closed.
	if IsPastDailyClose(&models.ChefCapacitySettings{CutoffEnabled: false, LunchCutoff: "10:00"}, istAt(23, 0)) {
		t.Fatal("disabled cutoffs never close the day")
	}
	// Both set; past both → closed.
	if !IsPastDailyClose(on("10:00", "16:00"), istAt(16, 30)) {
		t.Fatal("past both lunch+dinner cutoffs → closed")
	}
	// Both set; past lunch but not dinner → still open.
	if IsPastDailyClose(on("10:00", "16:00"), istAt(12, 0)) {
		t.Fatal("dinner still open at 12:00 → not closed")
	}
	// Only lunch set; past it → closed (dinner doesn't gate).
	if !IsPastDailyClose(on("10:00", ""), istAt(11, 0)) {
		t.Fatal("only-lunch cutoff past → closed")
	}
	// No cutoffs set → not closed.
	if IsPastDailyClose(on("", ""), istAt(23, 0)) {
		t.Fatal("no cutoffs set → not closed")
	}
}

func TestRemainingToday_Uncapped(t *testing.T) {
	if rem, sold := RemainingToday(uuid.New(), nil, CapacityDay(time.Now())); rem != nil || sold {
		t.Fatalf("uncapped item should return (nil,false), got (%v,%v)", rem, sold)
	}
	zero := 0
	if rem, _ := RemainingToday(uuid.New(), &zero, CapacityDay(time.Now())); rem != nil {
		t.Fatalf("cap 0 = unlimited → nil remaining, got %v", rem)
	}
}
