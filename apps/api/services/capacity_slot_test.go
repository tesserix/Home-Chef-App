package services

// Unit tests for the scheduled delivery-slot helpers (#51). Like the #48 caps,
// the atomic ReserveSlot/ReleaseSlot are DB-backed (exercised in handler/
// integration tests); here we cover the pure window/date/availability logic.

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

// slotSettings is a chef offering lunch (12:00–14:00, cutoff 10:00) and dinner
// (19:00–21:00, cutoff 17:00), both uncapped (so availability never hits the DB).
func slotSettings() *models.ChefCapacitySettings {
	return &models.ChefCapacitySettings{
		SlotsEnabled:    true,
		LunchSlotStart:  "12:00",
		LunchSlotEnd:    "14:00",
		LunchCutoff:     "10:00",
		DinnerSlotStart: "19:00",
		DinnerSlotEnd:   "21:00",
		DinnerCutoff:    "17:00",
	}
}

func TestSlotWindow(t *testing.T) {
	s := slotSettings()
	if start, end, ok := SlotWindow(s, SlotLunch); !ok || start != "12:00" || end != "14:00" {
		t.Fatalf("lunch window = (%q,%q,%v), want (12:00,14:00,true)", start, end, ok)
	}
	// A slot with no start time is not offered.
	s.DinnerSlotStart = ""
	if _, _, ok := SlotWindow(s, SlotDinner); ok {
		t.Fatal("dinner with blank start should not be offered")
	}
	if _, _, ok := SlotWindow(s, "brunch"); ok {
		t.Fatal("unknown slot should not be offered")
	}
}

func TestParseSlotDateIST(t *testing.T) {
	now := istAt(9, 0) // 2026-06-20 09:00 IST
	if d, ok := ParseSlotDateIST("", now); !ok || !d.Equal(CapacityDay(now)) {
		t.Fatalf("empty date should default to today IST, got (%v,%v)", d, ok)
	}
	if d, ok := ParseSlotDateIST("2026-06-21", now); !ok || d.Format("2006-01-02") != "2026-06-21" {
		t.Fatalf("valid date parse = (%v,%v)", d, ok)
	}
	if _, ok := ParseSlotDateIST("2026/06/21", now); ok {
		t.Fatal("malformed date should fail to parse")
	}
}

func TestResolveSlotSchedule(t *testing.T) {
	s := slotSettings()

	// Disabled slots → not offered.
	off := slotSettings()
	off.SlotsEnabled = false
	if _, _, err := ResolveSlotSchedule(off, SlotLunch, "", istAt(9, 0)); !errors.Is(err, ErrSlotNotOffered) {
		t.Fatalf("disabled slots → ErrSlotNotOffered, got %v", err)
	}
	// Unknown slot → not offered.
	if _, _, err := ResolveSlotSchedule(s, "brunch", "", istAt(9, 0)); !errors.Is(err, ErrSlotNotOffered) {
		t.Fatalf("unknown slot → ErrSlotNotOffered, got %v", err)
	}

	// Lunch today before the 10:00 cutoff → resolves to today's 12:00 window start.
	sf, day, err := ResolveSlotSchedule(s, SlotLunch, "", istAt(9, 0))
	if err != nil {
		t.Fatalf("lunch before cutoff should resolve, got %v", err)
	}
	if !sf.Equal(istAt(12, 0)) {
		t.Fatalf("scheduledFor = %v, want today 12:00 IST", sf)
	}
	if !day.Equal(CapacityDay(istAt(9, 0))) {
		t.Fatalf("booking day = %v, want today IST", day)
	}

	// Lunch today AFTER the 10:00 cutoff → closed.
	if _, _, err := ResolveSlotSchedule(s, SlotLunch, "", istAt(11, 0)); !errors.Is(err, ErrSlotClosed) {
		t.Fatalf("lunch past cutoff → ErrSlotClosed, got %v", err)
	}
	// But lunch TOMORROW is fine even though today's cutoff has passed.
	if _, _, err := ResolveSlotSchedule(s, SlotLunch, "2026-06-21", istAt(11, 0)); err != nil {
		t.Fatalf("tomorrow's lunch should resolve regardless of today's cutoff, got %v", err)
	}

	// Date in the past → invalid.
	if _, _, err := ResolveSlotSchedule(s, SlotLunch, "2026-06-19", istAt(9, 0)); !errors.Is(err, ErrSlotDateInvalid) {
		t.Fatalf("past date → ErrSlotDateInvalid, got %v", err)
	}
	// Date beyond the 6-day horizon (today 06-20 → max 06-26) → invalid.
	if _, _, err := ResolveSlotSchedule(s, SlotLunch, "2026-06-27", istAt(9, 0)); !errors.Is(err, ErrSlotDateInvalid) {
		t.Fatalf("date past horizon → ErrSlotDateInvalid, got %v", err)
	}
	// Malformed date → invalid.
	if _, _, err := ResolveSlotSchedule(s, SlotLunch, "soon", istAt(9, 0)); !errors.Is(err, ErrSlotDateInvalid) {
		t.Fatalf("malformed date → ErrSlotDateInvalid, got %v", err)
	}
}

func TestBuildSlotAvailability(t *testing.T) {
	// Disabled → empty (and never nil, so JSON renders []).
	if got := BuildSlotAvailability(&models.ChefCapacitySettings{SlotsEnabled: false}, uuid.New(), istAt(9, 0)); len(got) != 0 {
		t.Fatalf("disabled slots → empty, got %d", len(got))
	}

	chefID := uuid.New()
	// At 09:00 (before both cutoffs): 7 days × 2 uncapped slots = 14 entries,
	// today's lunch + dinner both available.
	got := BuildSlotAvailability(slotSettings(), chefID, istAt(9, 0))
	if len(got) != 14 {
		t.Fatalf("expected 14 slot entries (7 days × 2), got %d", len(got))
	}
	if got[0].Slot != SlotLunch || got[0].Date != "2026-06-20" || !got[0].Available {
		t.Fatalf("first entry should be today's available lunch, got %+v", got[0])
	}
	if got[0].Window != "12:00–14:00" || !got[0].ScheduledFor.Equal(istAt(12, 0)) {
		t.Fatalf("lunch window/scheduledFor wrong: %+v", got[0])
	}
	if got[0].Remaining != nil {
		t.Fatalf("uncapped slot should have nil remaining, got %v", got[0].Remaining)
	}

	// At 11:00 (past the 10:00 lunch cutoff): today's lunch is closed, today's
	// dinner (cutoff 17:00) still open.
	got2 := BuildSlotAvailability(slotSettings(), chefID, istAt(11, 0))
	if got2[0].Slot != SlotLunch || got2[0].Available {
		t.Fatalf("today's lunch should be unavailable past cutoff, got %+v", got2[0])
	}
	if got2[1].Slot != SlotDinner || !got2[1].Available {
		t.Fatalf("today's dinner should still be available, got %+v", got2[1])
	}
}
