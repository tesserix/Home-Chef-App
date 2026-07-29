package services

// meal_plan_refund_conservation_test.go — THE money-safety invariant for refund policy v3.
//
//	A customer can never be refunded more than they actually paid.
//
// v3 made this worth proving rather than assuming: the base went from a fee/GST-EXCLUDED
// slice (always comfortably below the gross) to the FULL gross itself, so the arithmetic now
// sits exactly on the ceiling with no slack. Anything that rounds the wrong way, or any pair
// of paths that both pay out, shows up as a customer refunded more than they were charged.
//
// The plan's snapshotted Total is the ceiling: it is what the Razorpay advance actually
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

// The headline case, in the customer's own terms: pay ₹100 for the plan, get back at most ₹100.
func TestRefundConservation_NeverMoreThanPaid_SingleDay(t *testing.T) {
	plan := buildPlan([]float64{92.59}, 8, 0) // ≈ ₹100 all-in
	require.Equal(t, 100.0, plan.Total)

	refund := MealPlanRefundAmount(plan, &plan.Days[0], 100)
	require.Equal(t, 100.0, refund, "a fully-refunded single-day plan returns exactly what was paid")
	require.LessOrEqual(t, refund, plan.Total)
}

// Refunding every day of a multi-day plan returns the whole charge and not a paisa more.
func TestRefundConservation_WholePlan(t *testing.T) {
	plan := buildPlan([]float64{160, 160}, 10, 10)
	require.Equal(t, 372.0, plan.Total) // 320 food + 32 GST + 20 delivery

	require.Equal(t, plan.Total, sumFullRefunds(plan),
		"refunding every day must reconstruct the charge exactly — no shortfall, no surplus")
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

// THE rounding sweep. Each day's refund is independently Round2'd, so in principle N days
// could each round UP and the total overshoot the charge by a few paise. This walks a wide
// spread of realistic plan shapes — awkward prices, odd day counts, real tax rates — and
// asserts the sum never crosses the ceiling.
func TestRefundConservation_RoundingSweep(t *testing.T) {
	prices := []float64{49.99, 75, 99.5, 123.45, 160, 233.33, 7.77}
	taxes := []float64{0, 5, 8, 12, 18}
	deliveries := []float64{0, 2.99, 15, 33.33}
	dayCounts := []int{1, 2, 3, 5, 6, 7, 13, 30}

	worst := 0.0
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
						"OVER-REFUND: %d days @ %.2f, tax %.0f%%, delivery %.2f — refunded %.2f of %.2f charged",
						n, price, tax, del, sum, plan.Total)
					if d := plan.Total - sum; d > worst {
						worst = d
					}
				}
			}
		}
	}
	// The residue is the platform's, never the customer's — but it must stay at rounding
	// scale rather than being a systematic shortfall.
	require.Less(t, worst, 1.0, "worst-case under-refund across the sweep was ₹%.2f", worst)
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
				require.LessOrEqual(t, sumFullRefunds(plan), plan.Total,
					"OVER-REFUND: shape %v, tax %.0f%%, delivery %.2f", shape, tax, del)
			}
		}
	}
}

// A legacy plan with no snapshotted totals refunds the bare food price — it must not
// invent GST or delivery it has no record of having collected.
func TestRefundConservation_LegacyPlanCannotInventTax(t *testing.T) {
	plan := &models.MealPlan{Days: []models.MealPlanDay{{Price: 200}}}
	require.Equal(t, 200.0, MealPlanRefundAmount(plan, &plan.Days[0], 100))
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
