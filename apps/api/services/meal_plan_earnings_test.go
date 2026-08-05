package services

import (
	"math"
	"testing"

	"github.com/homechef/api/models"
)

func planWithDays(prices []float64, statuses []models.MealPlanDayStatus) *models.MealPlan {
	subtotal := 0.0
	days := make([]models.MealPlanDay, len(prices))
	for i, p := range prices {
		subtotal += p
		days[i] = models.MealPlanDay{Price: p, Status: statuses[i]}
	}
	// Booking-time snapshot: 5% food GST, 4% platform fee (+18% GST on it), ₹25/day delivery (+18%).
	foodTax := Round2(subtotal * 0.05)
	fee := Round2(subtotal * 0.04)
	feeTax := Round2(fee * 0.18)
	delivery := 25.0 * float64(len(prices))
	deliveryTax := Round2(delivery * 0.18)
	return &models.MealPlan{
		Subtotal:    subtotal,
		PlatformFee: fee,
		TaxFood:     foodTax,
		TaxService:  feeTax,
		TaxDelivery: deliveryTax,
		Tax:         Round2(foodTax + feeTax + deliveryTax),
		Total:       Round2(subtotal + fee + foodTax + feeTax + delivery + deliveryTax),
		Currency:    "INR",
		Days:        days,
	}
}

func allStatuses(n int, s models.MealPlanDayStatus) []models.MealPlanDayStatus {
	out := make([]models.MealPlanDayStatus, n)
	for i := range out {
		out[i] = s
	}
	return out
}

// The escrow's per-day hold and the earnings engine must produce the identical
// rupee figure — a chef reading their plan breakdown is reading the money that
// actually moves, to the paise.
func TestPerDayNetPayoutMatchesOrderEarnings(t *testing.T) {
	prices := []float64{149.99, 87.33, 250, 33.37, 199.95, 1.01, 66.66}
	plan := planWithDays(prices, allStatuses(len(prices), models.MealPlanDayConfirmed))
	for i := range plan.Days {
		day := &plan.Days[i]
		got := perDayNetPayout(plan, day, DefaultCommissionRate)
		want := ComputeOrderEarnings(EarningsInput{
			ItemRevenue:    day.Price,
			Tax:            perDayFoodGST(plan, day),
			CommissionRate: DefaultCommissionRate,
		}, "").NetPayout
		if got != want {
			t.Errorf("day %d (₹%.2f): escrow hold %.4f, earnings engine %.4f", i, day.Price, got, want)
		}
	}
}

func TestComputeMealPlanChefEarnings_SumsPayableDaysOnly(t *testing.T) {
	prices := []float64{200, 150, 175}
	plan := planWithDays(prices, []models.MealPlanDayStatus{
		models.MealPlanDayDelivered,
		models.MealPlanDayConfirmed,
		models.MealPlanDayDeclined, // chef never cooks it — must not be paid for
	})

	got := ComputeMealPlanChefEarnings(plan, DefaultCommissionRate)

	if got.PayableDays != 2 || got.ExcludedDays != 1 {
		t.Fatalf("day split = %d payable / %d excluded, want 2/1", got.PayableDays, got.ExcludedDays)
	}
	if got.FoodSubtotal != 350 {
		t.Errorf("FoodSubtotal = %.2f, want 350", got.FoodSubtotal)
	}
	// Totals must equal the sum of the per-day rows the chef sees, not a
	// separately-derived figure that could disagree with them.
	var netFromRows float64
	for _, d := range got.Days {
		netFromRows += d.NetPayout
	}
	if math.Abs(netFromRows-got.NetPayout) > 0.001 {
		t.Errorf("NetPayout %.2f != sum of day rows %.2f", got.NetPayout, netFromRows)
	}
	if len(got.Days) != 2 {
		t.Errorf("got %d day rows, want 2 (payable only)", len(got.Days))
	}
}

// The chef's net must never quietly include money that is not theirs: delivery
// and the platform fee are the platform's, and the identity gross − commission
// − TDS must hold on the displayed (rounded) figures.
func TestComputeMealPlanChefEarnings_IdentityHolds(t *testing.T) {
	prices := []float64{149.99, 87.33, 250.5}
	plan := planWithDays(prices, allStatuses(3, models.MealPlanDayConfirmed))

	got := ComputeMealPlanChefEarnings(plan, DefaultCommissionRate)

	if diff := math.Abs(got.Gross - got.PlatformCommission - got.TDS - got.NetPayout); diff > 0.011 {
		t.Errorf("gross %.2f − commission %.2f − tds %.2f != net %.2f (off by %.4f)",
			got.Gross, got.PlatformCommission, got.TDS, got.NetPayout, diff)
	}
	if got.Gross >= plan.Total {
		t.Errorf("chef gross %.2f must exclude delivery and the platform fee (plan total %.2f)", got.Gross, plan.Total)
	}
	if got.CustomerTotal != plan.Total {
		t.Errorf("CustomerTotal = %.2f, want the plan total %.2f", got.CustomerTotal, plan.Total)
	}
}

// A plan awaiting the chef's response has every day still `requested`: the chef
// must see the projected payout, not a zeroed breakdown.
func TestComputeMealPlanChefEarnings_ProjectsRequestedDays(t *testing.T) {
	plan := planWithDays([]float64{120, 120}, allStatuses(2, models.MealPlanDayRequested))

	got := ComputeMealPlanChefEarnings(plan, DefaultCommissionRate)

	if got.PayableDays != 2 || got.NetPayout <= 0 {
		t.Fatalf("requested days must project: %d payable, net %.2f", got.PayableDays, got.NetPayout)
	}
}

func TestComputeMealPlanChefEarnings_EmptyPlan(t *testing.T) {
	got := ComputeMealPlanChefEarnings(&models.MealPlan{Currency: "INR"}, DefaultCommissionRate)
	if got.NetPayout != 0 || got.PayableDays != 0 || len(got.Days) != 0 {
		t.Errorf("empty plan produced %+v", got)
	}
	if got.Currency != "INR" {
		t.Errorf("Currency = %q, want INR", got.Currency)
	}
}

func TestComputeMealPlanChefEarnings_NilPlan(t *testing.T) {
	got := ComputeMealPlanChefEarnings(nil, DefaultCommissionRate)
	if got.NetPayout != 0 || got.Currency != EarningsCurrency {
		t.Errorf("nil plan produced %+v", got)
	}
}
