package database

// retired_gateway_columns_test.go — #1086 leftovers. The tesserix-k8s schema
// declares these columns dropped, but the bootstrap CronJob skips a provisioned
// database, so production still carries them. AutoMigrate never drops a column
// it no longer knows about, which is why the drop has to live here.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newChefProfilesForDrop(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id TEXT PRIMARY KEY,
		business_name TEXT DEFAULT '',
		razorpay_account_id TEXT,
		razorpay_settlement_requirements TEXT)`).Error)
	return db
}

func TestDropRetiredGatewayColumns_DropsWhenEveryRowIsEmpty(t *testing.T) {
	db := newChefProfilesForDrop(t)
	// The three shapes an unused column holds in production: NULL, empty string,
	// and the JSON literal null a nil payload was serialised to.
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, razorpay_account_id, razorpay_settlement_requirements)
		VALUES ('a', NULL, NULL), ('b', '', 'null'), ('c', NULL, 'null')`).Error)

	require.NoError(t, dropRetiredGatewayColumns(db, []retiredColumn{
		{"chef_profiles", "razorpay_account_id"},
		{"chef_profiles", "razorpay_settlement_requirements"},
	}))

	require.False(t, db.Migrator().HasColumn("chef_profiles", "razorpay_account_id"))
	require.False(t, db.Migrator().HasColumn("chef_profiles", "razorpay_settlement_requirements"))

	var rows int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM chef_profiles`).Scan(&rows).Error)
	require.EqualValues(t, 3, rows, "dropping a dead column must not touch the rows")
}

func TestDropRetiredGatewayColumns_KeepsAColumnThatStillHoldsData(t *testing.T) {
	db := newChefProfilesForDrop(t)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, razorpay_account_id)
		VALUES ('a', 'acc_live_123')`).Error)

	require.NoError(t, dropRetiredGatewayColumns(db, []retiredColumn{
		{"chef_profiles", "razorpay_account_id"},
	}))

	require.True(t, db.Migrator().HasColumn("chef_profiles", "razorpay_account_id"),
		"a column with a value left in it is a fact about a payout destination — never dropped silently")
	var got string
	require.NoError(t, db.Raw(`SELECT razorpay_account_id FROM chef_profiles WHERE id = 'a'`).Scan(&got).Error)
	require.Equal(t, "acc_live_123", got)
}

func TestDropRetiredGatewayColumns_DropsAFlagNobodyEverSet(t *testing.T) {
	db := newChefProfilesForDrop(t)
	require.NoError(t, db.Exec(`ALTER TABLE chef_profiles ADD COLUMN razorpay_stakeholder_created BOOLEAN DEFAULT false`).Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, razorpay_stakeholder_created)
		VALUES ('a', false), ('b', false)`).Error)

	require.NoError(t, dropRetiredGatewayColumns(db, []retiredColumn{
		{"chef_profiles", "razorpay_stakeholder_created"},
	}))

	require.False(t, db.Migrator().HasColumn("chef_profiles", "razorpay_stakeholder_created"),
		"false is the default nobody wrote — it records nothing worth keeping")
}

func TestDropRetiredGatewayColumns_KeepsAFlagSomebodySet(t *testing.T) {
	db := newChefProfilesForDrop(t)
	require.NoError(t, db.Exec(`ALTER TABLE chef_profiles ADD COLUMN razorpay_stakeholder_created BOOLEAN DEFAULT false`).Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, razorpay_stakeholder_created)
		VALUES ('a', false), ('b', true)`).Error)

	require.NoError(t, dropRetiredGatewayColumns(db, []retiredColumn{
		{"chef_profiles", "razorpay_stakeholder_created"},
	}))

	require.True(t, db.Migrator().HasColumn("chef_profiles", "razorpay_stakeholder_created"),
		"a flag that was actually set says a stakeholder exists at the gateway")
}

func TestDropRetiredGatewayColumns_IsIdempotentAndSkipsMissingTables(t *testing.T) {
	db := newChefProfilesForDrop(t)
	cols := []retiredColumn{
		{"chef_profiles", "razorpay_account_id"},
		{"delivery_partners", "razorpay_account_id"}, // table does not exist here
	}
	require.NoError(t, dropRetiredGatewayColumns(db, cols))
	require.NoError(t, dropRetiredGatewayColumns(db, cols), "a second boot must be a no-op")
	require.False(t, db.Migrator().HasColumn("chef_profiles", "razorpay_account_id"))
}
