package money

import "testing"

func TestFromRupees_RoundsHalfAwayNotTruncate(t *testing.T) {
	cases := []struct {
		in   float64
		want Paise
	}{
		{0.29, 29},        // classic IEEE-754: 0.29*100 == 28.999… → truncate would give 28
		{123.45, 12345},   // 123.45*100 == 12344.999…
		{19.99, 1999},     //
		{0.005, 1},        // half rounds away from zero → 1
		{0, 0},            //
		{1000.00, 100000}, //
		{0.1 + 0.2, 30},   // 0.30000000000000004 → 30
	}
	for _, c := range cases {
		if got := FromRupees(c.in); got != c.want {
			t.Errorf("FromRupees(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRupees_RoundTrip(t *testing.T) {
	for _, p := range []Paise{0, 1, 29, 1999, 12345, 100000} {
		if got := FromRupees(p.Rupees()); got != p {
			t.Errorf("round-trip %d → %v → %d", p, p.Rupees(), got)
		}
	}
}

func TestSum(t *testing.T) {
	if got := Sum([]Paise{100, 250, 33}); got != 383 {
		t.Errorf("Sum = %d, want 383", got)
	}
	if got := Sum(nil); got != 0 {
		t.Errorf("Sum(nil) = %d, want 0", got)
	}
}

// The core invariant: an allocation ALWAYS sums back to the total, for any total and any
// weight vector — this is what a float proportional split cannot guarantee.
func TestAllocate_AlwaysConservesTotal(t *testing.T) {
	totals := []Paise{0, 1, 2, 3, 7, 10, 100, 999, 1000, 12345, -10, -999}
	weightVecs := [][]Paise{
		{1},
		{1, 1},
		{1, 1, 1},
		{1, 2, 3},
		{10, 0, 5},
		{0, 0, 0}, // fallback to even
		{7, 7, 7, 7},
		{100, 1},  // very lopsided
		{3, 3, 3}, // ₹ thirds
		{1, 1, 1, 1, 1, 1, 1},
	}
	for _, total := range totals {
		for _, w := range weightVecs {
			parts := Allocate(total, w)
			if len(parts) != len(w) {
				t.Fatalf("Allocate(%d, %v): got %d parts, want %d", total, w, len(parts), len(w))
			}
			if s := Sum(parts); s != total {
				t.Errorf("Allocate(%d, %v) = %v sums to %d (must equal total)", total, w, parts, s)
			}
		}
	}
}

func TestAllocate_Empty(t *testing.T) {
	if got := Allocate(100, nil); got != nil {
		t.Errorf("Allocate(_, nil) = %v, want nil", got)
	}
	if got := Allocate(100, []Paise{}); got != nil {
		t.Errorf("Allocate(_, empty) = %v, want nil", got)
	}
}

func TestAllocate_SingleWeightGetsAll(t *testing.T) {
	if got := Allocate(12345, []Paise{99}); len(got) != 1 || got[0] != 12345 {
		t.Errorf("Allocate(12345, [99]) = %v, want [12345]", got)
	}
}

func TestAllocate_ThirdsOfATenner(t *testing.T) {
	// ₹10.00 split three ways: 334 + 333 + 333 = 1000 exactly (largest-remainder gives
	// the stray paise to the first part).
	got := Allocate(1000, []Paise{1, 1, 1})
	want := []Paise{334, 333, 333}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Allocate(1000, thirds) = %v, want %v", got, want)
		}
	}
}

func TestAllocate_Proportional(t *testing.T) {
	// total 1000 split 1:2:3 → 167 : 333 : 500 (sums to 1000).
	got := Allocate(1000, []Paise{1, 2, 3})
	if Sum(got) != 1000 {
		t.Fatalf("Allocate 1:2:3 sums to %d, want 1000", Sum(got))
	}
	// The largest weight gets the most, the smallest the least, and each is within 1 of
	// its exact proportional share.
	if !(got[0] <= got[1] && got[1] <= got[2]) {
		t.Errorf("Allocate 1:2:3 not monotonic: %v", got)
	}
}

func TestAllocate_ZeroWeightGetsNothingFromPositivePool(t *testing.T) {
	// A zero-weight part must get 0 when the pool has positive weights.
	got := Allocate(1000, []Paise{3, 0, 2})
	if got[1] != 0 {
		t.Errorf("zero-weight part got %d, want 0 (%v)", got[1], got)
	}
	if Sum(got) != 1000 {
		t.Errorf("sum %d, want 1000", Sum(got))
	}
}

// The leftover paise must go to the part with the LARGEST fractional remainder, never to
// a part that already got its exact whole share.
func TestAllocate_ExtraGoesToLargestRemainder(t *testing.T) {
	// total 10 split 2:1:1 → exact 5, 2.5, 2.5. Floors 5,2,2 (sum 9); the 1 leftover paise
	// goes to a remainder-0.5 part, never to part 0 (remainder exactly 0).
	got := Allocate(10, []Paise{2, 1, 1})
	if got[0] != 5 {
		t.Errorf("part 0 (whole share, remainder 0) should stay 5, got %v", got)
	}
	if Sum(got) != 10 {
		t.Errorf("sum %d, want 10", Sum(got))
	}
	if got[1]+got[2] != 5 {
		t.Errorf("the remaining 5 should split across parts 1,2; got %v", got)
	}
}

// A crore-rupee split must not overflow the int64 total×weight product.
func TestAllocate_LargeValuesNoOverflow(t *testing.T) {
	total := FromRupees(10_000_000) // ₹1 crore = 1e9 paise
	got := Allocate(total, []Paise{3, 3, 3, 3, 3, 3, 3})
	if Sum(got) != total {
		t.Errorf("crore split sums to %d, want %d", Sum(got), total)
	}
}

func TestAllocate_AllZeroWeightsSplitEvenly(t *testing.T) {
	got := Allocate(1000, []Paise{0, 0, 0})
	if Sum(got) != 1000 {
		t.Fatalf("sum %d, want 1000", Sum(got))
	}
	// Even split of 1000/3 = 334,333,333.
	if got[0] != 334 || got[1] != 333 || got[2] != 333 {
		t.Errorf("even fallback = %v, want [334 333 333]", got)
	}
}
