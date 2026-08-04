package handlers

// delivery_quote.go — checkout delivery-fee preview (#pickup-incentive).
//
// The customer app used to show "Delivery fee — Free" for every mode, hiding
// both the real fee and the saving from picking up. To show either honestly the
// app needs the fee BEFORE placing the order — and it must be the SAME number
// CreateOrder will charge, or the customer agrees to one total and is billed
// another. This endpoint returns exactly what CreateOrder computes, because both
// call services.QuoteOrderDeliveryFee.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type deliveryQuoteRequest struct {
	// The drop coordinates + city drive the 3PL / self-delivery distance fee.
	// Optional: without them the flat policy fee is returned, matching
	// CreateOrder's own fallback.
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	// Subtotal + State let the preview return the SAME service fee + tax the order
	// will be charged, so checkout shows an honest total (not just items+delivery).
	Subtotal float64 `json:"subtotal"`
	State    string  `json:"state"`
	// Discount is the applied promo, and the Credit fields are the customer's
	// wallet/loyalty intent. Together they let this preview return the SAME credit
	// allocation payment creation will make — the checkout screen shows the credit
	// card before an order exists, so without this it would have to do the money
	// arithmetic itself, which is exactly the drift this feature removes.
	Discount    float64 `json:"discount"`
	Fulfillment string  `json:"fulfillment"`
	// Tip rides in the total but never in the redeemable base — CreateOrder adds it
	// after tax and PlanCheckoutCredit's food branch excludes it. Sending it here
	// keeps the previewed `payable` equal to what the gateway will be asked for;
	// without it a tipped order previewed a payable short by the tip.
	Tip float64 `json:"tip"`
	services.CreditRequest
}

// fulfillmentForTax maps the client's requested mode to the one CreateOrder will
// actually record, so the quote resolves the delivery tax rule the same way the
// charge will. Anything but pickup is created as plain delivery — the chef picks
// the carrier at Mark Ready (resolveFulfillment).
func fulfillmentForTax(requested string) models.FulfillmentType {
	if requested == string(models.FulfillmentPickup) {
		return models.FulfillmentPickup
	}
	return models.FulfillmentDelivery
}

// QuoteDeliveryFee returns the per-mode delivery fee for a chef + drop address,
// so checkout can show the real delivery cost and pickup's saving.
//
// POST /v1/chefs/:id/delivery-quote
func (h *OrderHandler) QuoteDeliveryFee(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var req deliveryQuoteRequest
	_ = c.ShouldBindJSON(&req) // all fields optional; a bad body just means "no coords"

	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	// A quote is a price for an order this viewer could place. If they can't see
	// the kitchen, they can't be quoted by it.
	if proceed, reduced := chefVisibleTo(c, &chef); !proceed || reduced {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	country := req.Country
	if country == "" {
		country = "IN"
	}

	// Live conditions, resolved ONCE per quote: the headline fee, the itemised
	// breakdown and the pin below must all come from the same reading, or the
	// customer sees two different numbers for the same order.
	surge := services.CurrentSurge(c.Request.Context(), country, chef.Latitude, chef.Longitude, req.Latitude, req.Longitude)
	chargeSurge := 1.0
	if services.SurgeChargeEnabled() {
		chargeSurge = surge.Combined
	}

	// A customer's "delivery" request is always created as plain delivery (the
	// chef picks the carrier later at Mark Ready), so the fee it will be charged
	// is the delivery-mode fee — NOT chef_delivery. Quoting chef_delivery here
	// would show a number the order never uses.
	deliveryFee := services.QuoteOrderDeliveryFeeCtx(chef, models.FulfillmentDelivery, req.Latitude, req.Longitude, req.City, country, chargeSurge)
	// Pickup is always free — this zero is the saving the app advertises.
	const pickupFee = 0.0

	saving := deliveryFee - pickupFee
	if saving < 0 {
		saving = 0
	}

	// Serviceability (#709): auto-calculated distance from the chef to the drop and
	// whether it's within the kitchen's delivery range. The app uses this to block
	// delivery + warn BEFORE the customer tries to place an out-of-range order.
	// Skipped as a hard gate when a live 3PL covers the area.
	reach := services.DeliveryReach(chef, req.Latitude, req.Longitude)
	deliverable := reach.Deliverable || services.ThirdPartyDeliveryEnabled()

	policy := services.GetPlatformPolicy()
	taxRule := services.ResolveTaxRate(country, req.State)

	// Price the mode the customer actually chose — a pickup quote priced on the
	// delivery fee would preview a total the order never charges.
	effectiveDelivery := deliveryFee
	if req.Fulfillment == string(models.FulfillmentPickup) {
		effectiveDelivery = pickupFee
	}
	// The SAME function CreateOrder prices with (models/pricing.go), so the total
	// previewed here is the total charged, to the paise, and the tax lines the app
	// renders at checkout are the ones its receipt will carry. Clients render these
	// rather than recomputing — four surfaces doing their own arithmetic is how
	// checkout came to show ₹264.58 for an order the receipt called ₹264.57.
	pricing := models.ComputeOrderPricing(models.PricingInput{
		Subtotal:    req.Subtotal,
		DeliveryFee: effectiveDelivery,
		PlatformFee: req.Subtotal * (policy.PlatformFeePercent / 100.0),
		Discount:    req.Discount,
		Tip:         req.Tip,
		Rates:       taxRule.ComponentRates(services.DeliveryByPlatform(fulfillmentForTax(req.Fulfillment), services.ThirdPartyDeliveryEnabled())),
		Country:     country,
		IntraState:  services.IsIntraStateSupply(chef.State, req.State),
	})

	resp := gin.H{
		"deliveryFee": models.RoundAmount(deliveryFee),
		"pickupFee":   pickupFee,
		"platformFee": pricing.PlatformFee,
		// The priced breakdown for the chosen mode. effectiveDeliveryFee is the
		// delivery line that went INTO total (0 for pickup), where deliveryFee above
		// stays the delivery-mode fee the pickup-saving compares against.
		"effectiveDeliveryFee": pricing.DeliveryFee,
		"tax":                  pricing.Tax,
		"taxLines":             pricing.TaxLines,
		"total":                pricing.Total,
		// Kept for app builds already in the field that still compute their own tax.
		"taxRatePercent": taxRule.Rate,
		"taxName":        taxRule.TaxName,
		"taxInclusive":   taxRule.Inclusive,
		"taxCountry":     country,
		"taxIntraState":  services.IsIntraStateSupply(chef.State, req.State),
		// pickupSaving is what the customer keeps by collecting. 0 when delivery is
		// itself free — the app then shows no incentive rather than a fake one.
		"pickupSaving":       models.RoundAmount(saving),
		"currency":           services.CurrencyForCountry(chef.PayoutCountry),
		"offersPickup":       chef.OffersPickup,
		"offersSelfDelivery": chef.OffersSelfDelivery,
		// offersDelivery is the COMPUTED capability CreateOrder actually gates on
		// (resolveFulfillment): a delivery order is fulfillable only if the chef
		// self-delivers or a 3PL provider is live. It mirrors the same expression
		// used for ChefResponse.OffersDelivery in handlers/chefs.go.
		//
		// Without it on this response a client has to fetch the chef separately to
		// know whether Delivery is even offerable — and a client that assumes it is
		// offers a mode the server will 422. That is not hypothetical: with every
		// delivery_provider disabled and a chef who doesn't self-deliver, such an
		// order reaches `ready` with no carrier and strands there.
		"offersDelivery": chef.OffersSelfDelivery || services.ThirdPartyDeliveryEnabled(),
		// Delivery serviceability for the drop coords the app sent.
		"deliverable": deliverable,
		"distanceKm":  models.RoundAmount(reach.DistanceKm),
		"maxRadiusKm": reach.MaxRadiusKm,
		"rangeKnown":  reach.Known,
		// The conditions behind the fee, so the app can say WHY delivery costs more
		// today rather than showing an unexplained number.
		"surge": surge,
		// surgePin lets CreateOrder charge the multiplier quoted here instead of
		// re-reading conditions that may have moved since. Empty when there is no
		// surge to pin; an order without one is priced live.
		"surgePin": services.SignSurgePin(chef.ID, req.Latitude, req.Longitude, chargeSurge),
	}

	// Credit preview. The checkout screen renders the wallet/loyalty card BEFORE an
	// order exists, so it cannot use the order-scoped quote endpoint. Running the
	// SAME allocator here on the same fee/tax figures this endpoint already computes
	// keeps the preview and the eventual charge in agreement; payment creation still
	// recomputes from the real order and remains authoritative.
	if userID, ok := middleware.GetUserID(c); ok {
		// The RESOLVED gateway, not the chef's stored column. Two things read it:
		// the credit quote below, and — via the response — the RBI Payment
		// Aggregator disclosure the checkout screen must render. That disclosure
		// has to name the aggregator that will actually process the payment, so it
		// must come from the same resolution the charge itself uses rather than
		// from a column the selection may override.
		resolvedProvider := services.SelectCheckoutGateway(chef.PaymentProvider, chef.Mode)
		resp["paymentProvider"] = resolvedProvider

		preview := &models.Order{
			CustomerID: userID, Currency: services.CurrencyForCountry(chef.PayoutCountry),
			PaymentProvider: resolvedProvider,
			Subtotal:    pricing.Subtotal,
			Discount:    pricing.Discount,
			DeliveryFee: pricing.DeliveryFee,
			PlatformFee: pricing.PlatformFee,
			Tax:         pricing.Tax,
			Total:       pricing.Total,
		}
		if q, err := services.BuildCreditQuote(database.DB, preview, userID, req.CreditRequest, creditFlags()); err == nil {
			resp["credit"] = creditQuoteResponse(q)
		}
	}

	// Self-delivery estimate (#702). When the chef delivers themselves, the
	// customer sees an itemised, CAPPED "approx max" — base + distance beyond the
	// free radius, capped at the chef's max. selfDeliveryFee is that ceiling; the
	// chef can only bring it DOWN at accept (#703), never above it. Only computed
	// when the chef offers self-delivery, so plain-delivery chefs are unaffected.
	if chef.OffersSelfDelivery {
		// Built from the SAME surge reading as deliveryFee above, so the itemised
		// breakdown always explains the number the order will actually charge.
		b := services.SelfDeliveryBreakdownAt(chef, req.Latitude, req.Longitude, surge, chargeSurge)
		resp["selfDeliveryFee"] = models.RoundAmount(b.Fee)
		resp["selfDeliveryBreakdown"] = b
	}

	c.JSON(http.StatusOK, resp)
}
