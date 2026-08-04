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
	default: // FulfillmentDelivery
		// A live 3PL provider quotes the leg it will carry.
		if fee, ok := QuoteCheckoutDeliveryFee(chef, city, country, dropLat, dropLng); ok {
			return fee
		}
		// 3PL dark → the chef self-delivers this order, so charge the SELF-DELIVERY
		// fee (the recommended amount by distance, capped at the chef's max — #703).
		// This is the "approx max" taken upfront; the chef can bring it DOWN at
		// accept and the difference is refunded to the customer.
		if chef.OffersSelfDelivery {
			return ComputeSelfDeliveryFee(chef, dropLat, dropLng)
		}
		return GetPlatformPolicy().BaseDeliveryFee
	}
}

// DeliveryFeeFrom is the lowest delivery fee this chef's delivery can cost — the
// fee at zero distance — and whether that floor is also the ceiling.
//
// A chef card cannot state the actual fee: it is distance-based and the customer's
// address is not known until checkout. But it must not state a fee that is WRONG,
// which is what a hardcoded zero did — every kitchen advertised "Free delivery"
// while the order charged 39.12 (D-01). A floor the quote can only go up from is
// the strongest claim that is true for every customer.
//
// `flat` is what earns the unqualified "Free delivery": the fee has no distance
// component, so the floor holds however far away the customer is. Under a live 3PL
// the carrier prices the leg and no floor is knowable, so the platform's own base
// fee is quoted and flat is false.
func DeliveryFeeFrom(chef models.ChefProfile) (fee float64, flat bool) {
	if chef.OffersSelfDelivery {
		return chef.SelfDeliveryBaseFee, chef.SelfDeliveryPerKm <= 0
	}
	base := GetPlatformPolicy().BaseDeliveryFee
	return base, false
}
