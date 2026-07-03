package services

// payout_release_test.go — the admin payout release actuator (#388). Drives the
// #387 hold state machine from release_eligible into real (flag-gated) vendor
// payouts: list eligible holds, release / withhold / reverse them. Pure DB logic
// on an in-memory sqlite harness (the models' gen_random_uuid() default can't run
// on sqlite, so tables are hand-DDL'd).
//
// Money movement stays behind OrderPayoutAutoReleaseEnabled (regular orders) and
// MealPlanEscrowEnabled (meal-plan days); OFF ⇒ every action is a DB-only state
// advance with NO money moved. GetRazorpay() returns nil in tests (no Secret
// Manager), so even flag-ON tests reach the seam without a live gateway.

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

// setupReleaseDB mirrors setupHoldDB (payout_hold_test.go) but adds the columns
// ListPendingPayouts selects — orders.order_number, meal_plans.meal_plan_number,
// orders.chef_id/total, meal_plan_days.price — plus an audit_logs table so the
// full actuator surface exercises here without touching production Postgres.
func setupReleaseDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT,
			chef_id TEXT, status TEXT, razorpay_order_id TEXT DEFAULT '', total REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, delivered_at DATETIME,
			payout_settled_at DATETIME, payout_settle_attempts INTEGER DEFAULT 0,
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
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT, value TEXT, type TEXT, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
			next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
		`CREATE TABLE audit_logs (id TEXT PRIMARY KEY, user_id TEXT, action TEXT, entity_type TEXT,
			entity_id TEXT, old_value TEXT, new_value TEXT, ip_address TEXT, correlation_id TEXT, created_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

// seedOrderHold inserts a delivered, gateway-charged order in the given hold state.
func seedOrderHold(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, delivered time.Time) uuid.UUID {
	t.Helper()
	id, chef := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, razorpay_order_id, total, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		id.String(), "ORD-"+id.String()[:8], uuid.NewString(), chef.String(),
		"delivered", "order_rzp_123", 250.0, string(hold), delivered).Error)
	return id
}

// seedDayHold inserts a delivered meal-plan day (with a held transfer id) in the
// given hold state, plus its parent plan (for chef_id / meal_plan_number).
func seedDayHold(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, delivered time.Time) uuid.UUID {
	t.Helper()
	dayID, planID, chef := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status)
		VALUES (?,?,?,?,?)`, planID.String(), "MP-"+planID.String()[:8], uuid.NewString(), chef.String(), "active").Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, status, payout_transfer_id, price, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), "delivered", "trf_abc123", 120.0, string(hold), delivered).Error)
	return dayID
}

func loadDayHold(t *testing.T, db *gorm.DB, id uuid.UUID) models.PayoutHoldStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.PayoutHoldStatus(s)
}

// TestListPendingPayouts_ReturnsEligibleOnly — the queue surfaces only
// release_eligible order+day rows (awaiting only with the opt-in filter), never
// disputed / withheld / reversed / released.
func TestListPendingPayouts_ReturnsEligibleOnly(t *testing.T) {
	db := setupReleaseDB(t)
	now := time.Now()

	elOrder := seedOrderHold(t, db, models.PayoutHoldReleaseEligible, now.Add(-30*time.Hour))
	elDay := seedDayHold(t, db, models.PayoutHoldReleaseEligible, now.Add(-10*time.Hour))
	awOrder := seedOrderHold(t, db, models.PayoutHoldAwaitingConfirmation, now.Add(-5*time.Hour))
	seedOrderHold(t, db, models.PayoutHoldDisputed, now.Add(-5*time.Hour))
	seedOrderHold(t, db, models.PayoutHoldWithheld, now.Add(-5*time.Hour))

	rows, err := ListPendingPayouts(db, PendingFilter{})
	require.NoError(t, err)
	require.Len(t, rows, 2, "only the two release_eligible rows")

	ids := map[uuid.UUID]PendingPayout{}
	for _, r := range rows {
		ids[r.ID] = r
		require.Equal(t, models.PayoutHoldReleaseEligible, r.HoldStatus)
	}
	require.Contains(t, ids, elOrder)
	require.Contains(t, ids, elDay)
	require.Equal(t, 250.0, ids[elOrder].Amount, "order amount = Total")
	require.Equal(t, "order", ids[elOrder].AggType)
	require.Equal(t, 120.0, ids[elDay].Amount, "day amount = Price")
	require.Equal(t, "meal-plan-day", ids[elDay].AggType)
	require.Greater(t, ids[elOrder].AgeHours, 24.0, "age computed from delivered_at")
	// sorted oldest-first (age desc)
	require.Equal(t, elOrder, rows[0].ID)

	withAwaiting, err := ListPendingPayouts(db, PendingFilter{IncludeAwaiting: true})
	require.NoError(t, err)
	require.Len(t, withAwaiting, 3, "awaiting now surfaced too")
	found := false
	for _, r := range withAwaiting {
		if r.ID == awOrder {
			found = true
		}
		require.NotEqual(t, models.PayoutHoldDisputed, r.HoldStatus)
		require.NotEqual(t, models.PayoutHoldWithheld, r.HoldStatus)
	}
	require.True(t, found, "awaiting order surfaced with the opt-in filter")
}

// TestReleaseHold_FlagOff_AdvancesNoMoney — flag OFF: release_eligible → released
// is a pure DB advance that stages the hold_released event; GetRazorpay() is nil,
// no panic.
func TestReleaseHold_FlagOff_AdvancesNoMoney(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: false}

	db := setupReleaseDB(t)
	id := seedOrderHold(t, db, models.PayoutHoldReleaseEligible, time.Now().Add(-30*time.Hour))

	require.NoError(t, ReleaseHold(db, "order", id))
	require.Equal(t, models.PayoutHoldReleased, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleased), "one hold_released event staged")
}

// TestReleaseHold_FlagOn_ReachesSeam_NoCrash — flag ON with an unconfigured
// gateway (GetRazorpay()==nil): ReleaseOrderPayouts no-ops cleanly, row released.
func TestReleaseHold_FlagOn_ReachesSeam_NoCrash(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}

	db := setupReleaseDB(t)
	origDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = origDB })

	id := seedOrderHold(t, db, models.PayoutHoldReleaseEligible, time.Now().Add(-30*time.Hour))
	require.NoError(t, ReleaseHold(db, "order", id))
	require.Equal(t, models.PayoutHoldReleased, loadOrder(t, db, id).PayoutHoldStatus)
}

// TestReleaseHold_ReRelease_NotEligible — a second release on an already-released
// row is a conditional-update no-op → ErrHoldNotEligible, no second event.
func TestReleaseHold_ReRelease_NotEligible(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{}

	db := setupReleaseDB(t)
	id := seedOrderHold(t, db, models.PayoutHoldReleaseEligible, time.Now().Add(-30*time.Hour))

	require.NoError(t, ReleaseHold(db, "order", id))
	err := ReleaseHold(db, "order", id)
	require.ErrorIs(t, err, ErrHoldNotEligible)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleased), "no double-emit")
}

// TestWithholdHold_EligibleToWithheld — release_eligible → withheld drops the row
// out of the pending queue and moves no money.
func TestWithholdHold_EligibleToWithheld(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{}

	db := setupReleaseDB(t)
	id := seedOrderHold(t, db, models.PayoutHoldReleaseEligible, time.Now().Add(-30*time.Hour))

	require.NoError(t, WithholdHold(db, "order", id, "suspected fraud"))
	require.Equal(t, models.PayoutHoldWithheld, loadOrder(t, db, id).PayoutHoldStatus)

	rows, err := ListPendingPayouts(db, PendingFilter{})
	require.NoError(t, err)
	require.Empty(t, rows, "withheld row excluded from the queue")
}

// TestReverseHold_ReleasedToReversed — a released order can be clawed back to
// reversed with a reason.
func TestReverseHold_ReleasedToReversed(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{}

	db := setupReleaseDB(t)
	id := seedOrderHold(t, db, models.PayoutHoldReleased, time.Now().Add(-30*time.Hour))

	require.NoError(t, ReverseHold(db, "order", id, "chargeback"))
	require.Equal(t, models.PayoutHoldReversed, loadOrder(t, db, id).PayoutHoldStatus)
}

// TestReverseHold_MealPlanDay — a release_eligible meal-plan day with a held
// transfer reverses to reversed; flag OFF ⇒ no money.
func TestReverseHold_MealPlanDay(t *testing.T) {
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{}

	db := setupReleaseDB(t)
	id := seedDayHold(t, db, models.PayoutHoldReleaseEligible, time.Now().Add(-10*time.Hour))

	require.NoError(t, ReverseHold(db, "meal-plan-day", id, "bad food"))
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, id))
}

// TestReverseHold_MissingReason_400 — the service guards a non-empty reason.
func TestReverseHold_MissingReason(t *testing.T) {
	db := setupReleaseDB(t)
	id := seedOrderHold(t, db, models.PayoutHoldReleased, time.Now())
	require.Error(t, ReverseHold(db, "order", id, ""))
}

// TestGetPayoutAutoApproveHours_DefaultZero — default 0 (manual-first, sweep
// disabled); an explicit setting overrides.
func TestGetPayoutAutoApproveHours_DefaultZero(t *testing.T) {
	db := setupReleaseDB(t)
	require.Equal(t, 0, GetPayoutAutoApproveHours(db), "default 0 = disabled")

	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value, type) VALUES (?,?,?,?)`,
		uuid.NewString(), "payout.auto_approve_after_hours", "48", "number").Error)
	require.Equal(t, 48, GetPayoutAutoApproveHours(db), "override honoured")
}
