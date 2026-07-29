package services

// meal_plan_refund_partial_test.go — a refund covers ONLY the cancelled days.
//
// Cancel 2 days of a 7-day plan and the customer gets back those 2 days and nothing more: the
// food for them, their share of the GST, and their share of the delivery. The other 5 are
// untouched — still cooked, still charged, still payable to the chef.
//
// This is the invariant that makes per-day escrow work at all, and v3 put it under new
// pressure: the base now carries GST and delivery, which are charged across the WHOLE plan and
// have to be apportioned. Get the apportionment wrong and a 2-day cancellation either hands
// back 7 days' tax or none of it.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// sevenDayPlan: 7 days at ₹100, 8% GST, ₹10 delivery per day.
//
//	food     700.00
//	GST       56.00  (8% of 700)
//	delivery  70.00  (7 × 10)
//	TOTAL    826.00  → ₹118.00 per day
func sevenDayPlan() *models.MealPlan {
	days := make([]models.MealPlanDay, 7)
	for i := range days {
		days[i] = models.MealPlanDay{ID: uuid.New(), Price: 100, CommissionRate: 0.15}
	}
	return &models.MealPlan{Subtotal: 700, Tax: 56, Total: 826, Days: days}
}

// THE case the policy is about: 2 of 7 cancelled → 2 days' money back, not the plan.
func TestPartialCancellation_RefundsOnlyTheCancelledDays(t *testing.T) {
	plan := sevenDayPlan()

	// Day 3 and day 6 are cancelled at 100%.
	refund := MealPlanRefundAmount(plan, &plan.Days[2], 100) +
		MealPlanRefundAmount(plan, &plan.Days[5], 100)

	require.Equal(t, 236.0, Round2(refund), "2 days × ₹118 — food, their GST share, their delivery share")
	require.Equal(t, Round2(plan.Total*2/7), Round2(refund), "exactly two sevenths of the charge")
	require.Less(t, refund, plan.Total, "cancelling part of a plan never refunds the whole plan")

	// The 5 untouched days are still worth their full value to the chef and the platform.
	require.Equal(t, 590.0, Round2(plan.Total-refund), "5 days × ₹118 stays with the plan")
}

// Each cancelled day carries its own slice of every charge line — nothing is refunded twice
// and nothing is left behind.
func TestPartialCancellation_PerDayComposition(t *testing.T) {
	plan := sevenDayPlan()
	day := &plan.Days[0]

	require.Equal(t, 118.0, MealPlanRefundAmount(plan, day, 100), "100 food + 8 GST + 10 delivery")
	require.Equal(t, 8.0, MealPlanRefundGSTComponent(plan, day, 100), "one day's GST, not the plan's 56")
	require.Equal(t, 8.0, Round2(perDayFoodGST(plan, day)))
	require.Equal(t, 10.0, Round2(perDayDeliveryRefund(plan, day)), "one day's delivery, not the plan's 70")
}

// Cancelling every day one at a time reconstructs the whole charge — the apportionment is
// complete, with nothing double-counted and nothing stranded.
func TestPartialCancellation_AllDaysSumToTheCharge(t *testing.T) {
	plan := sevenDayPlan()
	var total, gst float64
	for i := range plan.Days {
		total += MealPlanRefundAmount(plan, &plan.Days[i], 100)
		gst += MealPlanRefundGSTComponent(plan, &plan.Days[i], 100)
	}
	require.Equal(t, plan.Total, Round2(total))
	require.Equal(t, plan.Tax, Round2(gst))
}

// A partial refund on a cancelled day is a slice of THAT DAY, never of the plan.
func TestPartialCancellation_ChefFloorAppliesPerDay(t *testing.T) {
	plan := sevenDayPlan()
	day := &plan.Days[0]

	require.Equal(t, 88.5, MealPlanRefundAmount(plan, day, 75), "75% of one day's ₹118")
	require.Equal(t, 59.0, MealPlanRefundAmount(plan, day, 50))
	require.Less(t, MealPlanRefundAmount(plan, day, 100), plan.Total/2,
		"even a full single-day refund is a small fraction of a 7-day plan")
}

// Days already served are excluded from a whole-plan cancel, so cancelling midway refunds only
// what is still owed. v2DayTerminal is the filter CancelMealPlan applies.
func TestPartialCancellation_ServedDaysAreNotRefundable(t *testing.T) {
	plan := sevenDayPlan()
	// Four days already delivered; the customer cancels the rest.
	for i := range 4 {
		plan.Days[i].Status = models.MealPlanDayDelivered
	}
	var refundable float64
	for i := range plan.Days {
		if plan.Days[i].Status == models.MealPlanDayDelivered {
			continue
		}
		refundable += MealPlanRefundAmount(plan, &plan.Days[i], 100)
	}
	require.Equal(t, 354.0, Round2(refundable), "only the 3 unserved days — meals already eaten are paid for")
	require.Equal(t, 472.0, Round2(plan.Total-refundable), "the 4 delivered days stay charged")
}

// Unequal day prices: a cancelled expensive day returns more than a cancelled cheap one, and
// the pair still adds up to less than the plan.
func TestPartialCancellation_UnequalPrices(t *testing.T) {
	plan := &models.MealPlan{
		Subtotal: 600, Tax: 48, Total: 678, // 3 days, 8% GST, 30 delivery
		Days: []models.MealPlanDay{{Price: 100}, {Price: 200}, {Price: 300}},
	}
	cheap := MealPlanRefundAmount(plan, &plan.Days[0], 100)
	dear := MealPlanRefundAmount(plan, &plan.Days[2], 100)

	require.Equal(t, 113.0, cheap, "100 food + 8 GST + 5 delivery (its 1/6 share)")
	require.Equal(t, 339.0, dear, "300 food + 24 GST + 15 delivery (its 1/2 share)")
	require.Equal(t, 3.0, Round2(dear/cheap), "shares track the day's price, not a flat split")
	require.Less(t, Round2(cheap+dear), plan.Total)
}
