package services

import (
	"crypto/sha256"
	"encoding/binary"
	"math"
)

// approxOffsetMeters is how far the displayed ("approximate area") point sits
// from the true location. Combined with the ~400m area circle the customer app
// draws around it, the true address is never the centre of the shown area and
// cannot be pin-pointed. Small enough that the area still reads as "nearby".
const approxOffsetMeters = 300.0

// metersPerDegreeLat is constant; longitude degrees shrink with latitude and are
// scaled by cos(lat) below.
const metersPerDegreeLat = 111320.0

// FuzzCoordinate returns an approximate location for a true (lat, lng): a point
// offset by a fixed ~300m in a deterministic direction derived from seed. It is
// stable for a given seed (so the same chef shows the same area on every screen
// and across reloads — no shifting circle), but the true point is never the
// centre, so it can't be recovered from the displayed value.
//
// Used for CUSTOMER-facing chef locations only. The 3PL delivery rider still
// receives the exact address server-to-server; this is purely to stop customers
// and chefs pin-pointing each other and arranging off-platform contact.
//
// (0,0) — the "no coordinates yet" sentinel used across this codebase — is
// passed through unchanged so the client keeps treating it as "no location".
func FuzzCoordinate(lat, lng float64, seed string) (float64, float64) {
	if lat == 0 && lng == 0 {
		return 0, 0
	}

	// Deterministic direction in [0, 2π) from the seed hash.
	h := sha256.Sum256([]byte(seed))
	angle := (float64(binary.BigEndian.Uint64(h[0:8])) / float64(math.MaxUint64)) * 2 * math.Pi

	dLat := (approxOffsetMeters * math.Cos(angle)) / metersPerDegreeLat
	cosLat := math.Cos(lat * math.Pi / 180)
	if cosLat == 0 {
		cosLat = 1e-6 // avoid div-by-zero at the poles (never hit in practice)
	}
	dLng := (approxOffsetMeters * math.Sin(angle)) / (metersPerDegreeLat * cosLat)

	return lat + dLat, lng + dLng
}
