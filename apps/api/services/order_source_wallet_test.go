package services

// order_source_wallet_test.go — WalletRefundEligible: only meal-plan, subscription, and group
// orders may refund to the wallet; a plain à-la-carte order may not.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestWalletRefundEligible(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		// deleted_at present so GORM's soft-delete scope (added for models with a DeletedAt)
		// doesn't error the classification query against a minimal table.
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, order_id TEXT, deleted_at DATETIME)`,
		`CREATE TABLE meal_subscription_fulfillments (id TEXT PRIMARY KEY, order_id TEXT, deleted_at DATETIME)`,
		`CREATE TABLE group_orders (id TEXT PRIMARY KEY, order_id TEXT, deleted_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}

	mealOrder, subOrder, groupOrder, alacarte := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id) VALUES (?,?)`, uuid.NewString(), mealOrder.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_subscription_fulfillments (id, order_id) VALUES (?,?)`, uuid.NewString(), subOrder.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, order_id) VALUES (?,?)`, uuid.NewString(), groupOrder.String()).Error)

	require.True(t, WalletRefundEligible(db, mealOrder), "meal-plan order → wallet ok")
	require.True(t, WalletRefundEligible(db, subOrder), "subscription order → wallet ok")
	require.True(t, WalletRefundEligible(db, groupOrder), "group order → wallet ok")
	require.False(t, WalletRefundEligible(db, alacarte), "à-la-carte order → wallet BLOCKED")
}
