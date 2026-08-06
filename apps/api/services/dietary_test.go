package services

// Unit tests for the dietary/allergen conflict matcher (#41). Pure logic — the
// endpoint that calls it is DB-backed and covered by integration.

import "testing"

func boolp(b bool) *bool { return &b }

func TestDietaryConflicts(t *testing.T) {
	cases := []struct {
		name           string
		diet, avoid    []string
		tags, allergen []string
		isVeg          *bool
		wantTypes      []string // expected conflict types, order-insensitive
	}{
		{
			name:      "no profile preferences → no conflicts",
			diet:      nil,
			avoid:     nil,
			allergen:  []string{"peanuts"},
			isVeg:     boolp(false),
			wantTypes: []string{},
		},
		{
			name:      "allergen the customer avoids is present",
			avoid:     []string{"peanuts"},
			allergen:  []string{"peanuts", "dairy"},
			wantTypes: []string{"allergen"},
		},
		{
			name:      "allergen match is case-insensitive",
			avoid:     []string{"Peanuts"},
			allergen:  []string{"peanuts"},
			wantTypes: []string{"allergen"},
		},
		{
			name:      "vegetarian customer + explicitly non-veg dish",
			diet:      []string{"vegetarian"},
			isVeg:     boolp(false),
			wantTypes: []string{"diet"},
		},
		{
			name:      "vegetarian customer + veg dish → ok",
			diet:      []string{"vegetarian"},
			isVeg:     boolp(true),
			wantTypes: []string{},
		},
		{
			name:      "veg customer + non-veg via tag (isVeg unset)",
			diet:      []string{"vegan"},
			tags:      []string{"non-veg"},
			wantTypes: []string{"diet"},
		},
		{
			name:      "non-veg customer + non-veg dish → no diet conflict",
			diet:      []string{"halal"},
			isVeg:     boolp(false),
			wantTypes: []string{},
		},
		{
			name:      "both an allergen and a diet conflict",
			diet:      []string{"jain"},
			avoid:     []string{"dairy", "peanuts"},
			allergen:  []string{"dairy"},
			isVeg:     boolp(false),
			wantTypes: []string{"allergen", "diet"},
		},
		{
			name:      "absence of a veg tag is NOT treated as non-veg (no false positive)",
			diet:      []string{"vegetarian"},
			tags:      []string{"gluten-free"},
			isVeg:     nil,
			wantTypes: []string{},
		},
	}

	for _, c := range cases {
		got := DietaryConflicts(c.diet, c.avoid, c.tags, c.allergen, c.isVeg)
		counts := map[string]int{}
		for _, g := range got {
			counts[g.Type]++
		}
		want := map[string]int{}
		for _, w := range c.wantTypes {
			want[w]++
		}
		if len(got) != len(c.wantTypes) {
			t.Errorf("%s: got %d conflicts %+v, want %d (%v)", c.name, len(got), got, len(c.wantTypes), c.wantTypes)
			continue
		}
		for typ, n := range want {
			if counts[typ] != n {
				t.Errorf("%s: type %q got %d, want %d", c.name, typ, counts[typ], n)
			}
		}
	}
}

// A diet implies allergens it forbids (#1065) — an eggless customer must be
// warned about a cake that declares eggs even when they never listed eggs as an
// allergy to avoid.
func TestDietaryConflicts_DietImpliedAllergens(t *testing.T) {
	cases := []struct {
		name      string
		diet      []string
		allergens []string
		wantLabel string
		wantCount int
	}{
		{"eggless customer, cake with egg", []string{"eggless"}, []string{"eggs"}, "Eggs", 1},
		{"eggless customer, eggless cake", []string{"eggless"}, []string{"dairy"}, "", 0},
		{"vegan customer, dairy frosting", []string{"vegan"}, []string{"dairy"}, "Dairy (milk)", 1},
		{"gluten-free customer, wheat sponge", []string{"gluten-free"}, []string{"gluten"}, "Gluten (wheat, barley, rye)", 1},
		{"nut-free customer, hazelnut cake", []string{"nut-free"}, []string{"tree-nuts"}, "Tree Nuts", 1},
		{"no diet set", nil, []string{"eggs"}, "", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DietaryConflicts(tc.diet, nil, nil, tc.allergens, nil)
			if len(got) != tc.wantCount {
				t.Fatalf("got %d conflicts %+v, want %d", len(got), got, tc.wantCount)
			}
			if tc.wantCount > 0 && got[0].Label != tc.wantLabel {
				t.Errorf("label = %q, want %q", got[0].Label, tc.wantLabel)
			}
		})
	}
}

// The same allergen must not be reported twice when the customer both avoids it
// and follows a diet that forbids it.
func TestDietaryConflicts_NoDuplicateAllergenWarning(t *testing.T) {
	got := DietaryConflicts([]string{"eggless"}, []string{"eggs"}, nil, []string{"eggs"}, nil)
	if len(got) != 1 {
		t.Fatalf("got %d conflicts %+v, want 1", len(got), got)
	}
}
