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

// DeliveryByPlatform decides which of the two delivery rules an order's delivery
// leg falls under, at the moment it is priced.
//
// Local delivery supplied THROUGH an e-commerce operator by a person not liable
// for registration became a notified §9(5) supply on 22 September 2025. A chef
// carrying their own food is a different thing: door delivery by the person who
// cooked it is arguable as part of the composite restaurant supply, at the food
// rate. See docs/gst-refunds-and-gateway-costs.md.
//
// The carrier is only chosen by the chef at Mark Ready, so at checkout this can
// be genuinely unknown. It resolves conservatively: when a third-party rider is
// still possible, the platform rule applies, because under-charging leaves the
// liability with the platform while over-charging is refundable.
func DeliveryByPlatform(fulfillment models.FulfillmentType, thirdPartyEnabled bool) bool {
	switch fulfillment {
	case models.FulfillmentPickup:
		return false // nothing is delivered
	case models.FulfillmentChefDelivery:
		return false // the chef is already committed to carrying it
	default:
		// Plain `delivery`. resolveFulfillment only admits it when someone can
		// carry it, so with 3PL dark the chef necessarily will.
		return thirdPartyEnabled
	}
}
