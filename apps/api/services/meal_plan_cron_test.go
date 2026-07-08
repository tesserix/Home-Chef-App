package services

// meal_plan_cron_test.go — #200 (tiffin E2E). Covers the negotiation-cutoff sweep
// (services/meal_plan_cron.go), which was previously untested. A lapsed chef-respond-by
// (pending_chef) or customer-approve-by (awaiting_customer) must expire the plan and
// (once escrow is enabled) fully refund it — issue #200 acceptance: "customer reject →
// full refund; chef no-response expiry → full refund".
//
// These tests exercise the CRON's own logic with escrow OFF: the correct plans are
// selected per cutoff, the status transition + cancel stamp are applied, and the right
// parties are notified (customer only for a chef no-response; customer AND chef for a
// lapsed customer-approval, since the chef cherry-picked and was waiting too). The money
// mechanics of the refund itself (RefundUndeliveredDays → RefundDay, wallet credit,
// idempotency) are covered separately by meal_plan_day_refund_atomic_test.go /
// meal_plan_day_refund_stamp_test.go; with escrow OFF RefundUndeliveredDays is a no-op,
// so these tests isolate the sweep's selection/transition/notification behaviour.

import (
	"context"
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

func setupMealPlanCronDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT,
		customer_id TEXT, chef_id TEXT, status TEXT, subtotal REAL, tax REAL, total REAL,
		chef_respond_by DATETIME, customer_approve_by DATETIME, cancelled_at DATETIME,
		cancel_reason TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT,
		status TEXT, payout_transfer_id TEXT, price REAL, payout_hold_status TEXT, refund_txn_id TEXT,
		date DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT,
		aggregate_type TEXT, aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
		next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedCronPlan inserts a plan (+ one requested day) with the given status and cutoffs
// (nil cutoff → NULL). Returns the plan id.
func seedCronPlan(t *testing.T, db *gorm.DB, status models.MealPlanStatus, customerID, chefID uuid.UUID, respondBy, approveBy *time.Time) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, subtotal, tax, total,
			chef_respond_by, customer_approve_by) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		planID.String(), "MP-"+planID.String()[:8], customerID.String(), chefID.String(), string(status),
		200.0, 20.0, 240.0, timeArg(respondBy), timeArg(approveBy)).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, date) VALUES (?,?,?,?,?)`,
		uuid.NewString(), planID.String(), string(models.MealPlanDayRequested), 120.0, time.Now().Add(72*time.Hour)).Error)
	return planID
}

func seedCronChefProfile(t *testing.T, db *gorm.DB, chefID, chefUserID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`, chefID.String(), chefUserID.String()).Error)
}

// timeArg maps a *time.Time to a bind value: nil → NULL, else the time value.
func timeArg(tm *time.Time) any {
	if tm == nil {
		return nil
	}
	return *tm
}

func cronPlanRow(t *testing.T, db *gorm.DB, id uuid.UUID) (status, reason string) {
	t.Helper()
	var row struct {
		Status       string
		CancelReason string
	}
	require.NoError(t, db.Raw(`SELECT status, cancel_reason FROM meal_plans WHERE id = ?`, id.String()).Scan(&row).Error)
	return row.Status, row.CancelReason
}

// expiredEventsFor counts the meal_plan.expired outbox events addressed to a user
// (EnqueueEvent stores the recipient user id in aggregate_id, aggregate_type "event").
func expiredEventsFor(t *testing.T, db *gorm.DB, userID uuid.UUID) int {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM outbox_events WHERE aggregate_type = 'event' AND aggregate_id = ?`,
		userID.String()).Scan(&n).Error)
	return int(n)
}

// A pending_chef plan past its chef_respond_by expires and notifies the customer.
func TestRunMealPlanSweep_ChefNoResponse_ExpiresAndNotifiesCustomer(t *testing.T) {
	escrowFlag(t, false) // RefundUndeliveredDays is a no-op with escrow off — isolate the sweep
	db := setupMealPlanCronDB(t)
	customerID, chefID := uuid.New(), uuid.New()
	past := time.Now().Add(-time.Hour)
	planID := seedCronPlan(t, db, models.MealPlanPendingChef, customerID, chefID, &past, nil)

	runMealPlanSweep(context.Background())

	status, reason := cronPlanRow(t, db, planID)
	require.Equal(t, string(models.MealPlanExpired), status, "lapsed chef-respond-by expires the plan")
	require.Contains(t, reason, "chef did not respond")
	require.Equal(t, 1, expiredEventsFor(t, db, customerID), "customer is notified")
}

// An awaiting_customer plan past its customer_approve_by expires and notifies BOTH the
// customer and the chef (the chef cherry-picked and was waiting on the customer).
func TestRunMealPlanSweep_CustomerNoApprove_ExpiresAndNotifiesBoth(t *testing.T) {
	escrowFlag(t, false)
	db := setupMealPlanCronDB(t)
	customerID, chefID, chefUserID := uuid.New(), uuid.New(), uuid.New()
	seedCronChefProfile(t, db, chefID, chefUserID)
	past := time.Now().Add(-time.Hour)
	planID := seedCronPlan(t, db, models.MealPlanAwaitingCustomer, customerID, chefID, nil, &past)

	runMealPlanSweep(context.Background())

	status, reason := cronPlanRow(t, db, planID)
	require.Equal(t, string(models.MealPlanExpired), status)
	require.Contains(t, reason, "customer did not approve")
	require.Equal(t, 1, expiredEventsFor(t, db, customerID), "customer is notified")
	require.Equal(t, 1, expiredEventsFor(t, db, chefUserID), "chef is also notified on an awaiting_customer expiry")
}

// A cutoff still in the future must not expire the plan.
func TestRunMealPlanSweep_BeforeCutoff_NotExpired(t *testing.T) {
	escrowFlag(t, false)
	db := setupMealPlanCronDB(t)
	future := time.Now().Add(time.Hour)
	planID := seedCronPlan(t, db, models.MealPlanPendingChef, uuid.New(), uuid.New(), &future, nil)

	runMealPlanSweep(context.Background())

	status, _ := cronPlanRow(t, db, planID)
	require.Equal(t, string(models.MealPlanPendingChef), status, "not yet past chef_respond_by → untouched")
}

// A NULL cutoff must never expire (the sweep guards on chef_respond_by IS NOT NULL).
func TestRunMealPlanSweep_NullCutoff_NotExpired(t *testing.T) {
	escrowFlag(t, false)
	db := setupMealPlanCronDB(t)
	planID := seedCronPlan(t, db, models.MealPlanPendingChef, uuid.New(), uuid.New(), nil, nil)

	runMealPlanSweep(context.Background())

	status, _ := cronPlanRow(t, db, planID)
	require.Equal(t, string(models.MealPlanPendingChef), status, "NULL chef_respond_by → never expires")
}

// A confirmed plan carrying a stale respond-by must not be swept (status guard).
func TestRunMealPlanSweep_ConfirmedPlan_NotExpired(t *testing.T) {
	escrowFlag(t, false)
	db := setupMealPlanCronDB(t)
	past := time.Now().Add(-time.Hour)
	planID := seedCronPlan(t, db, models.MealPlanConfirmed, uuid.New(), uuid.New(), &past, nil)

	runMealPlanSweep(context.Background())

	status, _ := cronPlanRow(t, db, planID)
	require.Equal(t, string(models.MealPlanConfirmed), status, "only pending_chef / awaiting_customer are swept")
}
