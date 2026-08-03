package models

// pricing_decimals_test.go — money is two decimal places, everywhere, always.
//
// Asserting that call-by-call is hopeless: one unrounded product anywhere and a
// receipt prints ₹11.976 or a JSON body carries 264.56999999999999. So this walks
// the SERIALISED output — the bytes clients actually receive — and fails on any
// number carrying a third decimal. A new field cannot be added without passing it.

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// assertTwoDecimals walks decoded JSON and reports the path of any number with
// more than two decimal places.
func assertTwoDecimals(t *testing.T, path string, v any) {
	t.Helper()
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			assertTwoDecimals(t, path+"."+k, child)
		}
	case []any:
		for i, child := range x {
			assertTwoDecimals(t, fmt.Sprintf("%s[%d]", path, i), child)
		}
	case float64:
		// Scale by 100 and compare against the nearest whole paise. The tolerance
		// absorbs IEEE-754 representation noise (0.1+0.2), not a real third decimal.
		scaled := x * 100
		require.InDelta(t, math.Round(scaled), scaled, 1e-6,
			"%s = %v carries more than two decimal places", path, x)
	}
}

func requireTwoDecimals(t *testing.T, label string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)

	// The serialised form is what a client parses, so check the text too: Go emits
	// the shortest round-tripping representation, and anything with a third
	// decimal here would reach the app exactly as written.
	var decoded any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assertTwoDecimals(t, label, decoded)
}

// The live Indian configuration: food 5% exclusive, platform fee 18% inclusive,
// chef delivery 5%, platform delivery 18%.
func liveIndiaRates(deliveryByPlatform bool) TaxRates {
	delivery := 5.0
	if deliveryByPlatform {
		delivery = 18
	}
	return TaxRates{
		Name: "GST", Food: 5, Service: 18, ServiceInclusive: true,
		Delivery: delivery, DeliveryByPlatform: deliveryByPlatform, Subscription: 18,
	}
}

func TestPricingSerialisesToTwoDecimals(t *testing.T) {
	// Deliberately awkward inputs: repeating fractions, odd paise, and amounts
	// whose 4.99% fee and 18%-inclusive split both land off a clean boundary.
	subtotals := []float64{500, 240, 333.33, 1, 0.05, 99.99, 1000.01, 777.77, 249.95}
	deliveries := []float64{0, 39, 45.5, 8.33}
	discounts := []float64{0, 50, 75.55}
	tips := []float64{0, 30, 7.77}

	for _, byPlatform := range []bool{false, true} {
		for _, sub := range subtotals {
			for _, del := range deliveries {
				for _, disc := range discounts {
					for _, tip := range tips {
						p := ComputeOrderPricing(PricingInput{
							Subtotal:    sub,
							DeliveryFee: del,
							PlatformFee: sub * 4.99 / 100,
							Discount:    disc,
							Tip:         tip,
							Rates:       liveIndiaRates(byPlatform),
							Country:     "IN",
							IntraState:  true,
						})
						label := fmt.Sprintf("sub=%v del=%v disc=%v tip=%v platform=%v", sub, del, disc, tip, byPlatform)
						requireTwoDecimals(t, label, p)

						// And the invariant, at every one of those inputs.
						sum := p.Subtotal + p.DeliveryFee + p.PlatformFee - p.Discount + p.Tip + p.Tax + p.Rounding
						require.InDelta(t, p.Total, RoundAmount(sum), 1e-9, "must foot: "+label)
					}
				}
			}
		}
	}
}

// Inter-state orders take the IGST branch, and a rate that halves unevenly must
// still print at two decimals.
func TestTaxLineRatesSerialiseToTwoDecimals(t *testing.T) {
	for _, rate := range []float64{5, 18, 13.5, 12.375, 7.77} {
		for _, intra := range []bool{true, false} {
			p := ComputeOrderPricing(PricingInput{
				Subtotal:   500,
				Rates:      TaxRates{Name: "GST", Food: rate, Service: rate, Delivery: rate},
				Country:    "IN",
				IntraState: intra,
			})
			requireTwoDecimals(t, fmt.Sprintf("rate=%v intra=%v", rate, intra), p)
			for _, l := range p.TaxLines {
				require.NotContains(t, l.Label, "e-", "a label must never print a rate in exponent form")
				require.LessOrEqual(t, len(strings.Split(fmt.Sprintf("%v", l.Rate), ".")), 2)
			}
		}
	}
}

// The DTO clients actually receive, over the same awkward inputs.
func TestOrderResponseSerialisesToTwoDecimals(t *testing.T) {
	for _, sub := range []float64{500, 333.33, 0.05, 777.77} {
		p := ComputeOrderPricing(PricingInput{
			Subtotal: sub, DeliveryFee: 39, PlatformFee: sub * 4.99 / 100, Tip: 7.77,
			Rates: liveIndiaRates(true), Country: "IN", IntraState: true,
		})
		o := Order{
			Subtotal: p.Subtotal, DeliveryFee: p.DeliveryFee, PlatformFee: p.PlatformFee,
			Tip: p.Tip, Tax: p.Tax, Total: p.Total,
			TaxFood: p.TaxFood, TaxService: p.TaxService, TaxDelivery: p.TaxDelivery,
			TaxRate: 5, TaxRateFood: 5, TaxRateService: 18, TaxRateDelivery: 18,
			TaxServiceInclusive: true, TaxName: "GST", DeliveryAddressCountry: "IN",
		}
		requireTwoDecimals(t, fmt.Sprintf("orderResponse sub=%v", sub), o.ToResponse())
	}
}

// A legacy order reconciled to its charged total must not leak a third decimal
// through the rounding row either.
func TestLegacyReconciliationSerialisesToTwoDecimals(t *testing.T) {
	for _, c := range []struct{ fee, tax, total float64 }{
		{11.976, 12.5988, 264.5748},
		{24.9500001, 26.2497, 551.1997},
		{4.9899999, 5.2394, 110.2294},
	} {
		p := PresentOrderPricing(PricingInput{
			Subtotal: 240, PlatformFee: c.fee, Rates: liveIndiaRates(false), Country: "IN", IntraState: true,
		}, TaxSnapshot{Total: c.tax}, c.total)
		requireTwoDecimals(t, fmt.Sprintf("legacy fee=%v", c.fee), p)

		sum := p.Subtotal + p.DeliveryFee + p.PlatformFee - p.Discount + p.Tip + p.Tax + p.Rounding
		require.InDelta(t, p.Total, RoundAmount(sum), 1e-9)
	}
}
