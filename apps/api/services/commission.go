package services

// commission.go — runtime resolution of the flat platform commission rate.
//
// The launch model (ADR-0001 / #390) charges every chef a flat 6%
// (DefaultCommissionRate). An admin can retune it at runtime by writing the
// PlatformSettings key `payout.commission_rate` — no redeploy. A missing or
// invalid setting silently falls back to the default, so a bad value never
// charges a nonsensical rate.

import (
	"strconv"

	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// commissionRateKey is the PlatformSettings key holding the flat commission rate.
const commissionRateKey = "payout.commission_rate"

// parseCommissionRate parses a stored setting value into a commission rate. It is
// valid only when 0 < rate < 1; any empty, non-numeric, or out-of-range value
// returns (0, false) so callers fall back to DefaultCommissionRate. Pure, no DB.
func parseCommissionRate(raw string) (float64, bool) {
	rate, err := strconv.ParseFloat(raw, 64)
	if err != nil || rate <= 0 || rate >= 1 {
		return 0, false
	}
	return rate, true
}

// GetCommissionRate resolves the flat platform commission rate from
// PlatformSettings (`payout.commission_rate`), falling back to
// DefaultCommissionRate when the setting is missing or invalid. Never panics.
func GetCommissionRate(db *gorm.DB) float64 {
	var setting models.PlatformSettings
	if err := db.Where("key = ?", commissionRateKey).First(&setting).Error; err != nil {
		return DefaultCommissionRate
	}
	if rate, ok := parseCommissionRate(setting.Value); ok {
		return rate
	}
	return DefaultCommissionRate
}
