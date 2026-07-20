package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// delivery_area_test.go — verifies the chef delivery-reach discovery helpers:
// the planar distance tracks haversine at city scale, the own-fleet reach test
// respects radius/coords/self-delivery, a live 3PL bypasses the radius gate, and
// the SQL predicate implements the hybrid keep-rule identically to the Go path.

func TestPlanarDistanceKm_TracksHaversineAtCityScale(t *testing.T) {
	// Bandra → ~10 km north in Mumbai. Planar must stay within 1% of haversine.
	custLat, custLng := 19.0596, 72.8295
	chefLat, chefLng := 19.1500, 72.8500

	planar := PlanarDistanceKm(chefLat, chefLng, custLat, custLng)
	hav := haversineDistance(custLat, custLng, chefLat, chefLng)

	assert.InEpsilon(t, hav, planar, 0.01,
		"planar (%.3f km) should be within 1%% of haversine (%.3f km)", planar, hav)
}

func TestChefDeliversTo_RespectsRadiusCoordsAndMode(t *testing.T) {
	custLat, custLng := 19.0760, 72.8777 // Mumbai CST-ish

	base := models.ChefProfile{
		OffersSelfDelivery: true,
		DeliveryRadius:     5,
		Latitude:           custLat,
		Longitude:          custLng,
	}
	assert.True(t, ChefDeliversTo(base, custLat, custLng), "chef at the customer's spot is deliverable")

	// ~8 km north, radius 5 → out of reach.
	far := base
	far.Latitude = 19.1480
	assert.False(t, ChefDeliversTo(far, custLat, custLng), "8 km away with a 5 km radius is not deliverable")

	// Same distance, radius 10 → in reach.
	farBig := far
	farBig.DeliveryRadius = 10
	assert.True(t, ChefDeliversTo(farBig, custLat, custLng), "8 km away with a 10 km radius is deliverable")

	// Self-delivery off, zero radius, or missing coords → never deliverable.
	noSelf := base
	noSelf.OffersSelfDelivery = false
	assert.False(t, ChefDeliversTo(noSelf, custLat, custLng), "non-self-delivering chef is not own-fleet deliverable")

	noRadius := base
	noRadius.DeliveryRadius = 0
	assert.False(t, ChefDeliversTo(noRadius, custLat, custLng), "zero radius is not deliverable")

	noCoords := base
	noCoords.Latitude, noCoords.Longitude = 0, 0
	assert.False(t, ChefDeliversTo(noCoords, custLat, custLng), "unset (0,0) coords are not deliverable")
}

func TestDeliverableToYou_SelfDeliveryReach(t *testing.T) {
	// Chef in central Bangalore. config.AppConfig is nil in this test, so
	// EffectiveDeliveryMaxKm falls back to the 10 km platform default.
	chefLat, chefLng := 12.9716, 77.5946
	nearLat, nearLng := 12.98, 77.60 // ~1.3 km — inside 10 km
	farLat, farLng := 13.25, 77.95   // ~50 km — outside 10 km

	self := models.ChefProfile{OffersSelfDelivery: true, Latitude: chefLat, Longitude: chefLng}

	// A live 3PL covers the whole area regardless of distance.
	assert.True(t, DeliverableToYou(self, farLat, farLng, true), "3PL live → deliverable regardless of distance")

	// max 0 → platform default 10 km cap (#709). Within → deliverable, beyond → NOT.
	assert.True(t, DeliverableToYou(self, nearLat, nearLng, false), "within the 10 km default cap → deliverable")
	assert.False(t, DeliverableToYou(self, farLat, farLng, false), "beyond the 10 km default cap → not deliverable")

	// A chef who set a SMALLER explicit max is gated by it.
	tight := self
	tight.SelfDeliveryMaxDistanceKm = 0.5
	assert.False(t, DeliverableToYou(tight, nearLat, nearLng, false), "1.3 km with a 0.5 km max → not deliverable")

	// A chef who set a LARGER explicit max reaches further.
	wide := self
	wide.SelfDeliveryMaxDistanceKm = 100
	assert.True(t, DeliverableToYou(wide, farLat, farLng, false), "50 km with a 100 km max → deliverable")

	// Not self-delivering and no 3PL → pickup only.
	noSelf := models.ChefProfile{OffersSelfDelivery: false, Latitude: chefLat, Longitude: chefLng}
	assert.False(t, DeliverableToYou(noSelf, nearLat, nearLng, false), "no self-delivery + no 3PL → not deliverable")
}

func TestDeliveryReach_DistanceAndCap(t *testing.T) {
	chef := models.ChefProfile{OffersSelfDelivery: true, Latitude: 12.9716, Longitude: 77.5946}

	// Within the 10 km default: deliverable, distance measured.
	in := DeliveryReach(chef, 12.98, 77.60)
	assert.True(t, in.Known)
	assert.True(t, in.Deliverable)
	assert.InDelta(t, 10.0, in.MaxRadiusKm, 1e-9)
	assert.Less(t, in.DistanceKm, 10.0)

	// Beyond the cap: not deliverable, with the distance for the error message.
	out := DeliveryReach(chef, 13.25, 77.95)
	assert.True(t, out.Known)
	assert.False(t, out.Deliverable)
	assert.Greater(t, out.DistanceKm, 10.0)

	// No customer coords → unmeasurable: Known false, defaults deliverable.
	unknown := DeliveryReach(chef, 0, 0)
	assert.False(t, unknown.Known)
	assert.True(t, unknown.Deliverable)
}

func TestDeliveryAreaKeepSQL_HybridFilter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	// Minimal table with only the columns the predicate touches — mirrors the
	// raw-DDL approach used elsewhere to avoid sqlite's lack of text[].
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', 
		id TEXT PRIMARY KEY, business_name TEXT DEFAULT '',
		offers_pickup INTEGER DEFAULT 0, offers_self_delivery INTEGER DEFAULT 0,
		delivery_radius REAL DEFAULT 0, latitude REAL DEFAULT 0, longitude REAL DEFAULT 0
	)`).Error)

	custLat, custLng := 19.0760, 72.8777
	const nearLat, nearLng = 19.0800, 72.8790 // ~0.5 km
	const farLat, farLng = 19.1850, 72.8900    // ~12 km north

	rows := []struct {
		name                        string
		pickup, self                int
		radius, lat, lng            float64
	}{
		{"near-selfdeliver", 0, 1, 5, nearLat, nearLng},         // in reach → keep
		{"far-selfdeliver-only", 0, 1, 5, farLat, farLng},       // out of reach, no pickup → DROP
		{"far-pickup-only", 1, 0, 0, farLat, farLng},            // pickup → keep (radius ignored)
		{"far-selfdeliver-bigradius", 0, 1, 30, farLat, farLng}, // big radius covers it → keep
		{"far-selfdeliver-and-pickup", 1, 1, 5, farLat, farLng}, // out of reach but pickup → keep
		{"nocoord-selfdeliver", 0, 1, 5, 0, 0},                  // no coords, no pickup → DROP
	}
	for i, r := range rows {
		require.NoError(t, db.Exec(
			`INSERT INTO chef_profiles (id, business_name, offers_pickup, offers_self_delivery, delivery_radius, latitude, longitude)
			 VALUES (?,?,?,?,?,?,?)`,
			i+1, r.name, r.pickup, r.self, r.radius, r.lat, r.lng).Error)
	}

	sql, vars := DeliveryAreaKeepSQL(custLat, custLng)
	var kept []struct{ BusinessName string }
	require.NoError(t, db.Table("chef_profiles").Where(sql, vars...).Find(&kept).Error)

	got := map[string]bool{}
	for _, k := range kept {
		got[k.BusinessName] = true
	}

	assert.True(t, got["near-selfdeliver"], "near self-delivering chef kept")
	assert.False(t, got["far-selfdeliver-only"], "far delivery-only chef dropped")
	assert.True(t, got["far-pickup-only"], "far pickup chef kept (pickup ignores radius)")
	assert.True(t, got["far-selfdeliver-bigradius"], "far chef with a wide radius kept")
	assert.True(t, got["far-selfdeliver-and-pickup"], "far chef offering pickup kept")
	assert.False(t, got["nocoord-selfdeliver"], "coordless delivery-only chef dropped")
}
