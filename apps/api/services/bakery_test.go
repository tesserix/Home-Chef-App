package services

// Unit tests for the bakery configurator's pricing + validation (#1065). Pure
// logic: the DB-backed handler that calls it is covered by integration.

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
)

// cakeSpec builds a per-kg priced cake with a shape, a flavour and an egg
// choice — the shape of a real bakery product.
func cakeSpec() (models.BakerySpec, map[string]uuid.UUID) {
	ids := map[string]uuid.UUID{}
	id := func(k string) uuid.UUID {
		ids[k] = uuid.New()
		return ids[k]
	}
	return models.BakerySpec{
		ID:              uuid.New(),
		ProductType:     models.BakeryProductCake,
		PricePerKg:      800,
		MinWeightKg:     0.5,
		MaxWeightKg:     5,
		WeightStepKg:    0.5,
		ServesPerKg:     8,
		AllowMessage:    true,
		MaxMessageChars: 25,
		LeadTimeHours:   24,
		Options: []models.BakeryOption{
			{ID: id("round"), Kind: models.BakeryOptionShape, Name: "Round", IsAvailable: true, IsDefault: true},
			{ID: id("heart"), Kind: models.BakeryOptionShape, Name: "Heart", PriceDelta: 150, PriceMode: models.BakeryPriceFlat, IsAvailable: true},
			{ID: id("vanilla"), Kind: models.BakeryOptionFlavour, Name: "Vanilla", IsAvailable: true, IsDefault: true},
			{ID: id("belgian"), Kind: models.BakeryOptionFlavour, Name: "Belgian chocolate", PriceDelta: 200, PriceMode: models.BakeryPricePerKg, IsAvailable: true},
			{ID: id("soldout"), Kind: models.BakeryOptionFlavour, Name: "Red velvet", IsAvailable: false},
			{ID: id("egg"), Kind: models.BakeryOptionEgg, Name: "With egg", IsAvailable: true, Allergens: []string{"eggs"}},
			{ID: id("eggless"), Kind: models.BakeryOptionEgg, Name: "Eggless", PriceDelta: 50, IsAvailable: true, DietaryTags: []string{"eggless"}},
		},
	}, ids
}

func TestPriceBakeryLine_PerKgWithDeltas(t *testing.T) {
	spec, ids := cakeSpec()

	unit, snap, err := PriceBakeryLine(spec, 0, BakeryLineInput{
		WeightKg:  1.5,
		OptionIDs: []uuid.UUID{ids["heart"], ids["belgian"], ids["eggless"]},
		Message:   "Happy Birthday Aarav",
		Occasion:  "birthday",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 800×1.5 base + 150 flat shape + 200×1.5 per-kg flavour + 50 flat eggless.
	want := 1200.0 + 150 + 300 + 50
	if math.Abs(unit-want) > 0.001 {
		t.Fatalf("unit price = %.2f, want %.2f", unit, want)
	}
	if snap.BasePrice != 1200 {
		t.Errorf("base price = %.2f, want 1200", snap.BasePrice)
	}
	if snap.Serves != 12 {
		t.Errorf("serves = %d, want 12", snap.Serves)
	}
	if len(snap.Selections) != 3 {
		t.Fatalf("selections = %d, want 3", len(snap.Selections))
	}
	if got := snap.Summary(); got != "1.5 kg · Heart · Belgian chocolate · Eggless · “Happy Birthday Aarav”" {
		t.Errorf("summary = %q", got)
	}
	if len(snap.DietaryTags) != 1 || snap.DietaryTags[0] != "eggless" {
		t.Errorf("dietary tags = %v, want [eggless]", snap.DietaryTags)
	}
	if len(snap.Allergens) != 0 {
		t.Errorf("allergens = %v, want none (eggless was chosen)", snap.Allergens)
	}
}

func TestPriceBakeryLine_AllergenRidesOnTheChosenOption(t *testing.T) {
	spec, ids := cakeSpec()
	_, snap, err := PriceBakeryLine(spec, 0, BakeryLineInput{
		WeightKg:  1,
		OptionIDs: []uuid.UUID{ids["round"], ids["vanilla"], ids["egg"]},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snap.Allergens) != 1 || snap.Allergens[0] != "eggs" {
		t.Errorf("allergens = %v, want [eggs]", snap.Allergens)
	}
}

func TestPriceBakeryLine_FlatPricedProduct(t *testing.T) {
	spec := models.BakerySpec{
		ProductType: models.BakeryProductCookie,
		Options: []models.BakeryOption{
			{ID: uuid.New(), Kind: models.BakeryOptionFlavour, Name: "Choc chip", IsAvailable: true},
		},
	}
	optID := spec.Options[0].ID

	unit, snap, err := PriceBakeryLine(spec, 250, BakeryLineInput{OptionIDs: []uuid.UUID{optID}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unit != 250 {
		t.Errorf("unit = %.2f, want 250 (menu price, no weight)", unit)
	}
	if snap.WeightKg != 0 {
		t.Errorf("weight = %.2f, want 0", snap.WeightKg)
	}
	if got := snap.Summary(); got != "Choc chip" {
		t.Errorf("summary = %q", got)
	}
}

func TestPriceBakeryLine_Rejections(t *testing.T) {
	spec, ids := cakeSpec()
	all := []uuid.UUID{ids["round"], ids["vanilla"], ids["egg"]}

	cases := []struct {
		name    string
		in      BakeryLineInput
		wantErr string
	}{
		{
			name:    "weight below the minimum",
			in:      BakeryLineInput{WeightKg: 0.25, OptionIDs: all},
			wantErr: "between 0.5 kg and 5 kg",
		},
		{
			name:    "weight above the maximum",
			in:      BakeryLineInput{WeightKg: 6, OptionIDs: all},
			wantErr: "between 0.5 kg and 5 kg",
		},
		{
			name:    "weight off the chef's step",
			in:      BakeryLineInput{WeightKg: 1.2, OptionIDs: all},
			wantErr: "0.5 kg steps",
		},
		{
			name:    "missing weight on a per-kg product",
			in:      BakeryLineInput{OptionIDs: all},
			wantErr: "choose a size",
		},
		{
			name:    "a required kind not chosen",
			in:      BakeryLineInput{WeightKg: 1, OptionIDs: []uuid.UUID{ids["round"], ids["vanilla"]}},
			wantErr: "Egg preference",
		},
		{
			name:    "two options from the same kind",
			in:      BakeryLineInput{WeightKg: 1, OptionIDs: []uuid.UUID{ids["round"], ids["heart"], ids["vanilla"], ids["egg"]}},
			wantErr: "one Shape",
		},
		{
			name:    "an unavailable option",
			in:      BakeryLineInput{WeightKg: 1, OptionIDs: []uuid.UUID{ids["round"], ids["soldout"], ids["egg"]}},
			wantErr: "no longer available",
		},
		{
			name:    "an option from another product",
			in:      BakeryLineInput{WeightKg: 1, OptionIDs: append(append([]uuid.UUID{}, all...), uuid.New())},
			wantErr: "invalid",
		},
		{
			name:    "a message longer than the chef allows",
			in:      BakeryLineInput{WeightKg: 1, OptionIDs: all, Message: strings.Repeat("x", 26)},
			wantErr: "25 characters",
		},
		{
			name:    "a photo the chef does not accept",
			in:      BakeryLineInput{WeightKg: 1, OptionIDs: all, ReferencePhotoURL: "https://x/y.jpg"},
			wantErr: "reference photo",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := PriceBakeryLine(spec, 0, tc.in)
			if err == nil {
				t.Fatalf("expected an error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestPriceBakeryLine_MessageRejectedWhenNotOffered(t *testing.T) {
	spec := models.BakerySpec{ProductType: models.BakeryProductBread}
	if _, _, err := PriceBakeryLine(spec, 100, BakeryLineInput{Message: "hi"}); err == nil {
		t.Fatal("expected an error when the chef does not offer a message")
	}
}

func TestBakeryWeightChoices(t *testing.T) {
	spec, _ := cakeSpec()
	got := BakeryWeightChoices(spec)
	want := []float64{0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5}
	if len(got) != len(want) {
		t.Fatalf("choices = %v, want %v", got, want)
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 0.001 {
			t.Fatalf("choices = %v, want %v", got, want)
		}
	}
	if flat := BakeryWeightChoices(models.BakerySpec{}); len(flat) != 0 {
		t.Errorf("a flat-priced product should offer no weights, got %v", flat)
	}
}

func TestEarliestBakeryFulfillment(t *testing.T) {
	now := time.Date(2026, 8, 6, 10, 0, 0, 0, time.UTC)

	if got := EarliestBakeryFulfillment(now, 24); !got.Equal(now.Add(24 * time.Hour)) {
		t.Errorf("earliest = %v, want +24h", got)
	}

	// A scheduled slot inside the lead time is refused; one outside it passes.
	tooSoon := now.Add(6 * time.Hour)
	if err := ValidateBakeryLeadTime(now, &tooSoon, 24); err == nil {
		t.Error("expected a lead-time error for a slot 6h away when 24h notice is needed")
	}
	ok := now.Add(30 * time.Hour)
	if err := ValidateBakeryLeadTime(now, &ok, 24); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// No lead time configured → any slot, including none at all, is fine.
	if err := ValidateBakeryLeadTime(now, nil, 0); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Lead time configured but no slot chosen → the customer must pick one.
	if err := ValidateBakeryLeadTime(now, nil, 24); err == nil {
		t.Error("expected an error when a lead-time product has no scheduled slot")
	}
}
