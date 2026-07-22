package payouts

import "testing"

func inr(minor int64) Money { return Money{Minor: minor, Currency: CurrencyINR} }

func TestAddAndSub(t *testing.T) {
	sum, err := inr(1050).Add(inr(2575))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if sum.Minor != 3625 {
		t.Fatalf("Add = %d, want 3625", sum.Minor)
	}

	diff, err := inr(1000).Sub(inr(2500))
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	if diff.Minor != -1500 {
		t.Fatalf("Sub = %d, want -1500", diff.Minor)
	}
}

func TestCurrencyMismatchIsAnError(t *testing.T) {
	// A zero-valued Money carries no currency and must not silently adopt one.
	if _, err := inr(100).Add(Money{Minor: 100, Currency: "USD"}); err == nil {
		t.Fatal("Add across currencies must error")
	}
	if _, err := inr(100).Sub(Money{Minor: 100, Currency: ""}); err == nil {
		t.Fatal("Sub against an unset currency must error")
	}
}

func TestZeroValueAddIsPermitted(t *testing.T) {
	// Sum() over an empty slice has no currency to report; adding a true zero
	// to a real amount is the one mismatch we tolerate, so accumulator loops work.
	got, err := Zero(CurrencyINR).Add(inr(500))
	if err != nil {
		t.Fatalf("zero + amount: %v", err)
	}
	if got.Minor != 500 || got.Currency != CurrencyINR {
		t.Fatalf("got %v, want 500 INR", got)
	}
}

func TestSum(t *testing.T) {
	got, err := Sum(inr(100), inr(250), inr(-75))
	if err != nil {
		t.Fatalf("Sum: %v", err)
	}
	if got.Minor != 275 {
		t.Fatalf("Sum = %d, want 275", got.Minor)
	}

	empty, err := Sum()
	if err != nil {
		t.Fatalf("Sum of nothing: %v", err)
	}
	if !empty.IsZero() {
		t.Fatalf("Sum of nothing = %v, want zero", empty)
	}
}

func TestApplyBasisPointsRoundsHalfAwayFromZero(t *testing.T) {
	cases := []struct {
		name string
		in   int64
		bps  int64
		want int64
	}{
		{"6 percent of 100.00", 10000, 600, 600},
		{"exact half rounds away from zero", 5, 5000, 3}, // 2.5 -> 3
		{"below half rounds down", 4, 5000, 2},           // 2.0 -> 2
		{"negative half rounds away from zero", -5, 5000, -3},
		{"zero bps", 12345, 0, 0},
		{"full", 12345, 10000, 12345},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inr(tc.in).ApplyBasisPoints(tc.bps)
			if got.Minor != tc.want {
				t.Fatalf("ApplyBasisPoints(%d, %d) = %d, want %d", tc.in, tc.bps, got.Minor, tc.want)
			}
			if got.Currency != CurrencyINR {
				t.Fatalf("currency lost: %v", got.Currency)
			}
		})
	}
}

func TestPredicatesAndClamp(t *testing.T) {
	if !inr(-1).IsNegative() {
		t.Fatal("-1 should be negative")
	}
	if inr(0).IsNegative() {
		t.Fatal("0 must not be negative")
	}
	if !inr(0).IsZero() {
		t.Fatal("0 should be zero")
	}
	if got := inr(-500).ClampAtZero(); got.Minor != 0 {
		t.Fatalf("ClampAtZero = %d, want 0", got.Minor)
	}
	if got := inr(500).ClampAtZero(); got.Minor != 500 {
		t.Fatalf("ClampAtZero must not alter positives, got %d", got.Minor)
	}
	if got := inr(-250).Neg(); got.Minor != 250 {
		t.Fatalf("Neg = %d, want 250", got.Minor)
	}
}

func TestCmp(t *testing.T) {
	lt, err := inr(100).Cmp(inr(200))
	if err != nil || lt != -1 {
		t.Fatalf("Cmp = %d, %v; want -1, nil", lt, err)
	}
	eq, _ := inr(200).Cmp(inr(200))
	if eq != 0 {
		t.Fatalf("Cmp equal = %d, want 0", eq)
	}
	if _, err := inr(1).Cmp(Money{Minor: 1, Currency: "USD"}); err == nil {
		t.Fatal("Cmp across currencies must error")
	}
}

func TestStringIsHumanReadable(t *testing.T) {
	if got := inr(123456).String(); got != "1234.56 INR" {
		t.Fatalf("String = %q, want %q", got, "1234.56 INR")
	}
	if got := inr(-5).String(); got != "-0.05 INR" {
		t.Fatalf("String = %q, want %q", got, "-0.05 INR")
	}
}
