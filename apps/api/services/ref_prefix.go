package services

import (
	"strings"
	"unicode"
)

// ref_prefix.go — kitchen-branded reference numbers.
//
// Every customer-facing reference (order number, meal-plan number) is prefixed
// with the kitchen it belongs to, so "AMMA-KA-KITCHEN-HC26072808359105" says who
// cooked it without a lookup. Support, payouts and receipts all read these
// numbers, and a bare HC… told nobody anything.

// chefRefPrefixMax caps the slug so a reference stays readable rather than
// carrying a kitchen's full trading name.
//
// It is sized for readability, not for any gateway field: the Cashfree order id
// is a UUID we mint, never this number, so no payment leg is length-constrained
// by a long kitchen name (#1086 retired the 40-char Razorpay `receipt`).
const chefRefPrefixMax = 24

// ChefRefPrefix converts a kitchen's business name into the uppercase slug used
// to prefix its reference numbers: "Amma Ka Kitchen" → "AMMA-KA-KITCHEN".
//
// Returns "" for a blank or unusable name (e.g. one written entirely in a script
// with no ASCII letters or digits), so callers fall back to the unprefixed
// number rather than emitting a stray leading hyphen.
func ChefRefPrefix(businessName string) string {
	var b strings.Builder
	lastHyphen := true // suppresses a leading separator
	for _, r := range businessName {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(unicode.ToUpper(r))
			lastHyphen = false
		case !lastHyphen:
			// Any run of spaces/punctuation/non-ASCII collapses to ONE hyphen.
			b.WriteByte('-')
			lastHyphen = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) <= chefRefPrefixMax {
		return slug
	}

	// Too long: cut at the last word boundary inside the budget so the prefix
	// reads as whole words ("SHRI-KRISHNA" rather than "SHRI-KRISHNA-CA").
	cut := slug[:chefRefPrefixMax]
	if i := strings.LastIndexByte(cut, '-'); i > 0 {
		return cut[:i]
	}
	return strings.TrimRight(cut, "-")
}

// ChefRef prefixes a reference number with the kitchen's slug.
//
// An unnamed kitchen (or one whose name yields no usable slug) returns the
// number untouched — a reference must never come back empty or malformed just
// because the profile is incomplete.
func ChefRef(businessName, number string) string {
	if p := ChefRefPrefix(businessName); p != "" {
		return p + "-" + number
	}
	return number
}
