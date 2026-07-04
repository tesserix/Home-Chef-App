package services

// earnings.go — single source of truth for the chef earnings math.
//
// India-specific settlement rules (shared by the live /chef/earnings/breakdown
// endpoint, the weekly settlement statement generator, and the TDS Form 16A
// certificate). Keeping the math in one place prevents the three surfaces
// drifting — a rate change here propagates everywhere.
//
//   - Platform commission on item revenue (subtotal only).
//   - gross = itemRevenue + Tax + chefTip. The food GST (Tax) is the chef's
//     income and enters gross; the delivery fee is the DRIVER's money and is
//     EXCLUDED from the chef's gross/net (#390).
//   - GST 18% on the commission: CGST 9% + SGST 9% (intra-state) or
//     IGST 18% (inter-state). This GST is the platform's downstream remittance
//     obligation on its own commission — NOT deducted from the chef's payout and
//     NOT part of the customer's captured order total.
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
	// DefaultCommissionRate is the flat platform commission on the food subtotal
	// (ADR-0001 / #390); runtime-tunable via GetCommissionRate. Every chef is
	// charged this by default — there is no premium reduced-rate tier in the
	// payout math.
	DefaultCommissionRate = 0.06

	// RateCommission is the platform's share of each order's item revenue. It is
	// the flat DefaultCommissionRate — a single source value that keeps the
	// display references (statement/earnings labels) showing the current rate.
	RateCommission = DefaultCommissionRate

	// RateGST is the total GST rate applied to the commission amount.
	RateGST = 0.18

	// RateTDS is the TDS rate deducted from gross payout under Section 194-O.
	RateTDS = 0.01

	// EarningsCurrency is the reporting currency for India-domiciled chefs.
	EarningsCurrency = "INR"
)

// EarningsInput is the per-order data the math operates on.
type EarningsInput struct {
	OrderID     uuid.UUID
	OrderNumber string
	CompletedAt time.Time
	ItemRevenue float64
	DeliveryFee float64
	// Tax is the order's food GST. The chef receives it, so it enters the chef's
	// gross (and thus TDS base) — unlike DeliveryFee, which is the driver's (#390).
	Tax           float64
	ChefTip       float64
	DeliveryState string
	// CommissionRate is the resolved flat commission rate for this order; 0/unset
	// uses DefaultCommissionRate. Callers resolve it from PlatformSettings via
	// GetCommissionRate and inject it here, keeping the math pure.
	CommissionRate float64
	// ChefFundedDiscount is the chef-funded promo discount the chef bears (#39).
	// It reduces the chef's food revenue before commission/gross/TDS. Zero for
	// platform-funded promos and ordinary orders, so existing callers are unaffected.
	ChefFundedDiscount float64
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
//	gross      = itemRevenue + Tax + chefTip   (delivery fee is the driver's, excluded)
//	intra-state (order.state == chef.state): CGST 9% + SGST 9% on commission
//	inter-state (order.state != chef.state): IGST 18% on commission
//	tds        = RateTDS × gross
//	netPayout  = gross − commission − tds   (GST is not deducted from chef)
func ComputeOrderEarnings(in EarningsInput, chefState string) OrderEarnings {
	// The flat commission rate is injected per order; 0/unset (or out of range)
	// falls back to the flat DefaultCommissionRate.
	rate := in.CommissionRate
	if rate <= 0 || rate >= 1 {
		rate = DefaultCommissionRate
	}
	// A chef-funded promo (#39) comes out of the chef's food revenue before
	// commission/gross/TDS — the chef bears the discount they funded. Floored at 0.
	itemRevenue := in.ItemRevenue - in.ChefFundedDiscount
	if itemRevenue < 0 {
		itemRevenue = 0
	}
	commission := Round2(rate * itemRevenue)
	// Gross is the chef's income: food revenue + food GST (Tax) + chef tip. The
	// delivery fee is intentionally NOT here — it is the driver's money (#390).
	gross := Round2(itemRevenue + in.Tax + in.ChefTip)

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
		OrderID:     in.OrderID,
		OrderNumber: in.OrderNumber,
		CompletedAt: in.CompletedAt,
		ItemRevenue: Round2(itemRevenue),
		// DeliveryFee is retained for display/context only — it does NOT enter
		// gross or net (it is the driver's money, settled separately) (#390).
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
