package services

import (
	"math"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// delivery_area.go — chef delivery-reach helpers for customer discovery.
//
// When a chef handles their own (self) delivery, their kitchen should only be
// orderable-for-delivery to customers the chef can actually reach. Discovery
// uses an equirectangular ("planar") distance in km rather than the exact
// haversine so the SAME arithmetic can run inside the SQL WHERE clause — the
// caller passes the longitude scale factor in as a bind var, so no trig runs in
// the database and the filter stays portable (sqlite in tests, postgres in
// prod). At city scale (<~25km) the planar error vs haversine is well under 1%,
// which is far tighter than the fuzz already applied to chef coordinates.

// kmPerDegLat is the approximate distance in km spanned by one degree of
// latitude. Constant everywhere on Earth (meridians are great circles).
const kmPerDegLat = 111.0

// KmPerDegLng returns the approximate km spanned by one degree of longitude at
// the given latitude. Lines of longitude converge toward the poles, so this
// scales the equatorial value by cos(latitude). Callers anchor it at the
// customer's latitude and reuse it for every candidate chef.
func KmPerDegLng(latDeg float64) float64 {
	return kmPerDegLat * math.Cos(latDeg*math.Pi/180)
}

// PlanarDistanceKm returns the approximate distance in km between a chef and a
// customer using an equirectangular projection anchored at the customer's
// latitude. This mirrors the expression compiled into the discovery SQL filter
// (see DeliveryAreaKeepSQL) so a chef sitting exactly on the radius boundary is
// classified identically in Go and in the query.
func PlanarDistanceKm(chefLat, chefLng, custLat, custLng float64) float64 {
	dLatKm := (chefLat - custLat) * kmPerDegLat
	dLngKm := (chefLng - custLng) * KmPerDegLng(custLat)
	return math.Hypot(dLatKm, dLngKm)
}

// hasRealCoords reports whether a chef has usable coordinates. A (0,0) pair is
// treated as "not captured" — it is Null Island in the Gulf of Guinea, never a
// real Indian kitchen — so such chefs are never classified as deliverable.
func hasRealCoords(lat, lng float64) bool {
	return lat != 0 || lng != 0
}

// ChefDeliversTo reports whether a self-delivering chef can reach the customer's
// coordinates: the chef offers self-delivery, has a positive delivery radius and
// real coordinates, and the customer falls within that radius. This is the
// per-customer own-fleet "deliverable to you" signal. Third-party delivery reach
// is orthogonal (see ThirdPartyDeliveryEnabled) — when a 3PL provider is live,
// discovery is not radius-gated at all.
func ChefDeliversTo(chef models.ChefProfile, custLat, custLng float64) bool {
	if !chef.OffersSelfDelivery || chef.DeliveryRadius <= 0 {
		return false
	}
	if !hasRealCoords(chef.Latitude, chef.Longitude) {
		return false
	}
	return PlanarDistanceKm(chef.Latitude, chef.Longitude, custLat, custLng) <= chef.DeliveryRadius
}

// DeliverableToYou is the value surfaced to the customer app for a single chef:
// whether the chef can deliver to the given coordinates, accounting for both
// own-fleet reach and any live third-party provider. When a 3PL provider is
// enabled, every chef is deliverable (the provider covers the area), so the
// radius gate is bypassed.
func DeliverableToYou(chef models.ChefProfile, custLat, custLng float64, tplEnabled bool) bool {
	if tplEnabled {
		return true // a live 3PL covers the whole area.
	}
	if chef.OffersSelfDelivery {
		reach := DeliveryReach(chef, custLat, custLng)
		return reach.Deliverable
	}
	return false // not self-delivering and no 3PL → pickup only.
}

// EffectiveDeliveryMaxKm is the serviceable self-delivery radius for a chef (#709):
// the chef's own SelfDeliveryMaxDistanceKm when they set one, else the platform
// default (DELIVERY_DEFAULT_MAX_RADIUS_KM, 10 km). This is the single source of
// truth for "how far will this kitchen deliver" — used at checkout AND at the
// hard order-creation guard so they can never disagree.
func EffectiveDeliveryMaxKm(chef models.ChefProfile) float64 {
	if chef.SelfDeliveryMaxDistanceKm > 0 {
		return chef.SelfDeliveryMaxDistanceKm
	}
	if config.AppConfig != nil && config.AppConfig.DeliveryDefaultMaxRadiusKm > 0 {
		return config.AppConfig.DeliveryDefaultMaxRadiusKm
	}
	return 10 // safe fallback if config isn't loaded (tests).
}

// DeliveryReachResult is the serviceability answer for a (chef → drop) pair.
type DeliveryReachResult struct {
	Deliverable bool    `json:"deliverable"`
	DistanceKm  float64 `json:"distanceKm"`  // straight-line chef → drop; 0 when unknown
	MaxRadiusKm float64 `json:"maxRadiusKm"` // the serviceable cap enforced
	Known       bool    `json:"known"`       // false when coords are missing (can't range-check)
}

// DeliveryReach computes whether a self-delivering chef can serve a drop and by
// how much margin. The rule (#709): a drop within EffectiveDeliveryMaxKm of the
// chef is deliverable; beyond it is NOT (hard cap — no order). When either the
// chef's or the customer's coordinates are missing we can't measure, so Known is
// false and Deliverable defaults true (the caller decides; the checkout requires
// coords for delivery, so this only affects legacy/coordless paths).
func DeliveryReach(chef models.ChefProfile, custLat, custLng float64) DeliveryReachResult {
	maxKm := EffectiveDeliveryMaxKm(chef)
	if !hasRealCoords(chef.Latitude, chef.Longitude) || (custLat == 0 && custLng == 0) {
		return DeliveryReachResult{Deliverable: true, MaxRadiusKm: maxKm, Known: false}
	}
	dist := PlanarDistanceKm(chef.Latitude, chef.Longitude, custLat, custLng)
	return DeliveryReachResult{
		Deliverable: dist <= maxKm,
		DistanceKm:  dist,
		MaxRadiusKm: maxKm,
		Known:       true,
	}
}

// DeliveryAreaKeepSQL returns a SQL predicate (and its ordered bind vars) that
// keeps only the chefs a customer at (custLat, custLng) can order from under the
// hybrid discovery rule:
//
//   - a chef that offers pickup is always kept (pickup ignores delivery reach);
//   - a self-delivering chef is kept only when the customer is inside the chef's
//     delivery_radius;
//   - a delivery-only chef outside its radius is dropped from discovery.
//
// Callers apply this predicate only when the customer supplied coordinates AND
// no third-party provider is live (a live 3PL covers the whole area, so there is
// no radius gate). The distance test is the planar approximation from
// PlanarDistanceKm, with the per-degree scale factors folded into bind vars so
// the database runs no trig and the clause stays portable (sqlite/postgres).
func DeliveryAreaKeepSQL(custLat, custLng float64) (string, []interface{}) {
	lngFactor := KmPerDegLng(custLat)
	lngFactor *= lngFactor                      // km² per squared degree of longitude
	const latFactor = kmPerDegLat * kmPerDegLat // km² per squared degree of latitude (12321)

	sql := "offers_pickup = TRUE OR (" +
		"offers_self_delivery = TRUE AND delivery_radius > 0 " +
		"AND (latitude != 0 OR longitude != 0) " +
		"AND ((latitude - ?) * (latitude - ?) * ? + (longitude - ?) * (longitude - ?) * ?) " +
		"<= delivery_radius * delivery_radius)"

	return sql, []interface{}{custLat, custLat, latFactor, custLng, custLng, lngFactor}
}
