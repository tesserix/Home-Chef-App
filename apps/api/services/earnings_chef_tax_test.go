package services

// earnings_chef_tax_test.go — D-02. A chef's gross must contain the GST on their
// FOOD and nothing else.
//
// order.Tax was passed straight into the earnings model, and order.Tax is the
// whole order's tax. Every chef was therefore credited the GST charged on the
// delivery fee and the platform fee as well as their own — ₹30.78 on one pending
// statement in the July test run. Rating the platform fee above the food makes it
// worse, not better.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A ₹500 order under the live configuration: food GST ₹25.00, delivery GST ₹1.95,
// platform-fee GST ₹3.81. Only the first is the chef's.
func TestChefAttributableTax_ExcludesFeeAndDeliveryGST(t *testing.T) {
	o := &models.Order{
		Subtotal: 500, DeliveryFee: 39, PlatformFee: 21.14,
		Tax: 30.76, TaxFood: 25.00, TaxDelivery: 1.95, TaxService: 3.81,
	}
	require.Equal(t, 25.00, ChefAttributableTax(o),
		"the chef's gross carries the food GST only — the rest is the platform's output tax")
	require.NotEqual(t, o.Tax, ChefAttributableTax(o),
		"crediting the whole order tax is exactly the D-02 leak")
}

// An order placed before tax was split has no snapshot, and its statement may
// already be settled — it keeps the figure it was reconciled against rather than
// being restated.
func TestChefAttributableTax_LegacyOrderKeepsItsSettledFigure(t *testing.T) {
	o := &models.Order{Subtotal: 500, Tax: 26.25}
	require.Equal(t, 26.25, ChefAttributableTax(o))
}

// ChefTaxOf is the same rule for the SQL projections. Every earnings path must
// agree, because the statement, the FY statement, the TDS certificate and the
// payout queue are documented to read one identical figure.
func TestChefTaxOf_MatchesTheModelRule(t *testing.T) {
	cases := []struct {
		name                             string
		tax, taxFood, taxService, expect float64
	}{
		{"per-supply snapshot", 30.76, 25.00, 3.81, 25.00},
		{"legacy, no snapshot", 26.25, 0, 0, 26.25},
		{"zero-rated order", 0, 0, 0, 0},
		{"food fully discounted, fee still taxed", 3.81, 0, 3.81, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.expect, ChefTaxOf(c.tax, c.taxFood, c.taxService))
		})
	}
}

// The whole point of the fix: a chef must not gain from the platform fee moving
// to a higher rate. Their gross is identical either way.
func TestChefGrossIsUnaffectedByThePlatformFeeRate(t *testing.T) {
	atFive := &models.Order{Subtotal: 500, Tax: 26.25, TaxFood: 25.00, TaxService: 1.25}
	atEighteen := &models.Order{Subtotal: 500, Tax: 28.81, TaxFood: 25.00, TaxService: 3.81}

	require.Equal(t, ChefAttributableTax(atFive), ChefAttributableTax(atEighteen),
		"re-rating the platform's own supply cannot change what the chef earns")
}
