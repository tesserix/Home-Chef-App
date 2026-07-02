package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
)

func TestResolveFulfillment(t *testing.T) {
	chefPickup := models.ChefProfile{OffersPickup: true}
	chefNoPickup := models.ChefProfile{OffersPickup: false}
	chefSelfDelivers := models.ChefProfile{OffersSelfDelivery: true}
	const tpl = true  // a 3PL provider is enabled
	const dark = false // 3PL dark (chef-only launch)

	// default → 3PL delivery when the chef does NOT self-deliver but 3PL is live
	if ft, err := resolveFulfillment(CreateOrderRequest{}, chefNoPickup, tpl); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("default should be delivery when 3PL is live, got %q err=%v", ft, err)
	}
	// CORE GUARD: chef doesn't self-deliver AND 3PL is dark → delivery is
	// unfulfillable and must be rejected (no silent dead-end at Mark Ready).
	if _, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "delivery"}, chefNoPickup, dark); err == nil {
		t.Fatal("delivery must be rejected when the chef doesn't self-deliver and 3PL is dark")
	}
	if _, err := resolveFulfillment(CreateOrderRequest{}, chefNoPickup, dark); err == nil {
		t.Fatal("empty (=delivery) must be rejected when the chef doesn't self-deliver and 3PL is dark")
	}
	// Chef-only launch: a self-delivering chef can take delivery orders with 3PL dark.
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "delivery"}, chefSelfDelivers, dark); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("self-delivering chef should accept delivery with 3PL dark, got %q err=%v", ft, err)
	}
	if ft, err := resolveFulfillment(CreateOrderRequest{}, chefSelfDelivers, dark); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("empty mode should be delivery for a self-delivering chef, got %q err=%v", ft, err)
	}
	// Defensive: an explicit chef_delivery from an old client is still honored.
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "chef_delivery"}, chefSelfDelivers, dark); err != nil || ft != models.FulfillmentChefDelivery {
		t.Fatalf("explicit chef_delivery should be honored, got %q err=%v", ft, err)
	}
	// pickup allowed when chef offers it (independent of delivery capability)
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "pickup"}, chefPickup, dark); err != nil || ft != models.FulfillmentPickup {
		t.Fatalf("pickup should be allowed, got %q err=%v", ft, err)
	}
	// pickup rejected when chef does not offer it
	if _, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "pickup"}, chefNoPickup, tpl); err == nil {
		t.Fatal("pickup must be rejected when chef does not offer pickup")
	}
	// unknown value rejected
	if _, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "teleport"}, chefPickup, tpl); err == nil {
		t.Fatal("unknown fulfillment must be rejected")
	}
}

func TestResolveReadyCarrier(t *testing.T) {
	self := models.ChefProfile{OffersSelfDelivery: true}
	noSelf := models.ChefProfile{}
	const tpl = true  // a 3PL provider is enabled
	const dark = false // 3PL dark (chef-only launch)

	// no carrier requested → keep current
	if ft, err := resolveReadyCarrier(models.FulfillmentDelivery, "", self, dark); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("empty keeps current; got %q err=%v", ft, err)
	}
	// self-delivering chef picks "I'll deliver" (works even with 3PL dark)
	if ft, err := resolveReadyCarrier(models.FulfillmentDelivery, "chef_delivery", self, dark); err != nil || ft != models.FulfillmentChefDelivery {
		t.Fatalf("want chef_delivery; got %q err=%v", ft, err)
	}
	// switch to rider is allowed only when 3PL is enabled
	if ft, err := resolveReadyCarrier(models.FulfillmentChefDelivery, "delivery", self, tpl); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("want delivery when 3PL live; got %q err=%v", ft, err)
	}
	// CORE GUARD: "hand to a rider" rejected when 3PL is dark
	if _, err := resolveReadyCarrier(models.FulfillmentChefDelivery, "delivery", self, dark); err == nil {
		t.Fatal("hand-to-rider must be rejected when 3PL is dark")
	}
	// chef who doesn't self-deliver cannot pick chef_delivery
	if _, err := resolveReadyCarrier(models.FulfillmentDelivery, "chef_delivery", noSelf, tpl); err == nil {
		t.Fatal("non-self-delivering chef must be rejected")
	}
	// pickup orders reject any carrier choice
	if _, err := resolveReadyCarrier(models.FulfillmentPickup, "chef_delivery", self, tpl); err == nil {
		t.Fatal("pickup orders must reject a carrier choice")
	}
}

func TestChefTrackCoords_PickupIsExact(t *testing.T) {
	o := models.Order{FulfillmentType: models.FulfillmentPickup}
	o.Chef.ID = uuid.New()
	o.Chef.Latitude, o.Chef.Longitude = 19.1499, 72.7967
	lat, lng, exact := chefTrackCoords(o)
	if !exact || lat != 19.1499 || lng != 72.7967 {
		t.Fatalf("pickup must return EXACT chef coords; got %v,%v exact=%v", lat, lng, exact)
	}
}

func TestChefTrackCoords_DeliveryIsFuzzed(t *testing.T) {
	o := models.Order{FulfillmentType: models.FulfillmentDelivery}
	o.Chef.ID = uuid.New()
	o.Chef.Latitude, o.Chef.Longitude = 19.1499, 72.7967
	lat, lng, exact := chefTrackCoords(o)
	if exact {
		t.Fatal("delivery must be fuzzed, not exact")
	}
	if lat == 19.1499 && lng == 72.7967 {
		t.Fatal("delivery coords must differ from the true location")
	}
}
