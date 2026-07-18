package services

// delivery_fee_test.go — the quote MUST equal the charge.
//
// These pin the per-mode fee so the checkout preview and CreateOrder can never
// disagree: pickup is always free (the whole incentive), chef self-delivery is
// the distance fee, and 3PL delivery falls back to the flat policy fee when there
// is no live quote.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// setPolicyBaseDeliveryFee overrides the cached platform policy for a test and
// restores it after. Writes the cache directly (same package) so no DB is needed.
func setPolicyBaseDeliveryFee(t *testing.T, fee float64) {
	t.Helper()
	platformPolicyMu.Lock()
	prev := platformPolicyCache
	p := PlatformPolicy{BaseDeliveryFee: fee}
	platformPolicyCache = &p
	platformPolicyFetchedAt = time.Now()
	platformPolicyMu.Unlock()
	t.Cleanup(func() {
		platformPolicyMu.Lock()
		platformPolicyCache = prev
		platformPolicyMu.Unlock()
	})
}

func TestQuoteOrderDeliveryFee_PickupIsAlwaysFree(t *testing.T) {
	// A chef with a fat self-delivery fee configured — pickup must STILL be 0,
	// because pickup has no delivery leg. This zero is the saving the incentive
	// shows, so it must never leak a fee.
	chef := models.ChefProfile{
		Latitude: 18.5, Longitude: 73.8,
		SelfDeliveryBaseFee: 40, SelfDeliveryPerKm: 10,
	}
	require.Equal(t, 0.0,
		QuoteOrderDeliveryFee(chef, models.FulfillmentPickup, 18.6, 73.9, "Pune", "IN"),
		"pickup has no delivery leg — the fee must be exactly 0, whatever the chef charges to deliver")
}

func TestQuoteOrderDeliveryFee_ChefDelivery_UsesTheSelfDeliveryFee(t *testing.T) {
	chef := models.ChefProfile{
		Latitude: 18.5, Longitude: 73.8,
		SelfDeliveryBaseFee: 30, SelfDeliveryFreeRadiusKm: 0, SelfDeliveryPerKm: 0,
	}
	// Same coords → 0 distance → just the base fee. Delegates to
	// ComputeSelfDeliveryFee, which has its own distance tests; here we only pin
	// that chef_delivery routes to it.
	require.Equal(t, 30.0,
		QuoteOrderDeliveryFee(chef, models.FulfillmentChefDelivery, 18.5, 73.8, "Pune", "IN"))
}

func TestQuoteOrderDeliveryFee_Delivery_FallsBackToPolicyFeeWithoutCoords(t *testing.T) {
	// No drop coords → the 3PL quote is skipped (it needs coords) → the flat
	// platform fee. This is exactly CreateOrder's fallback, so checkout shows the
	// same number that will be charged.
	setPolicyBaseDeliveryFee(t, 25)
	chef := models.ChefProfile{Latitude: 18.5, Longitude: 73.8}

	require.Equal(t, 25.0,
		QuoteOrderDeliveryFee(chef, models.FulfillmentDelivery, 0, 0, "Pune", "IN"),
		"with no coordinates the fee is the flat policy fee — the same fallback CreateOrder uses")
}

// The incentive is only honest when it reflects a REAL saving: pickup free while
// delivery costs the policy fee. When delivery is free (policy fee 0), pickup
// saves nothing and there is nothing to advertise — which is correct, not a bug.
func TestQuoteOrderDeliveryFee_SavingIsRealOnlyWhenDeliveryCosts(t *testing.T) {
	chef := models.ChefProfile{Latitude: 18.5, Longitude: 73.8}

	setPolicyBaseDeliveryFee(t, 0)
	delFree := QuoteOrderDeliveryFee(chef, models.FulfillmentDelivery, 0, 0, "Pune", "IN")
	pickFree := QuoteOrderDeliveryFee(chef, models.FulfillmentPickup, 0, 0, "Pune", "IN")
	require.Equal(t, delFree, pickFree, "delivery free → pickup saves nothing; do not fabricate a saving")

	setPolicyBaseDeliveryFee(t, 30)
	delPaid := QuoteOrderDeliveryFee(chef, models.FulfillmentDelivery, 0, 0, "Pune", "IN")
	pickPaid := QuoteOrderDeliveryFee(chef, models.FulfillmentPickup, 0, 0, "Pune", "IN")
	require.Equal(t, 30.0, delPaid-pickPaid, "delivery costs ₹30 → picking up saves exactly ₹30")
}
