package handlers

// Unit test for the catering deposit-default rule (#55). The lifecycle handlers
// are DB-backed (exercised in integration); here we cover the pure money math.

import "testing"

func TestDefaultCateringDeposit(t *testing.T) {
	cases := []struct {
		name           string
		deposit, total float64
		want           float64
	}{
		{"defaults to 25% when unset", 0, 1000, 250},
		{"defaults to 25% when negative", -50, 800, 200},
		{"keeps a chef-set deposit", 300, 1000, 300},
		{"clamps a deposit above total", 1500, 1000, 1000},
		{"rounds to paise", 0, 333, 83.25},
		{"zero total → zero", 0, 0, 0},
	}
	for _, c := range cases {
		if got := defaultCateringDeposit(c.deposit, c.total); got != c.want {
			t.Errorf("%s: defaultCateringDeposit(%v,%v) = %v, want %v", c.name, c.deposit, c.total, got, c.want)
		}
	}
}
