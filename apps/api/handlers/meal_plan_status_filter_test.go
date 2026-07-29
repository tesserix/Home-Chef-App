package handlers

import (
	"testing"

	"github.com/homechef/api/models"
)

// The chef plan list used to take ONE status defaulting to pending_chef, so an
// accepted plan vanished from every chef-facing surface until its day-orders
// generated 12h before each meal. Fetching a lane in one call is what makes the
// "upcoming plans" view possible.
func TestSplitStatuses_MultipleLanes(t *testing.T) {
	got := splitStatuses("confirmed,active")
	want := []models.MealPlanStatus{models.MealPlanConfirmed, models.MealPlanActive}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestSplitStatuses_TrimsAndDedupes(t *testing.T) {
	got := splitStatuses(" confirmed , active ,confirmed")
	if len(got) != 2 {
		t.Fatalf("want 2 unique statuses, got %v", got)
	}
}

// An unknown name must be dropped rather than reaching the query. Passing it
// through would return zero rows, which is indistinguishable from "this chef has
// no plans" — the exact silent-empty failure this endpoint already had once.
func TestSplitStatuses_DropsUnknown(t *testing.T) {
	got := splitStatuses("confirmed,not_a_status")
	if len(got) != 1 || got[0] != models.MealPlanConfirmed {
		t.Fatalf("unknown status must be dropped; got %v", got)
	}
	if len(splitStatuses("garbage")) != 0 {
		t.Error("an all-unknown filter must yield nothing so the handler can 400")
	}
	if len(splitStatuses("")) != 0 {
		t.Error("an empty filter must yield nothing so the handler can 400")
	}
}
