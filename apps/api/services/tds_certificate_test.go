package services

import (
	"testing"
	"time"
)

func TestFinancialYearWindow(t *testing.T) {
	start, end := FinancialYearWindow(2025)
	wantStart := time.Date(2025, time.April, 1, 0, 0, 0, 0, istLocation)
	wantEnd := time.Date(2026, time.April, 1, 0, 0, 0, 0, istLocation)
	if !start.Equal(wantStart) {
		t.Errorf("start = %s, want %s", start, wantStart.UTC())
	}
	if !end.Equal(wantEnd) {
		t.Errorf("end = %s, want %s", end, wantEnd.UTC())
	}
}

func TestCurrentFinancialYearStart(t *testing.T) {
	tests := []struct {
		now  time.Time
		want int
	}{
		{time.Date(2026, 5, 10, 6, 0, 0, 0, time.UTC), 2026},  // May → FY2026
		{time.Date(2026, 2, 10, 6, 0, 0, 0, time.UTC), 2025},  // Feb → FY2025
		{time.Date(2026, 4, 1, 0, 0, 0, 0, istLocation), 2026}, // 1 Apr boundary → FY2026
		{time.Date(2026, 3, 31, 23, 0, 0, 0, istLocation), 2025},
	}
	for _, tc := range tests {
		if got := CurrentFinancialYearStart(tc.now); got != tc.want {
			t.Errorf("CurrentFinancialYearStart(%s) = %d, want %d", tc.now, got, tc.want)
		}
	}
}

func TestFinancialQuarterIndex(t *testing.T) {
	tests := []struct {
		month time.Month
		want  int
	}{
		{time.April, 0}, {time.June, 0},
		{time.July, 1}, {time.September, 1},
		{time.October, 2}, {time.December, 2},
		{time.January, 3}, {time.March, 3},
	}
	for _, tc := range tests {
		d := time.Date(2025, tc.month, 15, 12, 0, 0, 0, istLocation)
		if got := financialQuarterIndex(d); got != tc.want {
			t.Errorf("financialQuarterIndex(%s) = %d, want %d", tc.month, got, tc.want)
		}
	}
}
