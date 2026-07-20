package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// premium_test.go — #44. PremiumChefIDs is the gate every perk reads, so it must
// only ever return chefs whose premium subscription is genuinely live: tier must
// be premium AND status trial/active. A lapsed (past_due/cancelled) premium sub,
// a standard-tier sub, or a soft-deleted row must NOT grant perks.

func setupPremiumDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, user_id TEXT)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE subscriptions (
			id         TEXT PRIMARY KEY,
			user_id    TEXT,
			tier       TEXT,
			status     TEXT,
			deleted_at DATETIME
		)
	`).Error)
	return db
}

func seedChefSub(t *testing.T, db *gorm.DB, tier models.SubscriptionTier, status models.SubscriptionStatus, deleted bool) uuid.UUID {
	t.Helper()
	chefID, userID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?, ?)`, chefID.String(), userID.String()).Error)
	del := "NULL"
	if deleted {
		del = "'2026-01-01 00:00:00'"
	}
	require.NoError(t, db.Exec(
		`INSERT INTO subscriptions (id, user_id, tier, status, deleted_at) VALUES (?, ?, ?, ?, `+del+`)`,
		uuid.New().String(), userID.String(), string(tier), string(status),
	).Error)
	return chefID
}

func TestPremiumChefIDs(t *testing.T) {
	db := setupPremiumDB(t)
	database.DB = db

	premiumActive := seedChefSub(t, db, models.TierPremium, models.SubStatusActive, false)
	premiumTrial := seedChefSub(t, db, models.TierPremium, models.SubStatusTrial, false)
	premiumPastDue := seedChefSub(t, db, models.TierPremium, models.SubStatusPastDue, false)
	premiumCancelled := seedChefSub(t, db, models.TierPremium, models.SubStatusCancelled, false)
	standardActive := seedChefSub(t, db, models.TierStandard, models.SubStatusActive, false)
	premiumDeleted := seedChefSub(t, db, models.TierPremium, models.SubStatusActive, true)

	ids := []uuid.UUID{premiumActive, premiumTrial, premiumPastDue, premiumCancelled, standardActive, premiumDeleted}
	got := PremiumChefIDs(ids)

	assert.True(t, got[premiumActive], "active premium grants perks")
	assert.True(t, got[premiumTrial], "trial premium grants perks")
	assert.False(t, got[premiumPastDue], "past-due premium does NOT grant perks")
	assert.False(t, got[premiumCancelled], "cancelled premium does NOT grant perks")
	assert.False(t, got[standardActive], "standard tier does NOT grant perks")
	assert.False(t, got[premiumDeleted], "soft-deleted subscription does NOT grant perks")

	assert.True(t, IsChefPremium(premiumActive))
	assert.False(t, IsChefPremium(standardActive))
}

func TestPremiumChefIDsEmpty(t *testing.T) {
	db := setupPremiumDB(t)
	database.DB = db
	assert.Empty(t, PremiumChefIDs(nil))
}
