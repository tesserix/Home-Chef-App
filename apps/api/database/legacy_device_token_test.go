package database

// legacy_device_token_test.go — #1164. Push now reads user_devices only, but
// the bootstrap CronJob does not replay DDL against a provisioned database
// (#1127), so its INSERT..SELECT never runs in prod: an account whose token
// still sits in users.fcm_token would go silent until the next sign-in.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func legacyTokenDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(
		`CREATE TABLE users (id TEXT PRIMARY KEY, fcm_token TEXT)`).Error)
	require.NoError(t, db.Exec(
		`CREATE TABLE user_devices (
			id TEXT PRIMARY KEY DEFAULT (hex(randomblob(16))),
			user_id TEXT NOT NULL, device_id TEXT NOT NULL, app TEXT NOT NULL,
			fcm_token TEXT, first_seen_at DATETIME, last_seen_at DATETIME,
			revoked_at DATETIME)`).Error)
	require.NoError(t, db.Exec(
		`CREATE UNIQUE INDEX idx_user_devices_user_device ON user_devices (user_id, device_id)`).Error)
	return db
}

func deviceTokens(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var got []string
	require.NoError(t, db.Raw(
		`SELECT fcm_token FROM user_devices ORDER BY user_id`).Scan(&got).Error)
	return got
}

func TestAdoptLegacyDeviceTokens_KeepsAnExistingInstallReachable(t *testing.T) {
	db := legacyTokenDB(t)
	require.NoError(t, db.Exec(`INSERT INTO users (id, fcm_token) VALUES
		('a','tok-a'), ('b','tok-b')`).Error)

	require.NoError(t, adoptLegacyDeviceTokens(db))

	require.Equal(t, []string{"tok-a", "tok-b"}, deviceTokens(t, db))
}

func TestAdoptLegacyDeviceTokens_IgnoresAccountsWithNothingToCarry(t *testing.T) {
	db := legacyTokenDB(t)
	require.NoError(t, db.Exec(`INSERT INTO users (id, fcm_token) VALUES
		('a', NULL), ('b','')`).Error)

	require.NoError(t, adoptLegacyDeviceTokens(db))

	require.Empty(t, deviceTokens(t, db))
}

// Runs on every boot, so a second pass must not duplicate the row or clobber a
// token the device has since re-registered for itself.
func TestAdoptLegacyDeviceTokens_IsIdempotent(t *testing.T) {
	db := legacyTokenDB(t)
	require.NoError(t, db.Exec(`INSERT INTO users (id, fcm_token) VALUES ('a','tok-a')`).Error)
	require.NoError(t, adoptLegacyDeviceTokens(db))
	require.NoError(t, db.Exec(
		`UPDATE user_devices SET fcm_token = 'tok-fresh' WHERE user_id = 'a'`).Error)

	require.NoError(t, adoptLegacyDeviceTokens(db))

	require.Equal(t, []string{"tok-fresh"}, deviceTokens(t, db))
}

func TestAdoptLegacyDeviceTokens_SkipsADatabaseWithoutTheLegacyColumn(t *testing.T) {
	db := legacyTokenDB(t)
	require.NoError(t, db.Exec(`DROP TABLE users`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY)`).Error)

	require.NoError(t, adoptLegacyDeviceTokens(db))
}
