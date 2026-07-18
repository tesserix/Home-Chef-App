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
			name: "distance-based beyond free radius (ROAD distance, #701)",
			// ROAD distance = haversine (~5.2 km) × 1.3 winding factor ≈ 6.7 km.
			// base 20 + (6.7 − 2 free) × 10 ≈ 20 + 47 = ~67. The road factor makes
			// this ~30% higher than the old straight-line ~52 — the intended fix so
			// the chef is paid for the distance actually driven.
			chef: models.ChefProfile{Latitude: chefLat, Longitude: chefLng, SelfDeliveryBaseFee: 20, SelfDeliveryFreeRadiusKm: 2, SelfDeliveryPerKm: 10},
			want: 67, tol: 4,
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

func TestComputeSelfDeliveryDistanceKm(t *testing.T) {
	const chefLat, chefLng = 12.9716, 77.5946
	const dropLat, dropLng = 12.9352, 77.6245 // ~4.4 km from the chef

	cases := []struct {
		name             string
		chef             models.ChefProfile
		dropLat, dropLng float64
		want             float64 // 0 means "unknown / exact zero"
		tol              float64
	}{
		{
			name:    "known coords → straight-line distance",
			chef:    models.ChefProfile{Latitude: chefLat, Longitude: chefLng},
			dropLat: dropLat, dropLng: dropLng,
			want: 4.4, tol: 1.0,
		},
		{
			name:    "missing chef coords → 0 (unknown)",
			chef:    models.ChefProfile{},
			dropLat: dropLat, dropLng: dropLng,
			want: 0, tol: 0,
		},
		{
			name:    "missing drop coords → 0 (unknown)",
			chef:    models.ChefProfile{Latitude: chefLat, Longitude: chefLng},
			dropLat: 0, dropLng: 0,
			want: 0, tol: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeSelfDeliveryDistanceKm(tc.chef, tc.dropLat, tc.dropLng)
			if diff := got - tc.want; diff < -tc.tol || diff > tc.tol {
				t.Errorf("ComputeSelfDeliveryDistanceKm = %.2f, want %.2f (±%.2f)", got, tc.want, tc.tol)
			}
		})
	}
}
