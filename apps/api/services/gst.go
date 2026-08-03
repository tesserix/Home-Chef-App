package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// gst.go — the service-side face of the GST split. The arithmetic lives in
// models/pricing.go so the API DTO, the invoice PDF and the checkout quote all
// share one implementation; this file supplies only the piece that needs the
// database, namely resolving two state spellings to the same state.

// normalizeState lowercases + trims so "Maharashtra" == " maharashtra ".
func normalizeState(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// IsIntraStateSupply reports whether the supply is intra-state (CGST+SGST)
// rather than inter-state (IGST), resolving each side through the seeded states
// table rather than comparing raw strings: a chef writes "Odisha" where a
// geocoded address writes "OR", and comparing those literally put IGST on a
// Bhubaneswar → Bhubaneswar invoice where CGST+SGST was due (state_resolve.go).
// Either side blank ⇒ intra, the safe case for a home kitchen.
func IsIntraStateSupply(sellerState, buyerState string) bool {
	return models.IntraStateSupply(sellerState, buyerState)
}
