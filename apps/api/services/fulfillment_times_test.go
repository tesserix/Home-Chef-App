package services

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

// istAt builds an IST instant for the test's "now".
func istMoment(y int, mo time.Month, d, h, m int) time.Time {
	return time.Date(y, mo, d, h, m, 0, 0, istLoc)
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
		h := s.At.In(istLoc).Hour()
		if s.Day == "Today" && h < 8 {
			t.Errorf("today slot before 08:00: %s (%s)", s.Label, s.Meal)
		}
	}
	first := times[0]
	if first.Meal != "Breakfast" || first.At.In(istLoc).Hour() < 8 {
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
		h := s.At.In(istLoc).Hour()
		mnt := s.At.In(istLoc).Minute()
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

func TestSuggestedTimes_CoversWholeOpenDay(t *testing.T) {
	// The reported bug: a chef open 10:00–22:00 was only offered the platform meal
	// windows (12:00–14:30 / 16:30–18:00 / 19:00–21:30), so 10:00–12:00, 14:30–16:30,
	// 18:00–19:00 and 21:00–22:00 were open but unbookable. Open hours are the offer.
	monday := istMoment(2026, time.July, 20, 6, 0)
	schedules := []models.ChefSchedule{
		{DayOfWeek: int(time.Monday), OpenTime: "10:00", CloseTime: "22:00"},
	}
	times := BuildSuggestedFulfillmentTimes(nil, schedules, 30, monday, 64)

	got := make(map[string]bool)
	for _, s := range times {
		if s.Day != "Today" {
			continue
		}
		got[s.At.In(istLoc).Format("15:04")] = true
		mins := s.At.In(istLoc).Hour()*60 + s.At.In(istLoc).Minute()
		if mins < 10*60 || mins >= 22*60 {
			t.Errorf("slot %s outside open hours 10:00-22:00", s.Label)
		}
		if s.Meal == "" {
			t.Errorf("slot %s has no meal label", s.Label)
		}
	}
	// Times inside the old platform gaps must now be offered.
	for _, want := range []string{"10:00", "11:00", "15:00", "16:00", "18:30", "21:30"} {
		if !got[want] {
			t.Errorf("expected slot %s inside the kitchen's open hours, not offered", want)
		}
	}
}

func TestSuggestedTimes_MealLabelsAreContiguous(t *testing.T) {
	// Every slot in an all-day kitchen gets a label, so the checkout groups cleanly.
	monday := istMoment(2026, time.July, 20, 6, 0)
	schedules := []models.ChefSchedule{
		{DayOfWeek: int(time.Monday), OpenTime: "09:00", CloseTime: "21:00"},
	}
	times := BuildSuggestedFulfillmentTimes(nil, schedules, 30, monday, 64)
	want := map[string]string{"09:30": "Breakfast", "12:00": "Lunch", "17:00": "Snacks", "20:00": "Dinner"}
	for _, s := range times {
		if s.Day != "Today" {
			continue
		}
		if w, ok := want[s.At.In(istLoc).Format("15:04")]; ok && s.Meal != w {
			t.Errorf("slot %s labelled %q, want %q", s.Label, s.Meal, w)
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
			mins := s.At.In(istLoc).Hour()*60 + s.At.In(istLoc).Minute()
			if mins < 13*60 || mins > 14*60 {
				t.Errorf("configured lunch slot %s outside 13:00-14:00", s.Label)
			}
		}
	}
}

func TestChefOpenAt(t *testing.T) {
	// Monday 10:00-22:00, Tuesday closed. (#969)
	schedules := []models.ChefSchedule{
		{DayOfWeek: int(time.Monday), OpenTime: "10:00", CloseTime: "22:00"},
		{DayOfWeek: int(time.Tuesday), IsClosed: true},
	}
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"inside open hours", istMoment(2026, time.July, 20, 13, 0), true},
		{"exactly at open", istMoment(2026, time.July, 20, 10, 0), true},
		{"exactly at close", istMoment(2026, time.July, 20, 22, 0), true},
		{"before open", istMoment(2026, time.July, 20, 9, 30), false},
		{"after close", istMoment(2026, time.July, 20, 22, 30), false},
		{"closed weekday", istMoment(2026, time.July, 21, 13, 0), false},
		{"weekday with no row is open", istMoment(2026, time.July, 22, 13, 0), true},
	}
	for _, c := range cases {
		if got := ChefOpenAt(schedules, c.at); got != c.want {
			t.Errorf("%s: ChefOpenAt(%s) = %v, want %v", c.name, c.at.Format(time.RFC3339), got, c.want)
		}
	}
	if !ChefOpenAt(nil, istMoment(2026, time.July, 20, 3, 0)) {
		t.Error("a chef with no schedule at all must not be blocked")
	}
}
