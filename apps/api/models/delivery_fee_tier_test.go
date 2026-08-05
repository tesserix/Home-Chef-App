package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ceiling used across these tests: ₹30 at the door + ₹12/km, which is the
// platform default shape (services.MaxChefDeliveryFee).
func testCeiling(km float64) float64 { return 30 + 12*km }

func TestParseDeliveryFeeTiers_SortsAndDropsJunk(t *testing.T) {
	tiers := ParseDeliveryFeeTiers(`[{"upToKm":10,"fee":150},{"upToKm":5,"fee":75},{"upToKm":0,"fee":20}]`)
	require.Equal(t, DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}}, tiers)
}

func TestParseDeliveryFeeTiers_EmptyAndInvalidAreNoLadder(t *testing.T) {
	require.Empty(t, ParseDeliveryFeeTiers(""))
	require.Empty(t, ParseDeliveryFeeTiers("[]"))
	require.Empty(t, ParseDeliveryFeeTiers("not json"))
}

func TestDeliveryFeeTiers_JSONRoundTrip(t *testing.T) {
	in := DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}}
	require.Equal(t, in, ParseDeliveryFeeTiers(in.JSON()))
}

func TestDeliveryFeeTiers_FeeForKmPicksTheFirstBandThatCovers(t *testing.T) {
	tiers := DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}}

	for _, tc := range []struct {
		km   float64
		want float64
	}{
		{0, 75},   // at the door still pays the first band — it is a flat rate
		{4.9, 75}, // inside the first band
		{5, 75},   // the boundary belongs to the band it names
		{5.1, 150},
		{10, 150},
		{12, 150}, // beyond the ladder the top band holds — never an unpriced order
	} {
		tier, ok := tiers.TierForKm(tc.km)
		require.True(t, ok, "km=%v", tc.km)
		require.Equal(t, tc.want, tier.Fee, "km=%v", tc.km)
	}
}

func TestDeliveryFeeTiers_TierForKmWithoutLadder(t *testing.T) {
	_, ok := DeliveryFeeTiers{}.TierForKm(3)
	require.False(t, ok)
}

func TestDeliveryFeeTiers_ValidateAcceptsAWellFormedLadder(t *testing.T) {
	tiers := DeliveryFeeTiers{{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 150}}
	require.NoError(t, tiers.Validate(testCeiling))
	require.NoError(t, DeliveryFeeTiers{}.Validate(testCeiling))
	// A free first band is legitimate — that is the chef's free-delivery zone.
	require.NoError(t, DeliveryFeeTiers{{UpToKm: 2, Fee: 0}, {UpToKm: 8, Fee: 60}}.Validate(testCeiling))
}

func TestDeliveryFeeTiers_ValidateRejectsAFeeOverThePlatformCeiling(t *testing.T) {
	// 5 km allows ₹90; ₹120 is the chef charging beyond the threshold.
	err := DeliveryFeeTiers{{UpToKm: 5, Fee: 120}}.Validate(testCeiling)
	require.ErrorContains(t, err, "5")
	require.ErrorContains(t, err, "90")
}

func TestDeliveryFeeTiers_ValidateRejectsMalformedLadders(t *testing.T) {
	for name, tiers := range map[string]DeliveryFeeTiers{
		"zero distance":     {{UpToKm: 0, Fee: 20}},
		"negative fee":      {{UpToKm: 5, Fee: -1}},
		"duplicate band":    {{UpToKm: 5, Fee: 40}, {UpToKm: 5, Fee: 75}},
		"cheaper when far":  {{UpToKm: 5, Fee: 75}, {UpToKm: 10, Fee: 50}},
		"beyond max radius": {{UpToKm: MaxDeliveryTierKm + 1, Fee: 10}},
	} {
		require.Error(t, tiers.Validate(testCeiling), name)
	}

	tooMany := make(DeliveryFeeTiers, 0, MaxDeliveryFeeTiers+1)
	for i := 1; i <= MaxDeliveryFeeTiers+1; i++ {
		tooMany = append(tooMany, DeliveryFeeTier{UpToKm: float64(i), Fee: float64(i)})
	}
	require.ErrorContains(t, tooMany.Validate(testCeiling), "bands")
}
