package services

// delivery_distance.go — road-distance abstraction for delivery pricing (#701,
// part of #699).
//
// The self-delivery fee was computed from haversineDistance — the straight line
// between chef and drop. Real driving is longer (roads wind, one-ways, rivers),
// so a straight-line fee under-pays the chef for the distance actually driven.
//
// This introduces one seam for "how far is the drive". A real router (OSRM /
// Google Distance Matrix, chosen in #700) plugs in behind DeliveryRouter and
// returns the driven distance; until then a winding-factor fallback scales the
// haversine — a genuine improvement with no external key or cost. Either way
// callers use RoadDistanceKm, so the fee path has a single distance source and
// the checkout preview and the charge can never disagree.

import "context"

// DeliveryRouter estimates the ROAD distance between two coordinates. Optional:
// when none is installed, RoadDistanceKm uses the fallback below.
type DeliveryRouter interface {
	// RoadDistanceKm returns the driving distance in km. ok=false means the
	// provider couldn't answer (out of coverage, timeout, quota) — the caller
	// then falls back, never blocks.
	RoadDistanceKm(ctx context.Context, fromLat, fromLng, toLat, toLng float64) (km float64, ok bool)
}

// deliveryRouter is the installed provider, or nil for fallback-only. Package
// global (not injected) to match the existing provider/geocode services; swapped
// via SetDeliveryRouter in main wiring and in tests.
var deliveryRouter DeliveryRouter

// SetDeliveryRouter installs (or clears, with nil) the road-distance provider.
func SetDeliveryRouter(r DeliveryRouter) { deliveryRouter = r }

// RoadDistanceFactor scales straight-line distance to approximate road distance
// when no router answers. ~1.3 is a common urban winding factor (the median
// road/straight-line ratio in dense cities). Exported so config (#700) or a test
// can tune it. Never below 1 — roads are never shorter than the straight line.
var RoadDistanceFactor = 1.3

// RoadDistanceKm returns the estimated road distance in km between two points.
// It NEVER fails: a configured router is used when it answers, otherwise the
// haversine × RoadDistanceFactor fallback — so the fee path always has a number
// and checkout never stalls on a routing call.
func RoadDistanceKm(fromLat, fromLng, toLat, toLng float64) float64 {
	if deliveryRouter != nil {
		if km, ok := deliveryRouter.RoadDistanceKm(context.Background(), fromLat, fromLng, toLat, toLng); ok && km >= 0 {
			return km
		}
	}
	factor := RoadDistanceFactor
	if factor < 1 {
		factor = 1 // a road is never shorter than the straight line
	}
	return haversineDistance(fromLat, fromLng, toLat, toLng) * factor
}
