package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// dietary.go — single source of truth for the dietary & allergen taxonomy (#41)
// and the pure conflict matcher reused by the /dietary/check endpoint and any
// future server-side safety check. Mobile mirrors these lists in
// packages/mobile-shared/src/dietary.ts; keep the two in sync.

// DietaryOption is one selectable diet or allergen (value is the canonical
// stored token; label is the display string).
type DietaryOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// DietOptions is the canonical diet vocabulary a customer can set on their
// profile and a chef can tag a dish with.
var DietOptions = []DietaryOption{
	{"vegetarian", "Vegetarian"},
	{"vegan", "Vegan"},
	{"jain", "Jain"},
	{"eggetarian", "Eggetarian"},
	{"halal", "Halal"},
	{"kosher", "Kosher"},
	{"gluten-free", "Gluten-Free"},
	{"dairy-free", "Dairy-Free"},
	{"nut-free", "Nut-Free"},
	{"low-carb", "Low-Carb"},
	// Bakery-led diets (#1065). Eggless is the single most-asked-for filter on an
	// Indian cake order, and diabetics ask for sugar-free by name — both matter
	// enough to be first-class tokens rather than free text in a description.
	{"eggless", "Eggless"},
	{"contains-egg", "Contains Egg"},
	{"sugar-free", "Sugar-Free"},
	{"no-added-sugar", "No Added Sugar"},
	{"whole-wheat", "Whole Wheat"},
	{"vegan-bake", "Vegan Bake"},
}

// BakeryDietOptions is the subset the bakery filters and the cake configurator
// surface — the rest of the taxonomy is noise on a cake.
var BakeryDietOptions = []DietaryOption{
	{"eggless", "Eggless"},
	{"contains-egg", "Contains Egg"},
	{"vegan", "Vegan"},
	{"sugar-free", "Sugar-Free"},
	{"no-added-sugar", "No Added Sugar"},
	{"gluten-free", "Gluten-Free"},
	{"dairy-free", "Dairy-Free"},
	{"nut-free", "Nut-Free"},
	{"whole-wheat", "Whole Wheat"},
	{"jain", "Jain"},
}

// BakeryAllergenOptions is the allergen subset a baker actually declares.
var BakeryAllergenOptions = []DietaryOption{
	{"gluten", "Gluten (wheat, barley, rye)"},
	{"eggs", "Eggs"},
	{"dairy", "Dairy (milk)"},
	{"peanuts", "Peanuts"},
	{"tree-nuts", "Tree Nuts"},
	{"soy", "Soy"},
	{"sesame", "Sesame"},
	{"sulphites", "Sulphites"},
}

// AllergenOptions is the canonical allergen vocabulary — the full set of major
// declarable allergens (the EU-14 / FDA "big 9" union, the basis of India's
// FSSAI labelling rules). A customer lists the ones they must avoid; a chef
// declares the ones a dish contains.
var AllergenOptions = []DietaryOption{
	{"gluten", "Gluten (wheat, barley, rye)"},
	{"peanuts", "Peanuts"},
	{"tree-nuts", "Tree Nuts"},
	{"dairy", "Dairy (milk)"},
	{"eggs", "Eggs"},
	{"soy", "Soy"},
	{"fish", "Fish"},
	{"shellfish", "Shellfish (crustaceans)"},
	{"molluscs", "Molluscs"},
	{"sesame", "Sesame"},
	{"mustard", "Mustard"},
	{"celery", "Celery"},
	{"lupin", "Lupin"},
	{"sulphites", "Sulphites"},
}

// vegDiets are the diet preferences that a non-vegetarian dish violates.
var vegDiets = map[string]bool{"vegetarian": true, "vegan": true, "jain": true}

// dietForbiddenAllergens maps a diet to the allergens it rules out by
// definition (#1065). An eggless customer never has to also list eggs as an
// allergy for us to warn them off an egg cake — the diet already said so.
var dietForbiddenAllergens = map[string][]string{
	"eggless":     {"eggs"},
	"vegan":       {"eggs", "dairy", "fish", "shellfish", "molluscs"},
	"vegan-bake":  {"eggs", "dairy"},
	"gluten-free": {"gluten"},
	"dairy-free":  {"dairy"},
	"nut-free":    {"peanuts", "tree-nuts"},
	"jain":        {"eggs"},
}

// DietaryConflict describes one reason a dish clashes with a customer's profile.
type DietaryConflict struct {
	// Type is "allergen" (dish contains an allergen the customer avoids) or
	// "diet" (dish violates the customer's diet, e.g. non-veg for a vegetarian).
	Type   string `json:"type"`
	Label  string `json:"label"`  // the offending allergen/diet, e.g. "Peanuts"
	Detail string `json:"detail"` // a short customer-facing explanation
}

// normTokens lowercases + trims a token slice into a set for comparison. The
// stored values may be Title Case ("Peanuts") or canonical ("peanuts"); we
// compare case-insensitively so either matches.
func normTokens(tokens []string) map[string]bool {
	out := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			out[t] = true
		}
	}
	return out
}

// labelFor returns a friendly label for a canonical token, falling back to the
// token itself (Title-cased) when it isn't in the taxonomy.
func labelFor(token string, options []DietaryOption) string {
	t := strings.ToLower(strings.TrimSpace(token))
	for _, o := range options {
		if o.Value == t {
			return o.Label
		}
	}
	if t == "" {
		return token
	}
	return strings.ToUpper(t[:1]) + t[1:]
}

// DietaryConflicts returns the conflicts between a customer's dietary profile
// and a single dish. Conservative on purpose — it only flags high-confidence
// clashes (an allergen the customer avoids is present; or the customer follows
// a veg diet and the dish is explicitly non-veg) so warnings stay trustworthy
// rather than noisy. Pure + unit-tested.
func DietaryConflicts(dietPrefs, avoidAllergens, itemDietaryTags, itemAllergens []string, itemIsVeg *bool) []DietaryConflict {
	conflicts := []DietaryConflict{}

	prefs := normTokens(dietPrefs)

	// Allergen clashes: any allergen the dish declares that the customer avoids,
	// either explicitly or by following a diet that rules it out.
	avoid := normTokens(avoidAllergens)
	for d := range prefs {
		for _, a := range dietForbiddenAllergens[d] {
			avoid[a] = true
		}
	}
	if len(avoid) > 0 {
		seen := map[string]bool{}
		for _, a := range itemAllergens {
			key := strings.ToLower(strings.TrimSpace(a))
			if !avoid[key] || seen[key] {
				continue
			}
			seen[key] = true
			conflicts = append(conflicts, DietaryConflict{
				Type:   "allergen",
				Label:  labelFor(a, AllergenOptions),
				Detail: "Contains " + labelFor(a, AllergenOptions) + ", which you avoid",
			})
		}
	}

	// Diet clash: a veg-following customer + an explicitly non-veg dish. We only
	// trust an explicit signal (IsVeg == false, or a non-veg dietary tag) — the
	// mere absence of a "vegetarian" tag is not treated as non-veg.
	wantsVeg := false
	var vegLabel string
	for d := range prefs {
		if vegDiets[d] {
			wantsVeg = true
			vegLabel = labelFor(d, DietOptions)
			break
		}
	}
	if wantsVeg && itemIsNonVeg(itemIsVeg, itemDietaryTags) {
		conflicts = append(conflicts, DietaryConflict{
			Type:   "diet",
			Label:  vegLabel,
			Detail: "Not " + vegLabel + " — this dish is non-vegetarian",
		})
	}

	return conflicts
}

// itemIsNonVeg reports whether a dish is explicitly non-vegetarian: IsVeg set to
// false, or a dietary tag that clearly marks it non-veg.
func itemIsNonVeg(isVeg *bool, dietaryTags []string) bool {
	if isVeg != nil {
		return !*isVeg
	}
	tags := normTokens(dietaryTags)
	for _, t := range []string{"non-veg", "non-vegetarian", "nonveg", "non veg"} {
		if tags[t] {
			return true
		}
	}
	return false
}

// DietaryConflictsForItem is a convenience wrapper over a loaded MenuItem.
func DietaryConflictsForItem(dietPrefs, avoidAllergens []string, item models.MenuItem) []DietaryConflict {
	return DietaryConflicts(dietPrefs, avoidAllergens, item.DietaryTags, item.Allergens, item.IsVeg)
}
