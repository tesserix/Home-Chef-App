package services

// meal_plan_refund_tiers.go — the lead-time tier table for refund policy v3
// (docs/refund-policy-v3-spec.md, #834).
//
// A customer cancellation is priced by how much notice the chef got before that day's
// cook-start. Each band carries a FLOOR the chef may not refund below, and the top band
// resolves automatically with no chef step at all:
//
//	lead > 12h  → 100%, automatic (prep has not started; the chef has incurred nothing)
//	12h – 6h    → floor 75%, the chef sets 75–100%
//	6h – 2h     → floor 50%, the chef sets 50–100%
//	< 2h        → floor 0%,  the chef may still grant up to 100% for a genuine case
//
// EVERY number above is configuration, not code: the whole table lives in the
// `platform_policy` PlatformSettings blob (PlatformPolicy.MealPlanRefundTiers) and ops can
// retune it — including flipping the top band off auto-approve and giving it a 75% floor,
// which is the alternative policy the issue describes — without a deploy. The spec asked
// for the existing `mealplan.refund_prep_cutoff_hours` config-driven pattern to be followed
// rather than hardcoding new cutoffs; a whole ordered table is that pattern generalised.

import "sort"

// MealPlanRefundTier is one lead-time band of the customer-cancellation policy. It applies
// when the remaining lead is at least MinLeadHours (bands are matched highest-first).
type MealPlanRefundTier struct {
	// MinLeadHours — the band applies when (cook_start − now) >= this many hours.
	MinLeadHours float64 `json:"minLeadHours"`
	// FloorPercent — the smallest refund the chef may agree to in this band, 0–100.
	FloorPercent int `json:"floorPercent"`
	// AutoApprove — resolve at FloorPercent immediately with NO chef decision step.
	// Only meaningful with FloorPercent = 100 (agreeing less than the full amount
	// without asking anyone would be a silent customer downgrade).
	AutoApprove bool `json:"autoApprove"`
}

// DefaultMealPlanRefundTiers is the shipped v3 table. The >12h band keeps the pre-v3
// behaviour — 100%, automatic, no chef step — deliberately: the 12h cutoff exists BECAUSE
// prep has not started, so withholding 25% there would be a penalty with no cost behind it,
// and it would add an approval step to the most common and most benign cancellation. The
// tiering starts where prep actually starts.
func DefaultMealPlanRefundTiers() []MealPlanRefundTier {
	return []MealPlanRefundTier{
		{MinLeadHours: 12, FloorPercent: 100, AutoApprove: true},
		{MinLeadHours: 6, FloorPercent: 75},
		{MinLeadHours: 2, FloorPercent: 50},
		{MinLeadHours: 0, FloorPercent: 0},
	}
}

// MealPlanRefundTiers returns the live, validated tier table from platform policy, falling
// back to the default whenever the configured one is unusable (empty, or every band out of
// range). Always returns at least one band, sorted highest-lead-first, so callers can index
// it without a length check.
func MealPlanRefundTiers() []MealPlanRefundTier {
	return normalizeRefundTiers(GetPlatformPolicy().MealPlanRefundTiers)
}

// normalizeRefundTiers drops unusable bands, clamps percentages into 0–100, guarantees a
// zero-lead catch-all, and sorts highest-lead-first. Pure, so the validation is unit-tested
// directly. An admin who saves a nonsense table gets the default rather than a policy hole.
func normalizeRefundTiers(in []MealPlanRefundTier) []MealPlanRefundTier {
	out := make([]MealPlanRefundTier, 0, len(in)+1)
	for _, t := range in {
		if t.MinLeadHours < 0 || t.FloorPercent < 0 || t.FloorPercent > 100 {
			continue
		}
		// Auto-approving anything below 100% would resolve a refund at less than the full
		// amount with nobody having agreed to it. Demote it to a chef decision instead.
		if t.FloorPercent < 100 {
			t.AutoApprove = false
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return DefaultMealPlanRefundTiers()
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].MinLeadHours > out[j].MinLeadHours })
	// Guarantee a catch-all so ResolveMealPlanRefundTier always matches. A configured table
	// that stops at, say, 2h would otherwise leave sub-2h cancellations unpriced.
	if out[len(out)-1].MinLeadHours > 0 {
		out = append(out, MealPlanRefundTier{MinLeadHours: 0, FloorPercent: 0})
	}
	return out
}

// ResolveMealPlanRefundTier returns the band that prices a cancellation made with leadHours
// of notice. A cancellation made AFTER cook-start (negative lead) falls to the last band.
func ResolveMealPlanRefundTier(leadHours float64) MealPlanRefundTier {
	tiers := MealPlanRefundTiers()
	for _, t := range tiers {
		if leadHours >= t.MinLeadHours {
			return t
		}
	}
	return tiers[len(tiers)-1]
}
