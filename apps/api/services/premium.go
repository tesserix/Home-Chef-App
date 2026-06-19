package services

import (
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// premium.go — the single source of truth for "is this chef currently premium"
// (#44). A chef gets premium perks (Verified-Pro badge, priority ranking, lower
// commission, advanced analytics) only while they hold a premium-tier
// subscription in a perks-granting status (trial or active). All perk call sites
// — chef-list ranking, badge population, commission, analytics gate — go through
// these helpers so the definition never drifts.

// PremiumChefIDs returns, for the given ChefProfile IDs, the subset that
// currently hold an active/trial premium subscription. Batched to one query so
// the chef-list ranking + badge population stay a single round-trip.
func PremiumChefIDs(chefIDs []uuid.UUID) map[uuid.UUID]bool {
	out := make(map[uuid.UUID]bool, len(chefIDs))
	if len(chefIDs) == 0 {
		return out
	}
	type row struct{ ChefID uuid.UUID }
	var rows []row
	database.DB.
		Table("chef_profiles AS c").
		Select("c.id AS chef_id").
		Joins("JOIN subscriptions s ON s.user_id = c.user_id").
		Where("c.id IN ?", chefIDs).
		Where("s.tier = ?", models.TierPremium).
		Where("s.status IN ?", []models.SubscriptionStatus{models.SubStatusTrial, models.SubStatusActive}).
		Where("s.deleted_at IS NULL").
		Scan(&rows)
	for _, r := range rows {
		out[r.ChefID] = true
	}
	return out
}

// IsChefPremium reports whether one chef currently has premium perks.
func IsChefPremium(chefID uuid.UUID) bool {
	return PremiumChefIDs([]uuid.UUID{chefID})[chefID]
}

// PremiumCommissionRateForChef returns the platform commission rate to charge a
// chef: the country's configured premium rate when the chef currently has premium
// perks, else 0 (callers fall back to the standard RateCommission via
// ComputeOrderEarnings). Returns 0 on any misconfiguration (rate ≤0 or ≥1) so a
// bad setting never charges a nonsensical rate. Resolve ONCE per chef and reuse
// across that chef's order loop — it issues a few queries.
func PremiumCommissionRateForChef(chefID uuid.UUID) float64 {
	if !IsChefPremium(chefID) {
		return 0
	}
	cfg, err := GetPlanSettings(chefCountryCode(chefID), models.SubscriberChef)
	if err != nil || cfg.PremiumCommissionRate <= 0 || cfg.PremiumCommissionRate >= 1 {
		return 0
	}
	return cfg.PremiumCommissionRate
}

// chefCountryCode returns the chef's payout country (defaults to IN), used to key
// the per-country premium commission setting.
func chefCountryCode(chefID uuid.UUID) string {
	var chef models.ChefProfile
	if err := database.DB.Select("payout_country").First(&chef, "id = ?", chefID).Error; err == nil && chef.PayoutCountry != "" {
		return chef.PayoutCountry
	}
	return "IN"
}
