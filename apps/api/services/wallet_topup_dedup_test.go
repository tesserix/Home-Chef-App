package services

// wallet_topup_dedup_test.go — #554. The platform-funded wallet top-up transfer had
// no idempotency guard, so a repeated VerifyPayment could issue the same real money
// transfer twice. ClaimWalletTopUp claims each (order, account) exactly once via the
// processed_events ledger; ReleaseWalletTopUp lets a failed transfer retry.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func setupTopUpDedupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE processed_events (
		consumer TEXT NOT NULL, msg_id TEXT NOT NULL, subject TEXT DEFAULT '',
		processed_at DATETIME, PRIMARY KEY (consumer, msg_id)
	)`).Error)
	return db
}

func TestClaimWalletTopUp_SecondClaimIsSkip(t *testing.T) {
	db := setupTopUpDedupDB(t)
	orderID := uuid.New()

	first, err := ClaimWalletTopUp(db, orderID, 0, "acc_chef")
	require.NoError(t, err)
	require.True(t, first, "first claim transfers")

	second, err := ClaimWalletTopUp(db, orderID, 0, "acc_chef")
	require.NoError(t, err)
	require.False(t, second, "repeat settlement skips — no double transfer")
}

func TestClaimWalletTopUp_DistinctAccountsEachClaimOnce(t *testing.T) {
	db := setupTopUpDedupDB(t)
	orderID := uuid.New()

	chef, err := ClaimWalletTopUp(db, orderID, 0, "acc_chef")
	require.NoError(t, err)
	driver, err := ClaimWalletTopUp(db, orderID, 0, "acc_driver")
	require.NoError(t, err)
	require.True(t, chef)
	require.True(t, driver, "a distinct account is a distinct top-up")
}

func TestReleaseWalletTopUp_AllowsRetryAfterFailure(t *testing.T) {
	db := setupTopUpDedupDB(t)
	orderID := uuid.New()

	first, err := ClaimWalletTopUp(db, orderID, 0, "acc_chef")
	require.NoError(t, err)
	require.True(t, first)

	// The transfer failed → release so the retry re-attempts it.
	ReleaseWalletTopUp(db, orderID, 0, "acc_chef")

	retry, err := ClaimWalletTopUp(db, orderID, 0, "acc_chef")
	require.NoError(t, err)
	require.True(t, retry, "after release the retry re-attempts the transfer")
}

// Different orders never collide even for the same account.
func TestClaimWalletTopUp_DistinctOrders(t *testing.T) {
	db := setupTopUpDedupDB(t)
	a, err := ClaimWalletTopUp(db, uuid.New(), 0, "acc_chef")
	require.NoError(t, err)
	b, err := ClaimWalletTopUp(db, uuid.New(), 0, "acc_chef")
	require.NoError(t, err)
	require.True(t, a)
	require.True(t, b)
}
