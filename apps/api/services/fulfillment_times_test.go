package services

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

// istAt builds an IST instant for the test's "now".
func istMoment(y int, mo time.Month, d, h, m int) time.Time {
	return time.Date(y, mo, d, h, m, 0, 0, capacityIST)
}

func TestParsePrepMinutes(t *testing.T) {
	cases := map[string]int{
		"30-45 min": 45,
		"45 min":    45,
		"1 hour":    60,
		"":          defaultPrepMinutes,
		"soon":      defaultPrepMinutes,
	}
	for in, want := range cases {
		if got := ParsePrepMinutes(in); got != want {
			t.Errorf("ParsePrepMinutes(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestSuggestedTimes_NoEarlyMorningSlots(t *testing.T) {
	// The reported bug: ordering at 6am proposed 9am. With default meal windows
	// and no schedule, a 6am order must NOT produce any pre-breakfast slot, and the
	// first suggestion must be a real meal time (>= 08:00).
	now := istMoment(2026, time.July, 20, 6, 0) // Monday 6:00am IST
	times := BuildSuggestedFulfillmentTimes(nil, nil, 45, now, 12)
	if len(times) == 0 {
		t.Fatal("expected suggestions")
	}
	for _, s := range times {
		h := s.At.In(capacityIST).Hour()
		if s.Day == "Today" && h < 8 {
			t.Errorf("today slot before 08:00: %s (%s)", s.Label, s.Meal)
		}
	}
	first := times[0]
	if first.Meal != "Breakfast" || first.At.In(capacityIST).Hour() < 8 {
		t.Errorf("first suggestion should be breakfast >=08:00, got %s %s", first.Meal, first.Label)
	}
}

func TestSuggestedTimes_RespectsPrepHeadroom(t *testing.T) {
	// At 12:10 with 45m prep, earliest = 12:55 → first lunch slot must be >= 13:00.
	now := istMoment(2026, time.July, 20, 12, 10)
	times := BuildSuggestedFulfillmentTimes(nil, nil, 45, now, 12)
	if len(times) == 0 {
		t.Fatal("expected suggestions")
	}
	for _, s := range times {
		if s.Day == "Today" && s.At.Before(now.Add(45*time.Minute)) {
			t.Errorf("slot %s violates prep headroom", s.Label)
		}
	}
}

func TestSuggestedTimes_ClipsToChefOpenHours(t *testing.T) {
	// Chef open only 12:00–15:00 on Monday → dinner window (19:00+) yields nothing
	// today; all Monday slots fall within 12:00–15:00.
	monday := istMoment(2026, time.July, 20, 6, 0)
	schedules := []models.ChefSchedule{
		{DayOfWeek: int(time.Monday), OpenTime: "12:00", CloseTime: "15:00"},
	}
	times := BuildSuggestedFulfillmentTimes(nil, schedules, 30, monday, 12)
	sawMonday := false
	for _, s := range times {
		if s.Day != "Today" {
			continue
		}
		sawMonday = true
		h := s.At.In(capacityIST).Hour()
		mnt := s.At.In(capacityIST).Minute()
		mins := h*60 + mnt
		if mins < 12*60 || mins > 15*60 {
			t.Errorf("Monday slot %s outside chef open hours 12:00-15:00", s.Label)
		}
	}
	if !sawMonday {
		t.Error("expected at least one Monday slot inside 12:00-15:00")
	}
}

func TestSuggestedTimes_SkipsClosedDay(t *testing.T) {
	// Kitchen closed Monday → the first suggestions must be Tomorrow (Tuesday).
	now := istMoment(2026, time.July, 20, 6, 0) // Monday
	schedules := []models.ChefSchedule{
		{DayOfWeek: int(time.Monday), IsClosed: true},
	}
	times := BuildSuggestedFulfillmentTimes(nil, schedules, 30, now, 12)
	if len(times) == 0 {
		t.Fatal("expected suggestions on later days")
	}
	for _, s := range times {
		if s.Day == "Today" {
			t.Errorf("got a Today slot on a closed Monday: %s", s.Label)
		}
	}
}

func TestSuggestedTimes_PrefersChefConfiguredWindows(t *testing.T) {
	// Chef lunch window 13:00–14:00 overrides the default 12:00 lunch start.
	now := istMoment(2026, time.July, 20, 6, 0)
	cap := &models.ChefCapacitySettings{LunchSlotStart: "13:00", LunchSlotEnd: "14:00"}
	times := BuildSuggestedFulfillmentTimes(cap, nil, 30, now, 20)
	for _, s := range times {
		if s.Meal == "Lunch" && s.Day == "Today" {
			mins := s.At.In(capacityIST).Hour()*60 + s.At.In(capacityIST).Minute()
			if mins < 13*60 || mins > 14*60 {
				t.Errorf("configured lunch slot %s outside 13:00-14:00", s.Label)
			}
		}
	}
}
