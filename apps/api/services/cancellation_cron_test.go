package services

// cancellation_cron_test.go — durability: an owed refund is never lost (the sweep
// retries until executed, idempotently) and a stuck vendor request always times
// out to admin review.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupSweepDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id text PRIMARY KEY, customer_id text, chef_id text, status text, payment_status text,
			payment_provider text DEFAULT '', razorpay_payment_id text DEFAULT '', subtotal real DEFAULT 0,
			delivery_fee real DEFAULT 0, service_fee real DEFAULT 0, tax real DEFAULT 0, total real DEFAULT 0,
			refund_amount real DEFAULT 0, refund_reason text, refunded_at datetime, created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE cancellation_requests (id text PRIMARY KEY, order_id text, customer_id text, chef_id text,
			status text, vendor_reason text, refund_destination text, refund_total_paise integer DEFAULT 0,
			vendor_kept_paise integer DEFAULT 0, platform_kept_paise integer DEFAULT 0, refund_executed integer DEFAULT 0,
			refund_ref text, vendor_respond_by datetime, resolved_at datetime, created_at datetime, updated_at datetime)`,
		`CREATE TABLE wallets (id text PRIMARY KEY, user_id text UNIQUE, balance real DEFAULT 0, currency text DEFAULT 'INR', created_at datetime, updated_at datetime)`,
		`CREATE TABLE wallet_txns (id text PRIMARY KEY, wallet_id text, user_id text, type text, source text,
			amount real, balance_after real, currency text, order_id text, reason text, created_by text,
			idempotency_key text UNIQUE, created_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// An approved request whose inline refund never ran (refund_executed=false) is
// picked up by the sweep, refunded to the wallet, and marked executed — once.
func TestSweepCancellationRefunds_RetriesOwedOnce(t *testing.T) {
	db := setupSweepDB(t)
	cust, chef := uuid.New(), uuid.New()
	oid, reqID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, chef_id, status, payment_status, subtotal, total, refund_amount)
		VALUES (?,?,?,?,?,500,630,0)`, oid.String(), cust.String(), chef.String(), "preparing", "completed").Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?,?,?,?,?,0)`, reqID.String(), oid.String(), cust.String(), chef.String(), "approved", "wallet", 25200).Error)

	SweepCancellationRefunds()

	var executed bool
	db.Raw(`SELECT refund_executed FROM cancellation_requests WHERE id = ?`, reqID.String()).Scan(&executed)
	require.True(t, executed, "the owed refund was retried and executed")
	var bal float64
	db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, cust.String()).Scan(&bal)
	require.InDelta(t, 252.0, bal, 0.001)

	// A second sweep must NOT double-refund (idempotent).
	SweepCancellationRefunds()
	db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, cust.String()).Scan(&bal)
	require.InDelta(t, 252.0, bal, 0.001, "no double refund on a second sweep")
}

func TestSweepCancellationTimeouts_RoutesStuckToAdmin(t *testing.T) {
	db := setupSweepDB(t)
	past := time.Now().Add(-time.Minute)
	future := time.Now().Add(time.Hour)
	stuck, fresh := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, vendor_respond_by)
		VALUES (?,?,?,?,?,?)`, stuck.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), "pending_vendor", past).Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, vendor_respond_by)
		VALUES (?,?,?,?,?,?)`, fresh.String(), uuid.NewString(), uuid.NewString(), uuid.NewString(), "pending_vendor", future).Error)

	SweepCancellationTimeouts(time.Now())

	var s1, s2 string
	db.Raw(`SELECT status FROM cancellation_requests WHERE id = ?`, stuck.String()).Scan(&s1)
	db.Raw(`SELECT status FROM cancellation_requests WHERE id = ?`, fresh.String()).Scan(&s2)
	require.Equal(t, string(models.CancelReqAdminReview), s1, "past-deadline request → admin review")
	require.Equal(t, "pending_vendor", s2, "in-window request untouched")
}
