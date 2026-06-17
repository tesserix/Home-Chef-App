package handlers

// chefs_search_test.go — the unit-testable core of the near-me filter (#36).
// ListChefs / SearchDishes themselves run Postgres-only SQL (NOW(), ANY(),
// ILIKE) so they're exercised in CI against Postgres, not the sqlite harness;
// the bounding-box math behind "near me" is pure and worth pinning down.

import (
	"math"
	"testing"
)

func TestChefBoundingBox(t *testing.T) {
	// Bangalore-ish coords, 15 km radius.
	lat, lng := 12.9716, 77.5946
	minLat, maxLat, minLng, maxLng := chefBoundingBox(lat, lng, 15)

	// Latitude span ≈ 2 * radius/111km.
	wantLatSpan := 2 * 15.0 / 111.0
	if math.Abs((maxLat-minLat)-wantLatSpan) > 0.001 {
		t.Fatalf("lat span = %v, want ~%v", maxLat-minLat, wantLatSpan)
	}

	// The center is inside its own box.
	if lat < minLat || lat > maxLat || lng < minLng || lng > maxLng {
		t.Fatal("center must be inside its own bounding box")
	}

	// A point ~10 km north is inside the 15 km box; ~25 km north is outside.
	inside := lat + 10.0/111.0
	outside := lat + 25.0/111.0
	if inside < minLat || inside > maxLat {
		t.Fatal("a point 10km north must be inside the 15km box")
	}
	if outside >= minLat && outside <= maxLat {
		t.Fatal("a point 25km north must be outside the 15km box")
	}

	// At this latitude longitude degrees are slightly shorter, so the longitude
	// half-width must exceed the latitude half-width (cos(lat) < 1).
	if (maxLng - minLng) <= (maxLat - minLat) {
		t.Fatalf("lng span (%v) should exceed lat span (%v) at lat %v",
			maxLng-minLng, maxLat-minLat, lat)
	}
}

func TestChefBoundingBox_PolarGuard(t *testing.T) {
	// Near the pole cos(lat)→0; the guard must keep the box finite (no div-by-0).
	minLat, maxLat, minLng, maxLng := chefBoundingBox(89.9, 0, 10)
	if math.IsInf(minLng, 0) || math.IsInf(maxLng, 0) || maxLat <= minLat {
		t.Fatalf("polar box must stay finite, got lat[%v,%v] lng[%v,%v]", minLat, maxLat, minLng, maxLng)
	}
}
