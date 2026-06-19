package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// premium_pricing_test.go — #44. The premium tier's prices and commission must be
// fully admin-configurable at runtime (no deploy). These tests pin that
// GetPlanSettings reads the `subscription.{CC}.chef.premium_*` PlatformSettings
// keys, falls back to sane defaults when unset, and that the premium suffixes
// don't collide with the standard `.monthly_price` keys. PriceFor is the single
// tier×interval price resolver the plan endpoints + invoicing share.

func setupPlatformSettingsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE platform_settings (
			id         TEXT PRIMARY KEY,
			key        TEXT NOT NULL,
			value      TEXT,
			type       TEXT DEFAULT 'string',
			updated_by TEXT,
			updated_at DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_platform_settings_key ON platform_settings(key)`).Error)
	return db
}

func setSetting(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value) VALUES (lower(hex(randomblob(16))), ?, ?)`,
		key, value,
	).Error)
}

func TestGetPlanSettingsPremiumDefaults(t *testing.T) {
	db := setupPlatformSettingsDB(t)
	database.DB = db

	cfg, err := GetPlanSettings("IN", models.SubscriberChef)
	require.NoError(t, err)
	// Premium defaults apply when no settings are stored.
	assert.Equal(t, 999.0, cfg.PremiumMonthlyPrice)
	assert.Equal(t, 2599.0, cfg.PremiumQuarterlyPrice)
	assert.Equal(t, 8999.0, cfg.PremiumYearlyPrice)
	assert.Equal(t, 0.12, cfg.PremiumCommissionRate)
	// Standard defaults are untouched.
	assert.Equal(t, 499.0, cfg.MonthlyPrice)
}

func TestGetPlanSettingsPremiumConfigurable(t *testing.T) {
	db := setupPlatformSettingsDB(t)
	database.DB = db

	// The admin sets premium pricing + commission at runtime.
	setSetting(t, db, "subscription.IN.chef.premium_monthly_price", "1499")
	setSetting(t, db, "subscription.IN.chef.premium_quarterly_price", "3999")
	setSetting(t, db, "subscription.IN.chef.premium_yearly_price", "13999")
	setSetting(t, db, "subscription.IN.chef.premium_commission_rate", "0.10")
	// And a standard price, to prove premium suffixes don't collide.
	setSetting(t, db, "subscription.IN.chef.monthly_price", "599")

	cfg, err := GetPlanSettings("IN", models.SubscriberChef)
	require.NoError(t, err)

	assert.Equal(t, 1499.0, cfg.PremiumMonthlyPrice)
	assert.Equal(t, 3999.0, cfg.PremiumQuarterlyPrice)
	assert.Equal(t, 13999.0, cfg.PremiumYearlyPrice)
	assert.Equal(t, 0.10, cfg.PremiumCommissionRate)
	// Standard monthly was overridden independently of premium.
	assert.Equal(t, 599.0, cfg.MonthlyPrice)
}

func TestPlanConfigPriceFor(t *testing.T) {
	cfg := &PlanConfig{
		MonthlyPrice: 499, QuarterlyPrice: 1299, YearlyPrice: 4499,
		PremiumMonthlyPrice: 999, PremiumQuarterlyPrice: 2599, PremiumYearlyPrice: 8999,
	}
	assert.Equal(t, 499.0, cfg.PriceFor(models.TierStandard, models.BillingMonthly))
	assert.Equal(t, 4499.0, cfg.PriceFor(models.TierStandard, models.BillingYearly))
	assert.Equal(t, 999.0, cfg.PriceFor(models.TierPremium, models.BillingMonthly))
	assert.Equal(t, 2599.0, cfg.PriceFor(models.TierPremium, models.BillingQuarterly))
	assert.Equal(t, 8999.0, cfg.PriceFor(models.TierPremium, models.BillingYearly))
}
