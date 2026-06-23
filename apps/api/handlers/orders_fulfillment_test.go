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

	// default → 3PL delivery when the chef does NOT self-deliver
	if ft, err := resolveFulfillment(CreateOrderRequest{}, chefNoPickup); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("default should be delivery, got %q err=%v", ft, err)
	}
	// Carrier is chosen at Mark Ready now, NOT at creation: a delivery order to a
	// self-delivering chef is still created as plain delivery.
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "delivery"}, chefSelfDelivers); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("delivery should stay delivery at creation, got %q err=%v", ft, err)
	}
	if ft, err := resolveFulfillment(CreateOrderRequest{}, chefSelfDelivers); err != nil || ft != models.FulfillmentDelivery {
		t.Fatalf("empty mode should be delivery at creation, got %q err=%v", ft, err)
	}
	// Defensive: an explicit chef_delivery from an old client is still honored.
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "chef_delivery"}, chefSelfDelivers); err != nil || ft != models.FulfillmentChefDelivery {
		t.Fatalf("explicit chef_delivery should be honored, got %q err=%v", ft, err)
	}
	// pickup allowed when chef offers it
	if ft, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "pickup"}, chefPickup); err != nil || ft != models.FulfillmentPickup {
		t.Fatalf("pickup should be allowed, got %q err=%v", ft, err)
	}
	// pickup rejected when chef does not offer it
	if _, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "pickup"}, chefNoPickup); err == nil {
		t.Fatal("pickup must be rejected when chef does not offer pickup")
	}
	// unknown value rejected
	if _, err := resolveFulfillment(CreateOrderRequest{FulfillmentType: "teleport"}, chefPickup); err == nil {
		t.Fatal("unknown fulfillment must be rejected")
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
