package services

import (
	"strings"
	"testing"
)

func TestChefRefPrefix(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"the canonical case", "Amma Ka Kitchen", "AMMA-KA-KITCHEN"},
		{"already uppercase", "AMMA KA KITCHEN", "AMMA-KA-KITCHEN"},
		{"punctuation collapses to one hyphen", "Amma's  Ka -- Kitchen!", "AMMA-S-KA-KITCHEN"},
		{"digits are kept", "Kitchen 24x7", "KITCHEN-24X7"},
		{"leading/trailing junk trimmed", "  ...Amma Ka...  ", "AMMA-KA"},
		{"single word", "Bhojanam", "BHOJANAM"},
		{"blank name yields no prefix", "   ", ""},
		// A name with no ASCII letters or digits must not produce a lone hyphen or
		// an empty-but-present prefix; callers fall back to the bare number.
		{"non-latin script yields no prefix", "अम्मा का किचन", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ChefRefPrefix(c.in); got != c.want {
				t.Errorf("ChefRefPrefix(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// A long name must be cut at a word boundary rather than mid-word, and must
// never exceed the cap that keeps receipts inside Razorpay's 40-char limit.
func TestChefRefPrefix_LongNameCutAtWordBoundary(t *testing.T) {
	got := ChefRefPrefix("Shri Krishna Rasoi Home Kitchen and Catering Services")
	if len(got) > chefRefPrefixMax {
		t.Fatalf("prefix %q is %d chars, over the %d cap", got, len(got), chefRefPrefixMax)
	}
	if strings.HasSuffix(got, "-") {
		t.Errorf("prefix %q ends in a hyphen", got)
	}
	if got != "SHRI-KRISHNA-RASOI-HOME" {
		t.Errorf("want SHRI-KRISHNA-RASOI-HOME (cut at a word boundary), got %q", got)
	}
}

// A single unbroken word longer than the cap has no boundary to cut at — it must
// still be truncated rather than blowing the budget.
func TestChefRefPrefix_LongSingleWord(t *testing.T) {
	got := ChefRefPrefix("Supercalifragilisticexpialidocious")
	if len(got) != chefRefPrefixMax {
		t.Fatalf("want %d chars, got %d (%q)", chefRefPrefixMax, len(got), got)
	}
}

func TestChefRef_FallsBackWhenUnnamed(t *testing.T) {
	if got := ChefRef("", "HC26072808359105"); got != "HC26072808359105" {
		t.Errorf("an unnamed kitchen must yield the bare number, got %q", got)
	}
	if got := ChefRef("Amma Ka Kitchen", "HC26072808359105"); got != "AMMA-KA-KITCHEN-HC26072808359105" {
		t.Errorf("got %q", got)
	}
}
