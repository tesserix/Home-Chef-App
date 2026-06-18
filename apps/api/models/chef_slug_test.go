package models

import "testing"

func TestChefSlug(t *testing.T) {
	cases := map[string]string{
		"Amma's Kitchen":      "ammas-kitchen",
		"  Spice & Co.  ":     "spice-co",
		"Rajesh's Home Food!": "rajeshs-home-food",
		"Café Déli":           "caf-d-li", // non-ascii collapses to dashes (ascii-only slug)
		"":                    "",
		"---":                 "",
	}
	for in, want := range cases {
		if got := ChefSlug(in); got != want {
			t.Errorf("ChefSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEffectiveSlug(t *testing.T) {
	// Stored slug wins.
	c := ChefProfile{BusinessName: "Amma's Kitchen", Slug: "ammas-kitchen-blr"}
	if got := c.EffectiveSlug(); got != "ammas-kitchen-blr" {
		t.Errorf("stored slug should win, got %q", got)
	}
	// Falls back to derived slug when unset.
	c2 := ChefProfile{BusinessName: "Amma's Kitchen"}
	if got := c2.EffectiveSlug(); got != "ammas-kitchen" {
		t.Errorf("derived fallback = %q, want ammas-kitchen", got)
	}
}
