package services

import (
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

// meal_plan_earnings.go — the chef-facing money breakdown for a tiffin plan.
//
// A plan is priced, taxed and settled exactly like a run of à la carte orders,
// so it is broken down by the SAME engine (ComputeOrderEarnings), one day at a
// time, and the plan totals are the sum of those day rows. Nothing here derives
// a figure independently: the number the chef reads is the number the escrow
// holds (perDayNetPayout) and the settlement statement pays.

// MealPlanDayEarnings is one payable day's settlement row.
type MealPlanDayEarnings struct {
	DayID              uuid.UUID `json:"dayId"`
	Date               time.Time `json:"date"`
	Slot               string    `json:"slot"`
	DishName           string    `json:"dishName,omitempty"`
	Status             string    `json:"status"`
	FoodPrice          float64   `json:"foodPrice"`
	FoodGST            float64   `json:"foodGst"`
	Gross              float64   `json:"gross"`
	PlatformCommission float64   `json:"platformCommission"`
	TDS                float64   `json:"tds"`
	NetPayout          float64   `json:"netPayout"`
}

// MealPlanChefEarnings is what the chef is paid for a plan, plus the customer-side
// totals the same plan was billed at, so both halves reconcile on one screen.
type MealPlanChefEarnings struct {
	Currency       string  `json:"currency"`
	CommissionRate float64 `json:"commissionRate"`
	TDSRate        float64 `json:"tdsRate"`

	PayableDays  int `json:"payableDays"`
	ExcludedDays int `json:"excludedDays"`

	FoodSubtotal       float64 `json:"foodSubtotal"`
	FoodGST            float64 `json:"foodGst"`
	Gross              float64 `json:"gross"`
	PlatformCommission float64 `json:"platformCommission"`
	CGST               float64 `json:"cgst"`
	SGST               float64 `json:"sgst"`
	TDS                float64 `json:"tds"`
	NetPayout          float64 `json:"netPayout"`

	// What the customer was charged for the whole plan — the receipt side.
	CustomerSubtotal    float64 `json:"customerSubtotal"`
	CustomerPlatformFee float64 `json:"customerPlatformFee"`
	CustomerDelivery    float64 `json:"customerDelivery"`
	CustomerTaxFood     float64 `json:"customerTaxFood"`
	CustomerTaxService  float64 `json:"customerTaxService"`
	CustomerTaxDelivery float64 `json:"customerTaxDelivery"`
	CustomerTax         float64 `json:"customerTax"`
	CustomerTotal       float64 `json:"customerTotal"`
	// What the excluded days return to the customer, on the policy each day's
	// own status carries (make-whole for a declined day, skip refund for a skip).
	RefundedToCustomer float64 `json:"refundedToCustomer"`

	Days []MealPlanDayEarnings `json:"days"`
}

// dayIsPayable reports whether a day still owes the chef money. `requested` counts:
// a plan awaiting the chef's response must show the payout they are being offered,
// not a zeroed breakdown.
func dayIsPayable(s models.MealPlanDayStatus) bool {
	switch s {
	case models.MealPlanDayDeclined, models.MealPlanDaySkipped,
		models.MealPlanDayCancelled, models.MealPlanDayRefunded:
		return false
	default:
		return true
	}
}

// ComputeMealPlanChefEarnings breaks a plan down into the chef's settlement.
func ComputeMealPlanChefEarnings(plan *models.MealPlan, rate float64) MealPlanChefEarnings {
	if rate <= 0 || rate >= 1 {
		rate = DefaultCommissionRate
	}
	out := MealPlanChefEarnings{
		Currency:       EarningsCurrency,
		CommissionRate: rate,
		TDSRate:        RateTDS,
		Days:           []MealPlanDayEarnings{},
	}
	if plan == nil {
		return out
	}
	if plan.Currency != "" {
		out.Currency = plan.Currency
	}
	out.CustomerSubtotal = Round2(plan.Subtotal)
	out.CustomerPlatformFee = Round2(plan.PlatformFee)
	out.CustomerDelivery = Round2(planDeliveryTotal(plan))
	out.CustomerTaxFood = Round2(planFoodTax(plan))
	out.CustomerTaxService = Round2(plan.TaxService)
	out.CustomerTaxDelivery = Round2(plan.TaxDelivery)
	out.CustomerTax = Round2(plan.Tax)
	out.CustomerTotal = Round2(plan.Total)

	for i := range plan.Days {
		day := &plan.Days[i]
		if !dayIsPayable(day.Status) {
			out.ExcludedDays++
			out.RefundedToCustomer += refundForExcludedDay(plan, day, rate)
			continue
		}
		foodGST := perDayFoodGST(plan, day)
		e := ComputeOrderEarnings(EarningsInput{
			ItemRevenue:    day.Price,
			Tax:            foodGST,
			CommissionRate: rate,
		}, "")
		out.PayableDays++
		out.FoodSubtotal += e.ItemRevenue
		out.FoodGST += Round2(foodGST)
		out.Gross += e.Gross
		out.PlatformCommission += e.PlatformCommission
		out.CGST += e.CGST
		out.SGST += e.SGST
		out.TDS += e.TDS
		out.NetPayout += e.NetPayout
		out.Days = append(out.Days, MealPlanDayEarnings{
			DayID:              day.ID,
			Date:               day.Date,
			Slot:               string(day.Slot),
			DishName:           day.DishName,
			Status:             string(day.Status),
			FoodPrice:          e.ItemRevenue,
			FoodGST:            Round2(foodGST),
			Gross:              e.Gross,
			PlatformCommission: e.PlatformCommission,
			TDS:                e.TDS,
			NetPayout:          e.NetPayout,
		})
	}

	out.FoodSubtotal = Round2(out.FoodSubtotal)
	out.FoodGST = Round2(out.FoodGST)
	out.Gross = Round2(out.Gross)
	out.PlatformCommission = Round2(out.PlatformCommission)
	out.CGST = Round2(out.CGST)
	out.SGST = Round2(out.SGST)
	out.TDS = Round2(out.TDS)
	out.NetPayout = Round2(out.NetPayout)
	out.RefundedToCustomer = Round2(out.RefundedToCustomer)
	return out
}

// refundForExcludedDay is what the customer gets back for a day the chef is not
// paid for, on that day's own policy: a day the chef declined (or one refunded
// outright) is made whole; a customer-initiated skip/cancel forfeits GST and the
// platform fee.
func refundForExcludedDay(plan *models.MealPlan, day *models.MealPlanDay, rate float64) float64 {
	switch day.Status {
	case models.MealPlanDaySkipped, models.MealPlanDayCancelled:
		return perDaySkipRefund(plan, day, rate)
	default:
		return perDayGross(plan, day)
	}
}
