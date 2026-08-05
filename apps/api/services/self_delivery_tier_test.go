package services

// self_delivery_tier_test.go — the chef's published distance→fee ladder.
//
// The ladder is the price the customer pays: fixed at checkout, never re-priced
// at accept, never moved by surge, and never above the platform's ceiling for
// that distance.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func setPolicy(t *testing.T, p PlatformPolicy) {
	t.Helper()
	platformPolicyMu.Lock()
	prev := platformPolicyCache
	platformPolicyCache = &p
	platformPolicyFetchedAt = time.Now()
	platformPolicyMu.Unlock()
	t.Cleanup(func() {
		platformPolicyMu.Lock()
		platformPolicyCache = prev
		platformPolicyMu.Unlock()
	})
}

// tieredChef delivers itself from a kitchen 0 km from (12.9716, 77.5946).
func tieredChef(tiers models.DeliveryFeeTiers) models.ChefProfile {
	return models.ChefProfile{
		OffersSelfDelivery: true,
		Latitude:           12.9716,
		Longitude:          77.5946,
		SelfDeliveryTiers:  tiers.JSON(),
	}
}

func TestMaxChefDeliveryFee_UsesThePolicyCeiling(t *testing.T) {
	setPolicy(t, PlatformPolicy{
		ChefDeliveryFeeCapBase:  30,
		ChefDeliveryFeeCapPerKm: 12,
		ChefDeliveryFeeCapMax:   200,
	})
	require.Equal(t, 30.0, MaxChefDeliveryFee(0))
	require.Equal(t, 90.0, MaxChefDeliveryFee(5))
	require.Equal(t, 150.0, MaxChefDeliveryFee(10))
	// The absolute ceiling bites before the per-km line runs away.
	require.Equal(t, 200.0, MaxChefDeliveryFee(40))
}

func TestMaxChefDeliveryFee_UnconfiguredPolicyFallsBackToDefaults(t *testing.T) {
	// A zero ceiling would reject every chef's pricing and clamp every fee to 0,
	// so an unset policy must read as the shipped default, not as "no fee allowed".
	setPolicy(t, PlatformPolicy{})
	def := DefaultPlatformPolicy()
	require.Equal(t, def.ChefDeliveryFeeCapBase+def.ChefDeliveryFeeCapPerKm*5, MaxChefDeliveryFee(5))
}

func TestSelfDeliveryBreakdown_TierPricesTheBandNotTheDistance(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	chef := tieredChef(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}})
	// ~4 km away: inside the first band, so the flat ₹75 applies.
	near := ComputeSelfDeliveryFeeBreakdown(chef, 13.0056, 77.5946)
	require.True(t, near.TierApplied)
	require.Equal(t, 5.0, near.TierUpToKm)
	require.Equal(t, 75.0, near.Fee)
	require.Zero(t, near.DistanceComponent)
}

func TestSelfDeliveryBreakdown_TierOverridesTheBaseAndPerKmFormula(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	chef := tieredChef(models.DeliveryFeeTiers{{UpToKm: 8, Fee: 60}})
	// Legacy columns still populated — the ladder wins outright.
	chef.SelfDeliveryBaseFee = 39
	chef.SelfDeliveryPerKm = 10
	chef.SelfDeliveryFreeRadiusKm = 1
	chef.SelfDeliveryMaxFee = 500
	require.Equal(t, 60.0, ComputeSelfDeliveryFeeBreakdown(chef, 13.0056, 77.5946).Fee)
}

func TestSelfDeliveryBreakdown_TierIgnoresSurge(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	chef := tieredChef(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 75}})
	surged := SelfDeliveryBreakdownAt(chef, 13.0056, 77.5946, SurgeFactors{Fuel: 1.5, Combined: 1.5}, 1.5)
	require.Equal(t, 75.0, surged.Fee, "a published fee is a promise — traffic can't move it")
}

func TestSelfDeliveryBreakdown_FreeFirstBandIsFreeDelivery(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	chef := tieredChef(models.DeliveryFeeTiers{{UpToKm: 3, Fee: 0}, {UpToKm: 10, Fee: 80}})
	b := ComputeSelfDeliveryFeeBreakdown(chef, 12.9800, 77.5946)
	require.Zero(t, b.Fee)
	require.True(t, b.WithinFreeZone)
}

func TestSelfDeliveryBreakdown_UnknownDistanceChargesTheNearestBand(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	chef := tieredChef(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}})
	b := ComputeSelfDeliveryFeeBreakdown(chef, 0, 0)
	require.False(t, b.DistanceKnown)
	require.Equal(t, 75.0, b.Fee, "never bill the far band for a distance nobody measured")
}

func TestSelfDeliveryBreakdown_PlatformCeilingClampsTheLegacyFormula(t *testing.T) {
	setPolicy(t, PlatformPolicy{ChefDeliveryFeeCapBase: 30, ChefDeliveryFeeCapPerKm: 12, ChefDeliveryFeeCapMax: 500})
	chef := models.ChefProfile{
		OffersSelfDelivery:  true,
		Latitude:            12.9716,
		Longitude:           77.5946,
		SelfDeliveryBaseFee: 400,
	}
	b := ComputeSelfDeliveryFeeBreakdown(chef, 13.0056, 77.5946)
	require.True(t, b.CappedByPlatform)
	require.Equal(t, MaxChefDeliveryFee(b.DistanceKm), b.Fee)
	require.Equal(t, MaxChefDeliveryFee(b.DistanceKm), b.PlatformMaxFee)
}

func TestSelfDeliveryBreakdown_PlatformCeilingUsesTheBandDistanceForATier(t *testing.T) {
	// A 5 km band priced at ₹75 must stay ₹75 for a 0.4 km drop: the band is a
	// flat rate the chef published, not a per-order distance calculation.
	setPolicy(t, PlatformPolicy{ChefDeliveryFeeCapBase: 30, ChefDeliveryFeeCapPerKm: 12, ChefDeliveryFeeCapMax: 500})
	chef := tieredChef(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 75}})
	b := ComputeSelfDeliveryFeeBreakdown(chef, 12.9750, 77.5946)
	require.Equal(t, 75.0, b.Fee)
	require.False(t, b.CappedByPlatform)
}

func TestQuoteOrderDelivery_ReportsWhosePriceItIs(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	chef := tieredChef(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 75}})

	// The chef's own ladder priced it, so the fee is the chef's to earn.
	q := QuoteOrderDelivery(chef, models.FulfillmentDelivery, 13.0056, 77.5946, "", "IN", 1.0)
	require.Equal(t, models.DeliveryFeeSourceChef, q.Source)
	require.Equal(t, 75.0, q.Fee)

	// A kitchen that doesn't self-deliver falls back to the platform's flat fee.
	q = QuoteOrderDelivery(models.ChefProfile{}, models.FulfillmentDelivery, 13.0056, 77.5946, "", "IN", 1.0)
	require.Equal(t, models.DeliveryFeeSourcePlatform, q.Source)

	// Pickup has no delivery leg and no fee to attribute.
	q = QuoteOrderDelivery(chef, models.FulfillmentPickup, 13.0056, 77.5946, "", "IN", 1.0)
	require.Zero(t, q.Fee)
	require.Empty(t, q.Source)
}

func TestValidateChefDeliveryTiers_AgainstTheLivePolicyCeiling(t *testing.T) {
	setPolicy(t, PlatformPolicy{ChefDeliveryFeeCapBase: 30, ChefDeliveryFeeCapPerKm: 12, ChefDeliveryFeeCapMax: 300})
	require.NoError(t, ValidateChefDeliveryTiers(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}}))
	require.Error(t, ValidateChefDeliveryTiers(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 91}}))
}

func TestSelfDeliveryBreakdown_PlatformCeilingBoundsSurge(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	chef := models.ChefProfile{
		OffersSelfDelivery: true,
		Latitude:           12.90, Longitude: 77.50,
		SelfDeliveryBaseFee: 20, SelfDeliveryPerKm: 10,
	}
	b := SelfDeliveryBreakdownAt(chef, 12.97, 77.59, SurgeFactors{Fuel: 1.5, Combined: 1.5}, 1.5)
	require.Equal(t, MaxChefDeliveryFee(b.DistanceKm), b.Fee, "surge can't push a fee past the platform ceiling")
}

func TestDeliveryFeeFrom_TieredChefQuotesTheFirstBand(t *testing.T) {
	setPolicy(t, DefaultPlatformPolicy())
	fee, flat := DeliveryFeeFrom(tieredChef(models.DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}}))
	require.Equal(t, 75.0, fee)
	require.False(t, flat, "it rises with distance, so it is a floor not a promise")

	// One band covering everything the chef delivers IS a flat promise.
	fee, flat = DeliveryFeeFrom(tieredChef(models.DeliveryFeeTiers{{UpToKm: 10, Fee: 40}}))
	require.Equal(t, 40.0, fee)
	require.True(t, flat)
}
