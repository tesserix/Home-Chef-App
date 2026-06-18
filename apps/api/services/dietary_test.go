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
