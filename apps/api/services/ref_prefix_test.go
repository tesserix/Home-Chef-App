package services

import (
	"fmt"
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

// The money-critical invariant: Razorpay rejects a receipt over 40 characters,
// which fails order creation — the customer cannot pay. Several call sites add
// their own prefix on top of an already kitchen-prefixed number, so check the
// worst combination survives the boundary clamp intact.
func TestChefRef_WorstCaseReceiptSurvivesClamp(t *testing.T) {
	number := fmt.Sprintf("HC%s%04d", "0601021504", 9999)
	longest := ChefRef(strings.Repeat("Kitchen ", 10), number)
	for _, prefix := range []string{"", "TIP-", "refund-", "GRP-"} {
		receipt := clampReceipt(prefix + longest)
		if len(receipt) > razorpayReceiptMax {
			t.Errorf("receipt %q is %d chars, over Razorpay's %d limit",
				receipt, len(receipt), razorpayReceiptMax)
		}
		// Reconciliation matches on the unique number, so it must survive.
		if !strings.HasSuffix(receipt, number) {
			t.Errorf("clamped receipt %q lost the order number %q", receipt, number)
		}
	}
}

// And if someone later adds a longer prefix anyway, the boundary clamp must keep
// the unique tail rather than truncating it away into a collision.
func TestClampReceipt_KeepsUniqueTail(t *testing.T) {
	long := "some-very-long-prefix-" + strings.Repeat("x", 30) + "-UNIQUE12345"
	got := clampReceipt(long)
	if len(got) != razorpayReceiptMax {
		t.Fatalf("want %d chars, got %d", razorpayReceiptMax, len(got))
	}
	if !strings.HasSuffix(got, "UNIQUE12345") {
		t.Errorf("clamp dropped the unique tail: %q", got)
	}
	if short := "AMMA-KA-KITCHEN-HC1"; clampReceipt(short) != short {
		t.Errorf("a short receipt must pass through untouched")
	}
}
