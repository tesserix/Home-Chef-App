package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The breakdown is what the customer sees at checkout (#702) — an itemised,
// CAPPED estimate. Its .Fee must equal ComputeSelfDeliveryFee exactly (they share
// one code path), and each component must add up to the final fee so the UI can
// render an honest receipt-style breakdown.
func TestComputeSelfDeliveryFeeBreakdown(t *testing.T) {
	const chefLat, chefLng = 12.9716, 77.5946
	const dropLat, dropLng = 12.9352, 77.6245 // ~5.2 km straight → ~6.7 km road

	t.Run("distance beyond free radius itemises correctly", func(t *testing.T) {
		chef := models.ChefProfile{
			Latitude: chefLat, Longitude: chefLng, OffersSelfDelivery: true,
			SelfDeliveryBaseFee: 20, SelfDeliveryFreeRadiusKm: 2, SelfDeliveryPerKm: 10,
		}
		b := ComputeSelfDeliveryFeeBreakdown(chef, dropLat, dropLng)

		require.InDelta(t, 6.7, b.DistanceKm, 1.0, "road distance")
		require.Equal(t, 20.0, b.BaseFee)
		require.Equal(t, 2.0, b.FreeRadiusKm)
		require.InDelta(t, b.DistanceKm-2, b.BillableKm, 1e-9, "billable = dist − free radius")
		require.InDelta(t, b.BillableKm*10, b.DistanceComponent, 1e-9)
		require.InDelta(t, b.BaseFee+b.DistanceComponent, b.Fee, 1e-9, "components sum to fee")
		require.False(t, b.Capped)
		require.False(t, b.WithinFreeZone)
		// The single-number path must agree to the rounded rupee.
		require.InDelta(t, ComputeSelfDeliveryFee(chef, dropLat, dropLng), b.Fee, 1e-9)
	})

	t.Run("within free radius → within-free-zone, no distance component", func(t *testing.T) {
		chef := models.ChefProfile{
			Latitude: chefLat, Longitude: chefLng, OffersSelfDelivery: true,
			SelfDeliveryBaseFee: 0, SelfDeliveryFreeRadiusKm: 50, SelfDeliveryPerKm: 10,
		}
		b := ComputeSelfDeliveryFeeBreakdown(chef, dropLat, dropLng)
		require.True(t, b.WithinFreeZone, "drop is inside the chef's free radius")
		require.Equal(t, 0.0, b.DistanceComponent)
		require.Equal(t, 0.0, b.Fee, "base 0 + free zone → free delivery")
	})

	t.Run("cap flags when the max fee bites", func(t *testing.T) {
		chef := models.ChefProfile{
			Latitude: chefLat, Longitude: chefLng, OffersSelfDelivery: true,
			SelfDeliveryBaseFee: 20, SelfDeliveryPerKm: 100, SelfDeliveryMaxFee: 50,
		}
		b := ComputeSelfDeliveryFeeBreakdown(chef, dropLat, dropLng)
		require.True(t, b.Capped, "uncapped fee exceeds the max")
		require.Equal(t, 50.0, b.Fee)
		require.Equal(t, 50.0, b.MaxFee)
	})

	t.Run("missing coords → base only, distance unknown", func(t *testing.T) {
		chef := models.ChefProfile{SelfDeliveryBaseFee: 25, SelfDeliveryFreeRadiusKm: 1, SelfDeliveryPerKm: 10}
		b := ComputeSelfDeliveryFeeBreakdown(chef, dropLat, dropLng)
		require.False(t, b.DistanceKnown)
		require.Equal(t, 0.0, b.DistanceComponent)
		require.Equal(t, 25.0, b.Fee)
	})
}
