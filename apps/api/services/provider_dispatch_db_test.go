package services

// provider_dispatch_db_test.go — CancelOrderDelivery must not read the global DB.
//
// It is called from a DETACHED goroutine on both cancellation paths
// (handlers/orders.go CancelOrder and handlers/chef_order_cancel.go), so it
// outlives the request that spawned it. Reading database.DB asynchronously means
// reading whatever that global happens to be by the time the goroutine is
// scheduled — which under test is nil, because the harness restores it on
// cleanup. The result was a nil-pointer panic inside gorm's getInstance, and a
// panic in a goroutine cannot be recovered by the test: it takes the whole binary
// down. That is what turned the API build red, intermittently.
//
// The fix is to hand the goroutine the handle that was live when the request was
// served. This test pins that: with a valid handle passed in, the call must
// succeed even when the global is nil.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

func TestCancelOrderDelivery_UsesThePassedHandleNotTheGlobal(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE deliveries (id TEXT PRIMARY KEY, order_id TEXT,
		provider_id TEXT, external_delivery_id TEXT, status TEXT, deleted_at DATETIME)`).Error)

	// Exactly the situation the goroutine races into: the request has finished
	// and the global has been torn down, but the goroutine still holds a handle.
	prev := database.DB
	database.DB = nil
	t.Cleanup(func() { database.DB = prev })

	require.NotPanics(t, func() {
		require.NoError(t, CancelOrderDelivery(db, uuid.New(), "chef cancelled"))
	}, "must use the handle it was given, not whatever the global is now")
}
