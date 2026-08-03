package services

// meal_plan_tax_parity_test.go — a meal plan's tax must split per supply exactly
// as an order's does, and the chef must be credited the FOOD GST either way.
//
// Meal plans priced themselves: their own fee, their own rate (policy.TaxPercent,
// defaulting to 8% while orders were taxed at the 5% in tax_rates), and no tax at
// all on the platform fee or the delivery. Two pricing models for one product,
// with the platform silently absorbing the difference.
//
// The risk in fixing it is perDayFoodGST: it is the single basis for the chef
// day-transfer, the TDS reporting, the credit note and the spawned day order's
// tax. Widening plan.Tax without splitting it would have handed every chef the
// GST on the platform fee and the delivery — the meal-plan half of D-02.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A 7-day plan at ₹100/day, priced the way an order is: food 5% exclusive,
// platform fee 18% quoted all-in, chef-carried delivery 5%.
func splitPlan() *models.MealPlan {
	p := models.ComputeOrderPricing(models.PricingInput{
		Subtotal: 700, DeliveryFee: 20.93, PlatformFee: 700 * 4.99 / 100,
		Rates:   models.TaxRates{Name: "GST", Food: 5, Service: 18, ServiceInclusive: true, Delivery: 5},
		Country: "IN", IntraState: true,
	})
	return &models.MealPlan{
		Subtotal: p.Subtotal, PlatformFee: p.PlatformFee, Total: p.Total,
		Tax: p.Tax, TaxFood: p.TaxFood, TaxService: p.TaxService, TaxDelivery: p.TaxDelivery,
		Days: make([]models.MealPlanDay, 7),
	}
}

// A plan booked before the split: Tax is food GST alone and there is no snapshot.
func legacyPlan() *models.MealPlan {
	return &models.MealPlan{
		Subtotal: 700, PlatformFee: 34.93, Tax: 35.00, Total: 790.86,
		Days: make([]models.MealPlanDay, 7),
	}
}

func TestMealPlanNowTaxesTheFeeAndDelivery(t *testing.T) {
	p := splitPlan()

	require.Greater(t, p.TaxService, 0.0,
		"a plan's platform fee used to carry no tax at all — the platform absorbed it")
	require.Greater(t, p.TaxDelivery, 0.0, "nor did its delivery")
	require.InDelta(t, p.Tax, p.TaxFood+p.TaxService+p.TaxDelivery, 1e-9,
		"the parts must sum to the whole")
	require.InDelta(t, 35.00, p.TaxFood, 1e-9, "700 of food at 5%")
}

// The chef's basis excludes everything that is not their food.
func TestPerDayFoodGST_ExcludesFeeAndDeliveryTax(t *testing.T) {
	p := splitPlan()
	day := &models.MealPlanDay{Price: 100}

	require.InDelta(t, 5.00, perDayFoodGST(p, day), 1e-9,
		"one day of 100 food at 5% — not a share of the fee's or delivery's GST")
	require.Greater(t, perDayTax(p, day), perDayFoodGST(p, day),
		"the customer paid more tax for that day than the chef is credited")
}

// A legacy plan's Tax WAS food GST, and its transfers are already reconciled
// against it — so it must keep exactly the figure it had.
func TestPerDayFoodGST_LegacyPlanKeepsItsSettledBasis(t *testing.T) {
	p := legacyPlan()
	day := &models.MealPlanDay{Price: 100}

	require.InDelta(t, 5.00, perDayFoodGST(p, day), 1e-9)
	require.InDelta(t, perDayFoodGST(p, day), perDayTax(p, day), 1e-9,
		"with no split there is nothing to distinguish — both read plan.Tax")
}

// A make-whole refund returns every rupee the customer paid for that day,
// including the GST on the fee and the delivery they never received.
func TestPerDayGross_ReturnsTheWholeTaxNotJustTheChefsShare(t *testing.T) {
	p := splitPlan()
	day := &models.MealPlanDay{Price: 100}

	gross := perDayGross(p, day)
	require.Greater(t, gross, 100+perDayFoodGST(p, day),
		"a make-whole refund is not just food plus the chef's tax")

	// Conservation: seven equal days must return the whole plan.
	var total float64
	for range p.Days {
		total += perDayGross(p, day)
	}
	require.InDelta(t, p.Total, Round2(total), 0.07,
		"the per-day refunds must sum back to what the customer paid, within per-day rounding")
}

// planDeliveryTotal derives delivery as Total − Subtotal − PlatformFee − Tax, so
// widening Tax to the whole tax must not make it swallow the delivery.
func TestPlanDeliveryTotal_SurvivesTheWiderTax(t *testing.T) {
	p := splitPlan()
	require.InDelta(t, 20.93, planDeliveryTotal(p), 0.01,
		"delivery is derived, so every component added to Total must also be subtracted here")
}
