package services

// delivery_fee.go — the ONE delivery-fee computation (#pickup-incentive).
//
// The fee a customer sees at checkout MUST equal the fee CreateOrder charges. If
// the checkout preview and the order-create path each compute it their own way,
// they WILL drift, and the customer gets charged a different number than the one
// they agreed to — a trust and money-correctness failure. So both call this.
//
// Before this, CreateOrder computed the fee inline (handlers/orders.go) and there
// was no preview at all — the app just showed "Free" for everything, hiding both
// the real delivery fee and pickup's saving. Extracting the logic here lets the
// checkout quote endpoint reuse the exact same code.

import (
	"github.com/homechef/api/models"
)

// QuoteOrderDeliveryFee returns the delivery fee for one order, by fulfillment
// mode. This is authoritative — CreateOrder charges exactly this.
//
//   - pickup        → 0 (the customer collects; no delivery leg). This is the
//     saving the pickup incentive advertises.
//   - chef_delivery → the chef's own distance-based self-delivery fee.
//   - delivery      → a live 3PL quote, falling back to the flat platform fee
//     when no coordinates are known yet or no provider can serve
//     the leg — so checkout never blocks on a quote.
//
// dropLat/dropLng may be 0 (address not yet chosen / no coords): the 3PL quote is
// skipped and the flat policy fee is returned, matching CreateOrder's fallback.
func QuoteOrderDeliveryFee(chef models.ChefProfile, fulfillment models.FulfillmentType, dropLat, dropLng float64, city, country string) float64 {
	switch fulfillment {
	case models.FulfillmentPickup:
		return 0
	case models.FulfillmentChefDelivery:
		return ComputeSelfDeliveryFee(chef, dropLat, dropLng)
	default: // FulfillmentDelivery (3PL)
		if fee, ok := QuoteCheckoutDeliveryFee(chef, city, country, dropLat, dropLng); ok {
			return fee
		}
		return GetPlatformPolicy().BaseDeliveryFee
	}
}
