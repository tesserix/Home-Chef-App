package services

import (
	"context"
	"math"

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
	// TierApplied says the chef's published distance→fee ladder priced this
	// delivery, in which case TierUpToKm names the band and every component above
	// is zero — a band is a flat rate, not a base plus a distance calculation.
	TierApplied bool    `json:"tierApplied"`
	TierUpToKm  float64 `json:"tierUpToKm"`
	// PlatformMaxFee is the platform's own ceiling for this delivery, and
	// CappedByPlatform says it had to bite. It should never bite in practice —
	// a chef's pricing is validated against it when they save it.
	PlatformMaxFee   float64 `json:"platformMaxFee"`
	CappedByPlatform bool    `json:"cappedByPlatform"`
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
	if chef.Latitude != 0 && chef.Longitude != 0 && dropLat != 0 && dropLng != 0 {
		b.DistanceKnown = true
		// Road distance, not straight line (#701): the chef drives roads, so the
		// per-km fee should reflect the driven distance. RoadDistanceKm uses a real
		// router when configured, else a winding-factor fallback — never blocks.
		b.DistanceKm = RoadDistanceKm(chef.Latitude, chef.Longitude, dropLat, dropLng)
	}

	if tiers := chef.DeliveryTiers(); len(tiers) > 0 {
		return applyTierPricing(b, tiers)
	}

	fee := chef.SelfDeliveryBaseFee
	if b.DistanceKnown {
		extra := b.DistanceKm - chef.SelfDeliveryFreeRadiusKm
		// A chef who set no free radius has no free zone, so a zero-distance drop
		// is not "inside" one — it just has nothing to bill for distance.
		if extra > 0 || chef.SelfDeliveryFreeRadiusKm <= 0 {
			if extra > 0 {
				b.BillableKm = extra
				// Surge scales the distance cost — a high-fuel day costs more to drive.
				b.DistanceComponent = extra * chef.SelfDeliveryPerKm * surge
				fee += b.DistanceComponent
			}
		} else {
			// Inside the chef's free-delivery radius the delivery is FREE — the flat
			// base fee is waived too, not just the distance component. A "free zone"
			// that still bills the base fee is not one, and the app would show a
			// delivery charge on an order the chef advertised as free.
			b.WithinFreeZone = true
			fee = 0
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
	return capToPlatformCeiling(b, b.DistanceKm)
}

// applyTierPricing prices the delivery from the chef's published ladder. The
// band is a FLAT rate: no base, no distance component, and no surge — the chef
// published a price for the distance and that is what the customer pays.
func applyTierPricing(b SelfDeliveryFeeBreakdown, tiers models.DeliveryFeeTiers) SelfDeliveryFeeBreakdown {
	km := b.DistanceKm
	if !b.DistanceKnown {
		// Nothing measured the drop, so bill the nearest band rather than guessing
		// upward against a customer who never agreed to the far one.
		km = 0
	}
	tier, ok := tiers.TierForKm(km)
	if !ok {
		return b
	}
	b.BaseFee, b.PerKm, b.MaxFee, b.FreeRadiusKm = 0, 0, 0, 0
	b.SurgeMultiplier = 1.0
	b.TierApplied = true
	b.TierUpToKm = tier.UpToKm
	b.Fee = models.RoundAmount(tier.Fee)
	b.WithinFreeZone = b.Fee == 0
	// Ceiling checked at the band's distance, not the drop's: a 5 km band is one
	// price for the whole band, so a short drop inside it isn't over-priced.
	return capToPlatformCeiling(b, tier.UpToKm)
}

// capToPlatformCeiling holds any chef fee to what the platform allows at `km`.
// Chef pricing is validated on save, so this is the backstop for pricing saved
// before a ceiling change.
func capToPlatformCeiling(b SelfDeliveryFeeBreakdown, km float64) SelfDeliveryFeeBreakdown {
	b.PlatformMaxFee = MaxChefDeliveryFee(km)
	if b.Fee > b.PlatformMaxFee {
		b.Fee = b.PlatformMaxFee
		b.CappedByPlatform = true
	}
	return b
}

// ValidateChefDeliveryTiers checks a ladder a chef is trying to save against
// the live platform ceiling. Every surface that lets a chef set delivery pricing
// must go through this — onboarding, the profile editor, admin edits.
func ValidateChefDeliveryTiers(tiers models.DeliveryFeeTiers) error {
	return tiers.Validate(MaxChefDeliveryFee)
}

// ChefDeliveryFeeCapPolicy is the ceiling in the shape the apps need it: enough
// to draw the limit beside each band the chef is editing, rather than letting
// them discover it from a rejected save.
type ChefDeliveryFeeCapPolicy struct {
	BaseFee  float64 `json:"baseFee"`
	PerKm    float64 `json:"perKm"`
	MaxFee   float64 `json:"maxFee"`
	MaxBands int     `json:"maxBands"`
	MaxKm    float64 `json:"maxKm"`
}

// ChefDeliveryFeeCap is the live ceiling. An unconfigured policy reads as the
// shipped default: a zero ceiling would clamp every kitchen's delivery to free
// rather than mean "no limit".
func ChefDeliveryFeeCap() ChefDeliveryFeeCapPolicy {
	p, def := GetPlatformPolicy(), DefaultPlatformPolicy()
	cap := ChefDeliveryFeeCapPolicy{
		BaseFee:  p.ChefDeliveryFeeCapBase,
		PerKm:    p.ChefDeliveryFeeCapPerKm,
		MaxFee:   p.ChefDeliveryFeeCapMax,
		MaxBands: models.MaxDeliveryFeeTiers,
		MaxKm:    models.MaxDeliveryTierKm,
	}
	if cap.BaseFee <= 0 {
		cap.BaseFee = def.ChefDeliveryFeeCapBase
	}
	if cap.PerKm <= 0 {
		cap.PerKm = def.ChefDeliveryFeeCapPerKm
	}
	if cap.MaxFee <= 0 {
		cap.MaxFee = def.ChefDeliveryFeeCapMax
	}
	return cap
}

// MaxChefDeliveryFee is the most a chef may charge to deliver `km`.
func MaxChefDeliveryFee(km float64) float64 {
	if km < 0 {
		km = 0
	}
	cap := ChefDeliveryFeeCap()
	return models.RoundAmount(math.Min(cap.BaseFee+cap.PerKm*km, cap.MaxFee))
}

// EstimateSelfDeliveryFeeBreakdown is the customer-facing ESTIMATE: the charge
// basis with the current surge factors (fuel now; traffic/weather later) folded
// into the distance component, still capped at the chef's max. This is the
// "approx max" shown at checkout — the chef can only bring it down at accept.
// Never blocks: surge degrades to neutral when no signal is available.
func EstimateSelfDeliveryFeeBreakdown(ctx context.Context, chef models.ChefProfile, dropLat, dropLng float64, country string) SelfDeliveryFeeBreakdown {
	surge := CurrentSurge(ctx, country, chef.Latitude, chef.Longitude, dropLat, dropLng)
	return SelfDeliveryBreakdownAt(chef, dropLat, dropLng, surge, surge.Combined)
}

// SelfDeliveryBreakdownAt builds the itemised breakdown at an ALREADY-resolved
// surge. The checkout quote needs both the headline fee and this breakdown, and
// they must agree; taking the multiplier as an argument means the live signals
// are resolved once per quote and both numbers come from that one reading.
//
// `applied` is the multiplier actually folded into the distance component, which
// is 1.0 while surge-charging is off; `factors` are the observed conditions,
// reported either way so the breakdown explains itself.
func SelfDeliveryBreakdownAt(chef models.ChefProfile, dropLat, dropLng float64, factors SurgeFactors, applied float64) SelfDeliveryFeeBreakdown {
	b := computeSelfDeliveryBreakdown(chef, dropLat, dropLng, applied)
	b.FuelSurge = factors.Fuel
	b.WeatherSurge = factors.Weather
	b.TrafficSurge = factors.Traffic
	b.SurgeMultiplier = applied
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
