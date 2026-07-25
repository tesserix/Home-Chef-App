package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/models"
)

func ctxWithEmail(email string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if email != "" {
		c.Set("userEmail", email)
	}
	return c
}

func TestViewerEmail(t *testing.T) {
	if got := viewerEmail(ctxWithEmail("a@b.com")); got != "a@b.com" {
		t.Fatalf("got %q, want a@b.com", got)
	}
	if got := viewerEmail(ctxWithEmail("")); got != "" {
		t.Fatalf("an anonymous caller must yield empty, got %q", got)
	}
}

func TestAssertMayOrderFromChef(t *testing.T) {
	now := time.Now()
	born := &models.ChefProfile{Mode: models.ChefModeTest}
	flipped := &models.ChefProfile{Mode: models.ChefModeTest, FirstLiveAt: &now}
	live := &models.ChefProfile{Mode: models.ChefModeLive}

	if err := assertMayOrderFromChef(ctxWithEmail("stranger@x.com"), born); err == nil {
		t.Fatal("a stranger must not be able to order from a born-test kitchen")
	}
	// Closed means closed: a kitchen flipped to test for debugging must not take
	// real orders from its regulars while it is being experimented on.
	if err := assertMayOrderFromChef(ctxWithEmail("stranger@x.com"), flipped); err == nil {
		t.Fatal("a stranger must not be able to order from a kitchen flipped to test")
	}
	if err := assertMayOrderFromChef(ctxWithEmail(""), born); err == nil {
		t.Fatal("an anonymous caller must never reach a test kitchen")
	}
	if err := assertMayOrderFromChef(ctxWithEmail("samyak.rout@gmail.com"), born); err != nil {
		t.Fatalf("a tester must be able to order from a test kitchen: %v", err)
	}
	// The allowlist grants visibility of fake kitchens, not discounts on real
	// ones. A tester ordering from a live chef is an ordinary paying customer.
	if err := assertMayOrderFromChef(ctxWithEmail("samyak.rout@gmail.com"), live); err != nil {
		t.Fatalf("a tester ordering from a live kitchen must be unaffected: %v", err)
	}
	if err := assertMayOrderFromChef(ctxWithEmail(""), live); err != nil {
		t.Fatalf("anonymous checkout against a live kitchen must be unaffected: %v", err)
	}
}

// The reduced payload must carry identity and nothing that could be acted on.
func TestClosedResponseLeaksNoCommerceFields(t *testing.T) {
	chef := &models.ChefProfile{
		BusinessName: "Amma ka Kitchen", City: "Bengaluru",
		MinimumOrder: 250, Rating: 4.8, TotalOrders: 900,
		AcceptingOrders: true, OffersPickup: true, OffersSelfDelivery: true,
	}
	r := chef.ToClosedResponse()

	if r.BusinessName != "Amma ka Kitchen" || r.City != "Bengaluru" {
		t.Fatal("the closed payload must keep the kitchen's identity")
	}
	if r.AcceptingOrders || r.IsOnline {
		t.Fatal("a kitchen in test mode must never read as open to its customers")
	}
	if r.MinimumOrder != 0 || r.Rating != 0 || r.TotalOrders != 0 {
		t.Fatalf("commerce fields must be absent from the closed payload: %+v", r)
	}
	if r.OffersPickup || r.OffersSelfDelivery || r.OffersDelivery {
		t.Fatal("no fulfilment option may be advertised by a closed kitchen")
	}
}
