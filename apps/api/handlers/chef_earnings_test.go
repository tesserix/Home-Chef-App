package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// TestComputeOrderBreakdown_IntraState verifies the earnings math for an
// intra-state order (delivery state == chef state → CGST + SGST, no IGST).
func TestComputeOrderBreakdown_IntraState(t *testing.T) {
	row := earningsOrderRow{
		OrderID:     uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		OrderNumber: "HC001",
		CompletedAt: time.Now(),
		ItemRevenue: 1000.00,
		// #390: gross carries the food GST (Tax), not the delivery fee.
		Tax:           50.00,
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

	// gross = 1000 + 50(tax) + 20 = 1070
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

// TestComputeOrderBreakdown_SurfacesPayoutHoldStatus verifies the per-order
// escrow hold status is carried through to the wire response (#617), and that a
// no-hold order (escrow flags off) surfaces the empty status so the pill hides.
func TestComputeOrderBreakdown_SurfacesPayoutHoldStatus(t *testing.T) {
	row := earningsOrderRow{
		OrderID:          uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		OrderNumber:      "HC004",
		CompletedAt:      time.Now(),
		ItemRevenue:      300.00,
		DeliveryState:    "Delhi",
		PayoutHoldStatus: string(models.PayoutHoldAwaitingConfirmation),
	}
	if got := computeOrderBreakdown(row, "Delhi", 0).PayoutHoldStatus; got != models.PayoutHoldAwaitingConfirmation {
		t.Errorf("PayoutHoldStatus: got %q, want awaiting_customer_confirmation", got)
	}

	noHold := earningsOrderRow{OrderID: uuid.New(), OrderNumber: "HC005", CompletedAt: time.Now(), ItemRevenue: 100, DeliveryState: "Delhi"}
	if got := computeOrderBreakdown(noHold, "Delhi", 0).PayoutHoldStatus; got != models.PayoutHoldNone {
		t.Errorf("no-hold order should surface empty status, got %q", got)
	}
}

// TestPayoutBucket verifies the Held / Released classification that drives the
// vendor earnings escrow split (#617): held = awaiting/eligible/disputed, released
// = released, and withheld/reversed/none fall into neither. Never both.
func TestPayoutBucket(t *testing.T) {
	cases := []struct {
		status            models.PayoutHoldStatus
		wantHeld, wantRel bool
	}{
		{models.PayoutHoldAwaitingConfirmation, true, false},
		{models.PayoutHoldReleaseEligible, true, false},
		{models.PayoutHoldDisputed, true, false},
		{models.PayoutHoldReleased, false, true},
		{models.PayoutHoldWithheld, false, false},
		{models.PayoutHoldReversed, false, false},
		{models.PayoutHoldNone, false, false}, // escrow flags off
	}
	for _, tc := range cases {
		held, rel := payoutBucket(tc.status)
		if held != tc.wantHeld || rel != tc.wantRel {
			t.Errorf("payoutBucket(%q) = (held=%v, released=%v), want (%v, %v)",
				tc.status, held, rel, tc.wantHeld, tc.wantRel)
		}
		if held && rel {
			t.Errorf("payoutBucket(%q) must never be in both buckets", tc.status)
		}
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

// TestResolvePeriod_Week pins the earnings week to the CURRENT settlement week:
// it opens on the Monday 00:00 IST the weekly statement opens on, and runs to
// the end of today. A rolling 7-day window would start mid-week and put this
// screen on a different week from the dashboard and the payout (#937).
func TestResolvePeriod_Week(t *testing.T) {
	start, end := resolvePeriod("week", uuid.Nil)

	ist := services.BusinessLocation()
	if got := start.In(ist).Weekday(); got != time.Monday {
		t.Errorf("week start = %v, want a Monday", got)
	}
	if h, m, s := start.In(ist).Clock(); h|m|s != 0 {
		t.Errorf("week start clock = %02d:%02d:%02d, want midnight IST", h, m, s)
	}
	if want := services.BusinessDayEnd(time.Now()); !end.Equal(want) {
		t.Errorf("week end = %v, want end of today IST %v", end, want)
	}
	// Same Monday the settlement statement closes on, so the chef's "this week"
	// and their payout week can never drift apart.
	if _, closedEnd := services.MostRecentClosedWeek(time.Now()); !start.Equal(closedEnd) {
		t.Errorf("week start %v != the Monday settlement closed on %v", start, closedEnd)
	}
	if !end.After(start) {
		t.Errorf("week window is inverted: %v .. %v", start, end)
	}
}

// TestResolvePeriod_Month pins the earnings month to the current IST calendar
// month-to-date, matching the "cycle" fallback rather than a rolling 30 days.
func TestResolvePeriod_Month(t *testing.T) {
	start, end := resolvePeriod("month", uuid.Nil)

	ist := services.BusinessLocation()
	if got := start.In(ist).Day(); got != 1 {
		t.Errorf("month start day = %d, want the 1st", got)
	}
	if h, m, s := start.In(ist).Clock(); h|m|s != 0 {
		t.Errorf("month start clock = %02d:%02d:%02d, want midnight IST", h, m, s)
	}
	if got, want := start.In(ist).Month(), time.Now().In(ist).Month(); got != want {
		t.Errorf("month start month = %v, want the current month %v", got, want)
	}
	if want := services.BusinessDayEnd(time.Now()); !end.Equal(want) {
		t.Errorf("month end = %v, want end of today IST %v", end, want)
	}
}

// TestResolvePeriod_UnknownFallsBackToWeek keeps an unrecognised ?period= on the
// settlement week rather than silently widening the money window.
func TestResolvePeriod_UnknownFallsBackToWeek(t *testing.T) {
	wantStart, wantEnd := resolvePeriod("week", uuid.Nil)
	gotStart, gotEnd := resolvePeriod("banana", uuid.Nil)

	if !gotStart.Equal(wantStart) || !gotEnd.Equal(wantEnd) {
		t.Errorf("unknown period = %v..%v, want the week window %v..%v",
			gotStart, gotEnd, wantStart, wantEnd)
	}
}
