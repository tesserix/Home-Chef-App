package services

// delivery_fee.go — the ONE delivery-fee computation (#pickup-incentive).
//
// The fee a customer sees at checkout MUST equal the fee CreateOrder charges. If
// the preview and the order-create path each compute it their own way, they WILL
// drift and the customer is billed a number they never agreed to. So both call
// QuoteOrderDeliveryFeeCtx with the SAME surge — the quote signs the multiplier
// it used (delivery_quote_pin.go) and CreateOrder replays it.

import (
	"context"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// SurgeChargeEnabled reports whether live conditions may move the CHARGED fee.
// When false, surge still shows in the estimate breakdown but the charge stays
// on the deterministic basis — the pre-#704 behaviour.
//
// Requires a pin key as well as the flag: without one a quoted multiplier can't
// be signed, so the charge could not be held to the number the customer saw.
func SurgeChargeEnabled() bool {
	return config.AppConfig != nil &&
		config.AppConfig.DeliverySurgeChargeEnabled &&
		config.AppConfig.DeliverySurgePinKey != ""
}

// ResolveChargeSurge returns the multiplier to CHARGE with. It prefers a valid
// pin (the multiplier the customer was quoted), falls back to live conditions,
// and is always 1.0 while surge-charging is off.
func ResolveChargeSurge(ctx context.Context, pin string, chef models.ChefProfile, dropLat, dropLng float64, country string) float64 {
	if !SurgeChargeEnabled() {
		return 1.0
	}
	if surge, ok := VerifySurgePin(pin, chef.ID, dropLat, dropLng); ok {
		return surge
	}
	// No usable pin (expired, or an older app build that doesn't send one): price
	// on current conditions rather than refusing the order.
	return CurrentSurge(ctx, country, chef.Latitude, chef.Longitude, dropLat, dropLng).Combined
}

// QuoteOrderDeliveryFee returns the delivery fee on the neutral basis (no surge).
// Retained for callers that price outside a customer quote — meal-plan and group
// paths, and tests — where there is no quoted multiplier to honour.
func QuoteOrderDeliveryFee(chef models.ChefProfile, fulfillment models.FulfillmentType, dropLat, dropLng float64, city, country string) float64 {
	return QuoteOrderDeliveryFeeCtx(chef, fulfillment, dropLat, dropLng, city, country, 1.0)
}

// QuoteOrderDeliveryFeeCtx returns the delivery fee for one order at a given
// surge. This is authoritative — CreateOrder charges exactly this.
//
//   - pickup        → 0 (the customer collects; no delivery leg)
//   - chef_delivery → the chef's distance-based self-delivery fee, surged
//   - delivery      → a live 3PL quote, else the chef's surged self-delivery fee,
//     else the flat platform fee
//
// Surge scales only the distance component (the flat base isn't a driving cost)
// and the chef's max-fee cap still bites, so a bad signal cannot run away.
// dropLat/dropLng may be 0: the distance component is then unknown and only the
// base applies, matching CreateOrder's fallback.
func QuoteOrderDeliveryFeeCtx(chef models.ChefProfile, fulfillment models.FulfillmentType, dropLat, dropLng float64, city, country string, surge float64) float64 {
	return QuoteOrderDelivery(chef, fulfillment, dropLat, dropLng, city, country, surge).Fee
}

// OrderDeliveryQuote is a delivery fee together with whose price it is. The
// source is persisted on the order (Order.DeliveryFeeSource) because a delivery
// order is created before anyone picks a carrier, and only the source says
// whether the money is the kitchen's.
type OrderDeliveryQuote struct {
	Fee    float64
	Source string
}

// QuoteOrderDelivery is QuoteOrderDeliveryFeeCtx with the attribution kept.
func QuoteOrderDelivery(chef models.ChefProfile, fulfillment models.FulfillmentType, dropLat, dropLng float64, city, country string, surge float64) OrderDeliveryQuote {
	selfDelivery := func() OrderDeliveryQuote {
		return OrderDeliveryQuote{
			Fee:    computeSelfDeliveryBreakdown(chef, dropLat, dropLng, surge).Fee,
			Source: models.DeliveryFeeSourceChef,
		}
	}
	switch fulfillment {
	case models.FulfillmentPickup:
		return OrderDeliveryQuote{}
	case models.FulfillmentChefDelivery:
		return selfDelivery()
	default: // FulfillmentDelivery
		// A live 3PL provider quotes the leg it will carry — their price already
		// reflects their own conditions, so platform surge must not double-count it.
		if fee, ok := QuoteCheckoutDeliveryFee(chef, city, country, dropLat, dropLng); ok {
			return OrderDeliveryQuote{Fee: fee, Source: models.DeliveryFeeSourceProvider}
		}
		// 3PL dark → the chef will carry it, so charge the chef's own published price.
		if chef.OffersSelfDelivery {
			return selfDelivery()
		}
		return OrderDeliveryQuote{Fee: GetPlatformPolicy().BaseDeliveryFee, Source: models.DeliveryFeeSourcePlatform}
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
		// A published ladder states the answer outright: the nearest band is the
		// cheapest this kitchen delivers for, and a single band holds at any
		// distance the chef covers.
		if tiers := chef.DeliveryTiers(); len(tiers) > 0 {
			return tiers[0].Fee, len(tiers) == 1
		}
		// Inside the chef's free radius the whole fee is waived — the flat base
		// included — so for anyone close enough the floor is zero, not the base.
		// Quoting the base here would overstate the cheapest this kitchen can be.
		if chef.SelfDeliveryFreeRadiusKm > 0 && (chef.SelfDeliveryBaseFee > 0 || chef.SelfDeliveryPerKm > 0) {
			return 0, false // beyond the radius it rises, so it is a floor not a promise
		}
		return chef.SelfDeliveryBaseFee, chef.SelfDeliveryPerKm <= 0
	}
	base := GetPlatformPolicy().BaseDeliveryFee
	return base, false
}
