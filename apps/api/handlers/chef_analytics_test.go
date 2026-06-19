package handlers

// Unit tests for the pure analytics helpers (#49). The aggregations are
// DB-backed; here we cover the rate + forecast math.

import "testing"

func TestPct(t *testing.T) {
	cases := []struct {
		num, den int64
		want     float64
	}{
		{0, 0, 0},    // no denominator → 0, not NaN
		{1, 4, 25},   // 25%
		{1, 3, 33.3}, // rounds to 1 decimal
		{3, 3, 100},  // all
		{0, 10, 0},   // none
		{7, 8, 87.5}, // exact half-decimal
	}
	for _, c := range cases {
		if got := pct(c.num, c.den); got != c.want {
			t.Errorf("pct(%d,%d) = %v, want %v", c.num, c.den, got, c.want)
		}
	}
}

func TestForecastAvg(t *testing.T) {
	cases := []struct {
		total, periods, want int
	}{
		{40, 4, 10}, // 10/week
		{42, 4, 11}, // 10.5 rounds to 11
		{41, 4, 10}, // 10.25 rounds to 10
		{0, 4, 0},   // no history → 0
		{10, 0, 0},  // guard divide-by-zero
		{3, 4, 1},   // 0.75 rounds to 1
	}
	for _, c := range cases {
		if got := forecastAvg(c.total, c.periods); got != c.want {
			t.Errorf("forecastAvg(%d,%d) = %d, want %d", c.total, c.periods, got, c.want)
		}
	}
}
