package services

import (
	"testing"
	"time"
)

func TestComputeOrderEarnings_PremiumCommissionOverride(t *testing.T) {
	// Premium chef (#44): a 12% commission override instead of the default 15%.
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:    1000,
		DeliveryFee:    50,
		ChefTip:        20,
		DeliveryState:  "maharashtra",
		CommissionRate: 0.12,
	}, "Maharashtra")

	// commission 12% of 1000 = 120 (not 150)
	if got.PlatformCommission != 120 {
		t.Errorf("commission = %.2f, want 120 (12%% override)", got.PlatformCommission)
	}
	// GST is computed off the (lower) commission: 9% each side of 120 = 10.8
	if got.CGST != 10.8 || got.SGST != 10.8 {
		t.Errorf("cgst/sgst = %.2f/%.2f, want 10.8/10.8", got.CGST, got.SGST)
	}
	// net = gross 1070 - commission 120 - tds 10.7 = 939.3
	if got.NetPayout != 939.3 {
		t.Errorf("netPayout = %.2f, want 939.3", got.NetPayout)
	}
}

func TestComputeOrderEarnings_ZeroRateFallsBackToDefault(t *testing.T) {
	// A 0 / unset CommissionRate must use the standard 15% (back-compat).
	got := ComputeOrderEarnings(EarningsInput{ItemRevenue: 1000, DeliveryState: "x"}, "y")
	if got.PlatformCommission != 150 {
		t.Errorf("commission = %.2f, want 150 (default rate)", got.PlatformCommission)
	}
}

func TestComputeOrderEarnings_IntraState(t *testing.T) {
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:   1000,
		DeliveryFee:   50,
		ChefTip:       20,
		DeliveryState: "maharashtra",
	}, "Maharashtra")

	// commission 15% of 1000 = 150; gross = 1070
	if got.PlatformCommission != 150 {
		t.Errorf("commission = %.2f, want 150", got.PlatformCommission)
	}
	if got.Gross != 1070 {
		t.Errorf("gross = %.2f, want 1070", got.Gross)
	}
	// intra-state: CGST 9% + SGST 9% of commission, IGST 0
	if got.CGST != 13.5 || got.SGST != 13.5 {
		t.Errorf("cgst/sgst = %.2f/%.2f, want 13.5/13.5", got.CGST, got.SGST)
	}
	if got.IGST != 0 {
		t.Errorf("igst = %.2f, want 0 (intra-state)", got.IGST)
	}
	// tds 1% of gross = 10.7; net = 1070 - 150 - 10.7
	if got.TDS != 10.7 {
		t.Errorf("tds = %.2f, want 10.7", got.TDS)
	}
	if got.NetPayout != 909.3 {
		t.Errorf("netPayout = %.2f, want 909.3", got.NetPayout)
	}
}

func TestComputeOrderEarnings_InterState(t *testing.T) {
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:   1000,
		DeliveryState: "Delhi",
	}, "Maharashtra")

	// inter-state: IGST 18% of commission (150) = 27; CGST/SGST 0
	if got.IGST != 27 {
		t.Errorf("igst = %.2f, want 27", got.IGST)
	}
	if got.CGST != 0 || got.SGST != 0 {
		t.Errorf("cgst/sgst = %.2f/%.2f, want 0/0 (inter-state)", got.CGST, got.SGST)
	}
}

func TestEarningsTotals_AddAndRound(t *testing.T) {
	var totals EarningsTotals
	for i := 0; i < 3; i++ {
		totals.Add(ComputeOrderEarnings(EarningsInput{
			ItemRevenue:   100.33,
			DeliveryState: "x",
		}, "y"))
	}
	totals.Round()
	if totals.OrdersCount != 3 {
		t.Errorf("ordersCount = %d, want 3", totals.OrdersCount)
	}
	// gross per order = round2(100.33) = 100.33; sum = 300.99
	if totals.GrossRevenue != 300.99 {
		t.Errorf("grossRevenue = %.2f, want 300.99", totals.GrossRevenue)
	}
}

func TestMostRecentClosedWeek(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		wantStart time.Time // Monday 00:00 IST
		wantEnd   time.Time // following Monday 00:00 IST
	}{
		{
			name:      "wednesday mid-week",
			now:       time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC), // Wed
			wantStart: time.Date(2026, 6, 1, 0, 0, 0, 0, istLocation),
			wantEnd:   time.Date(2026, 6, 8, 0, 0, 0, 0, istLocation),
		},
		{
			name:      "monday returns the week that just closed",
			now:       time.Date(2026, 6, 8, 6, 0, 0, 0, time.UTC), // Mon (IST 11:30)
			wantStart: time.Date(2026, 6, 1, 0, 0, 0, 0, istLocation),
			wantEnd:   time.Date(2026, 6, 8, 0, 0, 0, 0, istLocation),
		},
		{
			// On Sunday the current week (Jun 1–Jun 7) is still in progress,
			// so the most recent *closed* week is the one before it.
			name:      "sunday: current week not yet closed",
			now:       time.Date(2026, 6, 7, 12, 0, 0, 0, time.UTC), // Sun
			wantStart: time.Date(2026, 5, 25, 0, 0, 0, 0, istLocation),
			wantEnd:   time.Date(2026, 6, 1, 0, 0, 0, 0, istLocation),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			start, end := MostRecentClosedWeek(tc.now)
			if !start.Equal(tc.wantStart) {
				t.Errorf("start = %s, want %s", start, tc.wantStart.UTC())
			}
			if !end.Equal(tc.wantEnd) {
				t.Errorf("end = %s, want %s", end, tc.wantEnd.UTC())
			}
			if d := end.Sub(start); d != 7*24*time.Hour {
				t.Errorf("week length = %s, want 168h", d)
			}
		})
	}
}
