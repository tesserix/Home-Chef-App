package services

// delivery_distance_test.go — #701. RoadDistanceKm is the single distance source
// for the delivery fee, so its two behaviours matter: it scales straight-line up
// to a realistic road distance by default, and it uses a real router when one is
// installed but never blocks on it (falls back if the router can't answer).

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// A fake router that answers a fixed distance, or refuses (ok=false).
type fakeRouter struct {
	km      float64
	ok      bool
	calls   int
	lastLat float64
}

func (f *fakeRouter) RoadDistanceKm(_ context.Context, fromLat, _, _, _ float64) (float64, bool) {
	f.calls++
	f.lastLat = fromLat
	return f.km, f.ok
}

func TestRoadDistanceKm_FallbackScalesHaversine(t *testing.T) {
	SetDeliveryRouter(nil)
	prev := RoadDistanceFactor
	RoadDistanceFactor = 1.3
	t.Cleanup(func() { RoadDistanceFactor = prev })

	// Bangalore chef → ~4.4 km drop.
	const chefLat, chefLng = 12.9716, 77.5946
	const dropLat, dropLng = 12.9352, 77.6245

	hav := haversineDistance(chefLat, chefLng, dropLat, dropLng)
	road := RoadDistanceKm(chefLat, chefLng, dropLat, dropLng)

	require.InEpsilon(t, hav*1.3, road, 0.001,
		"with no router the road distance is haversine × the winding factor — the fair-distance improvement over straight line")
	require.Greater(t, road, hav, "a road is never shorter than the straight line")
}

func TestRoadDistanceKm_FactorNeverBelowOne(t *testing.T) {
	SetDeliveryRouter(nil)
	prev := RoadDistanceFactor
	RoadDistanceFactor = 0.5 // nonsensical — a road can't be shorter than the line
	t.Cleanup(func() { RoadDistanceFactor = prev })

	const a, b, c, d = 12.9716, 77.5946, 12.9352, 77.6245
	require.GreaterOrEqual(t, RoadDistanceKm(a, b, c, d), haversineDistance(a, b, c, d),
		"a sub-1 factor must be clamped — never quote less than the straight-line distance")
}

func TestRoadDistanceKm_UsesRouterWhenItAnswers(t *testing.T) {
	r := &fakeRouter{km: 7.2, ok: true}
	SetDeliveryRouter(r)
	t.Cleanup(func() { SetDeliveryRouter(nil) })

	got := RoadDistanceKm(12.9716, 77.5946, 12.9352, 77.6245)
	require.Equal(t, 7.2, got, "a configured router's driven distance is authoritative")
	require.Equal(t, 1, r.calls)
}

func TestRoadDistanceKm_FallsBackWhenRouterRefuses(t *testing.T) {
	r := &fakeRouter{ok: false} // out of coverage / timeout / quota
	SetDeliveryRouter(r)
	t.Cleanup(func() { SetDeliveryRouter(nil) })

	const a, b, c, d = 12.9716, 77.5946, 12.9352, 77.6245
	got := RoadDistanceKm(a, b, c, d)
	require.InEpsilon(t, haversineDistance(a, b, c, d)*RoadDistanceFactor, got, 0.001,
		"a router that can't answer must never block the fee — fall back to the factor")
	require.Equal(t, 1, r.calls)
}

// A negative distance from a misbehaving provider is rejected, not trusted.
func TestRoadDistanceKm_RejectsNegativeRouterResult(t *testing.T) {
	r := &fakeRouter{km: -3, ok: true}
	SetDeliveryRouter(r)
	t.Cleanup(func() { SetDeliveryRouter(nil) })

	const a, b, c, d = 12.9716, 77.5946, 12.9352, 77.6245
	require.Greater(t, RoadDistanceKm(a, b, c, d), 0.0, "a negative provider distance must fall back, not underflow the fee")
}
