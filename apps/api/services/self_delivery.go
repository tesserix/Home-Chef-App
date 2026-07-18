package services

import "github.com/homechef/api/models"

// ComputeSelfDeliveryFee returns the chef self-delivery fee for an order going
// to (dropLat, dropLng):
//
//	fee = BaseFee + max(0, distanceKm − FreeRadiusKm) × PerKm   (capped at MaxFee)
//
// MaxFee of 0 means uncapped. When either endpoint's coords are missing (0,0)
// the distance is unknown, so only the flat BaseFee applies. Never negative.
// Deterministic + pure so the checkout quote and any later re-quote agree.
func ComputeSelfDeliveryFee(chef models.ChefProfile, dropLat, dropLng float64) float64 {
	fee := chef.SelfDeliveryBaseFee
	if chef.Latitude != 0 && chef.Longitude != 0 && dropLat != 0 && dropLng != 0 {
		// Road distance, not straight line (#701): the chef drives roads, so the
		// per-km fee should reflect the driven distance. RoadDistanceKm uses a real
		// router when configured, else a winding-factor fallback — never blocks.
		distKm := RoadDistanceKm(chef.Latitude, chef.Longitude, dropLat, dropLng)
		if extra := distKm - chef.SelfDeliveryFreeRadiusKm; extra > 0 {
			fee += extra * chef.SelfDeliveryPerKm
		}
	}
	if chef.SelfDeliveryMaxFee > 0 && fee > chef.SelfDeliveryMaxFee {
		fee = chef.SelfDeliveryMaxFee
	}
	if fee < 0 {
		fee = 0
	}
	return fee
}

// ComputeSelfDeliveryDistanceKm returns the chef→drop straight-line distance in
// km, or 0 when either endpoint's coords are missing (distance unknown). Uses
// the same haversine + coords as ComputeSelfDeliveryFee so the fee quote and the
// vendor's distance warning can never disagree.
func ComputeSelfDeliveryDistanceKm(chef models.ChefProfile, dropLat, dropLng float64) float64 {
	if chef.Latitude == 0 || chef.Longitude == 0 || dropLat == 0 || dropLng == 0 {
		return 0
	}
	return haversineDistance(chef.Latitude, chef.Longitude, dropLat, dropLng)
}
