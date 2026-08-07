package services

// meal_plan_refund_conservation_test.go — THE money-safety invariant for refund policy v3.
//
//	A customer can never be refunded more than they actually paid.
//
// v3 made this worth proving rather than assuming: the base picked up the GST, which is
// charged across the whole plan and has to be apportioned per day, and each day's share is
// rounded independently. Anything that rounds the wrong way, or any pair of paths that both
// pay out, shows up here as a customer refunded more than their share.
//
// TWO ceilings are asserted throughout. The HARD one is plan.Total — what the advance actually
// captured. The TIGHT one is the exact unrounded sum of the per-day shares; since the platform
// retains its commission the hard ceiling now has slack, so only the tight one still catches
// rounding drift.
//
// The plan's snapshotted Total is the ceiling: it is what the retired gateway advance actually
// captured (VerifyMealPlanAdvance binds the payment to it), so it is the real amount of money
// that entered escrow for this plan.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// buildPlan mirrors what booking does: food subtotal, GST + per-day delivery on top
// (MealPlanFeeTotals), Total = the sum — the exact figure the customer is charged.
func buildPlan(dayPrices []float64, taxPercent, perDayDelivery float64) *models.MealPlan {
	var subtotal float64
	days := make([]models.MealPlanDay, 0, len(dayPrices))
	for _, p := range dayPrices {
		subtotal += p
		days = append(days, models.MealPlanDay{Price: p, CommissionRate: 0.15})
	}
	tax := Round2(subtotal * taxPercent / 100)
	delivery := Round2(perDayDelivery * float64(len(dayPrices)))
	return &models.MealPlan{
		Subtotal: Round2(subtotal), Tax: tax, Total: Round2(subtotal + tax + delivery), Days: days,
	}
}

// sumFullRefunds is the worst case: EVERY day of the plan refunded at 100%.
func sumFullRefunds(plan *models.MealPlan) float64 {
	var total float64
	for i := range plan.Days {
		total += MealPlanRefundAmount(plan, &plan.Days[i], 100)
	}
	return Round2(total)
}

// The headline case, in the customer's own terms: pay ₹100 for the plan, never get back
// more than ₹100. Under v3 the platform also retains its commission, so the refund lands
// strictly below the charge.
func TestRefundConservation_NeverMoreThanPaid_SingleDay(t *testing.T) {
	plan := buildPlan([]float64{92.59}, 8, 0) // ≈ ₹100 all-in
	require.Equal(t, 100.0, plan.Total)

	refund := MealPlanRefundAmount(plan, &plan.Days[0], 100)
	require.Less(t, refund, plan.Total, "the retained commission keeps the refund below the charge")
	require.Equal(t, Round2(92.59*(1-0.15)+7.41), Round2(refund), "food-net + GST, no delivery on this plan")
}

// Refunding every day of a multi-day plan returns the charge LESS the platform's commission —
// the whole of what is owed back, and not a paisa more.
func TestRefundConservation_WholePlan(t *testing.T) {
	plan := buildPlan([]float64{160, 160}, 10, 10)
	require.Equal(t, 372.0, plan.Total) // 320 food + 32 GST + 20 delivery

	commission := Round2(plan.Subtotal * 0.15) // 48
	require.Equal(t, Round2(plan.Total-commission), sumFullRefunds(plan),
		"every day refunded reconstructs the charge minus the retained commission")
	require.Less(t, sumFullRefunds(plan), plan.Total)
}

// Unequal day prices: the GST and delivery split is proportional, so the identity holds.
func TestRefundConservation_UnequalDayPrices(t *testing.T) {
	plan := buildPlan([]float64{120, 200, 80}, 8, 15)
	require.LessOrEqual(t, sumFullRefunds(plan), plan.Total)

	// The expensive day carries proportionally more GST and delivery than the cheap one.
	dear := MealPlanRefundAmount(plan, &plan.Days[1], 100)
	cheap := MealPlanRefundAmount(plan, &plan.Days[2], 100)
	require.Greater(t, dear, cheap)
}

// A partial refund is always strictly less than the day's full charge.
func TestRefundConservation_PartialNeverExceedsFull(t *testing.T) {
	plan := buildPlan([]float64{160, 160}, 10, 10)
	day := &plan.Days[0]
	full := MealPlanRefundAmount(plan, day, 100)
	for _, pct := range []int{0, 25, 50, 75, 99} {
		require.Less(t, MealPlanRefundAmount(plan, day, pct), full, "percent %d", pct)
	}
	// And an out-of-range percent is clamped, never extrapolated past the ceiling.
	require.Equal(t, full, MealPlanRefundAmount(plan, day, 1000))
}

// THE rounding sweep. Each day's refund is rounded independently, so in principle N days could
// each round UP and the total overshoot. This walks a wide spread of realistic plan shapes —
// awkward prices, odd day counts, real tax rates — and asserts two ceilings: the hard one (the
// amount actually captured) and the exact one (the unrounded sum of the per-day shares, which
// is what catches rounding drift now that the retained commission gives the hard ceiling
// plenty of slack).
func TestRefundConservation_RoundingSweep(t *testing.T) {
	prices := []float64{49.99, 75, 99.5, 123.45, 160, 233.33, 7.77}
	taxes := []float64{0, 5, 8, 12, 18}
	deliveries := []float64{0, 2.99, 15, 33.33}
	dayCounts := []int{1, 2, 3, 5, 6, 7, 13, 30}

	worstDrift := 0.0
	for _, price := range prices {
		for _, tax := range taxes {
			for _, del := range deliveries {
				for _, n := range dayCounts {
					dayPrices := make([]float64, n)
					for i := range dayPrices {
						dayPrices[i] = price
					}
					plan := buildPlan(dayPrices, tax, del)
					sum := sumFullRefunds(plan)

					require.LessOrEqual(t, sum, plan.Total,
						"OVER-REFUND past the capture: %d days @ %.2f, tax %.0f%%, delivery %.2f — refunded %.2f of %.2f charged",
						n, price, tax, del, sum, plan.Total)

					exact := exactFullRefundSum(plan)
					require.LessOrEqual(t, sum, exact+1e-9,
						"OVER-REFUND past the exact share: %d days @ %.2f, tax %.0f%%, delivery %.2f — refunded %.2f, exact %.4f",
						n, price, tax, del, sum, exact)
					if d := exact - sum; d > worstDrift {
						worstDrift = d
					}
				}
			}
		}
	}
	// Truncation keeps at most a paise per day, retained by the platform. Anything larger
	// would be a systematic shortfall rather than rounding.
	require.Less(t, worstDrift, 1.0, "worst-case rounding shortfall across the sweep was ₹%.2f", worstDrift)
}

// exactFullRefundSum is the sum of the UNROUNDED per-day shares — the figure the rounded
// refunds must never exceed.
func exactFullRefundSum(plan *models.MealPlan) float64 {
	var total float64
	for i := range plan.Days {
		d := &plan.Days[i]
		rate := d.CommissionRate
		if rate <= 0 || rate >= 1 {
			rate = DefaultCommissionRate
		}
		total += d.Price - rate*d.Price + perDayFoodGST(plan, d) + perDayDeliveryRefund(plan, d)
	}
	return total
}

// Mixed day prices under rounding pressure — the proportional split is where drift would
// accumulate if it were going to.
func TestRefundConservation_RoundingSweep_MixedPrices(t *testing.T) {
	shapes := [][]float64{
		{33.33, 33.33, 33.34},
		{10, 20, 30, 40, 50, 60, 70},
		{0.01, 999.99},
		{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		{149.95, 87.5, 62.55},
	}
	for _, tax := range []float64{5, 8, 18} {
		for _, del := range []float64{0, 2.99, 40} {
			for _, shape := range shapes {
				plan := buildPlan(shape, tax, del)
				sum := sumFullRefunds(plan)
				require.LessOrEqual(t, sum, plan.Total,
					"OVER-REFUND past the capture: shape %v, tax %.0f%%, delivery %.2f", shape, tax, del)
				require.LessOrEqual(t, sum, exactFullRefundSum(plan)+1e-9,
					"OVER-REFUND past the exact share: shape %v, tax %.0f%%, delivery %.2f", shape, tax, del)
			}
		}
	}
}

// A legacy plan with no snapshotted totals refunds food net of the default commission — it
// must not invent GST or delivery it has no record of having collected.
func TestRefundConservation_LegacyPlanCannotInventTax(t *testing.T) {
	plan := &models.MealPlan{Days: []models.MealPlanDay{{Price: 200}}}
	require.Equal(t, 188.0, MealPlanRefundAmount(plan, &plan.Days[0], 100), "200 − 6% default commission")
	require.Equal(t, 0.0, MealPlanRefundGSTComponent(plan, &plan.Days[0], 100))
}

// A day whose price exceeds the whole plan subtotal is corrupt data; the share formula would
// otherwise hand it MORE than the plan's total. Pins that it cannot exceed the ceiling.
func TestRefundConservation_CorruptDayPriceCannotExceedPlanTotal(t *testing.T) {
	// Subtotal says 100 but the day claims 500 — a mismatch no legitimate flow produces.
	plan := &models.MealPlan{Subtotal: 100, Tax: 8, Total: 113,
		Days: []models.MealPlanDay{{Price: 500}}}
	refund := MealPlanRefundAmount(plan, &plan.Days[0], 100)
	require.LessOrEqual(t, refund, plan.Total,
		"a day priced above the plan subtotal must not be refunded more than the plan was charged")
}
