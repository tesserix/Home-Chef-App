package services

// payout_crossguard_refund_test.go — the refund↔payout-hold cross-guard suite
// (#457, P0). Proves that (1) every order refund drives the hold via the shared
// WithholdOrReverseOrderHoldForRefund helper, (2) ReleaseHold refuses any
// aggregate whose underlying order is refunded / has refunded_at set / has a
// pending issue (order + meal-plan-day + group-order), and (3) the admin queue
// excludes refunded/cancelled/refunded_at orders and flags open-issue rows.
//
// The harness (setupCrossguardDB) extends setupReleaseDB's schema with the
// refund columns + the wallet tables so the auto-refund path (RefundIssueToWallet
// → CreditWallet) actually EXECUTES here — the AutoRefundDrivesHold test is real,
// not a vacuous -run match. Money seams hit GetRazorpay()==nil (no-op) and the
// escrow flags are OFF, so every transition is a pure DB state advance.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// setupCrossguardDB builds the extended sqlite harness: setupReleaseDB's tables
// with the full refund column set on orders + order_issues, plus wallets /
// wallet_txns so CreditWallet runs. It swaps database.DB (LogSystemAudit +
// CreditWallet's audit reach through it) and restores on cleanup.
func setupCrossguardDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT,
			chef_id TEXT, status TEXT, razorpay_order_id TEXT DEFAULT '', total REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, delivered_at DATETIME,
			payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
			refund_amount REAL DEFAULT 0, refund_reason TEXT, refund_initiated_by TEXT,
			refunded_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT,
			status TEXT, payout_transfer_id TEXT DEFAULT '', price REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, delivered_at DATETIME,
			payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
			date DATETIME, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT DEFAULT '',
			customer_id TEXT, chef_id TEXT, status TEXT)`,
		`CREATE TABLE group_orders (id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT, order_id TEXT,
			status TEXT, payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			customer_confirmed_at DATETIME, delivered_at DATETIME, payout_settled_at DATETIME,
			payout_settle_attempts INTEGER DEFAULT 0, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0,
			currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, photo_urls TEXT, affected_item_ids TEXT,
			requested_amount REAL DEFAULT 0, refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending',
			resolved_by TEXT, resolved_at DATETIME, refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0,
			currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT, source TEXT,
			amount REAL, balance_after REAL, currency TEXT, order_id TEXT, reason TEXT, created_by TEXT,
			idempotency_key TEXT UNIQUE, created_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT, value TEXT, type TEXT, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
			next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
		`CREATE TABLE audit_logs (id TEXT PRIMARY KEY, user_id TEXT, action TEXT, entity_type TEXT,
			entity_id TEXT, old_value TEXT, new_value TEXT, ip_address TEXT, correlation_id TEXT, created_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// seedCrossOrder inserts a delivered, gateway-charged order with an explicit
// status + refunded_at (the release-side backstop shapes) and returns its id and
// customer id (the wallet target for the issue path).
func seedCrossOrder(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, status string, refundedAt *time.Time) (uuid.UUID, uuid.UUID) {
	t.Helper()
	id, customer := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, razorpay_order_id, total, payout_hold_status, delivered_at, refunded_at)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		id.String(), "ORD-"+id.String()[:8], customer.String(), uuid.NewString(),
		status, "order_rzp_123", 250.0, string(hold), time.Now().Add(-30*time.Hour), refundedAt).Error)
	return id, customer
}

// seedCrossDay inserts a delivered meal-plan day in the given hold linked to the
// (optional) underlying order id, plus its parent plan.
func seedCrossDay(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, orderID *uuid.UUID) uuid.UUID {
	t.Helper()
	dayID, planID, chef := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status)
		VALUES (?,?,?,?,?)`, planID.String(), "MP-"+planID.String()[:8], uuid.NewString(), chef.String(), "active").Error)
	var orderStr any
	if orderID != nil {
		orderStr = orderID.String()
	}
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, order_id, status, payout_transfer_id, price, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), orderStr, "delivered", "trf_abc123", 120.0,
		string(hold), time.Now().Add(-10*time.Hour)).Error)
	return dayID
}

// seedCrossGroup inserts a delivered group order in the given hold linked to the
// (optional) underlying consolidated order id.
func seedCrossGroup(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, orderID *uuid.UUID) uuid.UUID {
	t.Helper()
	id, chef := uuid.New(), uuid.New()
	var orderStr any
	if orderID != nil {
		orderStr = orderID.String()
	}
	require.NoError(t, db.Exec(`INSERT INTO group_orders
		(id, host_id, chef_id, order_id, status, payout_hold_status, delivered_at, subtotal, tax)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), chef.String(), orderStr, "delivered",
		string(hold), time.Now().Add(-10*time.Hour), 200.0, 20.0).Error)
	return id
}

// seedPendingIssue inserts a pending OrderIssue for the given order/customer.
func seedPendingIssue(t *testing.T, db *gorm.DB, orderID, customerID uuid.UUID) *models.OrderIssue {
	t.Helper()
	iss := &models.OrderIssue{
		OrderID: orderID, ChefID: uuid.New(), CustomerID: customerID,
		Reason: models.IssueMissingItem, Status: models.IssuePending,
	}
	require.NoError(t, db.Create(iss).Error)
	return iss
}

// loadOrderHold reads the current payout_hold_status of an order.
func loadOrderHold(t *testing.T, db *gorm.DB, id uuid.UUID) models.PayoutHoldStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM orders WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.PayoutHoldStatus(s)
}

func flagsOff(t *testing.T) {
	t.Helper()
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{}
}

// ── Direct helper ──────────────────────────────────────────────────────────

func TestCrossguard_EligibleWithheld(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)

	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, id, "customer refund"))
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, id))
	require.ErrorIs(t, ReleaseHold(db, aggTypeOrder, id), ErrHoldNotEligible)
}

func TestCrossguard_AwaitingWithheld(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldAwaitingConfirmation, "delivered", nil)

	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, id, "customer refund"))
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, id))
}

func TestCrossguard_ReleasedReversed(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleased, "delivered", nil)

	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, id, "chargeback"))
	require.Equal(t, models.PayoutHoldReversed, loadOrderHold(t, db, id))
	require.NotNil(t, loadOrder(t, db, id).PayoutSettledAt, "reverse seam stamped settled once")
}

func TestCrossguard_NoopStates(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	for _, hold := range []models.PayoutHoldStatus{
		models.PayoutHoldNone, models.PayoutHoldWithheld,
		models.PayoutHoldReversed, models.PayoutHoldDisputed,
	} {
		id, _ := seedCrossOrder(t, db, hold, "delivered", nil)
		require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, id, "refund"))
		require.Equal(t, hold, loadOrderHold(t, db, id), "no-op for %s", hold)
	}
}

func TestCrossguard_Idempotent(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleased, "delivered", nil)

	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, id, "refund"))
	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, id, "refund"))
	require.Equal(t, models.PayoutHoldReversed, loadOrderHold(t, db, id))
	settled := loadOrder(t, db, id).PayoutSettledAt
	require.NotNil(t, settled)
}

// ── Wiring / issue path ────────────────────────────────────────────────────

func TestCrossguard_AutoRefundDrivesHold(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, customer := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)
	iss := seedPendingIssue(t, db, orderID, customer)

	// Auto-refund the issue (by="system"). This ACTUALLY credits the wallet and
	// then must drive the hold to withheld via the best-effort wiring.
	require.NoError(t, RefundIssueToWallet(db, iss, 120, "system", nil))

	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, orderID),
		"auto-refund drove the hold to withheld")
	require.ErrorIs(t, ReleaseHold(db, aggTypeOrder, orderID), ErrHoldNotEligible)
}

// ── Release-side backstop (refunded_at-aware) ──────────────────────────────

func TestCrossguard_ReleaseHold_BlocksRefundedAtOnly(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	now := time.Now()
	// The issue-path shape: status stays 'delivered' but refunded_at is set.
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", &now)
	require.ErrorIs(t, ReleaseHold(db, aggTypeOrder, id), ErrHoldNotEligible)
	require.Equal(t, models.PayoutHoldReleaseEligible, loadOrderHold(t, db, id), "no state change")
}

func TestCrossguard_ReleaseHold_BlocksRefundedStatus(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, string(models.OrderStatusRefunded), nil)
	require.ErrorIs(t, ReleaseHold(db, aggTypeOrder, id), ErrHoldNotEligible)
}

func TestCrossguard_ReleaseHold_BlocksPendingIssue(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, customer := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)
	seedPendingIssue(t, db, id, customer)
	require.ErrorIs(t, ReleaseHold(db, aggTypeOrder, id), ErrHoldNotEligible)
}

func TestCrossguard_ReleaseHold_AllowsClean(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)
	require.NoError(t, ReleaseHold(db, aggTypeOrder, id))
	require.Equal(t, models.PayoutHoldReleased, loadOrderHold(t, db, id))
}

// ── All-aggregate pre-check ────────────────────────────────────────────────

func TestCrossguard_ReleaseHold_Day_BlocksRefundedOrder(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	// A day linked to a refunded order is blocked.
	refOrder, _ := seedCrossOrder(t, db, models.PayoutHoldWithheld, string(models.OrderStatusRefunded), nil)
	blocked := seedCrossDay(t, db, models.PayoutHoldReleaseEligible, &refOrder)
	require.ErrorIs(t, ReleaseHold(db, aggTypeMealPlanDay, blocked), ErrHoldNotEligible)

	// A day with no underlying order releases fine.
	clean := seedCrossDay(t, db, models.PayoutHoldReleaseEligible, nil)
	require.NoError(t, ReleaseHold(db, aggTypeMealPlanDay, clean))
	require.Equal(t, models.PayoutHoldReleased, loadDayHold(t, db, clean))
}

func TestCrossguard_ReleaseHold_Group_BlocksRefundedOrder(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	refOrder, _ := seedCrossOrder(t, db, models.PayoutHoldWithheld, string(models.OrderStatusRefunded), nil)
	blocked := seedCrossGroup(t, db, models.PayoutHoldReleaseEligible, &refOrder)
	require.ErrorIs(t, ReleaseHold(db, aggTypeGroupOrder, blocked), ErrHoldNotEligible)

	clean := seedCrossGroup(t, db, models.PayoutHoldReleaseEligible, nil)
	require.NoError(t, ReleaseHold(db, aggTypeGroupOrder, clean))
}

// ── Queue ──────────────────────────────────────────────────────────────────

func TestCrossguard_ListPendingOrders_ExcludesRefunded(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	now := time.Now()
	clean, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)
	seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, string(models.OrderStatusRefunded), nil)
	seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, string(models.OrderStatusCancelled), nil)
	seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", &now) // refunded_at set

	rows, err := listPendingOrders(db, PendingFilter{})
	require.NoError(t, err)
	require.Len(t, rows, 1, "only the clean delivered order")
	require.Equal(t, clean, rows[0].ID)
}

func TestCrossguard_ListPendingPayouts_FlagsOpenIssue(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	withIssue, customer := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)
	seedPendingIssue(t, db, withIssue, customer)
	clean, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)

	rows, err := ListPendingPayouts(db, PendingFilter{})
	require.NoError(t, err)
	byID := map[uuid.UUID]PendingPayout{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	require.True(t, byID[withIssue].HasOpenIssue, "open-issue row flagged")
	require.False(t, byID[clean].HasOpenIssue, "clean row not flagged")
}
