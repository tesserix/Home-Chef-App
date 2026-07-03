package services

import (
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// premium.go — the single source of truth for "is this chef currently premium"
// (#44). A chef gets premium perks (Verified-Pro badge, priority ranking,
// advanced analytics) only while they hold a premium-tier subscription in a
// perks-granting status (trial or active). All perk call sites — chef-list
// ranking, badge population, analytics gate — go through these helpers so the
// definition never drifts. Note: premium no longer affects payout commission —
// the platform charges a flat runtime rate (ADR-0001 / #390, see commission.go).

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
