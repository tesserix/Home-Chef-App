package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// gst.go — Indian GST compliance: a single "GST" figure must be presented as
// CGST + SGST for an intra-state supply, or IGST for an inter-state supply, on
// the tax invoice (and the checkout preview). Which one applies is decided by the
// seller's (chef's) state vs the buyer's (delivery) state.
//
// This is presentation only — it splits an already-computed tax amount; it never
// changes the total. Non-IN orders keep a single tax line (handled by callers).

// GSTBreakdown is the compliant split of a GST amount.
type GSTBreakdown struct {
	Intra bool `json:"intra"` // true → CGST+SGST; false → IGST
	// Amounts (major units) — for intra, CGST+SGST = tax; for inter, IGST = tax.
	CGST float64 `json:"cgst"`
	SGST float64 `json:"sgst"`
	IGST float64 `json:"igst"`
	// Component rates in percent (each half of the total rate for intra).
	CGSTRate float64 `json:"cgstRate"`
	SGSTRate float64 `json:"sgstRate"`
	IGSTRate float64 `json:"igstRate"`
}

// normalizeState lowercases + trims so "Maharashtra" == " maharashtra ".
func normalizeState(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// IsIntraStateSupply reports whether the supply is intra-state (seller state ==
// buyer state, both known). When either state is blank we default to INTRA — the
// safe, common case for a home kitchen (the food service's place of supply is the
// kitchen), rather than wrongly charging IGST.
func IsIntraStateSupply(sellerState, buyerState string) bool {
	s, b := normalizeState(sellerState), normalizeState(buyerState)
	if s == "" || b == "" {
		return true
	}
	return s == b
}

// SplitIndiaGST breaks a GST amount (at ratePercent) into the compliant
// components for the seller/buyer states. Intra → CGST+SGST (rate halved each,
// amount split so the two sum EXACTLY to tax with no rounding drift); inter → IGST.
func SplitIndiaGST(taxAmount, ratePercent float64, sellerState, buyerState string) GSTBreakdown {
	if IsIntraStateSupply(sellerState, buyerState) {
		cgst := models.RoundAmount(taxAmount / 2)
		return GSTBreakdown{
			Intra:    true,
			CGST:     cgst,
			SGST:     models.RoundAmount(taxAmount - cgst), // remainder → no drift
			CGSTRate: ratePercent / 2,
			SGSTRate: ratePercent / 2,
		}
	}
	return GSTBreakdown{Intra: false, IGST: models.RoundAmount(taxAmount), IGSTRate: ratePercent}
}
