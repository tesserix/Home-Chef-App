package database

// payment_id_index_test.go — #395·1. Proves the partial unique index semantics the postMigrate
// block relies on: real gateway ids are unique, but the empty default (wallet-only / unpaid /
// Stripe / meal-plan-day shell orders) may repeat freely. sqlite honours the same
// `WHERE col <> ''` partial-index syntax as Postgres, so this validates the constraint behaviour
// without a live Postgres.

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPaymentIDPartialUniqueIndex(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, razorpay_payment_id TEXT DEFAULT '')`).Error)
	// The exact index the postMigrate block creates.
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_razorpay_payment_id
		ON orders (razorpay_payment_id) WHERE razorpay_payment_id <> ''`).Error)

	// Many rows with the empty default are allowed — the partial WHERE excludes them.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, razorpay_payment_id) VALUES ('a',''),('b',''),('c','')`).Error)

	// A real payment id inserts once.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, razorpay_payment_id) VALUES ('d','pay_123')`).Error)

	// The same real payment id on a second order is BLOCKED — the money-integrity backstop.
	require.Error(t, db.Exec(`INSERT INTO orders (id, razorpay_payment_id) VALUES ('e','pay_123')`).Error,
		"a duplicate non-empty payment id must violate the partial unique index")

	// Re-creating the index is idempotent (IF NOT EXISTS) — the postMigrate block re-runs each boot.
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_razorpay_payment_id
		ON orders (razorpay_payment_id) WHERE razorpay_payment_id <> ''`).Error)
}
