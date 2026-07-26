package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// RecalcChefRating recomputes a kitchen's average rating and review count for
// ONE mode and stores it in chef_mode_stats.
//
// When the mode is live it additionally mirrors the figures onto chef_profiles,
// which stays the customer-facing source of truth: every existing query that
// reads chef.Rating keeps working untouched, and — because a test review is
// never counted into the live figures — a fake order can never move a real
// kitchen's public rating.
//
// Hidden, unapproved and deleted reviews are excluded exactly as before.
func RecalcChefRating(db *gorm.DB, chefID uuid.UUID, mode string) error {
	mode = models.NormalizeMode(mode)

	var stats struct {
		AvgRating    float64
		TotalReviews int64
	}
	if err := db.Model(&models.Review{}).
		Where("chef_id = ? AND mode = ? AND is_approved = ? AND is_hidden = ? AND deleted_at IS NULL",
			chefID, mode, true, false).
		Select("COALESCE(AVG(overall_rating), 0) as avg_rating, COUNT(*) as total_reviews").
		Scan(&stats).Error; err != nil {
		return err
	}

	row := models.ChefModeStats{
		ChefID:       chefID,
		Mode:         mode,
		Rating:       stats.AvgRating,
		TotalReviews: int(stats.TotalReviews),
	}
	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "chef_id"}, {Name: "mode"}},
		DoUpdates: clause.AssignmentColumns([]string{"rating", "total_reviews", "updated_at"}),
	}).Create(&row).Error; err != nil {
		return err
	}

	if models.IsTestMode(mode) {
		// Deliberately does NOT touch chef_profiles. That row is what customers
		// see, and sandbox activity must never reach it.
		return nil
	}
	return db.Model(&models.ChefProfile{}).Where("id = ?", chefID).
		Updates(map[string]interface{}{
			"rating":        stats.AvgRating,
			"total_reviews": stats.TotalReviews,
		}).Error
}

// GetChefModeStats reads a kitchen's counters for one mode, returning a zero row
// when the kitchen has no activity in that mode yet (a brand-new sandbox
// session legitimately has none).
func GetChefModeStats(db *gorm.DB, chefID uuid.UUID, mode string) (models.ChefModeStats, error) {
	out := models.ChefModeStats{ChefID: chefID, Mode: models.NormalizeMode(mode)}
	err := db.Where("chef_id = ? AND mode = ?", chefID, out.Mode).First(&out).Error
	if err == gorm.ErrRecordNotFound {
		return models.ChefModeStats{ChefID: chefID, Mode: out.Mode}, nil
	}
	return out, err
}
