package services

import (
	"math"
	"testing"
)

// haversineMeters returns the great-circle distance between two points.
func haversineMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371000.0
	p1 := lat1 * math.Pi / 180
	p2 := lat2 * math.Pi / 180
	dp := (lat2 - lat1) * math.Pi / 180
	dl := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return r * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func TestFuzzCoordinate_ZeroPassthrough(t *testing.T) {
	lat, lng := FuzzCoordinate(0, 0, "chef-1")
	if lat != 0 || lng != 0 {
		t.Fatalf("expected (0,0) sentinel to pass through, got (%v,%v)", lat, lng)
	}
}

func TestFuzzCoordinate_Deterministic(t *testing.T) {
	// Mumbai-ish point.
	const lat, lng = 19.149911, 72.7967402
	a1, b1 := FuzzCoordinate(lat, lng, "chef-1")
	a2, b2 := FuzzCoordinate(lat, lng, "chef-1")
	if a1 != a2 || b1 != b2 {
		t.Fatalf("same seed must be stable: (%v,%v) != (%v,%v)", a1, b1, a2, b2)
	}
}

func TestFuzzCoordinate_DifferentSeedsDiffer(t *testing.T) {
	const lat, lng = 19.149911, 72.7967402
	a1, b1 := FuzzCoordinate(lat, lng, "chef-1")
	a2, b2 := FuzzCoordinate(lat, lng, "chef-2")
	if a1 == a2 && b1 == b2 {
		t.Fatal("different seeds should produce different offsets")
	}
}

func TestFuzzCoordinate_OffsetIsApprox300m(t *testing.T) {
	const lat, lng = 19.149911, 72.7967402
	// The true point must NOT be the centre of the shown area — it should sit a
	// fixed ~300m away (so it can't be recovered as the circle centre).
	for _, seed := range []string{"chef-1", "chef-2", "chef-abc", "0191"} {
		fLat, fLng := FuzzCoordinate(lat, lng, seed)
		d := haversineMeters(lat, lng, fLat, fLng)
		if d < 280 || d > 320 {
			t.Fatalf("seed %q: offset %.1fm outside expected ~300m band", seed, d)
		}
	}
}
