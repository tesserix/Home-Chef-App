package services

// chef_availability_status_test.go — the real-time availability decision, especially the reported
// prod bug: a chef past their daily cutoff must read Closed on the card, not Open.

import (
	"testing"
	"time"

	"github.com/homechef/api/models"
)

// baseOpen is a fully orderable kitchen (no schedule/cutoff/platform gating) at 12:00 IST.
func baseOpen(nowMin int) availInputs {
	return availInputs{
		nowMin:           nowMin,
		acceptingOrders:  true,
		pausedUntilMin:   -1,
		schedOpenMin:     -1,
		schedCloseMin:    -1,
		dailyCloseMin:    -1,
		platformOpen:     true,
		platformOpenMin:  -1,
		platformCloseMin: -1,
	}
}

func TestAvail_PlainOpen(t *testing.T) {
	got := decideAvailability(baseOpen(12 * 60))
	if !got.Orderable || got.Status != AvailOpen {
		t.Fatalf("want open+orderable, got %+v", got)
	}
}

// THE BUG: accepting_orders is true but the daily cutoff has passed → must be Closed, not Open.
func TestAvail_PastDailyCutoff_IsClosed(t *testing.T) {
	in := baseOpen(15 * 60) // 3pm
	in.dailyCloseMin = 14 * 60
	in.pastDailyClose = true
	got := decideAvailability(in)
	if got.Orderable {
		t.Fatalf("past-cutoff kitchen must NOT be orderable, got %+v", got)
	}
	if got.Status != AvailClosed {
		t.Fatalf("want closed, got %+v", got)
	}
}

// Closing soon: orderable and the cutoff is 20 min out → closing_soon pill with the countdown.
func TestAvail_ClosingSoon_Cutoff(t *testing.T) {
	in := baseOpen(13*60 + 40) // 1:40pm
	in.dailyCloseMin = 14 * 60 // 2:00pm → 20 min
	got := decideAvailability(in)
	if got.Status != AvailClosingSoon || got.MinutesToChange != 20 || !got.Orderable {
		t.Fatalf("want closing_soon in 20, got %+v", got)
	}
}

// A cutoff more than the soon-window away stays plain Open (no premature pill).
func TestAvail_CutoffBeyondWindow_StaysOpen(t *testing.T) {
	in := baseOpen(13 * 60)    // 1pm
	in.dailyCloseMin = 14 * 60 // 2pm → 60 min > 30
	if got := decideAvailability(in); got.Status != AvailOpen {
		t.Fatalf("want open, got %+v", got)
	}
}

// Opening soon: auto-schedule kitchen, before today's open, opens in 15 min.
func TestAvail_OpeningSoon_Schedule(t *testing.T) {
	in := baseOpen(9*60 + 45)  // 9:45am, not yet open
	in.acceptingOrders = false // cron hasn't flipped it on yet
	in.scheduleGates = true
	in.schedOpenMin = 10 * 60 // opens 10:00 → 15 min
	in.schedCloseMin = 22 * 60
	got := decideAvailability(in)
	if got.Status != AvailOpeningSoon || got.MinutesToChange != 15 {
		t.Fatalf("want opening_soon in 15, got %+v", got)
	}
}

// Auto-schedule kitchen PAST today's close reads Closed even though accepting_orders is still true
// (the 5-minute cron lag the card used to show through).
func TestAvail_AutoSchedulePastClose_ClosesLag(t *testing.T) {
	in := baseOpen(22*60 + 10) // 10:10pm
	in.acceptingOrders = true  // cron hasn't closed it yet
	in.scheduleGates = true
	in.schedOpenMin = 10 * 60
	in.schedCloseMin = 22 * 60 // closed at 10:00pm
	got := decideAvailability(in)
	if got.Orderable {
		t.Fatalf("past schedule close must not be orderable, got %+v", got)
	}
}

// Closing soon driven by the schedule close (auto-schedule), 10 min out.
func TestAvail_ClosingSoon_Schedule(t *testing.T) {
	in := baseOpen(21*60 + 50) // 9:50pm
	in.scheduleGates = true
	in.schedOpenMin = 10 * 60
	in.schedCloseMin = 22 * 60 // 10:00pm → 10 min
	got := decideAvailability(in)
	if got.Status != AvailClosingSoon || got.MinutesToChange != 10 {
		t.Fatalf("want closing_soon in 10, got %+v", got)
	}
}

// A timed pause resuming beyond the window reads paused (not opening_soon).
func TestAvail_PausedBeyondWindow(t *testing.T) {
	in := baseOpen(12 * 60)
	in.pausedUntilMin = 13 * 60 // back in 60 min
	got := decideAvailability(in)
	if got.Status != AvailPaused {
		t.Fatalf("want paused, got %+v", got)
	}
}

// A pause resuming within the window reads opening_soon with the countdown.
func TestAvail_PausedWithinWindow_OpeningSoon(t *testing.T) {
	in := baseOpen(12 * 60)
	in.pausedUntilMin = 12*60 + 20 // back in 20 min
	got := decideAvailability(in)
	if got.Status != AvailOpeningSoon || got.MinutesToChange != 20 {
		t.Fatalf("want opening_soon in 20, got %+v", got)
	}
}

// Manual off (accepting=false, no schedule) → plain Closed, no phantom open time.
func TestAvail_ManualOff_Closed(t *testing.T) {
	in := baseOpen(12 * 60)
	in.acceptingOrders = false
	got := decideAvailability(in)
	if got.Orderable || got.Status != AvailClosed {
		t.Fatalf("want closed, got %+v", got)
	}
}

// Platform closed gates everything even if the chef is accepting.
func TestAvail_PlatformClosed(t *testing.T) {
	in := baseOpen(23 * 60)
	in.platformOpen = false
	if got := decideAvailability(in); got.Orderable {
		t.Fatalf("platform-closed must not be orderable, got %+v", got)
	}
}

// The earliest close wins: schedule closes at 22:00 but the cutoff is 12:20 (20 min) → closing_soon 20.
func TestAvail_EarliestCloseWins(t *testing.T) {
	in := baseOpen(12 * 60)
	in.scheduleGates = true
	in.schedOpenMin = 10 * 60
	in.schedCloseMin = 22 * 60
	in.dailyCloseMin = 12*60 + 20
	got := decideAvailability(in)
	if got.Status != AvailClosingSoon || got.MinutesToChange != 20 {
		t.Fatalf("want closing_soon in 20 (cutoff beats schedule), got %+v", got)
	}
}

// hhmmLabel formats a 12-hour clock label.
func TestHHMMLabel(t *testing.T) {
	cases := map[int]string{0: "12:00 am", 9 * 60: "9:00 am", 12 * 60: "12:00 pm", 13*60 + 30: "1:30 pm", 22 * 60: "10:00 pm"}
	for min, want := range cases {
		if got := hhmmLabel(min); got != want {
			t.Errorf("hhmmLabel(%d) = %q, want %q", min, got, want)
		}
	}
}

// ── Resolver tests (model → decision), the exact path the list/detail handlers take. ──
// Platform hours read the default policy (nil DB → always open) so these need no database.

// THE BUG, end-to-end: an accepting chef past its cutoff resolves to Closed (not Orderable).
func TestComputeChefAvailability_PastCutoff_Closed(t *testing.T) {
	now := time.Date(2026, 7, 25, 15, 0, 0, 0, istLoc) // 3:00pm IST
	chef := &models.ChefProfile{AcceptingOrders: true, IsActive: true}
	cap := &models.ChefCapacitySettings{CutoffEnabled: true, LunchCutoff: "10:00", DinnerCutoff: "14:00"}
	got := ComputeChefAvailability(chef, nil, cap, now)
	if got.Orderable || got.Status != AvailClosed {
		t.Fatalf("past-cutoff chef must be closed, got %+v", got)
	}
}

// Closing-soon resolves from a cutoff 15 min out.
func TestComputeChefAvailability_ClosingSoon(t *testing.T) {
	now := time.Date(2026, 7, 25, 13, 45, 0, 0, istLoc) // 1:45pm
	chef := &models.ChefProfile{AcceptingOrders: true, IsActive: true}
	cap := &models.ChefCapacitySettings{CutoffEnabled: true, DinnerCutoff: "14:00"} // 2:00pm → 15 min
	got := ComputeChefAvailability(chef, nil, cap, now)
	if got.Status != AvailClosingSoon || got.MinutesToChange != 15 || !got.Orderable {
		t.Fatalf("want closing_soon 15, got %+v", got)
	}
}

// Auto-schedule kitchen past today's close resolves to not-orderable even with accepting=true.
func TestComputeChefAvailability_AutoSchedulePastClose(t *testing.T) {
	now := time.Date(2026, 7, 25, 22, 30, 0, 0, istLoc) // 10:30pm
	chef := &models.ChefProfile{AcceptingOrders: true, IsActive: true, AutoScheduleEnabled: true}
	sched := &models.ChefSchedule{DayOfWeek: int(now.In(istLoc).Weekday()), OpenTime: "10:00", CloseTime: "22:00"}
	got := ComputeChefAvailability(chef, sched, nil, now)
	if got.Orderable {
		t.Fatalf("past schedule close must not be orderable, got %+v", got)
	}
}

// A plain accepting chef with no cutoffs/schedule is simply Open.
func TestComputeChefAvailability_PlainOpen(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, istLoc)
	got := ComputeChefAvailability(&models.ChefProfile{AcceptingOrders: true, IsActive: true}, nil, nil, now)
	if !got.Orderable || got.Status != AvailOpen {
		t.Fatalf("want open, got %+v", got)
	}
}

// laterCutoffMinutes returns the LATER of the set cutoffs (IsPastDailyClose needs both passed).
func TestLaterCutoffMinutes(t *testing.T) {
	if got := laterCutoffMinutes(&models.ChefCapacitySettings{CutoffEnabled: true, LunchCutoff: "10:00", DinnerCutoff: "16:00"}); got != 16*60 {
		t.Errorf("later cutoff = %d, want %d", got, 16*60)
	}
	if got := laterCutoffMinutes(&models.ChefCapacitySettings{CutoffEnabled: true, LunchCutoff: "10:00"}); got != 10*60 {
		t.Errorf("lunch-only cutoff = %d, want %d", got, 10*60)
	}
	if got := laterCutoffMinutes(&models.ChefCapacitySettings{CutoffEnabled: false, LunchCutoff: "10:00"}); got != -1 {
		t.Errorf("disabled cutoff = %d, want -1", got)
	}
}

// ── Account-level pause ───────────────────────────────────────────────────────
//
// A chef who pauses their whole account must never read as orderable. The
// listing query already excludes them, but a direct link or a favourite reaches
// the detail handler, which used to render a normal, orderable-looking kitchen
// that only failed at checkout with "Chef not found or not available".

func TestDecideAvailability_AccountPaused_IsNotOrderable(t *testing.T) {
	in := baseOpen(720) // midday, everything else wide open
	in.accountPaused = true

	got := decideAvailability(in)

	if got.Orderable {
		t.Fatal("a paused account must never be orderable")
	}
	if got.Status != AvailPaused {
		t.Fatalf("status = %q, want %q", got.Status, AvailPaused)
	}
	if got.Label == "" {
		t.Fatal("paused kitchens need a label for the greyed-out pill")
	}
}

// The account pause outranks the clock: no reopening time is quoted, because
// only the chef can lift it.
func TestDecideAvailability_AccountPause_OutranksTimedPause(t *testing.T) {
	in := baseOpen(720)
	in.accountPaused = true
	in.pausedUntilMin = 800 // a timed pause that would otherwise quote "Back at ..."

	got := decideAvailability(in)

	if got.Status != AvailPaused {
		t.Fatalf("status = %q, want %q", got.Status, AvailPaused)
	}
	if got.MinutesToChange != 0 {
		t.Fatalf("MinutesToChange = %d, want 0 — there is no known reopen time",
			got.MinutesToChange)
	}
}

func TestComputeChefAvailability_InactiveChefIsPaused(t *testing.T) {
	chef := &models.ChefProfile{AcceptingOrders: true, IsActive: false}

	got := ComputeChefAvailability(chef, nil, nil, time.Now())

	if got.Orderable {
		t.Fatal("an inactive chef profile must not be orderable")
	}
	if got.Status != AvailPaused {
		t.Fatalf("status = %q, want %q", got.Status, AvailPaused)
	}
}
