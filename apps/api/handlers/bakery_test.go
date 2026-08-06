package handlers

// Validation of the baker's configurator input (#1065). Pure — the persistence
// it guards is covered by the order-path integration tests.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

func TestValidateBakerySpecInput(t *testing.T) {
	cases := []struct {
		name    string
		in      BakerySpecInput
		wantErr string
	}{
		{
			name: "a per-kg cake with sizes is valid",
			in:   BakerySpecInput{ProductType: "cake", PricePerKg: 800, MinWeightKg: 0.5, MaxWeightKg: 5, LeadTimeHours: 24},
		},
		{
			name: "a flat-priced loaf needs no sizes",
			in:   BakerySpecInput{ProductType: "bread"},
		},
		{
			name:    "an unknown product type",
			in:      BakerySpecInput{ProductType: "biryani"},
			wantErr: "what kind of bakery product",
		},
		{
			name:    "per-kg pricing without sizes",
			in:      BakerySpecInput{ProductType: "cake", PricePerKg: 800},
			wantErr: "smallest and largest size",
		},
		{
			name:    "max size below min size",
			in:      BakerySpecInput{ProductType: "cake", PricePerKg: 800, MinWeightKg: 3, MaxWeightKg: 1},
			wantErr: "at least the smallest size",
		},
		{
			name:    "a negative price per kg",
			in:      BakerySpecInput{ProductType: "cake", PricePerKg: -1},
			wantErr: "cannot be negative",
		},
		{
			name:    "an absurd notice period",
			in:      BakerySpecInput{ProductType: "cake", LeadTimeHours: 5000},
			wantErr: "between 0 and 720 hours",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBakerySpecInput(&tc.in)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected an error containing %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestSanitizeOccasions(t *testing.T) {
	got := sanitizeOccasions([]string{"Birthday", "birthday", " WEDDING ", "poker-night", ""})
	if len(got) != 2 || got[0] != "birthday" || got[1] != "wedding" {
		t.Errorf("occasions = %v, want [birthday wedding]", got)
	}
	if out := sanitizeOccasions(nil); out == nil || len(out) != 0 {
		t.Errorf("nil input should render as an empty slice, got %v", out)
	}
}

func TestSaveItemBakerySpec_RejectsAKitchenThatDoesNotBake(t *testing.T) {
	err := saveItemBakerySpec(uuid.New(), uuid.New(), false, &BakerySpecInput{ProductType: "cake"})
	if err == nil || !strings.Contains(err.Error(), "cakes and bakes") {
		t.Fatalf("error = %v, want a bakery-only rejection", err)
	}
}

func TestOccasionLabel(t *testing.T) {
	if got := occasionLabel("baby-shower"); got != "Baby Shower" {
		t.Errorf("label = %q, want %q", got, "Baby Shower")
	}
}

// An update that never mentions the configurator must leave it alone — an
// availability toggle sends only isAvailable, and stripping a cake's options
// there would silently un-sell the product (#1065).
func TestParseBakeryUpdate(t *testing.T) {
	cases := []struct {
		name        string
		raw         string
		wantTouched bool
		wantSpec    bool
	}{
		{name: "absent leaves the spec alone", raw: `{"isAvailable":false}`},
		{name: "explicit null strips the spec", raw: `{"bakery":null}`, wantTouched: true},
		{name: "an object replaces the spec", raw: `{"bakery":{"productType":"cake"}}`, wantTouched: true, wantSpec: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req UpdateMenuItemRequest
			if err := json.Unmarshal([]byte(tc.raw), &req); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			spec, touched, err := parseBakeryUpdate(req.Bakery)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if touched != tc.wantTouched {
				t.Errorf("touched = %v, want %v", touched, tc.wantTouched)
			}
			if (spec != nil) != tc.wantSpec {
				t.Errorf("spec present = %v, want %v", spec != nil, tc.wantSpec)
			}
			if tc.wantSpec && spec.ProductType != "cake" {
				t.Errorf("productType = %q, want cake", spec.ProductType)
			}
		})
	}
}

func TestAttachBakeryChefs(t *testing.T) {
	chefA, chefB := uuid.New(), uuid.New()
	items := []models.MenuItem{
		{ChefID: chefA, Name: "Belgian Truffle"},
		{ChefID: chefB, Name: "Sourdough"},
	}
	chefs := map[uuid.UUID]models.ChefProfile{
		chefA: {BusinessName: "Crumb & Co", City: "Pune", Rating: 4.6},
	}

	out := attachBakeryChefs(items, chefs)
	if len(out) != 2 {
		t.Fatalf("got %d products, want 2", len(out))
	}
	if out[0].ChefName != "Crumb & Co" || out[0].ChefCity != "Pune" || out[0].ChefRating != 4.6 {
		t.Errorf("attribution = %+v", out[0])
	}
	// A product whose kitchen row is missing still lists — losing the cake is
	// worse than losing the byline.
	if out[1].ChefName != "" || out[1].Name != "Sourdough" {
		t.Errorf("unattributed product = %+v", out[1])
	}
}
