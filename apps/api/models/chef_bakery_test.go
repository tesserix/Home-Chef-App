package models

import "testing"

// One store, both shelves (#1065): a meals kitchen can also sell bakes, so the
// configurator is gated on the capability, not on an exclusive vertical.
func TestSellsBakery(t *testing.T) {
	cases := []struct {
		name string
		chef ChefProfile
		want bool
	}{
		{"plain kitchen", ChefProfile{Vertical: VerticalKitchen}, false},
		{"blank vertical", ChefProfile{}, false},
		{"dedicated bakery", ChefProfile{Vertical: VerticalBakery}, true},
		{"kitchen that also bakes", ChefProfile{Vertical: VerticalKitchen, SellsBakery: true}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.chef.OffersBakery(); got != tc.want {
				t.Errorf("OffersBakery() = %v, want %v", got, tc.want)
			}
		})
	}
	// The primary shelf is unaffected by the extra capability — a kitchen that
	// also bakes is still listed and branded as a kitchen.
	c := ChefProfile{Vertical: VerticalKitchen, SellsBakery: true}
	if c.EffectiveVertical() != VerticalKitchen {
		t.Errorf("EffectiveVertical() = %q, want kitchen", c.EffectiveVertical())
	}
}
