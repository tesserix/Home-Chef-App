package handlers

// pricing_simulator.go — the admin's pricing sandbox.
//
// It answers "what happens to a ₹500 order?" for every fulfilment mode and every
// refund tier at once, so a rate change can be seen before it is saved.
//
// The single rule this file lives by: it calls models.ComputeOrderPricing and
// services.ComputeCancellationRefund — the SAME functions that charge and refund
// real money. A simulator that reimplements the arithmetic is worse than none,
// because it tells you what someone once believed the system does.

import (
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type PricingSimulatorHandler struct{}

func NewPricingSimulatorHandler() *PricingSimulatorHandler { return &PricingSimulatorHandler{} }

type simulateRequest struct {
	Subtotal float64 `json:"subtotal"`
	// Optional overrides; each falls back to the live configuration so the default
	// simulation is of the platform as it stands right now.
	DeliveryFee        *float64 `json:"deliveryFee"`
	PlatformFeePercent *float64 `json:"platformFeePercent"`
	Discount           float64  `json:"discount"`
	Tip                float64  `json:"tip"`
	Country            string   `json:"country"`
	Region             string   `json:"region"`
	// IntraState drives CGST+SGST vs IGST. Defaults true — a home kitchen serving
	// its own city, which is every order today.
	IntraState *bool `json:"intraState"`
	// GatewayFeePercent is the payment gateway's MDR. It is NOT returned on a
	// refund, so it is what makes a cancellation cost real money.
	GatewayFeePercent *float64 `json:"gatewayFeePercent"`
	// RefundTiers to model. Defaults to the ones the cancellation policy uses.
	RefundTiers []int `json:"refundTiers"`
}

type simulatedScenario struct {
	Fulfillment string              `json:"fulfillment"`
	Label       string              `json:"label"`
	Pricing     models.OrderPricing `json:"pricing"`
	// Validations are the invariants checked on THIS scenario, so the admin sees
	// that the numbers hold rather than being asked to take it on trust.
	Validations []validationCheck `json:"validations"`
	Rates       models.TaxRates   `json:"rates"`
	Gateway     gatewayCost       `json:"gateway"`
	Refunds     []simulatedRefund `json:"refunds"`
}

type gatewayCost struct {
	Percent  float64 `json:"percent"`
	Fee      float64 `json:"fee"`
	TaxOnFee float64 `json:"taxOnFee"`
	Gross    float64 `json:"gross"`
	// Real is what it costs after reclaiming the tax on it, which is only possible
	// where the platform has a standard-rated output supply to set it against.
	Real float64 `json:"real"`
}

type simulatedRefund struct {
	Label          string  `json:"label"`
	FoodRefundPct  int     `json:"foodRefundPct"`
	Dispatched     bool    `json:"dispatched"`
	CustomerGets   float64 `json:"customerGets"`
	FoodRefund     float64 `json:"foodRefund"`
	DeliveryRefund float64 `json:"deliveryRefund"`
	TaxRefund      float64 `json:"taxRefund"`
	ChefKeeps      float64 `json:"chefKeeps"`
	PlatformKeeps  float64 `json:"platformKeeps"`
	// CreditNote is the tax to reverse under §34 — exactly the tax refunded.
	CreditNote float64 `json:"creditNote"`
	// PlatformMargin is what the platform is left with once the gateway fee (sunk
	// at capture, never returned) is taken off what it kept.
	PlatformMargin float64 `json:"platformMargin"`
	// TaxRefundFood / TaxRefundDelivery break the credit note down by supply. The
	// platform fee's tax never appears: the fee is kept, that supply happened, and
	// its tax cannot be credit-noted.
	TaxRefundFood     float64 `json:"taxRefundFood"`
	TaxRefundDelivery float64 `json:"taxRefundDelivery"`
	TaxKeptOnFee      float64 `json:"taxKeptOnFee"`
	// Conserves is the invariant: nothing appears or disappears.
	Conserves bool `json:"conserves"`
	// Warnings are conditions worth an operator's attention — a refund that costs
	// the platform money, or tax handed back that cannot be reclaimed.
	Warnings []string `json:"warnings,omitempty"`
}

// SimulatePricing models an order end to end at the CURRENT configuration.
//
// POST /admin/pricing/simulate
func (h *PricingSimulatorHandler) SimulatePricing(c *gin.Context) {
	var req simulateRequest
	_ = c.ShouldBindJSON(&req)
	if req.Subtotal <= 0 {
		req.Subtotal = 500 // a round number to reason about
	}
	country := req.Country
	if country == "" {
		country = "IN"
	}
	intraState := true
	if req.IntraState != nil {
		intraState = *req.IntraState
	}
	gatewayPct := 1.95
	if req.GatewayFeePercent != nil {
		gatewayPct = *req.GatewayFeePercent
	}
	tiers := req.RefundTiers
	if len(tiers) == 0 {
		tiers = []int{100, 75, 50, 0}
	}

	policy := services.GetPlatformPolicy()
	feePct := policy.PlatformFeePercent
	if req.PlatformFeePercent != nil {
		feePct = *req.PlatformFeePercent
	}
	deliveryFee := policy.BaseDeliveryFee
	if req.DeliveryFee != nil {
		deliveryFee = *req.DeliveryFee
	}
	rule := services.ResolveTaxRate(country, req.Region)
	thirdParty := services.ThirdPartyDeliveryEnabled()

	modes := []struct {
		fulfillment models.FulfillmentType
		label       string
		delivery    float64
	}{
		{models.FulfillmentPickup, "Pickup — customer collects", 0},
		{models.FulfillmentChefDelivery, "Delivery — the chef carries it", deliveryFee},
		{models.FulfillmentDelivery, "Delivery — a platform rider carries it", deliveryFee},
	}

	scenarios := make([]simulatedScenario, 0, len(modes))
	for _, m := range modes {
		// FulfillmentDelivery only resolves to the platform rule when a third-party
		// rider is actually possible; force it here so the admin can see BOTH answers
		// side by side rather than only the one their current config produces.
		byPlatform := services.DeliveryByPlatform(m.fulfillment, thirdParty)
		if m.fulfillment == models.FulfillmentDelivery {
			byPlatform = true
		}
		rates := rule.ComponentRates(byPlatform)
		pricing := models.ComputeOrderPricing(models.PricingInput{
			Subtotal:    req.Subtotal,
			DeliveryFee: m.delivery,
			PlatformFee: req.Subtotal * feePct / 100,
			Discount:    req.Discount,
			Tip:         req.Tip,
			Rates:       rates,
			Country:     country,
			IntraState:  intraState,
		})
		gw := computeGatewayCost(pricing.Total, gatewayPct, rates.Service > rates.Food)
		refunds := simulateRefunds(pricing, tiers, gw.Real)
		scenarios = append(scenarios, simulatedScenario{
			Fulfillment: string(m.fulfillment),
			Label:       m.label,
			Pricing:     pricing,
			Rates:       rates,
			Gateway:     gw,
			Refunds:     refunds,
			Validations: validateScenario(pricing, refunds),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"input": gin.H{
			"subtotal": req.Subtotal, "platformFeePercent": feePct, "deliveryFee": deliveryFee,
			"discount": req.Discount, "tip": req.Tip, "country": country, "region": req.Region,
			"intraState": intraState, "gatewayFeePercent": gatewayPct,
		},
		"taxRule":                rule,
		"thirdPartyDeliveryLive": thirdParty,
		"scenarios":              scenarios,
	})
}

// computeGatewayCost models the MDR. reclaimable is true when the platform has a
// standard-rated output supply to set the gateway's own tax against — which is
// precisely what charging the platform fee above the restaurant rate creates.
// validationCheck is one asserted property of a scenario, reported rather than
// merely assumed. The admin page shows these so a rate change that breaks an
// invariant is visible immediately, not at the next reconciliation.
type validationCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// validateScenario re-checks, on the produced numbers, the properties the whole
// pricing model is supposed to guarantee.
func validateScenario(p models.OrderPricing, refunds []simulatedRefund) []validationCheck {
	sum := func(ls []models.TaxLine) float64 {
		var t float64
		for _, l := range ls {
			t = models.RoundAmount(t + l.Amount)
		}
		return t
	}
	rows := models.RoundAmount(p.Subtotal + p.DeliveryFee + p.PlatformFee - p.Discount + p.Tip + p.Tax + p.Rounding)
	parts := models.RoundAmount(p.TaxFood + p.TaxService + p.TaxDelivery)

	checks := []validationCheck{
		{"Rows add up to the total", rows == p.Total,
			fmt.Sprintf("rows %.2f vs total %.2f", rows, p.Total)},
		{"Tax per supply sums to the tax", parts == models.RoundAmount(p.Tax),
			fmt.Sprintf("food %.2f + fee %.2f + delivery %.2f = %.2f vs %.2f", p.TaxFood, p.TaxService, p.TaxDelivery, parts, p.Tax)},
		{"Customer view sums to the tax", sum(p.TaxLines) == models.RoundAmount(p.Tax),
			fmt.Sprintf("%d rows totalling %.2f", len(p.TaxLines), sum(p.TaxLines))},
		{"Invoice view sums to the tax", sum(p.TaxBreakdown) == models.RoundAmount(p.Tax),
			fmt.Sprintf("%d rows totalling %.2f", len(p.TaxBreakdown), sum(p.TaxBreakdown))},
		{"Every figure is two decimals", twoDecimals(p),
			"no amount carries a third decimal place"},
	}

	conserved := true
	for _, r := range refunds {
		if !r.Conserves {
			conserved = false
		}
	}
	checks = append(checks, validationCheck{
		"Every refund conserves money", conserved,
		"refund + chef + platform equals the order total at every tier",
	})
	return checks
}

// twoDecimals reports whether every money figure on the breakdown is whole paise.
func twoDecimals(p models.OrderPricing) bool {
	vals := []float64{p.Subtotal, p.DeliveryFee, p.PlatformFee, p.Discount, p.Tip,
		p.Tax, p.TaxFood, p.TaxService, p.TaxDelivery, p.Rounding, p.Total}
	for _, l := range p.TaxBreakdown {
		vals = append(vals, l.Amount)
	}
	for _, v := range vals {
		if math.Abs(v*100-math.Round(v*100)) > 1e-6 {
			return false
		}
	}
	return true
}

func computeGatewayCost(total, percent float64, reclaimable bool) gatewayCost {
	fee := models.RoundAmount(total * percent / 100)
	taxOnFee := models.RoundAmount(fee * 0.18)
	g := gatewayCost{Percent: percent, Fee: fee, TaxOnFee: taxOnFee, Gross: models.RoundAmount(fee + taxOnFee)}
	g.Real = g.Gross
	if reclaimable {
		g.Real = fee
	}
	return g
}

// simulateRefunds runs every tier through the real cancellation model, in both
// the dispatched and not-dispatched states where that changes the answer.
func simulateRefunds(p models.OrderPricing, tiers []int, gatewayReal float64) []simulatedRefund {
	base := services.CancellationOrder{
		FoodPaise:        services.ToPaise(p.Subtotal),
		DeliveryPaise:    services.ToPaise(p.DeliveryFee),
		PlatformFeePaise: services.ToPaise(p.PlatformFee),
		TaxPaise:         services.ToPaise(p.Tax),
		DiscountPaise:    services.ToPaise(p.Discount),
		TaxFoodPaise:     services.ToPaise(p.TaxFood),
		TaxDeliveryPaise: services.ToPaise(p.TaxDelivery),
		TaxServicePaise:  services.ToPaise(p.TaxService),
	}

	out := []simulatedRefund{}
	for _, pct := range tiers {
		dispatchStates := []bool{false}
		if p.DeliveryFee > 0 {
			dispatchStates = []bool{false, true}
		}
		for _, dispatched := range dispatchStates {
			o := base
			o.Dispatched = dispatched
			r := services.ComputeCancellationRefund(o, pct)

			label := refundLabel(pct)
			if dispatched {
				label += " (rider already dispatched)"
			}
			platformKeeps := services.FromPaise(r.PlatformKept)

			// The credit note, split the way it will actually be raised: the tax on
			// the food refunded, plus the delivery's when the delivery is refunded.
			taxFood, taxDelivery := 0.0, 0.0
			if p.Subtotal > 0 {
				taxFood = models.RoundAmount(p.TaxFood * services.FromPaise(r.FoodRefund) / p.Subtotal)
			}
			if r.DeliveryRefund > 0 {
				taxDelivery = p.TaxDelivery
			}

			margin := models.RoundAmount(platformKeeps - p.TaxService - gatewayReal)
			warnings := []string{}
			if margin < 0 {
				warnings = append(warnings,
					"This refund costs the platform money: the gateway fee is sunk at capture and is not returned. The chef-fault gateway-fee levy recovers it.")
			}
			if !(r.Total+r.VendorKept+r.PlatformKept == o.GrandPaise()) {
				warnings = append(warnings,
					"Money is not conserved — refund + chef + platform does not equal the order total. This is a bug, not a policy.")
			}

			out = append(out, simulatedRefund{
				Label:          label,
				FoodRefundPct:  pct,
				Dispatched:     dispatched,
				CustomerGets:   services.FromPaise(r.Total),
				FoodRefund:     services.FromPaise(r.FoodRefund),
				DeliveryRefund: services.FromPaise(r.DeliveryRefund),
				TaxRefund:      services.FromPaise(r.TaxRefund),
				ChefKeeps:      services.FromPaise(r.VendorKept),
				PlatformKeeps:  platformKeeps,
				CreditNote:     services.FromPaise(r.TaxRefund),
				// The gateway fee is sunk at capture whatever is refunded, so it comes
				// off whatever the platform was left holding.
				PlatformMargin:    margin,
				TaxRefundFood:     taxFood,
				TaxRefundDelivery: taxDelivery,
				TaxKeptOnFee:      p.TaxService,
				Conserves:         r.Total+r.VendorKept+r.PlatformKept == o.GrandPaise(),
				Warnings:          warnings,
			})
		}
	}
	return out
}

func refundLabel(pct int) string {
	switch pct {
	case 100:
		return "Full refund — chef cancels or pre-accept"
	case 0:
		return "No food refund — post-cutoff cancellation"
	default:
		return "Partial refund"
	}
}
