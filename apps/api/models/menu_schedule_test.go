package models

import (
	"testing"

	"github.com/lib/pq"
)

func TestMenuItemIsAvailableOn(t *testing.T) {
	// Empty schedule = available every day (default / legacy rows).
	everyDay := MenuItem{}
	for wd := 0; wd <= 6; wd++ {
		if !everyDay.IsAvailableOn(wd) {
			t.Fatalf("empty AvailableDays must be available every day; failed on %d", wd)
		}
	}

	// Weekdays-only schedule (Mon..Fri = 1..5).
	weekdaysOnly := MenuItem{AvailableDays: pq.Int64Array{1, 2, 3, 4, 5}}
	if weekdaysOnly.IsAvailableOn(0) || weekdaysOnly.IsAvailableOn(6) {
		t.Fatal("weekdays-only dish must not be available on Sun(0)/Sat(6)")
	}
	for wd := 1; wd <= 5; wd++ {
		if !weekdaysOnly.IsAvailableOn(wd) {
			t.Fatalf("weekdays-only dish must be available on %d", wd)
		}
	}
}
