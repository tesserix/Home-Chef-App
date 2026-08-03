package models

// order_pricing_response_test.go — the DTO contract every client renders against.
// Whatever OrderResponse says must add up, because the app, the web page and the
// invoice PDF now print it without doing any arithmetic of their own.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Every case below is an Indian exclusive-GST order, the only kind the platform
// charges today; an inclusive rate keeps its tax inside the subtotal and is
// labelled "(incl.)" instead of being added.
func assertResponseFoots(t *testing.T, r OrderResponse) {
	t.Helper()
	sum := r.Subtotal + r.DeliveryFee + r.PlatformFee - r.Discount + r.Tip + r.Tax + r.Rounding
	require.InDelta(t, r.Total, RoundAmount(sum), 1e-9, "the rows on the receipt must sum to its total")

	var lines float64
	for _, l := range r.TaxLines {
		lines += l.Amount
	}
	if len(r.TaxLines) > 0 {
		require.InDelta(t, r.Tax, RoundAmount(lines), 1e-9)
	}
}

// The order behind the receipt this work started from: ₹240 pickup, platform fee
// stored as the raw 240 × 4.99% = 11.976, GST 12.5988, charged 264.5748. The app
// rendered 240.00 + 11.98 + 6.30 + 6.30 = 264.58 above a total of ₹264.57.
func TestOrderToResponse_LegacyOrderFootsToTheChargedTotal(t *testing.T) {
	o := Order{
		Subtotal:               240,
		PlatformFee:            11.976,
		Tax:                    12.5988,
		TaxRate:                5,
		TaxName:                "GST",
		Total:                  264.5748,
		DeliveryAddressCountry: "IN",
	}
	r := o.ToResponse()

	require.Equal(t, 264.57, r.Total, "the total is what the customer paid, never recomputed")
	require.Len(t, r.TaxLines, 2)
	require.Equal(t, "CGST (2.5%)", r.TaxLines[0].Label)
	require.Equal(t, 6.3, r.TaxLines[0].Amount)
	require.Equal(t, 6.3, r.TaxLines[1].Amount)
	require.Equal(t, -0.01, r.Rounding, "the paise the rounded lines overshoot by, stated openly")
	assertResponseFoots(t, r)
}

// An order priced by ComputeOrderPricing needs no rounding row at all.
func TestOrderToResponse_PricedOrderNeedsNoRoundingRow(t *testing.T) {
	p := ComputeOrderPricing(PricingInput{
		Subtotal: 240, PlatformFee: 240 * 4.99 / 100,
		TaxRate: 5, TaxName: "GST", Country: "IN", IntraState: true,
	})
	o := Order{
		Subtotal: p.Subtotal, PlatformFee: p.PlatformFee, Tax: p.Tax, Total: p.Total,
		TaxRate: 5, TaxName: "GST", DeliveryAddressCountry: "IN",
	}
	r := o.ToResponse()

	require.Zero(t, r.Rounding)
	require.Equal(t, 264.58, r.Total)
	assertResponseFoots(t, r)
}

// A tipped, discounted delivery order — every row present at once.
func TestOrderToResponse_FullOrderFoots(t *testing.T) {
	p := ComputeOrderPricing(PricingInput{
		Subtotal: 749.5, DeliveryFee: 39, PlatformFee: 749.5 * 4.99 / 100,
		Discount: 75, Tip: 30,
		TaxRate: 5, TaxName: "GST", Country: "IN", IntraState: true,
	})
	o := Order{
		Subtotal: p.Subtotal, DeliveryFee: p.DeliveryFee, PlatformFee: p.PlatformFee,
		Discount: p.Discount, Tip: p.Tip, Tax: p.Tax, Total: p.Total,
		TaxRate: 5, TaxName: "GST", DeliveryAddressCountry: "IN",
	}
	r := o.ToResponse()

	require.Zero(t, r.Rounding)
	assertResponseFoots(t, r)
}

// A chef in one state delivering to another is a single IGST row, and the chef
// view must show the customer the identical figures.
func TestOrderToChefResponse_MatchesCustomerBreakdown(t *testing.T) {
	o := Order{
		Subtotal: 500, PlatformFee: 24.95, Tax: 26.25, TaxRate: 5, TaxName: "GST",
		Total: 551.2, DeliveryAddressCountry: "IN", DeliveryAddressState: "Maharashtra",
	}
	o.Chef.State = "Odisha"

	customer, chef := o.ToResponse(), o.ToChefResponse()
	require.Len(t, customer.TaxLines, 1)
	require.Equal(t, TaxLineIGST, customer.TaxLines[0].Code)
	require.Equal(t, customer.TaxLines, chef.TaxLines)
	require.Equal(t, customer.Total, chef.Total)
	assertResponseFoots(t, customer)
	assertResponseFoots(t, chef)
}
