package services

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
)

// bakery.go — pricing and validation for a configured bakery line (#1065).
// Pure (no DB) so the checkout path, the reorder path and the cart preview all
// price a cake identically and can be unit-tested.

// BakeryLineInput is the customer's configuration of one bakery line.
type BakeryLineInput struct {
	WeightKg          float64     `json:"weightKg"`
	OptionIDs         []uuid.UUID `json:"bakeryOptionIds"`
	Message           string      `json:"messageOnCake"`
	ReferencePhotoURL string      `json:"referencePhotoUrl"`
	Occasion          string      `json:"occasion"`
}

// weightTolerance absorbs float noise in the step check — the client sends
// 1.5000000000000002 often enough to matter.
const weightTolerance = 0.001

// PriceBakeryLine validates the customer's configuration against the chef's
// spec and returns the per-unit price plus the snapshot to persist on the order
// line. basePrice is the MenuItem's flat price, used only when the spec is not
// priced per kg. Errors are customer-facing.
func PriceBakeryLine(spec models.BakerySpec, basePrice float64, in BakeryLineInput) (float64, models.OrderItemBakery, error) {
	var empty models.OrderItemBakery

	weight, base, err := bakeryBasePrice(spec, basePrice, in.WeightKg)
	if err != nil {
		return 0, empty, err
	}

	selections, tags, allergens, delta, err := resolveBakeryOptions(spec, weight, in.OptionIDs)
	if err != nil {
		return 0, empty, err
	}

	message := strings.TrimSpace(in.Message)
	if message != "" {
		if !spec.AllowMessage {
			return 0, empty, fmt.Errorf("this baker doesn't offer a message on this item")
		}
		max := spec.MaxMessageChars
		if max <= 0 {
			max = 40
		}
		if len([]rune(message)) > max {
			return 0, empty, fmt.Errorf("keep the message to %d characters or fewer", max)
		}
	}

	photo := strings.TrimSpace(in.ReferencePhotoURL)
	if photo != "" && !spec.AllowReferencePhoto {
		return 0, empty, fmt.Errorf("this baker doesn't accept a reference photo on this item")
	}

	snap := models.OrderItemBakery{
		ProductType:       spec.ProductType,
		WeightKg:          weight,
		Serves:            bakeryServes(spec, weight),
		BasePrice:         round2(base),
		Selections:        selections,
		MessageOnCake:     message,
		ReferencePhotoURL: photo,
		Occasion:          strings.TrimSpace(in.Occasion),
		DietaryTags:       tags,
		Allergens:         allergens,
	}
	return round2(base + delta), snap, nil
}

// bakeryBasePrice resolves the line's base price and validates the weight for a
// per-kg product.
func bakeryBasePrice(spec models.BakerySpec, basePrice, weight float64) (float64, float64, error) {
	if spec.PricePerKg <= 0 {
		return 0, basePrice, nil
	}
	if weight <= 0 {
		return 0, 0, fmt.Errorf("choose a size for this item")
	}
	min, max := spec.MinWeightKg, spec.MaxWeightKg
	if min <= 0 {
		min = 0.5
	}
	if max <= 0 {
		max = 10
	}
	if weight < min-weightTolerance || weight > max+weightTolerance {
		return 0, 0, fmt.Errorf("size must be between %s kg and %s kg", trimNum(min), trimNum(max))
	}
	if step := spec.WeightStepKg; step > 0 {
		steps := (weight - min) / step
		if math.Abs(steps-math.Round(steps)) > weightTolerance {
			return 0, 0, fmt.Errorf("size is sold in %s kg steps", trimNum(step))
		}
	}
	return weight, spec.PricePerKg * weight, nil
}

// resolveBakeryOptions enforces one choice per offered kind and prices the
// deltas. Every kind the chef defined options for is required — a cake with no
// flavour picked is not something a baker can start.
func resolveBakeryOptions(spec models.BakerySpec, weight float64, selected []uuid.UUID) ([]models.BakerySelection, []string, []string, float64, error) {
	sel := map[uuid.UUID]bool{}
	for _, id := range selected {
		sel[id] = true
	}

	byKind := map[models.BakeryOptionKind][]models.BakeryOption{}
	for _, o := range spec.Options {
		byKind[o.Kind] = append(byKind[o.Kind], o)
	}

	selections := []models.BakerySelection{}
	tagSet, allergenSet := map[string]bool{}, map[string]bool{}
	matched := map[uuid.UUID]bool{}
	var delta float64

	for _, kind := range models.BakeryOptionKinds {
		options := byKind[kind]
		if len(options) == 0 {
			continue
		}
		label := models.BakeryOptionKindLabels[kind]

		chosen := []models.BakeryOption{}
		for _, o := range options {
			if !sel[o.ID] {
				continue
			}
			if !o.IsAvailable {
				return nil, nil, nil, 0, fmt.Errorf("%s is no longer available", o.Name)
			}
			chosen = append(chosen, o)
			matched[o.ID] = true
		}

		if len(chosen) == 0 {
			// Only require a kind that still has something orderable in it.
			if !hasAvailable(options) {
				continue
			}
			return nil, nil, nil, 0, fmt.Errorf("please choose an option for %q", label)
		}
		if len(chosen) > 1 {
			return nil, nil, nil, 0, fmt.Errorf("pick just one %s", label)
		}

		o := chosen[0]
		d := o.PriceDelta
		if o.PriceMode == models.BakeryPricePerKg && weight > 0 {
			d = o.PriceDelta * weight
		}
		delta += d
		selections = append(selections, models.BakerySelection{
			Kind: kind, Label: label, Name: o.Name, PriceDelta: round2(d),
		})
		for _, t := range o.DietaryTags {
			tagSet[t] = true
		}
		for _, a := range o.Allergens {
			allergenSet[a] = true
		}
	}

	for _, id := range selected {
		if !matched[id] {
			return nil, nil, nil, 0, fmt.Errorf("invalid option selection")
		}
	}

	return selections, sortedKeys(tagSet), sortedKeys(allergenSet), delta, nil
}

func hasAvailable(options []models.BakeryOption) bool {
	for _, o := range options {
		if o.IsAvailable {
			return true
		}
	}
	return false
}

// BakeryWeightChoices expands the spec's min/max/step into the sizes the
// configurator shows. Empty for a flat-priced product.
func BakeryWeightChoices(spec models.BakerySpec) []float64 { return spec.WeightChoices() }

// bakeryServes converts a weight into a guest count using the chef's own
// serves-per-kg guidance.
func bakeryServes(spec models.BakerySpec, weight float64) int {
	if weight <= 0 {
		return 0
	}
	perKg := spec.ServesPerKg
	if perKg <= 0 {
		perKg = 8
	}
	return int(math.Round(weight * float64(perKg)))
}

// EarliestBakeryFulfillment is the soonest slot a product needing leadHours of
// notice can be collected or delivered.
func EarliestBakeryFulfillment(now time.Time, leadHours int) time.Time {
	if leadHours <= 0 {
		return now
	}
	return now.Add(time.Duration(leadHours) * time.Hour)
}

// ValidateBakeryLeadTime checks a scheduled slot against the longest lead time
// on the order. A product needing notice must be scheduled — a cake cannot be
// an "as soon as possible" order.
func ValidateBakeryLeadTime(now time.Time, scheduledFor *time.Time, leadHours int) error {
	if leadHours <= 0 {
		return nil
	}
	earliest := EarliestBakeryFulfillment(now, leadHours)
	if scheduledFor == nil {
		return fmt.Errorf("this order needs %dh notice — choose a delivery date and time", leadHours)
	}
	if scheduledFor.Before(earliest) {
		return fmt.Errorf("this order needs %dh notice — the earliest slot is %s", leadHours, earliest.Format("2 Jan, 3:04 PM"))
	}
	return nil
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// trimNum renders 0.50 as "0.5" and 5.00 as "5" for customer-facing copy.
func trimNum(f float64) string {
	s := strings.TrimRight(fmt.Sprintf("%.2f", f), "0")
	return strings.TrimSuffix(s, ".")
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }
