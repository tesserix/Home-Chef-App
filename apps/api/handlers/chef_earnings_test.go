package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestComputeOrderBreakdown_IntraState verifies the earnings math for an
// intra-state order (delivery state == chef state → CGST + SGST, no IGST).
func TestComputeOrderBreakdown_IntraState(t *testing.T) {
	row := earningsOrderRow{
		OrderID:       uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		OrderNumber:   "HC001",
		CompletedAt:   time.Now(),
		ItemRevenue:   1000.00,
		DeliveryFee:   50.00,
		ChefTip:       20.00,
		DeliveryState: "Maharashtra",
	}
	chefState := "Maharashtra"

	got := computeOrderBreakdown(row, chefState, 0)

	// commission = 0.06 × 1000 = 60 (flat default, 0 rate falls back to 6%)
	wantCommission := 60.00
	if got.PlatformCommission != wantCommission {
		t.Errorf("PlatformCommission: got %.2f, want %.2f", got.PlatformCommission, wantCommission)
	}

	// gross = 1000 + 50 + 20 = 1070
	wantGross := 1070.00
	if got.Gross != wantGross {
		t.Errorf("Gross: got %.2f, want %.2f", got.Gross, wantGross)
	}

	// CGST = 9% × 60 = 5.40
	wantCGST := 5.40
	if got.CGST != wantCGST {
		t.Errorf("CGST: got %.2f, want %.2f", got.CGST, wantCGST)
	}

	// SGST = 9% × 60 = 5.40
	wantSGST := 5.40
	if got.SGST != wantSGST {
		t.Errorf("SGST: got %.2f, want %.2f", got.SGST, wantSGST)
	}

	// IGST must be 0 for intra-state
	if got.IGST != 0 {
		t.Errorf("IGST: got %.2f, want 0 (intra-state)", got.IGST)
	}

	// TDS = 1% × 1070 = 10.70
	wantTDS := 10.70
	if got.TDS != wantTDS {
		t.Errorf("TDS: got %.2f, want %.2f", got.TDS, wantTDS)
	}

	// netPayout = 1070 - 60 - 10.70 = 999.30
	wantNet := 999.30
	if got.NetPayout != wantNet {
		t.Errorf("NetPayout: got %.2f, want %.2f", got.NetPayout, wantNet)
	}
}

// TestComputeOrderBreakdown_InterState verifies that IGST (not CGST+SGST) is
// applied when the delivery state differs from the chef's state.
func TestComputeOrderBreakdown_InterState(t *testing.T) {
	row := earningsOrderRow{
		OrderID:       uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		OrderNumber:   "HC002",
		CompletedAt:   time.Now(),
		ItemRevenue:   500.00,
		DeliveryFee:   30.00,
		ChefTip:       0.00,
		DeliveryState: "Karnataka",
	}
	chefState := "Maharashtra"

	got := computeOrderBreakdown(row, chefState, 0)

	// CGST and SGST must both be 0 for inter-state
	if got.CGST != 0 {
		t.Errorf("CGST: got %.2f, want 0 (inter-state)", got.CGST)
	}
	if got.SGST != 0 {
		t.Errorf("SGST: got %.2f, want 0 (inter-state)", got.SGST)
	}

	// IGST = 18% × (0.06 × 500) = 18% × 30 = 5.40
	wantIGST := 5.40
	if got.IGST != wantIGST {
		t.Errorf("IGST: got %.2f, want %.2f", got.IGST, wantIGST)
	}
}

// TestComputeOrderBreakdown_ZeroTip verifies correct behaviour when chefTip is 0.
func TestComputeOrderBreakdown_ZeroTip(t *testing.T) {
	row := earningsOrderRow{
		OrderID:       uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		OrderNumber:   "HC003",
		CompletedAt:   time.Now(),
		ItemRevenue:   200.00,
		DeliveryFee:   0.00,
		ChefTip:       0.00,
		DeliveryState: "Delhi",
	}

	got := computeOrderBreakdown(row, "Delhi", 0)

	wantGross := 200.00
	if got.Gross != wantGross {
		t.Errorf("Gross: got %.2f, want %.2f", got.Gross, wantGross)
	}

	wantCommission := 12.00 // 6% of 200
	if got.PlatformCommission != wantCommission {
		t.Errorf("PlatformCommission: got %.2f, want %.2f", got.PlatformCommission, wantCommission)
	}

	wantTDS := 2.00 // 1% of 200
	if got.TDS != wantTDS {
		t.Errorf("TDS: got %.2f, want %.2f", got.TDS, wantTDS)
	}

	// netPayout = 200 - 12 - 2 = 186
	wantNet := 186.00
	if got.NetPayout != wantNet {
		t.Errorf("NetPayout: got %.2f, want %.2f", got.NetPayout, wantNet)
	}
}

// TestNormaliseState verifies that state comparison is case-insensitive and
// ignores leading/trailing whitespace.
func TestNormaliseState(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Maharashtra", "maharashtra", true},
		{"MAHARASHTRA", "maharashtra", true},
		{" Maharashtra ", "Maharashtra", true},
		{"Karnataka", "Maharashtra", false},
		{"", "", true},
	}
	for _, tc := range cases {
		got := normaliseState(tc.a) == normaliseState(tc.b)
		if got != tc.want {
			t.Errorf("normaliseState(%q) == normaliseState(%q): got %v, want %v",
				tc.a, tc.b, got, tc.want)
		}
	}
}

// TestRound2 confirms the helper rounds to 2 decimal places correctly.
func TestRound2(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{13.505, 13.51},
		{13.504, 13.50},
		{0.0, 0.0},
		{-1.01, -1.01},
		{-1.004, -1.00},
		{100.999, 101.00},
	}
	for _, tc := range cases {
		got := round2(tc.in)
		if got != tc.want {
			t.Errorf("round2(%.4f) = %.4f, want %.4f", tc.in, got, tc.want)
		}
	}
}

// TestResolvePeriod_Week checks that the week period returns a 7-day window.
func TestResolvePeriod_Week(t *testing.T) {
	start, end := resolvePeriod("week", uuid.Nil)
	diff := end.Sub(start)
	// Must be at least 6 days and at most 8 days (approximate)
	if diff < 6*24*time.Hour || diff > 8*24*time.Hour {
		t.Errorf("week period diff = %v, expected ~7 days", diff)
	}
}

// TestResolvePeriod_Month checks that the month period returns a ~30-day window.
func TestResolvePeriod_Month(t *testing.T) {
	start, end := resolvePeriod("month", uuid.Nil)
	diff := end.Sub(start)
	if diff < 29*24*time.Hour || diff > 31*24*time.Hour {
		t.Errorf("month period diff = %v, expected ~30 days", diff)
	}
}
