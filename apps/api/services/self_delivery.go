package services

import (
	"context"

	"github.com/homechef/api/models"
)

// SelfDeliveryFeeBreakdown is the itemised, CAPPED self-delivery estimate shown
// to the customer at checkout (#702). Every component is exposed so the app can
// render an honest breakdown, and the chef's later accept fee (#703) is bounded
// by Fee (the "approx max"). Components sum to Fee unless Capped is true, in
// which case Fee is MaxFee.
type SelfDeliveryFeeBreakdown struct {
	// BaseFee is the flat per-trip fee the chef configured.
	BaseFee float64 `json:"baseFee"`
	// DistanceKnown is false when either endpoint's coords are missing (0,0), so
	// the distance component can't be computed and only BaseFee applies.
	DistanceKnown bool `json:"distanceKnown"`
	// DistanceKm is the ROAD distance chef→drop (#701 winding factor / router).
	DistanceKm float64 `json:"distanceKm"`
	// FreeRadiusKm is the chef's free-delivery radius; distance inside it is free.
	FreeRadiusKm float64 `json:"freeRadiusKm"`
	// BillableKm is the distance actually charged: max(0, DistanceKm − FreeRadiusKm).
	BillableKm float64 `json:"billableKm"`
	// PerKm is the chef's per-km rate beyond the free radius.
	PerKm float64 `json:"perKm"`
	// DistanceComponent = BillableKm × PerKm.
	DistanceComponent float64 `json:"distanceComponent"`
	// WithinFreeZone is true when the drop is inside the free radius (no distance
	// component) — the app shows "Free delivery" when this yields Fee 0.
	WithinFreeZone bool `json:"withinFreeZone"`
	// MaxFee is the chef's cap (0 = uncapped); Capped is true when it bit.
	MaxFee float64 `json:"maxFee"`
	Capped bool    `json:"capped"`
	// FuelSurge / SurgeMultiplier are the surge factors folded into the DISTANCE
	// component for the customer ESTIMATE (#704+). Both are 1.0 on the charge-basis
	// path (ComputeSelfDeliveryFeeBreakdown) — surge never changes what's charged.
	FuelSurge       float64 `json:"fuelSurge"`
	WeatherSurge    float64 `json:"weatherSurge"`
	TrafficSurge    float64 `json:"trafficSurge"`
	SurgeMultiplier float64 `json:"surgeMultiplier"`
	// Fee is the final, capped, non-negative self-delivery fee — the approx MAX
	// the customer is quoted and the ceiling the chef can charge at accept.
	Fee float64 `json:"fee"`
}

// ComputeSelfDeliveryFeeBreakdown computes the itemised self-delivery CHARGE
// basis (no surge):
//
//	fee = BaseFee + max(0, roadKm − FreeRadiusKm) × PerKm   (capped at MaxFee)
//
// MaxFee of 0 means uncapped. When either endpoint's coords are missing the
// distance is unknown, so only BaseFee applies. Never negative. This is the ONE
// place the fee is computed; ComputeSelfDeliveryFee is a thin wrapper so the
// single-number and itemised paths can never drift. Surge is NOT applied here —
// this feeds the actual order charge; the customer estimate adds surge via
// EstimateSelfDeliveryFeeBreakdown.
func ComputeSelfDeliveryFeeBreakdown(chef models.ChefProfile, dropLat, dropLng float64) SelfDeliveryFeeBreakdown {
	return computeSelfDeliveryBreakdown(chef, dropLat, dropLng, 1.0)
}

// computeSelfDeliveryBreakdown is the core, parameterised by a surge multiplier
// applied to the DISTANCE component only (the flat base isn't a driving cost).
// surge of 1.0 is the neutral charge basis.
func computeSelfDeliveryBreakdown(chef models.ChefProfile, dropLat, dropLng, surge float64) SelfDeliveryFeeBreakdown {
	if surge < 1.0 {
		surge = 1.0
	}
	b := SelfDeliveryFeeBreakdown{
		BaseFee:         chef.SelfDeliveryBaseFee,
		FreeRadiusKm:    chef.SelfDeliveryFreeRadiusKm,
		PerKm:           chef.SelfDeliveryPerKm,
		MaxFee:          chef.SelfDeliveryMaxFee,
		SurgeMultiplier: surge,
		FuelSurge:       1.0,
		WeatherSurge:    1.0,
		TrafficSurge:    1.0,
	}
	fee := chef.SelfDeliveryBaseFee

	if chef.Latitude != 0 && chef.Longitude != 0 && dropLat != 0 && dropLng != 0 {
		b.DistanceKnown = true
		// Road distance, not straight line (#701): the chef drives roads, so the
		// per-km fee should reflect the driven distance. RoadDistanceKm uses a real
		// router when configured, else a winding-factor fallback — never blocks.
		b.DistanceKm = RoadDistanceKm(chef.Latitude, chef.Longitude, dropLat, dropLng)
		if extra := b.DistanceKm - chef.SelfDeliveryFreeRadiusKm; extra > 0 {
			b.BillableKm = extra
			// Surge scales the distance cost — a high-fuel day costs more to drive.
			b.DistanceComponent = extra * chef.SelfDeliveryPerKm * surge
			fee += b.DistanceComponent
		} else {
			// Drop sits inside the free radius — no distance charge.
			b.WithinFreeZone = true
		}
	}

	if chef.SelfDeliveryMaxFee > 0 && fee > chef.SelfDeliveryMaxFee {
		fee = chef.SelfDeliveryMaxFee
		b.Capped = true
	}
	if fee < 0 {
		fee = 0
	}
	b.Fee = fee
	return b
}

// EstimateSelfDeliveryFeeBreakdown is the customer-facing ESTIMATE: the charge
// basis with the current surge factors (fuel now; traffic/weather later) folded
// into the distance component, still capped at the chef's max. This is the
// "approx max" shown at checkout — the chef can only bring it down at accept.
// Never blocks: surge degrades to neutral when no signal is available.
func EstimateSelfDeliveryFeeBreakdown(ctx context.Context, chef models.ChefProfile, dropLat, dropLng float64, country string) SelfDeliveryFeeBreakdown {
	surge := CurrentSurge(ctx, country, dropLat, dropLng)
	b := computeSelfDeliveryBreakdown(chef, dropLat, dropLng, surge.Combined)
	b.FuelSurge = surge.Fuel
	b.WeatherSurge = surge.Weather
	b.TrafficSurge = surge.Traffic
	return b
}

// ComputeSelfDeliveryFee returns just the final capped self-delivery fee for an
// order going to (dropLat, dropLng). Deterministic + pure so the checkout quote
// and any later re-quote agree; delegates to ComputeSelfDeliveryFeeBreakdown so
// the number always matches the itemised estimate. No surge — this is the charge
// basis.
func ComputeSelfDeliveryFee(chef models.ChefProfile, dropLat, dropLng float64) float64 {
	return ComputeSelfDeliveryFeeBreakdown(chef, dropLat, dropLng).Fee
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
