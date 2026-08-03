package services

// observed_orders_test.go — every figure observed on screen tonight, asserted against
// the production functions. Nothing here is my arithmetic; the numbers on the
// left came off the app, the gateway and the receipts, and the code has to
// reproduce them or this fails.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The live Indian configuration, as returned by the production lookup endpoint.
func liveRates() models.TaxRates {
	return models.TaxRates{
		Name: "GST", Food: 5, Service: 18, ServiceInclusive: true, Delivery: 5,
	}
}

func priceIt(subtotal, delivery float64) models.OrderPricing {
	return models.ComputeOrderPricing(models.PricingInput{
		Subtotal:    subtotal,
		DeliveryFee: delivery,
		PlatformFee: subtotal * 4.99 / 100, // PlatformFeePercent
		Rates:       liveRates(),
		Country:     "IN",
		IntraState:  true,
	})
}

// Pre-accept cancellation: 100% food, delivery refunded, platform fee retained.
func cancelIt(p models.OrderPricing) CancellationRefund {
	return ComputeCancellationRefund(CancellationOrder{
		FoodPaise:        ToPaise(p.Subtotal),
		DeliveryPaise:    ToPaise(p.DeliveryFee),
		PlatformFeePaise: ToPaise(p.PlatformFee),
		TaxPaise:         ToPaise(p.Tax),
		TaxFoodPaise:     ToPaise(p.TaxFood),
		TaxDeliveryPaise: ToPaise(p.TaxDelivery),
		TaxServicePaise:  ToPaise(p.TaxService),
	}, 100)
}

// Order 1 — HC26080315534435. Butter Chicken 320 + Chicken Korma 330 +
// Garden Salad Bowl 180 = 830, delivered by the chef.
func TestObserved_Order1(t *testing.T) {
	p := priceIt(830, 39.12)

	require.Equal(t, 830.00, p.Subtotal)
	require.Equal(t, 39.12, p.DeliveryFee)
	require.Equal(t, 35.10, p.PlatformFee, "screen: Platform fee 35.10")
	require.Equal(t, 49.78, p.Tax, "screen: Tax 49.78")
	require.Equal(t, 954.00, p.Total, "screen + Cashfree capture: 954.00")

	require.Equal(t, 41.50, p.TaxFood)
	require.Equal(t, 1.96, p.TaxDelivery)
	require.Equal(t, 6.32, p.TaxService)

	require.Len(t, p.TaxLines, 2)
	require.Equal(t, 24.89, p.TaxLines[0].Amount, "screen: CGST 24.89")
	require.Equal(t, 24.89, p.TaxLines[1].Amount, "screen: SGST 24.89")

	r := cancelIt(p)
	require.Equal(t, 912.58, FromPaise(r.Total), "screen: refunded 912.58")
	require.Equal(t, 41.42, FromPaise(r.PlatformKept), "screen: retained 41.42")
	require.Equal(t, ToPaise(p.Total), r.Total+r.VendorKept+r.PlatformKept, "must conserve")
}

// Order 2 — HC26080316029688. Salad 180 + Thali 240 + Chicken Biryani 340 +
// Mutton Biryani 440 = 1200, delivered by the chef, 248.60 wallet applied.
func TestObserved_Order2(t *testing.T) {
	p := priceIt(1200, 39.12)

	require.Equal(t, 1200.00, p.Subtotal)
	require.Equal(t, 39.12, p.DeliveryFee)
	require.Equal(t, 50.75, p.PlatformFee, "screen: Platform fee 50.75")
	require.Equal(t, 71.09, p.Tax)
	require.Equal(t, 1360.96, p.Total, "screen: Total 1360.96")

	require.Equal(t, 60.00, p.TaxFood)
	require.Equal(t, 1.96, p.TaxDelivery)
	require.Equal(t, 9.13, p.TaxService)

	require.Equal(t, 35.55, p.TaxLines[0].Amount, "screen: CGST 35.55")
	require.Equal(t, 35.54, p.TaxLines[1].Amount, "screen: SGST 35.54 — the odd paise goes here")

	// Checkout: "To pay" after wallet credit.
	require.Equal(t, 1112.36, models.RoundAmount(p.Total-248.60), "checkout: To pay 1112.36")
	// Checkout: "Fees & taxes are always paid separately".
	require.Equal(t, 121.84, models.RoundAmount(p.PlatformFee+p.Tax), "checkout: 121.84")

	r := cancelIt(p)
	require.Equal(t, 1301.08, FromPaise(r.Total), "screen: refunded 1301.08")
	require.Equal(t, 59.88, FromPaise(r.PlatformKept), "screen: retained 59.88")
	require.Equal(t, ToPaise(p.Total), r.Total+r.VendorKept+r.PlatformKept, "must conserve")

	// Retained is the whole all-in fee: its net plus its own GST.
	require.Equal(t, 59.88, models.RoundAmount(p.PlatformFee+p.TaxService))

	// Cashfree refunded 1063.42 to the card; the wallet took the rest.
	card := models.RoundAmount(FromPaise(r.Total) * 1112.36 / p.Total)
	require.Equal(t, 1063.42, card, "Cashfree: partial refund 1063.42")
	require.Equal(t, 237.66, models.RoundAmount(FromPaise(r.Total)-card), "wallet leg 237.66")
}

// The refund the OLD proportional-by-value formula would have produced — the
// figure that proves the splitter fix was not cosmetic.
func TestObserved_OldFormulaWouldHaveOverRefunded(t *testing.T) {
	for _, c := range []struct{ subtotal, expectedOld, expectedNew float64 }{
		{830, 916.96, 912.58},
		{1200, 1307.41, 1301.08},
	} {
		p := priceIt(c.subtotal, 39.12)

		legacy := ComputeCancellationRefund(CancellationOrder{
			FoodPaise:        ToPaise(p.Subtotal),
			DeliveryPaise:    ToPaise(p.DeliveryFee),
			PlatformFeePaise: ToPaise(p.PlatformFee),
			TaxPaise:         ToPaise(p.Tax),
			// no per-supply snapshot ⇒ the old proportional path
		}, 100)
		require.Equal(t, c.expectedOld, FromPaise(legacy.Total), "old proportional formula")
		require.Equal(t, c.expectedNew, FromPaise(cancelIt(p).Total), "per-supply formula")
	}
}
