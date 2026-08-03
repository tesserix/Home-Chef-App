package models

import (
	"strconv"
	"strings"
)

// pricing.go — the one place an order's money is added up and its tax is split.
//
// Checkout, order detail, the receipt, the invoice PDF, web and mobile all render
// what this file produces, so one order can never show two different totals. Each
// surface used to do its own arithmetic and they disagreed: CreateOrder stored
// unrounded fees, every client rounded each line independently for display, and a
// receipt's lines summed to a paise more than the total printed underneath them.
//
// Two entry points, and the difference matters:
//
//	ComputeOrderPricing — money not yet charged (checkout quote, CreateOrder).
//	  Components are rounded to paise BEFORE summing, so the total is the exact sum
//	  of the lines the customer is shown.
//	PresentOrderPricing — an order already charged. The stored total is the money
//	  that left the customer's account and is never recomputed; the residual left by
//	  an order placed before this file existed surfaces as an explicit rounding line
//	  rather than as an invoice that quietly fails to add up.

// Tax line codes. Clients switch on Code and print Label verbatim.
const (
	TaxLineCGST  = "cgst"
	TaxLineSGST  = "sgst"
	TaxLineIGST  = "igst"
	TaxLineOther = "tax"
)

// TaxLine is one statutory tax row on an invoice. Label already carries the rate
// ("CGST (2.5%)") so every surface prints the same string without reformatting it.
type TaxLine struct {
	Code   string  `json:"code"`
	Label  string  `json:"label"`
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
}

// SameStateFunc decides whether two state spellings name the same state. The real
// implementation resolves both through the seeded `states` table and is installed
// by the services package at init — models cannot import services without an
// import cycle. Plain comparison is the same degradation services.SameState
// already falls back to when that table is unavailable.
var SameStateFunc = func(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// IntraStateSupply reports whether a supply is intra-state (CGST+SGST) rather
// than inter-state (IGST). Either side unknown ⇒ intra: a home kitchen's place of
// supply is the kitchen, so that is the safe default over wrongly charging IGST.
func IntraStateSupply(sellerState, buyerState string) bool {
	if strings.TrimSpace(sellerState) == "" || strings.TrimSpace(buyerState) == "" {
		return true
	}
	return SameStateFunc(sellerState, buyerState)
}

// PricingInput is everything needed to price an order. Country and IntraState
// decide the shape of the tax lines; the rest is money.
type PricingInput struct {
	Subtotal    float64
	DeliveryFee float64
	PlatformFee float64
	Discount    float64
	Tip         float64

	TaxRate      float64
	TaxName      string
	TaxInclusive bool
	Country      string
	IntraState   bool
}

// OrderPricing is the rendered breakdown. The JSON tags are the contract every
// client renders against.
type OrderPricing struct {
	Subtotal    float64   `json:"subtotal"`
	DeliveryFee float64   `json:"deliveryFee"`
	PlatformFee float64   `json:"platformFee"`
	Discount    float64   `json:"discount"`
	Tip         float64   `json:"tip"`
	Tax         float64   `json:"tax"`
	TaxLines    []TaxLine `json:"taxLines"`
	// Rounding reconciles a pre-rounding order's stored total to the sum of its
	// displayed lines. Always 0 for anything ComputeOrderPricing produced.
	Rounding float64 `json:"rounding,omitempty"`
	Total    float64 `json:"total"`
}

// ComputeOrderPricing prices an order that has not been charged yet. Every
// component is rounded to paise first, so Total is the exact sum of the lines.
func ComputeOrderPricing(in PricingInput) OrderPricing {
	p := OrderPricing{
		Subtotal:    RoundAmount(nonNegative(in.Subtotal)),
		DeliveryFee: RoundAmount(nonNegative(in.DeliveryFee)),
		PlatformFee: RoundAmount(nonNegative(in.PlatformFee)),
		Discount:    RoundAmount(nonNegative(in.Discount)),
		Tip:         RoundAmount(nonNegative(in.Tip)),
	}
	// A promo can only discount what there is to discount. Beyond that the charge
	// floors at zero while the discount line keeps growing, and the invoice stops
	// adding up — so cap it and record the amount actually applied.
	if charge := p.Subtotal + p.DeliveryFee + p.PlatformFee; p.Discount > charge {
		p.Discount = charge
	}
	// Tip rides in the total but never in the tax base — it is a pass-through to
	// the chef or rider, not consideration for the platform's supply.
	base := nonNegative(p.Subtotal + p.DeliveryFee + p.PlatformFee - p.Discount)
	if in.TaxRate > 0 {
		if in.TaxInclusive {
			p.Tax = RoundAmount(base - base/(1+in.TaxRate/100.0))
		} else {
			p.Tax = RoundAmount(base * in.TaxRate / 100.0)
		}
	}
	// An inclusive rate is already inside base; adding it would charge tax twice.
	charged := base
	if !in.TaxInclusive {
		charged += p.Tax
	}
	p.Total = RoundAmount(charged + p.Tip)
	p.TaxLines = BuildTaxLines(p.Tax, in)
	return p
}

// PresentOrderPricing renders an order that has already been charged.
// chargedTotal and taxCharged are what the order was actually billed and are
// never recomputed — a rate row edited since then must not restate a historical
// invoice. Any paise the stored figures fail to reconcile becomes Rounding.
func PresentOrderPricing(in PricingInput, taxCharged, chargedTotal float64) OrderPricing {
	p := OrderPricing{
		Subtotal:    RoundAmount(in.Subtotal),
		DeliveryFee: RoundAmount(in.DeliveryFee),
		PlatformFee: RoundAmount(in.PlatformFee),
		Discount:    RoundAmount(in.Discount),
		Tip:         RoundAmount(in.Tip),
		Tax:         RoundAmount(taxCharged),
		Total:       RoundAmount(chargedTotal),
	}
	lines := p.Subtotal + p.DeliveryFee + p.PlatformFee - p.Discount + p.Tip
	if !in.TaxInclusive {
		lines += p.Tax
	}
	p.Rounding = RoundAmount(p.Total - lines)
	p.TaxLines = BuildTaxLines(p.Tax, in)
	return p
}

// BuildTaxLines splits a tax amount into the rows an invoice must carry: CGST +
// SGST for an intra-state Indian supply, IGST for inter-state, the configured
// single line elsewhere. The two Indian halves sum EXACTLY to tax — the second
// takes the remainder rather than being rounded on its own.
func BuildTaxLines(tax float64, in PricingInput) []TaxLine {
	tax = RoundAmount(tax)
	if tax <= 0 {
		return nil
	}
	// Orders written before the meal-plan path snapshotted TaxRate carry a real tax
	// at rate 0. Name the tax but print no rate, never a 0% the amount contradicts.
	rated := in.TaxRate > 0
	if strings.EqualFold(strings.TrimSpace(in.Country), "IN") {
		if in.IntraState {
			half := in.TaxRate / 2
			cgst := RoundAmount(tax / 2)
			return []TaxLine{
				{Code: TaxLineCGST, Label: taxLineLabel("CGST", half, rated), Rate: half, Amount: cgst},
				{Code: TaxLineSGST, Label: taxLineLabel("SGST", half, rated), Rate: half, Amount: RoundAmount(tax - cgst)},
			}
		}
		return []TaxLine{{Code: TaxLineIGST, Label: taxLineLabel("IGST", in.TaxRate, rated), Rate: in.TaxRate, Amount: tax}}
	}
	name := strings.TrimSpace(in.TaxName)
	if name == "" {
		name = "Tax"
	}
	if in.TaxInclusive {
		name += " (incl.)"
	}
	return []TaxLine{{Code: TaxLineOther, Label: name, Rate: in.TaxRate, Amount: tax}}
}

func taxLineLabel(name string, rate float64, rated bool) string {
	if !rated {
		return name
	}
	return name + " (" + formatTaxRate(rate) + "%)"
}

// formatTaxRate prints 2.5 as "2.5" and 18 as "18", capped at two decimals.
func formatTaxRate(r float64) string {
	s := strings.TrimRight(strconv.FormatFloat(r, 'f', 2, 64), "0")
	return strings.TrimSuffix(s, ".")
}

func nonNegative(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}
