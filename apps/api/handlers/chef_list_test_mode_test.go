package handlers

import (
	"testing"

	"github.com/homechef/api/models"
)

func testChef(name, mode string) models.ChefProfile {
	return models.ChefProfile{
		BusinessName: name, Mode: mode, City: "Bengaluru",
		MinimumOrder: 199, Rating: 4.6, AcceptingOrders: true,
		OffersPickup: true, OffersSelfDelivery: true,
	}
}

// The list is where a customer first meets a kitchen, so it is where the
// visibility rule has to bite. A card that still reads "Open" is a card they
// will tap.
func TestListPresentationClosesTestKitchensOutsideTheAllowlist(t *testing.T) {
	chefs := []models.ChefProfile{
		testChef("Saffron Home Kitchen", models.ChefModeTest),
		testChef("Amma ka Kitchen", models.ChefModeLive),
	}
	responses := []models.ChefProfileResponse{chefs[0].ToResponse(), chefs[1].ToResponse()}

	applyTestModePresentation(chefs, responses, false)

	if responses[0].AcceptingOrders || responses[0].IsOnline {
		t.Fatal("a test kitchen must not read as open to a customer outside the allowlist")
	}
	if responses[0].UnavailableMessage != models.ClosedForMaintenanceMessage {
		t.Fatalf("the closed card must say why it cannot be ordered from: %+v", responses[0])
	}
	if responses[0].TestMode {
		t.Fatal("test status must never be disclosed outside the allowlist")
	}
	if responses[0].BusinessName != "Saffron Home Kitchen" {
		t.Fatal("the closed card keeps the kitchen's name — it is not hidden, it is disabled")
	}
	if !responses[1].AcceptingOrders || responses[1].TestMode {
		t.Fatalf("a live kitchen must be untouched: %+v", responses[1])
	}
}

// An allowlisted tester needs the kitchen fully orderable — that is the whole
// point of the allowlist — plus a flag so the app can mark what it is.
func TestListPresentationBadgesTestKitchensForTheAllowlist(t *testing.T) {
	chefs := []models.ChefProfile{
		testChef("Saffron Home Kitchen", models.ChefModeTest),
		testChef("Amma ka Kitchen", models.ChefModeLive),
	}
	responses := []models.ChefProfileResponse{chefs[0].ToResponse(), chefs[1].ToResponse()}

	applyTestModePresentation(chefs, responses, true)

	if !responses[0].TestMode {
		t.Fatal("a tester must be told which kitchen is the test one")
	}
	if !responses[0].AcceptingOrders || !responses[0].IsOnline {
		t.Fatal("a tester must still be able to order from the test kitchen")
	}
	if responses[0].UnavailableMessage != "" {
		t.Fatal("a tester sees the open kitchen, not the maintenance copy")
	}
	if responses[1].TestMode {
		t.Fatal("a live kitchen is never badged as test")
	}
}

// Enrichment runs over the whole page before this does; a short response slice
// must not panic the listing.
func TestListPresentationToleratesAShortResponseSlice(t *testing.T) {
	chefs := []models.ChefProfile{testChef("Saffron Home Kitchen", models.ChefModeTest)}

	applyTestModePresentation(chefs, nil, false)
}
