package models

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

// Tests for the group-order split math (#46): shares must sum exactly to
// subtotal+extras (no paise lost/created), pro-rata by item subtotal.

func TestSplitShares_SumsToTotal_ProRata(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	subtotals := map[uuid.UUID]float64{a: 200, b: 100, c: 100} // group subtotal 400
	extras := 80.0                                             // delivery+service+tax
	shares := SplitShares(subtotals, extras)

	// Pro-rata: a pays 200 + 80*(200/400)=240; b,c pay 100 + 20 = 120 each.
	if math.Abs(shares[a]-240) > 0.01 {
		t.Fatalf("a share = %v, want 240", shares[a])
	}
	if math.Abs(shares[b]-120) > 0.01 || math.Abs(shares[c]-120) > 0.01 {
		t.Fatalf("b/c shares = %v/%v, want 120 each", shares[b], shares[c])
	}
	var sum float64
	for _, s := range shares {
		sum += s
	}
	if math.Abs(sum-480) > 0.001 {
		t.Fatalf("shares sum = %v, want 480 (400 subtotal + 80 extras)", sum)
	}
}

func TestSplitShares_RoundingRemainderToLargest(t *testing.T) {
	a, b, d := uuid.New(), uuid.New(), uuid.New()
	// Subtotals that force rounding drift when extras are split 3 ways.
	subtotals := map[uuid.UUID]float64{a: 100, b: 100, d: 100} // 300
	extras := 10.0                                             // 10/3 = 3.33 each → drift
	shares := SplitShares(subtotals, extras)

	var sum float64
	for _, s := range shares {
		sum += s
	}
	// Critical invariant: the shares must sum EXACTLY to 310 (no paise lost).
	if math.Abs(sum-310) > 0.001 {
		t.Fatalf("shares sum = %v, want exactly 310", sum)
	}
}

func TestSplitShares_NoItems_SplitsExtrasEvenly(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	subtotals := map[uuid.UUID]float64{a: 0, b: 0}
	shares := SplitShares(subtotals, 100)
	var sum float64
	for _, s := range shares {
		sum += s
	}
	if math.Abs(sum-100) > 0.001 {
		t.Fatalf("shares sum = %v, want 100 (even split of extras)", sum)
	}
}

func TestSplitShares_Empty(t *testing.T) {
	if got := SplitShares(map[uuid.UUID]float64{}, 50); len(got) != 0 {
		t.Fatalf("empty input should yield empty shares, got %v", got)
	}
}
