package services

// topaise_test.go — #524/#396 Phase 1. ToPaise truncated (int(amount*100)), so
// IEEE-754 near-integer products lost a paise (0.29 → 28.99… → 28). Now that #518
// feeds Round2-ed NET payouts straight into ToPaise on the Route-transfer path,
// that under-pays real money. ToPaise must round, matching ToMinor's INR path.

import (
	"testing"
)

func TestToPaise_RoundsInsteadOfTruncating(t *testing.T) {
	cases := []struct {
		amount float64
		want   int
	}{
		{0.29, 29},      // classic: 0.29*100 == 28.9999… → truncated 28
		{123.45, 12345}, // 123.45*100 == 12344.9999…
		{19.99, 1999},
		{1.10, 110},
		{499.00, 49900},
		{0.01, 1},
		{0.10, 10},
		{2.30, 230}, // 2.30*100 == 229.9999…
		{0.0, 0},
	}
	for _, c := range cases {
		if got := ToPaise(c.amount); got != c.want {
			t.Errorf("ToPaise(%v) = %d, want %d", c.amount, got, c.want)
		}
	}
}

// ToPaise must agree with ToMinor(amount, "INR") for every rupee amount so the two
// gateway paths mint the identical minor-unit value (#396 acceptance criterion).
func TestToPaise_MatchesToMinorINR(t *testing.T) {
	for cents := 0; cents <= 100000; cents++ {
		amount := float64(cents) / 100.0
		if got, want := ToPaise(amount), ToMinor(amount, "INR"); got != want {
			t.Fatalf("ToPaise(%v)=%d != ToMinor=%d", amount, got, want)
		}
	}
}

// Money-conservation: splitting a total into Round2-ed parts and summing their
// paise must equal the paise of the whole (no truncation drift), the property the
// escrow invariant depends on.
func TestToPaise_PartsSumToWhole(t *testing.T) {
	// three equal-ish parts of 100.00 → 33.33 + 33.33 + 33.34
	parts := []float64{33.33, 33.33, 33.34}
	sum := 0
	for _, p := range parts {
		sum += ToPaise(p)
	}
	if want := ToPaise(100.00); sum != want {
		t.Errorf("sum of part paise = %d, want whole = %d", sum, want)
	}
}
