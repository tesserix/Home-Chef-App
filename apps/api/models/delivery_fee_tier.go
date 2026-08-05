package models

// delivery_fee_tier.go — the chef's distance→fee ladder.
//
// A chef publishes one price per distance band ("up to 5 km ₹75, up to 10 km
// ₹150") and that price is what the customer pays. It replaces the platform's
// base fee and the base+per-km formula for that kitchen, and it is fixed at
// checkout — nobody re-prices the delivery after the customer has paid.

import (
	"encoding/json"
	"fmt"
	"sort"
)

const (
	// MaxDeliveryFeeTiers keeps the ladder legible on a phone and in the chef's head.
	MaxDeliveryFeeTiers = 6
	// MaxDeliveryTierKm is the farthest band a home kitchen may price.
	MaxDeliveryTierKm = 50.0
)

// DeliveryFeeTier prices every drop within UpToKm of the kitchen at Fee.
type DeliveryFeeTier struct {
	UpToKm float64 `json:"upToKm"`
	Fee    float64 `json:"fee"`
}

// DeliveryFeeTiers is a chef's ladder, ascending by distance.
type DeliveryFeeTiers []DeliveryFeeTier

// ParseDeliveryFeeTiers reads the stored JSONB column. Anything unreadable is
// "no ladder" rather than an error: pricing must never fail on a bad column, it
// falls back to the base+per-km formula.
func ParseDeliveryFeeTiers(raw string) DeliveryFeeTiers {
	if raw == "" {
		return nil
	}
	var tiers DeliveryFeeTiers
	if err := json.Unmarshal([]byte(raw), &tiers); err != nil {
		return nil
	}
	return tiers.Normalized()
}

// JSON renders the ladder for the JSONB column.
func (t DeliveryFeeTiers) JSON() string {
	b, err := json.Marshal(t.Normalized())
	if err != nil {
		return "[]"
	}
	return string(b)
}

// Normalized drops bands with no distance and sorts the rest by distance, so
// every reader sees the same ordered ladder however it was entered.
func (t DeliveryFeeTiers) Normalized() DeliveryFeeTiers {
	out := make(DeliveryFeeTiers, 0, len(t))
	for _, tier := range t {
		if tier.UpToKm <= 0 {
			continue
		}
		if tier.Fee < 0 {
			tier.Fee = 0
		}
		out = append(out, DeliveryFeeTier{UpToKm: tier.UpToKm, Fee: RoundAmount(tier.Fee)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].UpToKm < out[j].UpToKm })
	if len(out) == 0 {
		return nil
	}
	return out
}

// DeliveryTiers is the chef's published ladder, empty when they never set one.
func (c ChefProfile) DeliveryTiers() DeliveryFeeTiers {
	return ParseDeliveryFeeTiers(c.SelfDeliveryTiers)
}

// TierForKm returns the band that prices a drop `km` from the kitchen: the first
// band that covers it, or the top band when the drop is beyond the ladder (range
// is gated separately — an in-range order must always have a price).
func (t DeliveryFeeTiers) TierForKm(km float64) (DeliveryFeeTier, bool) {
	if len(t) == 0 {
		return DeliveryFeeTier{}, false
	}
	for _, tier := range t {
		if km <= tier.UpToKm {
			return tier, true
		}
	}
	return t[len(t)-1], true
}

// Validate checks the ladder a chef submitted, including that no band charges
// more than the platform allows at that distance. `ceiling` is the platform's
// max fee for a distance (services.MaxChefDeliveryFee) — passed in so this stays
// free of the policy layer.
func (t DeliveryFeeTiers) Validate(ceiling func(km float64) float64) error {
	if len(t) == 0 {
		return nil
	}
	if len(t) > MaxDeliveryFeeTiers {
		return fmt.Errorf("delivery pricing supports at most %d distance bands", MaxDeliveryFeeTiers)
	}
	sorted := append(DeliveryFeeTiers(nil), t...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].UpToKm < sorted[j].UpToKm })

	prevKm, prevFee := 0.0, 0.0
	for i, tier := range sorted {
		if tier.UpToKm <= 0 || tier.UpToKm > MaxDeliveryTierKm {
			return fmt.Errorf("distance band must be between 0 and %.0f km", MaxDeliveryTierKm)
		}
		if tier.Fee < 0 {
			return fmt.Errorf("delivery fee for the %.0f km band can't be negative", tier.UpToKm)
		}
		if i > 0 && tier.UpToKm <= prevKm {
			return fmt.Errorf("each distance band must be farther than the one before %.0f km", prevKm)
		}
		// A farther band that costs less would price a distant customer below a
		// near one — the ladder must only ever climb.
		if i > 0 && tier.Fee < prevFee {
			return fmt.Errorf("the %.0f km band can't cost less than the %.0f km band", tier.UpToKm, prevKm)
		}
		if max := ceiling(tier.UpToKm); tier.Fee > max {
			return fmt.Errorf("the %.0f km band can't be more than %.0f", tier.UpToKm, max)
		}
		prevKm, prevFee = tier.UpToKm, tier.Fee
	}
	return nil
}
