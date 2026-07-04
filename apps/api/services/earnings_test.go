package services

import (
	"testing"
	"time"
)

func TestComputeOrderEarnings_FlatSixPercent_MoneyConserved(t *testing.T) {
	// The launch model (ADR-0001 / #390): a flat 6% platform commission. Gross
	// carries the food GST (Tax), NOT the delivery fee — delivery is the driver's.
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:    1000,
		Tax:            50,
		ChefTip:        20,
		DeliveryState:  "maharashtra",
		CommissionRate: 0.06,
	}, "Maharashtra")

	// commission 6% of 1000 = 60
	if got.PlatformCommission != 60 {
		t.Errorf("commission = %.2f, want 60 (6%% flat)", got.PlatformCommission)
	}
	// gross = 1000 + 50 + 20 = 1070
	if got.Gross != 1070 {
		t.Errorf("gross = %.2f, want 1070", got.Gross)
	}
	// intra-state GST off the commission: 9% each side of 60 = 5.40
	if got.CGST != 5.4 || got.SGST != 5.4 {
		t.Errorf("cgst/sgst = %.2f/%.2f, want 5.40/5.40", got.CGST, got.SGST)
	}
	// tds 1% of gross = 10.70
	if got.TDS != 10.7 {
		t.Errorf("tds = %.2f, want 10.70", got.TDS)
	}
	// net = gross 1070 - commission 60 - tds 10.70 = 999.30
	if got.NetPayout != 999.3 {
		t.Errorf("netPayout = %.2f, want 999.30", got.NetPayout)
	}

	// Money conservation: commission + tds + netPayout == gross.
	if got.PlatformCommission+got.TDS+got.NetPayout != got.Gross {
		t.Errorf("money not conserved: commission %.2f + tds %.2f + net %.2f = %.2f, want gross %.2f",
			got.PlatformCommission, got.TDS, got.NetPayout,
			got.PlatformCommission+got.TDS+got.NetPayout, got.Gross)
	}

	// Captured-conservation: platformRetained (commission + tds) + netPayout +
	// refunds (0 on a clean order) == gross. Every rupee the customer paid is
	// accounted for as platform-retained, chef payout, or refund.
	const refunds = 0.0
	platformRetained := got.PlatformCommission + got.TDS
	if platformRetained+got.NetPayout+refunds != got.Gross {
		t.Errorf("captured-conservation broken: retained %.2f + net %.2f + refunds %.2f = %.2f, want gross %.2f",
			platformRetained, got.NetPayout, refunds,
			platformRetained+got.NetPayout+refunds, got.Gross)
	}
}

func TestComputeOrderEarnings_GrossUsesTaxNotDelivery(t *testing.T) {
	// #390: the chef's gross is food revenue + food GST (Tax) + chef tip. The
	// delivery fee is the DRIVER's money and must be EXCLUDED from gross/net, even
	// though OrderEarnings.DeliveryFee is retained as a display/context field.
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:    1000,
		Tax:            50,
		DeliveryFee:    70,
		ChefTip:        20,
		DeliveryState:  "maharashtra",
		CommissionRate: 0.06,
	}, "Maharashtra")

	// gross = 1000 + 50(tax) + 20(tip) = 1070 — the ₹70 delivery is NOT in gross.
	if got.Gross != 1070 {
		t.Errorf("gross = %.2f, want 1070 (delivery 70 excluded)", got.Gross)
	}
	// net = 1070 - commission 60 - tds 10.70 = 999.30
	if got.NetPayout != 999.3 {
		t.Errorf("netPayout = %.2f, want 999.30", got.NetPayout)
	}
	// DeliveryFee is still surfaced for display, just not folded into gross/net.
	if got.DeliveryFee != 70 {
		t.Errorf("DeliveryFee = %.2f, want 70 (display context retained)", got.DeliveryFee)
	}
}

func TestComputeOrderEarnings_DefaultRateIsSixPercent(t *testing.T) {
	// A 0 / unset CommissionRate must fall back to the flat 6% default, NOT 15%.
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:   1000,
		DeliveryState: "maharashtra",
	}, "Maharashtra")

	if got.PlatformCommission != 60 {
		t.Errorf("commission = %.2f, want 60 (6%% default, not 150)", got.PlatformCommission)
	}
}

func TestComputeOrderEarnings_InjectedRate(t *testing.T) {
	// The per-order rate is injected via EarningsInput.CommissionRate; injecting
	// 0.12 still yields a 12% commission (the injection point survives the flat
	// default; only the "premium" framing is gone).
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:    1000,
		Tax:            50,
		ChefTip:        20,
		DeliveryState:  "maharashtra",
		CommissionRate: 0.12,
	}, "Maharashtra")

	// commission 12% of 1000 = 120
	if got.PlatformCommission != 120 {
		t.Errorf("commission = %.2f, want 120 (0.12 injected)", got.PlatformCommission)
	}
	// GST is computed off the injected commission: 9% each side of 120 = 10.8
	if got.CGST != 10.8 || got.SGST != 10.8 {
		t.Errorf("cgst/sgst = %.2f/%.2f, want 10.8/10.8", got.CGST, got.SGST)
	}
	// gross = 1000 + 50(tax) + 20 = 1070; net = 1070 - commission 120 - tds 10.7 = 939.3
	if got.NetPayout != 939.3 {
		t.Errorf("netPayout = %.2f, want 939.3", got.NetPayout)
	}
}

func TestComputeOrderEarnings_ChefFundedDiscount(t *testing.T) {
	// A chef-funded promo (#39) of 100 comes out of the chef's food revenue
	// before commission/gross/TDS. Effective itemRevenue = 1000 - 100 = 900.
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:        1000,
		Tax:                50,
		ChefTip:            20,
		DeliveryState:      "maharashtra",
		ChefFundedDiscount: 100,
	}, "Maharashtra")

	if got.ItemRevenue != 900 {
		t.Errorf("itemRevenue = %.2f, want 900 (net of chef-funded discount)", got.ItemRevenue)
	}
	// commission 6% of 900 = 54
	if got.PlatformCommission != 54 {
		t.Errorf("commission = %.2f, want 54", got.PlatformCommission)
	}
	// gross = 900 + 50(tax) + 20 = 970; tds 1% = 9.7; net = 970 - 54 - 9.7 = 906.3
	if got.Gross != 970 {
		t.Errorf("gross = %.2f, want 970", got.Gross)
	}
	if got.NetPayout != 906.3 {
		t.Errorf("netPayout = %.2f, want 906.3", got.NetPayout)
	}
}

func TestComputeOrderEarnings_PlatformFundedLeavesChefWhole(t *testing.T) {
	// Platform-funded promo → ChefFundedDiscount 0 → chef earnings unchanged.
	got := ComputeOrderEarnings(EarningsInput{ItemRevenue: 1000, DeliveryState: "x"}, "y")
	if got.ItemRevenue != 1000 || got.PlatformCommission != 60 {
		t.Errorf("platform-funded should leave chef whole: itemRevenue=%.2f commission=%.2f", got.ItemRevenue, got.PlatformCommission)
	}
}

func TestComputeOrderEarnings_IntraState(t *testing.T) {
	// New formula (#390): gross carries Tax (the food GST the chef receives), not
	// the delivery fee. Tax:50 drives gross 1070; DeliveryFee is intentionally
	// absent from gross/net.
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:   1000,
		Tax:           50,
		ChefTip:       20,
		DeliveryState: "maharashtra",
	}, "Maharashtra")

	// commission 6% of 1000 = 60; gross = 1070
	if got.PlatformCommission != 60 {
		t.Errorf("commission = %.2f, want 60", got.PlatformCommission)
	}
	if got.Gross != 1070 {
		t.Errorf("gross = %.2f, want 1070", got.Gross)
	}
	// intra-state: CGST 9% + SGST 9% of commission (60), IGST 0
	if got.CGST != 5.4 || got.SGST != 5.4 {
		t.Errorf("cgst/sgst = %.2f/%.2f, want 5.4/5.4", got.CGST, got.SGST)
	}
	if got.IGST != 0 {
		t.Errorf("igst = %.2f, want 0 (intra-state)", got.IGST)
	}
	// tds 1% of gross = 10.7; net = 1070 - 60 - 10.7
	if got.TDS != 10.7 {
		t.Errorf("tds = %.2f, want 10.7", got.TDS)
	}
	if got.NetPayout != 999.3 {
		t.Errorf("netPayout = %.2f, want 999.3", got.NetPayout)
	}
}

// TestComputeOrderEarnings_GSTSplitReconciles — the CGST/SGST split must reconcile
// EXACTLY to the full GST on commission, even for odd-paise commission (#462). The
// old split (cgst=sgst=Round2(GST/2*commission)) could sum ±1 paise off the total.
// commission 0.50 → fullGST = Round2(0.18*0.50) = 0.09; halving 0.045 doesn't
// reconcile under the old code. Compared in integer paise to avoid float noise.
func TestComputeOrderEarnings_GSTSplitReconciles(t *testing.T) {
	intra := ComputeOrderEarnings(EarningsInput{
		ItemRevenue: 5, CommissionRate: 0.10, DeliveryState: "maharashtra",
	}, "Maharashtra")
	if paise(intra.PlatformCommission) != 50 {
		t.Fatalf("commission = %.2f, want 0.50 (odd-paise GST setup)", intra.PlatformCommission)
	}
	fullGST := paise(Round2(RateGST * intra.PlatformCommission)) // 9 paise
	if paise(intra.CGST)+paise(intra.SGST) != fullGST {
		t.Errorf("cgst+sgst = %d+%d paise, want %d (must reconcile to full GST)",
			paise(intra.CGST), paise(intra.SGST), fullGST)
	}
	if intra.IGST != 0 {
		t.Errorf("igst = %.2f, want 0 (intra-state)", intra.IGST)
	}

	inter := ComputeOrderEarnings(EarningsInput{
		ItemRevenue: 5, CommissionRate: 0.10, DeliveryState: "delhi",
	}, "Maharashtra")
	if paise(inter.IGST) != paise(Round2(RateGST*inter.PlatformCommission)) {
		t.Errorf("igst = %.2f, want full GST", inter.IGST)
	}
	if inter.CGST != 0 || inter.SGST != 0 {
		t.Errorf("cgst/sgst = %.2f/%.2f, want 0/0 (inter-state)", inter.CGST, inter.SGST)
	}
}

// paise converts a rupee amount to integer paise for exact comparison.
func paise(rupees float64) int64 { return int64(rupees*100 + 0.5) }

func TestComputeOrderEarnings_InterState(t *testing.T) {
	got := ComputeOrderEarnings(EarningsInput{
		ItemRevenue:   1000,
		DeliveryState: "Delhi",
	}, "Maharashtra")

	// inter-state: IGST 18% of commission (60) = 10.8; CGST/SGST 0
	if got.IGST != 10.8 {
		t.Errorf("igst = %.2f, want 10.8", got.IGST)
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
