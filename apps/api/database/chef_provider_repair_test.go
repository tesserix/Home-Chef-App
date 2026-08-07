package database

// chef_provider_repair_test.go — #1122. A chef row is CONFIGURATION, not a
// record of a payment: it says which gateway the next order should be routed
// through. Since #1086 razorpay can no longer take money, so a chef still
// carrying it is a row that names a rail nothing can settle on.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func providerRepairDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	for _, table := range []string{"chef_profiles", "delivery_partners"} {
		require.NoError(t, db.Exec(
			`CREATE TABLE `+table+` (id TEXT PRIMARY KEY, payment_provider TEXT DEFAULT '')`).Error)
	}
	return db
}

func providers(t *testing.T, db *gorm.DB, table string) []string {
	t.Helper()
	var got []string
	require.NoError(t, db.Raw(`SELECT payment_provider FROM `+table+` ORDER BY id`).Scan(&got).Error)
	return got
}

func TestNormalizeConfiguredProviders_MovesAnUnsettleableChefToCashfree(t *testing.T) {
	db := providerRepairDB(t)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, payment_provider) VALUES
		('a','razorpay'), ('b','cashfree'), ('c','stripe')`).Error)

	require.NoError(t, normalizeConfiguredProviders(db))

	require.Equal(t, []string{"cashfree", "cashfree", "stripe"}, providers(t, db, "chef_profiles"),
		"razorpay moves; stripe is still a live rail and must be left alone")
}

func TestNormalizeConfiguredProviders_RepairsBlankAndGarbledValues(t *testing.T) {
	db := providerRepairDB(t)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, payment_provider) VALUES
		('a',''), ('b','RAZORPAY'), ('c','razorpy'), ('d','wallet')`).Error)

	require.NoError(t, normalizeConfiguredProviders(db))

	// wallet is an order OUTCOME, never a chef's configuration — it cannot route
	// a new order either, so it is repaired like the rest.
	require.Equal(t, []string{"cashfree", "cashfree", "cashfree", "cashfree"},
		providers(t, db, "chef_profiles"))
}

func TestNormalizeConfiguredProviders_CoversDeliveryPartners(t *testing.T) {
	db := providerRepairDB(t)
	require.NoError(t, db.Exec(`INSERT INTO delivery_partners (id, payment_provider) VALUES ('a','razorpay')`).Error)

	require.NoError(t, normalizeConfiguredProviders(db))

	require.Equal(t, []string{"cashfree"}, providers(t, db, "delivery_partners"))
}

func TestNormalizeConfiguredProviders_IsIdempotent(t *testing.T) {
	db := providerRepairDB(t)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, payment_provider) VALUES ('a','razorpay')`).Error)

	for range 3 {
		require.NoError(t, normalizeConfiguredProviders(db))
	}
	require.Equal(t, []string{"cashfree"}, providers(t, db, "chef_profiles"))
}

func TestNormalizeConfiguredProviders_SkipsAMissingTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, normalizeConfiguredProviders(db))
}
