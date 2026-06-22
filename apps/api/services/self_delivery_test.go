package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestComputeSelfDeliveryFee(t *testing.T) {
	// Chef in central Bangalore; drop ~3.9 km away.
	const chefLat, chefLng = 12.9716, 77.5946
	const dropLat, dropLng = 12.9352, 77.6245 // ~4.4 km

	cases := []struct {
		name string
		chef models.ChefProfile
		want float64 // approx; distance-based cases use a tolerance
		tol  float64
	}{
		{
			name: "free self-delivery (all zero)",
			chef: models.ChefProfile{Latitude: chefLat, Longitude: chefLng, OffersSelfDelivery: true},
			want: 0, tol: 0,
		},
		{
			name: "flat base fee only",
			chef: models.ChefProfile{Latitude: chefLat, Longitude: chefLng, SelfDeliveryBaseFee: 30},
			want: 30, tol: 0,
		},
		{
			name: "free radius covers the whole trip",
			chef: models.ChefProfile{Latitude: chefLat, Longitude: chefLng, SelfDeliveryBaseFee: 20, SelfDeliveryFreeRadiusKm: 50, SelfDeliveryPerKm: 10},
			want: 20, tol: 0,
		},
		{
			name: "distance-based beyond free radius",
			// base 20 + (~4.4km − 2km free) × 10 ≈ 20 + ~24 = ~44
			chef: models.ChefProfile{Latitude: chefLat, Longitude: chefLng, SelfDeliveryBaseFee: 20, SelfDeliveryFreeRadiusKm: 2, SelfDeliveryPerKm: 10},
			want: 44, tol: 8,
		},
		{
			name: "capped at max fee",
			chef: models.ChefProfile{Latitude: chefLat, Longitude: chefLng, SelfDeliveryBaseFee: 20, SelfDeliveryFreeRadiusKm: 0, SelfDeliveryPerKm: 100, SelfDeliveryMaxFee: 50},
			want: 50, tol: 0,
		},
		{
			name: "missing chef coords → base only (distance unknown)",
			chef: models.ChefProfile{SelfDeliveryBaseFee: 25, SelfDeliveryFreeRadiusKm: 1, SelfDeliveryPerKm: 10},
			want: 25, tol: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeSelfDeliveryFee(tc.chef, dropLat, dropLng)
			if diff := got - tc.want; diff < -tc.tol || diff > tc.tol {
				t.Errorf("ComputeSelfDeliveryFee = %.2f, want %.2f (±%.2f)", got, tc.want, tc.tol)
			}
		})
	}
}
