package services

// earnings.go — single source of truth for the chef earnings math.
//
// India-specific settlement rules (shared by the live /chef/earnings/breakdown
// endpoint, the weekly settlement statement generator, and the TDS Form 16A
// certificate). Keeping the math in one place prevents the three surfaces
// drifting — a rate change here propagates everywhere.
//
//   - Platform commission on item revenue (subtotal only).
//   - GST 18% on the commission: CGST 9% + SGST 9% (intra-state) or
//     IGST 18% (inter-state). GST is levied on the platform's revenue,
//     NOT deducted from the chef's payout.
//   - TDS 1% under Section 194-O on gross order value.
//   - netPayout = gross − platformCommission − tds.

import (
	"time"

	"github.com/google/uuid"
)

// Earnings rate constants. Exported so every surface references the same
// values; defined as consts so they can be lifted to env/DB config later
// without touching call sites.
const (
	// RateCommission is the platform's share of each order's item revenue.
	RateCommission = 0.15

	// RateGST is the total GST rate applied to the commission amount.
	RateGST = 0.18

	// RateTDS is the TDS rate deducted from gross payout under Section 194-O.
	RateTDS = 0.01

	// EarningsCurrency is the reporting currency for India-domiciled chefs.
	EarningsCurrency = "INR"
)

// EarningsInput is the per-order data the math operates on.
type EarningsInput struct {
	OrderID       uuid.UUID
	OrderNumber   string
	CompletedAt   time.Time
	ItemRevenue   float64
	DeliveryFee   float64
	ChefTip       float64
	DeliveryState string
	// CommissionRate overrides the platform take-rate for this order (#44 premium
	// tier). Zero/unset uses the standard RateCommission, so existing callers are
	// unaffected.
	CommissionRate float64
}

// OrderEarnings is the computed per-order breakdown.
type OrderEarnings struct {
	OrderID            uuid.UUID
	OrderNumber        string
	CompletedAt        time.Time
	ItemRevenue        float64
	DeliveryFee        float64
	Tip                float64
	Gross              float64
	PlatformCommission float64
	CGST               float64
	SGST               float64
	IGST               float64
	TDS                float64
	NetPayout          float64
}

// EarningsTotals accumulates per-order earnings into period totals.
type EarningsTotals struct {
	GrossRevenue       float64
	PlatformCommission float64
	CGST               float64
	SGST               float64
	IGST               float64
	TDS                float64
	NetPayout          float64
	OrdersCount        int
}

// ComputeOrderEarnings applies the settlement rules to a single order.
//
//	commission = RateCommission × itemRevenue
//	gross      = itemRevenue + deliveryFee + chefTip
//	intra-state (order.state == chef.state): CGST 9% + SGST 9% on commission
//	inter-state (order.state != chef.state): IGST 18% on commission
//	tds        = RateTDS × gross
//	netPayout  = gross − commission − tds   (GST is not deducted from chef)
func ComputeOrderEarnings(in EarningsInput, chefState string) OrderEarnings {
	// Premium chefs (#44) carry a lower commission rate; 0/unset = the standard rate.
	rate := in.CommissionRate
	if rate <= 0 || rate >= 1 {
		rate = RateCommission
	}
	commission := Round2(rate * in.ItemRevenue)
	gross := Round2(in.ItemRevenue + in.DeliveryFee + in.ChefTip)

	var cgst, sgst, igst float64
	halfGST := Round2(RateGST / 2 * commission)
	fullGST := Round2(RateGST * commission)

	if NormaliseState(in.DeliveryState) == NormaliseState(chefState) {
		cgst = halfGST
		sgst = halfGST
	} else {
		igst = fullGST
	}

	tds := Round2(RateTDS * gross)
	netPayout := Round2(gross - commission - tds)

	return OrderEarnings{
		OrderID:            in.OrderID,
		OrderNumber:        in.OrderNumber,
		CompletedAt:        in.CompletedAt,
		ItemRevenue:        Round2(in.ItemRevenue),
		DeliveryFee:        Round2(in.DeliveryFee),
		Tip:                Round2(in.ChefTip),
		Gross:              gross,
		PlatformCommission: commission,
		CGST:               cgst,
		SGST:               sgst,
		IGST:               igst,
		TDS:                tds,
		NetPayout:          netPayout,
	}
}

// Add folds a single order's earnings into the running totals. The caller
// invokes Round once after the loop so intermediate sums don't compound
// rounding error.
func (t *EarningsTotals) Add(e OrderEarnings) {
	t.GrossRevenue += e.Gross
	t.PlatformCommission += e.PlatformCommission
	t.CGST += e.CGST
	t.SGST += e.SGST
	t.IGST += e.IGST
	t.TDS += e.TDS
	t.NetPayout += e.NetPayout
	t.OrdersCount++
}

// Round normalises every accumulated total to 2dp.
func (t *EarningsTotals) Round() {
	t.GrossRevenue = Round2(t.GrossRevenue)
	t.PlatformCommission = Round2(t.PlatformCommission)
	t.CGST = Round2(t.CGST)
	t.SGST = Round2(t.SGST)
	t.IGST = Round2(t.IGST)
	t.TDS = Round2(t.TDS)
	t.NetPayout = Round2(t.NetPayout)
}

// Round2 rounds a float64 to 2 decimal places (half-up). Monetary values
// here are non-negative on the hot path; the negative branch is retained
// for correctness.
func Round2(v float64) float64 {
	const factor = 100.0
	if v >= 0 {
		return float64(int64(v*factor+0.5)) / factor
	}
	return -float64(int64(-v*factor+0.5)) / factor
}

// NormaliseState lowercases and strips whitespace from a state name so
// "Maharashtra" == "maharashtra" and " MAHARASHTRA " == "maharashtra".
func NormaliseState(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			out = append(out, r+32)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
