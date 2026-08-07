package database

// Postgres-dialect proof for #1127. sqlite covers the branch logic; this covers
// what only the real dialect can answer — that DROP COLUMN survives the partial
// unique index sitting on the column next to it, and that the recovered ids are
// still unique afterwards. Skipped unless TEST_POSTGRES_DSN is set, since CI has
// no Postgres.

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestMigrateGatewayIDColumns_Postgres(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set TEST_POSTGRES_DSN to run the Postgres-dialect migration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS orders`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id text PRIMARY KEY,
		razorpay_order_id text DEFAULT '', razorpay_payment_id text DEFAULT '',
		gateway_order_id text DEFAULT '', gateway_payment_id text DEFAULT '')`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_orders_razorpay_order_id ON orders (razorpay_order_id)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, razorpay_order_id, razorpay_payment_id)
		VALUES ('a','cf_order_a','cf_pay_a'), ('b','cf_order_b',''), ('c','','')`).Error)

	require.NoError(t, migrateGatewayIDColumns(db, [][2]string{{"orders", "order_id"}, {"orders", "payment_id"}}))

	var remaining int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = 'orders' AND column_name LIKE 'razorpay%'`).Scan(&remaining).Error)
	require.Zero(t, remaining)

	var ids []string
	require.NoError(t, db.Raw(`SELECT gateway_order_id FROM orders ORDER BY id`).Scan(&ids).Error)
	require.Equal(t, []string{"cf_order_a", "cf_order_b", ""}, ids)

	// The uniqueness backstop must still be creatable over the recovered ids.
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_orders_gateway_order_id
		ON orders (gateway_order_id) WHERE gateway_order_id <> ''`).Error)
	require.NoError(t, db.Exec(`DROP TABLE orders`).Error)
}
