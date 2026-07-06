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
		`CREATE TABLE group_orders (id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT, order_id TEXT, status TEXT,
			payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			created_at DATETIME, updated_at DATETIME)`,
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

// ── shell-order reconcile (#594): meal-plan-day + group shells have no
// razorpay_order_id, so the gateway reconcile above excludes them. These sweeps key
// off the shell aggregate (meal_plan_days.status / group_orders.status) instead. ──

// seedStrandedDay inserts a meal-plan-day SHELL order (no razorpay_order_id) with a
// failed/returned delivery aged `age` ago and a meal_plan_days row in `dayStatus`.
func seedStrandedDay(t *testing.T, db *gorm.DB, dayStatus models.MealPlanDayStatus, hold models.PayoutHoldStatus, delStatus models.DeliveryStatus, age time.Duration) (orderID, dayID uuid.UUID) {
	t.Helper()
	orderID, dayID, delID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, status, razorpay_order_id, chef_id, customer_id, payout_hold_status)
		VALUES (?,?,?,?,?,?)`, orderID.String(), string(models.OrderStatusDelivering), "", uuid.NewString(), uuid.NewString(), string(models.PayoutHoldNone)).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, payout_hold_status)
		VALUES (?,?,?,?,?)`, dayID.String(), uuid.NewString(), orderID.String(), string(dayStatus), string(hold)).Error)
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, status, updated_at) VALUES (?,?,?,?)`,
		delID.String(), orderID.String(), string(delStatus), time.Now().Add(-age)).Error)
	return orderID, dayID
}

// seedStrandedGroup inserts a consolidated GROUP SHELL order (no razorpay_order_id) with
// a failed/returned delivery aged `age` ago and a group_orders row in `groupStatus`.
func seedStrandedGroup(t *testing.T, db *gorm.DB, groupStatus models.GroupOrderStatus, hold models.PayoutHoldStatus, delStatus models.DeliveryStatus, age time.Duration) (orderID, groupID uuid.UUID) {
	t.Helper()
	orderID, groupID, delID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, status, razorpay_order_id, chef_id, customer_id, payout_hold_status)
		VALUES (?,?,?,?,?,?)`, orderID.String(), string(models.OrderStatusDelivering), "", uuid.NewString(), uuid.NewString(), string(models.PayoutHoldNone)).Error)
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, host_id, chef_id, order_id, status, payout_hold_status)
		VALUES (?,?,?,?,?,?)`, groupID.String(), uuid.NewString(), uuid.NewString(), orderID.String(), string(groupStatus), string(hold)).Error)
	require.NoError(t, db.Exec(`INSERT INTO deliveries (id, order_id, status, updated_at) VALUES (?,?,?,?)`,
		delID.String(), orderID.String(), string(delStatus), time.Now().Add(-age)).Error)
	return orderID, groupID
}

func dayHoldOf(t *testing.T, db *gorm.DB, dayID uuid.UUID) models.PayoutHoldStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&s).Error)
	return models.PayoutHoldStatus(s)
}

func groupHoldOf(t *testing.T, db *gorm.DB, groupID uuid.UUID) models.PayoutHoldStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM group_orders WHERE id = ?`, groupID.String()).Scan(&s).Error)
	return models.PayoutHoldStatus(s)
}

// ── meal-plan-day shell reconcile ──

func TestReconcileMealPlanDayFailures_FreezesStrandedDay(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	orderID, dayID := seedStrandedDay(t, db, models.MealPlanDayPrepared, models.PayoutHoldNone, models.DeliveryFailed, time.Hour)

	require.Equal(t, 1, reconcileStrandedMealPlanDayFailures())
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID), "day marked failed")
	require.Equal(t, models.PayoutHoldDisputed, dayHoldOf(t, db, dayID), "day hold frozen")
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayFailed))
	// The gateway sweep must leave the shell alone (its hold stays none, no order issue).
	require.Equal(t, models.PayoutHoldNone, holdOf(t, db, orderID), "shell order hold untouched by gateway sweep")
}

func TestReconcileMealPlanDayFailures_ReturnedAlsoFrozen(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	_, dayID := seedStrandedDay(t, db, models.MealPlanDayConfirmed, models.PayoutHoldNone, models.DeliveryReturned, time.Hour)
	require.Equal(t, 1, reconcileStrandedMealPlanDayFailures())
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))
}

func TestReconcileMealPlanDayFailures_AlreadyFailedSkipped(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	seedStrandedDay(t, db, models.MealPlanDayFailed, models.PayoutHoldDisputed, models.DeliveryFailed, time.Hour)
	require.Equal(t, 0, reconcileStrandedMealPlanDayFailures(), "already frozen day skipped")
	require.Equal(t, 0, countOutbox(t, db, SubjectMealPlanDayFailed), "no re-emit")
}

func TestReconcileMealPlanDayFailures_TerminalDaySkipped(t *testing.T) {
	// A day already resolved (delivered/refunded/cancelled) is not a strand — the freeze
	// would no-op on it, so the sweep must not select it and loop forever.
	for _, st := range []models.MealPlanDayStatus{
		models.MealPlanDayDelivered, models.MealPlanDayRefunded, models.MealPlanDayCancelled,
		models.MealPlanDaySkipped, models.MealPlanDayDeclined,
	} {
		t.Run(string(st), func(t *testing.T) {
			db := setupDeliveryReconcileDB(t)
			_, dayID := seedStrandedDay(t, db, st, models.PayoutHoldReleased, models.DeliveryFailed, time.Hour)
			require.Equal(t, 0, reconcileStrandedMealPlanDayFailures())
			require.Equal(t, st, loadDayStatus(t, db, dayID), "terminal day untouched")
		})
	}
}

func TestReconcileMealPlanDayFailures_WithinGraceSkipped(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	_, dayID := seedStrandedDay(t, db, models.MealPlanDayPrepared, models.PayoutHoldNone, models.DeliveryFailed, time.Minute)
	require.Equal(t, 0, reconcileStrandedMealPlanDayFailures(), "inside grace window")
	require.Equal(t, models.MealPlanDayPrepared, loadDayStatus(t, db, dayID))
}

func TestReconcileMealPlanDayFailures_IgnoresGatewayOrder(t *testing.T) {
	// A gateway order (razorpay_order_id set, no meal_plan_days row) is the gateway
	// sweep's job — the day sweep must not touch it.
	db := setupDeliveryReconcileDB(t)
	orderID, _ := seedStranded(t, db, models.OrderStatusDelivering, models.PayoutHoldNone, true, models.DeliveryFailed, time.Hour)
	require.Equal(t, 0, reconcileStrandedMealPlanDayFailures())
	require.Equal(t, models.PayoutHoldNone, holdOf(t, db, orderID))
}

// ── group shell reconcile ──

func TestReconcileGroupFailures_FreezesStrandedGroup(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	orderID, groupID := seedStrandedGroup(t, db, models.GroupOrderConfirmed, models.PayoutHoldNone, models.DeliveryFailed, time.Hour)

	require.Equal(t, 1, reconcileStrandedGroupFailures())
	require.Equal(t, models.GroupOrderFailed, groupStatusOf(t, db, groupID), "group marked failed")
	require.Equal(t, models.PayoutHoldDisputed, groupHoldOf(t, db, groupID), "group hold frozen")
	require.Equal(t, 1, countOutbox(t, db, SubjectGroupOrderFailed))
	require.Equal(t, models.PayoutHoldNone, holdOf(t, db, orderID), "shell order hold untouched")
}

func TestReconcileGroupFailures_ReturnedAlsoFrozen(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	_, groupID := seedStrandedGroup(t, db, models.GroupOrderPlaced, models.PayoutHoldNone, models.DeliveryReturned, time.Hour)
	require.Equal(t, 1, reconcileStrandedGroupFailures())
	require.Equal(t, models.GroupOrderFailed, groupStatusOf(t, db, groupID))
}

func TestReconcileGroupFailures_AlreadyFailedSkipped(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	seedStrandedGroup(t, db, models.GroupOrderFailed, models.PayoutHoldDisputed, models.DeliveryFailed, time.Hour)
	require.Equal(t, 0, reconcileStrandedGroupFailures())
	require.Equal(t, 0, countOutbox(t, db, SubjectGroupOrderFailed))
}

func TestReconcileGroupFailures_TerminalGroupSkipped(t *testing.T) {
	for _, st := range []models.GroupOrderStatus{
		models.GroupOrderDelivered, models.GroupOrderCancelled, models.GroupOrderExpired,
	} {
		t.Run(string(st), func(t *testing.T) {
			db := setupDeliveryReconcileDB(t)
			_, groupID := seedStrandedGroup(t, db, st, models.PayoutHoldReleased, models.DeliveryFailed, time.Hour)
			require.Equal(t, 0, reconcileStrandedGroupFailures())
			require.Equal(t, st, groupStatusOf(t, db, groupID))
		})
	}
}

func TestReconcileGroupFailures_WithinGraceSkipped(t *testing.T) {
	db := setupDeliveryReconcileDB(t)
	_, groupID := seedStrandedGroup(t, db, models.GroupOrderConfirmed, models.PayoutHoldNone, models.DeliveryFailed, time.Minute)
	require.Equal(t, 0, reconcileStrandedGroupFailures())
	require.Equal(t, models.GroupOrderConfirmed, groupStatusOf(t, db, groupID))
}

func TestReconcileShellFailures_DayAndGroupSweepsAreIsolated(t *testing.T) {
	// A stranded day and a stranded group in the same DB: each sweep freezes ONLY its own
	// shape (the JOIN scopes it), and neither touches the other. Guards against a future
	// query change that would let one sweep pick up the wrong shell shape.
	db := setupDeliveryReconcileDB(t)
	_, dayID := seedStrandedDay(t, db, models.MealPlanDayPrepared, models.PayoutHoldNone, models.DeliveryFailed, time.Hour)
	_, groupID := seedStrandedGroup(t, db, models.GroupOrderConfirmed, models.PayoutHoldNone, models.DeliveryFailed, time.Hour)

	require.Equal(t, 1, reconcileStrandedMealPlanDayFailures(), "day sweep freezes only the day")
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))
	require.Equal(t, models.GroupOrderConfirmed, groupStatusOf(t, db, groupID), "group untouched by day sweep")

	require.Equal(t, 1, reconcileStrandedGroupFailures(), "group sweep freezes only the group")
	require.Equal(t, models.GroupOrderFailed, groupStatusOf(t, db, groupID))
}
