package services

// payout_hold_test.go — the payout hold state machine (#387). Delivery parks a
// hold (awaiting_customer_confirmation, no money moves); an explicit customer
// confirmation advances awaiting -> release_eligible unless an open OrderIssue
// disputes it. Pure DB logic on an in-memory sqlite harness (hand-DDL'd because
// the models' gen_random_uuid() default can't run on sqlite).

import (
	"context"
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

func setupHoldDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, customer_id TEXT, status TEXT,
			razorpay_order_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
			customer_confirmed_at DATETIME, delivered_at DATETIME, refunded_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT,
			status TEXT, payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
			customer_confirmed_at DATETIME, delivered_at DATETIME, date DATETIME,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, status TEXT)`,
		`CREATE TABLE group_orders (id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT, order_id TEXT,
			status TEXT, payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			customer_confirmed_at DATETIME, delivered_at DATETIME, payout_settled_at DATETIME,
			payout_settle_attempts INTEGER DEFAULT 0, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0,
			currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT, value TEXT, type TEXT, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
			next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

// countOutbox returns how many staged outbox rows carry the given subject.
func countOutbox(t *testing.T, db *gorm.DB, subject string) int {
	t.Helper()
	var n int
	require.NoError(t, db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, subject).Scan(&n).Error)
	return n
}

func seedRegularOrder(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status) VALUES (?,?,?,?,?)`,
		id.String(), uuid.NewString(), "delivered", "order_rzp_123", string(hold)).Error)
	return id
}

// seedReleasedOrder inserts a gateway-charged order already flipped to the given
// terminal hold status with the given (nullable) settled_at — the drift shape the
// reconcile re-drives (released/reversed + settled_at NULL).
func seedReleasedOrder(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, settledAt *time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status, payout_settled_at) VALUES (?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), "delivered", "order_rzp_123", string(hold), settledAt).Error)
	return id
}

func loadOrder(t *testing.T, db *gorm.DB, id uuid.UUID) models.Order {
	t.Helper()
	var o models.Order
	require.NoError(t, db.First(&o, "id = ?", id).Error)
	return o
}

// SetOrderHoldAwaitingConfirmation parks a gateway-charged order in the hold; a
// meal-plan/consolidated order (no razorpay_order_id) is a no-op.
func TestSetOrderHoldAwaiting_RegularVsConsolidated(t *testing.T) {
	db := setupHoldDB(t)

	reg := seedRegularOrder(t, db, models.PayoutHoldNone)
	require.NoError(t, SetOrderHoldAwaitingConfirmation(db, reg))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadOrder(t, db, reg).PayoutHoldStatus)

	// Consolidated order with no razorpay id → no-op, stays "".
	cons := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status) VALUES (?,?,?,?,?)`,
		cons.String(), uuid.NewString(), "delivered", "", "").Error)
	require.NoError(t, SetOrderHoldAwaitingConfirmation(db, cons))
	require.Equal(t, models.PayoutHoldNone, loadOrder(t, db, cons).PayoutHoldStatus)
}

func TestSetMealPlanDayHoldAwaiting(t *testing.T) {
	db := setupHoldDB(t)
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, payout_hold_status) VALUES (?,?,?,?)`,
		dayID.String(), uuid.NewString(), "delivered", "").Error)
	require.NoError(t, SetMealPlanDayHoldAwaitingConfirmation(db, dayID))
	var day models.MealPlanDay
	require.NoError(t, db.First(&day, "id = ?", dayID).Error)
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, day.PayoutHoldStatus)
}

// BLOCKER 3: with the auto-release flag ON, driving the settle saga seam for a
// regular order parks the hold and releases nothing — the flag no longer implies
// release on delivery.
func TestSettleSaga_RegularOrder_HoldsNoRelease_FlagOn(t *testing.T) {
	saved := config.AppConfig
	defer func() { config.AppConfig = saved }()
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}

	db := setupHoldDB(t)
	origDB := database.DB
	database.DB = db
	defer func() { database.DB = origDB }()

	id := seedRegularOrder(t, db, models.PayoutHoldNone)
	require.NoError(t, SettleOrderPayouts(context.Background(), id))

	o := loadOrder(t, db, id)
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, o.PayoutHoldStatus, "delivery must park the hold, not release")
	require.Nil(t, o.RefundedAt, "no release/refund side-effect on the order")
}

// Confirm advances awaiting -> release_eligible and stamps CustomerConfirmedAt.
func TestConfirmOrderHold_AwaitingToReleaseEligible(t *testing.T) {
	db := setupHoldDB(t)
	id := seedRegularOrder(t, db, models.PayoutHoldAwaitingConfirmation)
	order := loadOrder(t, db, id)

	status, err := ConfirmOrderHold(db, &order)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldReleaseEligible, status)

	got := loadOrder(t, db, id)
	require.Equal(t, models.PayoutHoldReleaseEligible, got.PayoutHoldStatus)
	require.NotNil(t, got.CustomerConfirmedAt)

	// The confirm-endpoint path emits exactly one release-eligible event.
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))
	require.Equal(t, 0, countOutbox(t, db, SubjectHoldDisputed))
}

// An open (pending) OrderIssue forces disputed, never release_eligible.
func TestConfirmOrderHold_OpenIssueDisputes(t *testing.T) {
	db := setupHoldDB(t)
	id := seedRegularOrder(t, db, models.PayoutHoldAwaitingConfirmation)
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), id.String(), string(models.IssuePending)).Error)
	order := loadOrder(t, db, id)

	status, err := ConfirmOrderHold(db, &order)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldDisputed, status)
	require.Equal(t, models.PayoutHoldDisputed, loadOrder(t, db, id).PayoutHoldStatus)

	// A disputed transition emits hold_disputed and never release_eligible.
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed))
	require.Equal(t, 0, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// Re-confirming an already-confirmed order is an idempotent no-op that keeps the
// terminal status and does not overwrite CustomerConfirmedAt.
func TestConfirmOrderHold_Idempotent(t *testing.T) {
	db := setupHoldDB(t)
	id := seedRegularOrder(t, db, models.PayoutHoldAwaitingConfirmation)
	order := loadOrder(t, db, id)

	first, err := ConfirmOrderHold(db, &order)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldReleaseEligible, first)
	stamp := loadOrder(t, db, id).CustomerConfirmedAt
	require.NotNil(t, stamp)

	time.Sleep(2 * time.Millisecond)
	again := loadOrder(t, db, id)
	second, err := ConfirmOrderHold(db, &again)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldReleaseEligible, second)
	require.WithinDuration(t, *stamp, *loadOrder(t, db, id).CustomerConfirmedAt, time.Millisecond, "confirmed stamp must not be overwritten")

	// The no-op re-confirm must not double-emit.
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// A disputed hold can never flip to release_eligible.
func TestConfirmOrderHold_DisputedStaysDisputed(t *testing.T) {
	db := setupHoldDB(t)
	id := seedRegularOrder(t, db, models.PayoutHoldDisputed)
	order := loadOrder(t, db, id)

	status, err := ConfirmOrderHold(db, &order)
	require.NoError(t, err)
	require.NotEqual(t, models.PayoutHoldReleaseEligible, status)
	require.Equal(t, models.PayoutHoldDisputed, loadOrder(t, db, id).PayoutHoldStatus)
}

func TestGetCustomerConfirmWindowHours(t *testing.T) {
	db := setupHoldDB(t)
	require.Equal(t, 24, GetCustomerConfirmWindowHours(db), "default is 24h")

	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value, type) VALUES (?,?,?,?)`,
		uuid.NewString(), "payout.customer_confirm_window_hours", "48", "number").Error)
	require.Equal(t, 48, GetCustomerConfirmWindowHours(db), "override honoured")
}

// ConfirmMealPlanDayHold parallels ConfirmOrderHold; dispute keyed on OrderID.
func TestConfirmMealPlanDayHold(t *testing.T) {
	db := setupHoldDB(t)

	orderID := uuid.New()
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, payout_hold_status) VALUES (?,?,?,?,?)`,
		dayID.String(), uuid.NewString(), orderID.String(), "delivered", string(models.PayoutHoldAwaitingConfirmation)).Error)
	var day models.MealPlanDay
	require.NoError(t, db.First(&day, "id = ?", dayID).Error)

	status, err := ConfirmMealPlanDayHold(db, &day)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldReleaseEligible, status)

	// Now with an open issue on the day's order → disputed.
	orderID2 := uuid.New()
	dayID2 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, payout_hold_status) VALUES (?,?,?,?,?)`,
		dayID2.String(), uuid.NewString(), orderID2.String(), "delivered", string(models.PayoutHoldAwaitingConfirmation)).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), orderID2.String(), string(models.IssuePending)).Error)
	var day2 models.MealPlanDay
	require.NoError(t, db.First(&day2, "id = ?", dayID2).Error)
	status2, err := ConfirmMealPlanDayHold(db, &day2)
	require.NoError(t, err)
	require.Equal(t, models.PayoutHoldDisputed, status2)
}
