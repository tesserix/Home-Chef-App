package models

// pricing_test.go — money conservation. Every case here asserts the same two
// invariants, because breaking either is what put a tax invoice on a customer's
// phone whose lines summed to a paise more than the total printed under them:
//
//  1. the displayed components (plus Rounding) sum EXACTLY to Total;
//  2. the tax lines sum EXACTLY to Tax.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// assertFoots is invariants 1 and 2, checked at the paise the customer sees.
func assertFoots(t *testing.T, p OrderPricing, inclusive bool) {
	t.Helper()
	sum := p.Subtotal + p.DeliveryFee + p.PlatformFee - p.Discount + p.Tip + p.Rounding
	if !inclusive {
		sum += p.Tax
	}
	require.InDelta(t, p.Total, RoundAmount(sum), 1e-9, "lines must sum to the total")

	if len(p.TaxLines) > 0 {
		var lines float64
		for _, l := range p.TaxLines {
			lines += l.Amount
		}
		require.InDelta(t, p.Tax, RoundAmount(lines), 1e-9, "tax lines must sum to the tax")
	}
}

// indiaGST is the live rule: 5% exclusive, intra-state.
func indiaGST(subtotal, deliveryFee, discount, tip float64) PricingInput {
	return PricingInput{
		Subtotal:    subtotal,
		DeliveryFee: deliveryFee,
		// 4.99% of subtotal — the rate that produced the un-round 11.976.
		PlatformFee: subtotal * 4.99 / 100,
		Discount:    discount,
		Tip:         tip,
		TaxRate:     5,
		TaxName:     "GST",
		Country:     "IN",
		IntraState:  true,
	}
}

// The order from the receipt that started this: HC26080306287649, a ₹240 pickup.
// It charged 264.5748 and displayed 240.00 + 11.98 + 6.30 + 6.30 = 264.58 above a
// total of ₹264.57.
func TestComputeOrderPricing_RealPickupOrderFoots(t *testing.T) {
	p := ComputeOrderPricing(indiaGST(240, 0, 0, 0))

	require.Equal(t, 11.98, p.PlatformFee, "the fee is stored at paise, not as 11.976")
	require.Equal(t, 12.6, p.Tax)
	require.Equal(t, 264.58, p.Total)
	require.Zero(t, p.Rounding, "nothing to reconcile — this was priced, not patched")

	require.Len(t, p.TaxLines, 2)
	require.Equal(t, TaxLineCGST, p.TaxLines[0].Code)
	require.Equal(t, "CGST (2.5%)", p.TaxLines[0].Label)
	require.Equal(t, 6.3, p.TaxLines[0].Amount)
	require.Equal(t, 6.3, p.TaxLines[1].Amount)
	assertFoots(t, p, false)
}

// The same order as it is STORED today. The charged total is authoritative, so
// the paise the rounded lines overshoot by becomes an explicit rounding row
// rather than an invoice that silently fails to add up.
func TestPresentOrderPricing_ReconcilesLegacyOrderToChargedTotal(t *testing.T) {
	in := indiaGST(240, 0, 0, 0)
	in.PlatformFee = 11.976 // as written before the fee was rounded
	p := PresentOrderPricing(in, 12.5988, 264.5748)

	require.Equal(t, 264.57, p.Total, "the customer paid 264.57 and the invoice must say so")
	require.Equal(t, 11.98, p.PlatformFee)
	require.Equal(t, 12.6, p.Tax)
	require.Equal(t, -0.01, p.Rounding)
	assertFoots(t, p, false)
}

func TestComputeOrderPricing_FootsAcrossAmounts(t *testing.T) {
	cases := []struct{ subtotal, delivery, discount, tip float64 }{
		{240, 0, 0, 0},
		{249.99, 39, 0, 0},
		{1, 0, 0, 0},
		{333.33, 45.5, 50, 20},
		{1000.01, 0, 0, 7.77},
		{499.95, 29.99, 100, 0},
		{0.05, 0, 0, 0},
	}
	for _, c := range cases {
		p := ComputeOrderPricing(indiaGST(c.subtotal, c.delivery, c.discount, c.tip))
		assertFoots(t, p, false)
	}
}

// A discount bigger than the order cannot produce a negative charge, and the
// discount LINE must shrink with it or the invoice stops adding up. The clients
// floored the total at zero and the server did not.
func TestComputeOrderPricing_OverDiscountIsCappedAtTheCharge(t *testing.T) {
	p := ComputeOrderPricing(indiaGST(100, 0, 500, 25))
	require.Equal(t, 104.99, p.Discount, "capped at subtotal + fees, not the 500 requested")
	require.Equal(t, 25.0, p.Total, "nothing to pay but the tip")
	require.Zero(t, p.Tax)
	require.Empty(t, p.TaxLines, "no tax means no tax line, not a ₹0.00 CGST row")
	assertFoots(t, p, false)
}

func TestBuildTaxLines_InterStateIsOneIGSTLine(t *testing.T) {
	in := indiaGST(240, 0, 0, 0)
	in.IntraState = false
	p := ComputeOrderPricing(in)

	require.Len(t, p.TaxLines, 1)
	require.Equal(t, TaxLineIGST, p.TaxLines[0].Code)
	require.Equal(t, "IGST (5%)", p.TaxLines[0].Label)
	require.Equal(t, p.Tax, p.TaxLines[0].Amount)
	assertFoots(t, p, false)
}

// An inclusive rate is already inside the base — the tax line is informational
// and must not be added to the total a second time.
func TestComputeOrderPricing_InclusiveTaxIsNotAddedToTotal(t *testing.T) {
	p := ComputeOrderPricing(PricingInput{
		Subtotal:     100,
		TaxRate:      20,
		TaxName:      "VAT",
		TaxInclusive: true,
		Country:      "GB",
	})
	require.Equal(t, 100.0, p.Total)
	require.Equal(t, 16.67, p.Tax)
	require.Equal(t, "VAT (incl.)", p.TaxLines[0].Label)
	assertFoots(t, p, true)
}

// A tax carried by an order written before TaxRate was snapshotted must be named
// without claiming a rate the amount contradicts.
func TestBuildTaxLines_UnratedTaxPrintsNoPercentage(t *testing.T) {
	lines := BuildTaxLines(11.20, PricingInput{Country: "IN", IntraState: true})
	require.Len(t, lines, 2)
	require.Equal(t, "CGST", lines[0].Label)
	require.InDelta(t, 11.20, lines[0].Amount+lines[1].Amount, 1e-9)
}

// An odd paise cannot be halved evenly; the second line takes the remainder so
// the pair still sums to the tax exactly.
func TestBuildTaxLines_OddPaiseSplitsWithoutDrift(t *testing.T) {
	lines := BuildTaxLines(12.59, PricingInput{TaxRate: 5, Country: "IN", IntraState: true})
	require.InDelta(t, 12.59, lines[0].Amount+lines[1].Amount, 1e-9)
}

func TestIntraStateSupply_BlankSideDefaultsToIntra(t *testing.T) {
	require.True(t, IntraStateSupply("", "Odisha"))
	require.True(t, IntraStateSupply("Odisha", ""))
	require.True(t, IntraStateSupply("Odisha", " odisha "))
	require.False(t, IntraStateSupply("Odisha", "Maharashtra"))
}
