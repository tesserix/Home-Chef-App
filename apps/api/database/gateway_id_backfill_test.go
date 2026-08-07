package database

// gateway_id_backfill_test.go — #1127. AutoMigrate owns the schema in production
// (the db-schema-bootstrap CronJob logs "Database already provisioned — skipping
// DDL"), and AutoMigrate ADDS gateway_order_id beside razorpay_order_id rather
// than renaming it. So the #1119 rename left every historical id in a column
// nothing selects any more. This proves the repair, both shapes of it.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMigrateGatewayIDColumns_CopiesThenDropsWhenBothExist(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// The shape AutoMigrate actually left in production: both columns, ids in the old one.
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id TEXT PRIMARY KEY,
		razorpay_order_id TEXT DEFAULT '', razorpay_payment_id TEXT DEFAULT '',
		gateway_order_id TEXT DEFAULT '', gateway_payment_id TEXT DEFAULT '')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, razorpay_order_id, razorpay_payment_id)
		VALUES ('paid','cf_order_1','cf_pay_1'), ('unpaid','',''), ('order-only','cf_order_2','')`).Error)

	require.NoError(t, migrateGatewayIDColumns(db, [][2]string{{"orders", "order_id"}, {"orders", "payment_id"}}))

	require.False(t, db.Migrator().HasColumn("orders", "razorpay_order_id"),
		"the orphaned column must be dropped, not left to drift")
	require.False(t, db.Migrator().HasColumn("orders", "razorpay_payment_id"))

	var got []struct{ ID, GatewayOrderID, GatewayPaymentID string }
	require.NoError(t, db.Raw(`SELECT id, gateway_order_id, gateway_payment_id FROM orders ORDER BY id`).Scan(&got).Error)
	require.Len(t, got, 3)
	require.Equal(t, "cf_order_2", got[0].GatewayOrderID) // order-only
	require.Equal(t, "", got[0].GatewayPaymentID)
	require.Equal(t, "cf_order_1", got[1].GatewayOrderID) // paid
	require.Equal(t, "cf_pay_1", got[1].GatewayPaymentID)
	require.Equal(t, "", got[2].GatewayOrderID) // unpaid
}

func TestMigrateGatewayIDColumns_RenamesWhenTargetAbsent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// A database that never ran the new AutoMigrate: only the old column exists.
	require.NoError(t, db.Exec(`CREATE TABLE tips (id TEXT PRIMARY KEY, razorpay_order_id TEXT DEFAULT '')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO tips (id, razorpay_order_id) VALUES ('t1','cf_tip_1')`).Error)

	require.NoError(t, migrateGatewayIDColumns(db, [][2]string{{"tips", "order_id"}}))

	require.False(t, db.Migrator().HasColumn("tips", "razorpay_order_id"))
	require.True(t, db.Migrator().HasColumn("tips", "gateway_order_id"))
	var id string
	require.NoError(t, db.Raw(`SELECT gateway_order_id FROM tips WHERE id = 't1'`).Scan(&id).Error)
	require.Equal(t, "cf_tip_1", id, "a rename must not lose the id")
}

func TestMigrateGatewayIDColumns_IsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY,
		razorpay_order_id TEXT DEFAULT '', gateway_order_id TEXT DEFAULT '')`).Error)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, razorpay_order_id) VALUES ('a','cf_1')`).Error)

	cols := [][2]string{{"orders", "order_id"}}
	require.NoError(t, migrateGatewayIDColumns(db, cols))
	// Every boot re-runs Migrate(); the second pass must be a no-op, not an error.
	require.NoError(t, migrateGatewayIDColumns(db, cols))
	require.NoError(t, migrateGatewayIDColumns(db, cols))

	var id string
	require.NoError(t, db.Raw(`SELECT gateway_order_id FROM orders WHERE id = 'a'`).Scan(&id).Error)
	require.Equal(t, "cf_1", id)
}

func TestMigrateGatewayIDColumns_NeverOverwritesANewerID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY,
		razorpay_order_id TEXT DEFAULT '', gateway_order_id TEXT DEFAULT '')`).Error)
	// An order minted AFTER the new image rolled: the new column is authoritative.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, razorpay_order_id, gateway_order_id)
		VALUES ('a','cf_stale','cf_current')`).Error)

	require.NoError(t, migrateGatewayIDColumns(db, [][2]string{{"orders", "order_id"}}))

	var id string
	require.NoError(t, db.Raw(`SELECT gateway_order_id FROM orders WHERE id = 'a'`).Scan(&id).Error)
	require.Equal(t, "cf_current", id, "backfill must not clobber an id the new code already wrote")
}

func TestMigrateGatewayIDColumns_SkipsMissingTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	// A table this deployment does not have must not fail the whole boot.
	require.NoError(t, migrateGatewayIDColumns(db, [][2]string{{"not_a_table", "order_id"}}))
}
