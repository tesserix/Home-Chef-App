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

// PricingInput is everything needed to price an order. Rates come from the one
// tax_rates row for the delivery address (TaxRate.ComponentRates); Country and
// IntraState decide the shape of the tax lines; the rest is money.
type PricingInput struct {
	Subtotal    float64
	DeliveryFee float64
	PlatformFee float64
	Discount    float64
	Tip         float64

	Rates      TaxRates
	Country    string
	IntraState bool
}

// OrderPricing is the rendered breakdown. The JSON tags are the contract every
// client renders against.
type OrderPricing struct {
	Subtotal    float64 `json:"subtotal"`
	DeliveryFee float64 `json:"deliveryFee"`
	PlatformFee float64 `json:"platformFee"`
	Discount    float64 `json:"discount"`
	Tip         float64 `json:"tip"`
	Tax         float64 `json:"tax"`
	// Tax per supply, because the food (restaurant service) and the platform's own
	// fee are not the same supply and need not share a rate. They always sum to Tax.
	TaxFood     float64   `json:"taxFood"`
	TaxService  float64   `json:"taxService"`
	TaxDelivery float64   `json:"taxDelivery"`
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
	// A promo discounts the food, and only spills onto delivery and the fee once
	// the food is exhausted — so a rate that differs per component is applied to
	// the base that component actually kept.
	food, delivery, fee := spendDiscount(p.Subtotal, p.DeliveryFee, p.PlatformFee, p.Discount)
	incl := in.Rates.Inclusive

	if in.Rates.Uniform() {
		// One rate over one base and ONE rounding — bit-for-bit the figure this
		// charged before tax became per-component, so splitting the model moved no
		// money. The parts below are an apportionment of that single number.
		p.Tax = taxOn(food+delivery+fee, in.Rates.Food, incl)
		p.TaxFood, p.TaxDelivery, p.TaxService = apportion(p.Tax, food, delivery, fee)
	} else {
		p.TaxFood = taxOn(food, in.Rates.Food, incl)
		p.TaxDelivery = taxOn(delivery, in.Rates.Delivery, incl)
		p.TaxService = taxOn(fee, in.Rates.Service, incl)
		p.Tax = RoundAmount(p.TaxFood + p.TaxDelivery + p.TaxService)
	}

	// Tip rides in the total but never in the tax base — it is a pass-through to
	// the chef or rider, not consideration for the platform's supply. An inclusive
	// rate is already inside the base; adding it would charge the tax twice.
	charged := food + delivery + fee
	if !incl {
		charged += p.Tax
	}
	p.Total = RoundAmount(charged + p.Tip)
	p.TaxLines = BuildTaxLines(p, in)
	return p
}

// taxOn is the tax carried by one base at one rate, to the paise.
func taxOn(base, ratePercent float64, inclusive bool) float64 {
	if base <= 0 || ratePercent <= 0 {
		return 0
	}
	if inclusive {
		return RoundAmount(base - base/(1+ratePercent/100.0))
	}
	return RoundAmount(base * ratePercent / 100.0)
}

// spendDiscount takes the discount off the food first, then delivery, then the
// platform fee, returning what each component keeps.
func spendDiscount(food, delivery, fee, discount float64) (float64, float64, float64) {
	for _, part := range []*float64{&food, &delivery, &fee} {
		take := discount
		if take > *part {
			take = *part
		}
		*part = RoundAmount(*part - take)
		discount = RoundAmount(discount - take)
	}
	return food, delivery, fee
}

// apportion splits one tax figure across three bases so the parts sum EXACTLY to
// it — the last takes the remainder rather than being rounded on its own.
func apportion(tax, food, delivery, fee float64) (float64, float64, float64) {
	base := food + delivery + fee
	if base <= 0 || tax == 0 {
		return 0, 0, 0
	}
	tFood := RoundAmount(tax * food / base)
	tDelivery := RoundAmount(tax * delivery / base)
	return tFood, tDelivery, RoundAmount(tax - tFood - tDelivery)
}

// TaxSnapshot is the tax an order was actually charged, per supply. Food/Service
// /Delivery are all zero for an order placed before tax became per-component.
type TaxSnapshot struct {
	Total    float64
	Food     float64
	Service  float64
	Delivery float64
}

// PresentOrderPricing renders an order that has already been charged. charged
// and chargedTotal are what the order was actually billed and are never
// recomputed — a rate row edited since then must not restate a historical
// invoice. Any paise the stored figures fail to reconcile becomes Rounding.
func PresentOrderPricing(in PricingInput, charged TaxSnapshot, chargedTotal float64) OrderPricing {
	p := OrderPricing{
		Subtotal:    RoundAmount(in.Subtotal),
		DeliveryFee: RoundAmount(in.DeliveryFee),
		PlatformFee: RoundAmount(in.PlatformFee),
		Discount:    RoundAmount(in.Discount),
		Tip:         RoundAmount(in.Tip),
		Tax:         RoundAmount(charged.Total),
		TaxFood:     RoundAmount(charged.Food),
		TaxService:  RoundAmount(charged.Service),
		TaxDelivery: RoundAmount(charged.Delivery),
		Total:       RoundAmount(chargedTotal),
	}
	lines := p.Subtotal + p.DeliveryFee + p.PlatformFee - p.Discount + p.Tip
	if !in.Rates.Inclusive {
		lines += p.Tax
	}
	p.Rounding = RoundAmount(p.Total - lines)
	p.TaxLines = BuildTaxLines(p, in)
	return p
}

// taxGroup is the supplies sharing one rate — the unit an invoice states a tax
// head against.
type taxGroup struct {
	rate    float64
	amount  float64
	sources []string
}

// BuildTaxLines produces the rows an invoice must carry: CGST + SGST for an
// intra-state Indian supply, IGST for inter-state, the configured single line
// elsewhere. The halves of each pair sum EXACTLY to that pair's tax.
//
// Rows are grouped by RATE, not by component. While every component shares a
// rate — the case today — that is one CGST/SGST pair and reads exactly as it
// always has. Give the platform fee its own rate and it becomes a pair per rate,
// each naming what it is charged on, which is what an invoice must show when one
// bill carries two rates.
func BuildTaxLines(p OrderPricing, in PricingInput) []TaxLine {
	groups := groupTaxByRate(p, in)
	if len(groups) == 0 {
		return nil
	}
	india := strings.EqualFold(strings.TrimSpace(in.Country), "IN")
	single := len(groups) == 1

	lines := make([]TaxLine, 0, len(groups)*2)
	for _, g := range groups {
		// A source suffix is noise when there is only one group and everything is
		// charged at that rate; it is essential the moment there are two.
		on := ""
		if !single {
			on = " on " + strings.Join(g.sources, " + ")
		}
		switch {
		case india && in.IntraState:
			half := g.rate / 2
			cgst := RoundAmount(g.amount / 2)
			lines = append(lines,
				TaxLine{Code: TaxLineCGST, Label: taxLineLabel("CGST", half, g.rate > 0) + on, Rate: half, Amount: cgst},
				TaxLine{Code: TaxLineSGST, Label: taxLineLabel("SGST", half, g.rate > 0) + on, Rate: half, Amount: RoundAmount(g.amount - cgst)},
			)
		case india:
			lines = append(lines, TaxLine{
				Code: TaxLineIGST, Label: taxLineLabel("IGST", g.rate, g.rate > 0) + on, Rate: g.rate, Amount: g.amount,
			})
		default:
			name := strings.TrimSpace(in.Rates.Name)
			if name == "" {
				name = "Tax"
			}
			if in.Rates.Inclusive {
				name += " (incl.)"
			}
			lines = append(lines, TaxLine{Code: TaxLineOther, Label: name + on, Rate: g.rate, Amount: g.amount})
		}
	}
	return lines
}

// groupTaxByRate collects the taxed components into one group per distinct rate,
// in the order they appear on the invoice.
func groupTaxByRate(p OrderPricing, in PricingInput) []taxGroup {
	parts := []struct {
		source string
		rate   float64
		amount float64
	}{
		{"food", in.Rates.Food, RoundAmount(p.TaxFood)},
		{"delivery", in.Rates.Delivery, RoundAmount(p.TaxDelivery)},
		{"platform fee", in.Rates.Service, RoundAmount(p.TaxService)},
	}
	// An order placed before tax was split per component carries only a total. It
	// is one supply at one rate, which is exactly how it was charged.
	if p.TaxFood == 0 && p.TaxDelivery == 0 && p.TaxService == 0 {
		if RoundAmount(p.Tax) <= 0 {
			return nil
		}
		return []taxGroup{{rate: in.Rates.Food, amount: RoundAmount(p.Tax), sources: []string{"this order"}}}
	}

	var groups []taxGroup
	for _, part := range parts {
		if part.amount <= 0 {
			continue
		}
		found := false
		for i := range groups {
			if groups[i].rate == part.rate {
				groups[i].amount = RoundAmount(groups[i].amount + part.amount)
				groups[i].sources = append(groups[i].sources, part.source)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, taxGroup{rate: part.rate, amount: part.amount, sources: []string{part.source}})
		}
	}
	return groups
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
