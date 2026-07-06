package services

// delivery_failure_reconcile_test.go — #594. The delivery-failure FREEZE (open a
// delivery_failed issue + dispute the hold) happens synchronously when a delivery is
// reported failed/returned. If that synchronous path errored (a 3PL webhook that gave up
// after its retry budget, an own-fleet retry that hit the cap but whose freeze crashed),
// the order is left in status=failed/returned with the hold NEVER disputed and no dispute
// ticket — money-safe (the hold stays `none`, so the chef is never wrongly paid) but the
// customer is never refunded and no admin is alerted. This reconcile is the durable
// safety net: it re-drives TerminalizeDeliveryFailure for any stranded failed/returned
// delivery. Runs regardless of the escrow flags — the freeze is plain DB state.

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

func setupDeliveryReconcileDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE deliveries (id TEXT PRIMARY KEY, order_id TEXT, status TEXT DEFAULT 'pending',
			failure_reason TEXT DEFAULT '', updated_at DATETIME, created_at DATETIME)`,
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT,
			chef_id TEXT, status TEXT, razorpay_order_id TEXT DEFAULT '', total REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', refund_amount REAL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, photo_urls TEXT, affected_item_ids TEXT, requested_amount REAL DEFAULT 0,
			refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending', resolved_by TEXT, resolved_at DATETIME,
			refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT, status TEXT,
			payout_transfer_id TEXT DEFAULT '', price REAL DEFAULT 0, payout_hold_status TEXT DEFAULT '',
			delivered_at DATETIME, refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
			next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedStranded inserts a gateway order + a failed/returned delivery updated `age` ago.
func seedStranded(t *testing.T, db *gorm.DB, orderStatus models.OrderStatus, hold models.PayoutHoldStatus, gateway bool, delStatus models.DeliveryStatus, age time.Duration) (uuid.UUID, uuid.UUID) {
	t.Helper()
	orderID, delID := uuid.New(), uuid.New()
	rzp := ""
	if gateway {
		rzp = "order_rzp_" + orderID.String()[:8]
	}
	require.NoError(t, db.Exec(`INSERT INTO orders (id, status, razorpay_order_id, chef_id, customer_id, payout_hold_status)
		VALUES (?,?,?,?,?,?)`, orderID.String(), string(orderStatus), rzp, uuid.NewString(), uuid.NewString(), string(hold)).Error)
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, status, updated_at) VALUES (?,?,?,?)`,
		delID.String(), orderID.String(), string(delStatus), time.Now().Add(-age)).Error)
	return orderID, delID
}

func holdOf(t *testing.T, db *gorm.DB, orderID uuid.UUID) models.PayoutHoldStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM orders WHERE id = ?`, orderID.String()).Scan(&s).Error)
	return models.PayoutHoldStatus(s)
}

func TestReconcileDeliveryFailures_FreezesStrandedOrder(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivering, models.PayoutHoldNone, true, models.DeliveryFailed, time.Hour)

	n := reconcileStrandedDeliveryFailures()
	require.Equal(t, 1, n)
	require.Equal(t, models.PayoutHoldDisputed, holdOf(t, db, orderID), "stranded order now frozen")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))
}

func TestReconcileDeliveryFailures_ReturnedAlsoFrozen(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivering, models.PayoutHoldNone, true, models.DeliveryReturned, time.Hour)
	require.Equal(t, 1, reconcileStrandedDeliveryFailures())
	require.Equal(t, models.PayoutHoldDisputed, holdOf(t, db, orderID))
}

func TestReconcileDeliveryFailures_AlreadyFrozenSkipped(t *testing.T) {
	// An order already frozen (pending delivery_failed issue) is not re-driven — no
	// duplicate issue, no re-dispute.
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivering, models.PayoutHoldDisputed, true, models.DeliveryFailed, time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, reason, status) VALUES (?,?,?,?)`,
		uuid.NewString(), orderID.String(), string(models.IssueDeliveryFailed), string(models.IssuePending)).Error)

	require.Equal(t, 0, reconcileStrandedDeliveryFailures(), "already frozen → skipped")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending), "no duplicate issue")
}

func TestReconcileDeliveryFailures_ResolvedOrderSkipped(t *testing.T) {
	// A resolved order (hold already released, issue no longer pending) must not be
	// re-frozen.
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivered, models.PayoutHoldReleased, true, models.DeliveryFailed, time.Hour)
	require.Equal(t, 0, reconcileStrandedDeliveryFailures())
	require.Equal(t, models.PayoutHoldReleased, holdOf(t, db, orderID), "settled hold untouched")
}

func TestReconcileDeliveryFailures_NonGatewaySkipped(t *testing.T) {
	// A group order (no razorpay_order_id) has no freeze handler yet (#594) — the
	// reconcile must not pick it every sweep and re-drive a froze=false no-op forever.
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivering, models.PayoutHoldNone, false, models.DeliveryFailed, time.Hour)
	require.Equal(t, 0, reconcileStrandedDeliveryFailures())
	require.Equal(t, models.PayoutHoldNone, holdOf(t, db, orderID))
}

func TestReconcileDeliveryFailures_WithinGraceSkipped(t *testing.T) {
	// A just-failed delivery (updated recently) is inside the grace window — its
	// synchronous freeze may be mid-retry, so the reconcile leaves it alone to avoid a
	// duplicate-issue race.
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivering, models.PayoutHoldNone, true, models.DeliveryFailed, time.Minute)
	require.Equal(t, 0, reconcileStrandedDeliveryFailures())
	require.Equal(t, models.PayoutHoldNone, holdOf(t, db, orderID), "not yet reconciled")
}

func TestReconcileDeliveryFailures_DeliveredNotTouched(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivered, models.PayoutHoldAwaitingConfirmation, true, models.DeliveryDelivered, time.Hour)
	require.Equal(t, 0, reconcileStrandedDeliveryFailures())
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, holdOf(t, db, orderID))
}
