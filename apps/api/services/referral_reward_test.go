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
)

// referral_reward_test.go — #38. The reward engine: it must pay BOTH wallets
// exactly once on the referee's first paid order, be idempotent against
// duplicate webhooks, NOT pay on a non-first order, and respect the monthly
// spend cap + the enabled toggle. Money correctness is the whole point, so each
// path is pinned.

func setupRewardDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, customer_id TEXT, payment_status TEXT, deleted_at DATETIME)`,
		`CREATE TABLE referrals (id TEXT PRIMARY KEY, referrer_user_id TEXT, referee_user_id TEXT UNIQUE, code TEXT,
			status TEXT, order_id TEXT, referrer_reward REAL DEFAULT 0, referee_reward REAL DEFAULT 0,
			rewarded_at DATETIME, referee_device TEXT, referee_ip TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0, currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT, source TEXT,
			amount REAL, balance_after REAL, currency TEXT, order_id TEXT, reason TEXT, created_by TEXT,
			idempotency_key TEXT UNIQUE, created_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT, type TEXT, updated_by TEXT, updated_at DATETIME)`,
		`CREATE TABLE users (email_enc text DEFAULT '', email_bidx text DEFAULT '', first_name_enc text DEFAULT '', last_name_enc text DEFAULT '', phone_enc text DEFAULT '', phone_bidx text DEFAULT '', id TEXT PRIMARY KEY, fcm_token TEXT, deleted_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT,
			payload TEXT, status TEXT, attempts INT, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func seedPendingReferral(t *testing.T, db *gorm.DB, referrer, referee uuid.UUID, device string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO referrals (id, referrer_user_id, referee_user_id, code, status, referee_device) VALUES (?, ?, ?, 'CODE2345', 'pending', ?)`,
		id.String(), referrer.String(), referee.String(), device,
	).Error)
	return id
}

func seedPaidOrder(t *testing.T, db *gorm.DB, customer uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, payment_status) VALUES (?, ?, 'completed')`, id.String(), customer.String()).Error)
	return id
}

func balanceOf(t *testing.T, db *gorm.DB, userID uuid.UUID) float64 {
	t.Helper()
	var bal float64
	db.Raw(`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = ?`, userID.String()).Scan(&bal)
	return bal
}

func setReferralSetting(t *testing.T, db *gorm.DB, key, value string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value) VALUES (?, ?, ?)`, uuid.New().String(), key, value).Error)
}

func TestMaybeGrantReward_HappyPath(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	refID := seedPendingReferral(t, db, referrer, referee, "")
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)

	assert.Equal(t, 100.0, balanceOf(t, db, referrer), "referrer credited default reward")
	assert.Equal(t, 100.0, balanceOf(t, db, referee), "referee credited default reward")

	var status string
	db.Raw(`SELECT status FROM referrals WHERE id = ?`, refID.String()).Scan(&status)
	assert.Equal(t, "rewarded", status)

	// An outbox notification was enqueued.
	var n int64
	db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, SubjectReferralRewarded).Scan(&n)
	assert.Equal(t, int64(1), n)
}

func TestMaybeGrantReward_Idempotent(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)
	MaybeGrantReward(db, orderID) // duplicate webhook

	assert.Equal(t, 100.0, balanceOf(t, db, referrer), "no double credit on retry")
	assert.Equal(t, 100.0, balanceOf(t, db, referee))
}

func TestMaybeGrantReward_NotFirstPaidOrder(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	_ = seedPaidOrder(t, db, referee) // a PRIOR paid order
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)

	assert.Equal(t, 0.0, balanceOf(t, db, referrer), "no reward when it isn't the first paid order")
}

func TestMaybeGrantReward_SpendCap(t *testing.T) {
	db := setupRewardDB(t)
	setReferralSetting(t, db, "referral.monthly_spend_cap", "150") // < 100+100
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)

	assert.Equal(t, 0.0, balanceOf(t, db, referrer), "grant skipped when it would exceed the monthly cap")
}

func TestMaybeGrantReward_Disabled(t *testing.T) {
	db := setupRewardDB(t)
	setReferralSetting(t, db, "referral.enabled", "false")
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)

	assert.Equal(t, 0.0, balanceOf(t, db, referrer), "no reward when the program is disabled")
}

func TestMaybeGrantReward_DeviceDedupe(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	// Referrer and referee share a device token → self-referral signal.
	require.NoError(t, db.Exec(`INSERT INTO users (id, fcm_token) VALUES (?, 'SHARED-DEVICE')`, referrer.String()).Error)
	refID := seedPendingReferral(t, db, referrer, referee, "SHARED-DEVICE")
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)

	assert.Equal(t, 0.0, balanceOf(t, db, referrer), "device-shared referral is rejected, not paid")
	var status string
	db.Raw(`SELECT status FROM referrals WHERE id = ?`, refID.String()).Scan(&status)
	assert.Equal(t, "rejected", status)
}
