package services

// meal_plan_stuck_day_test.go — #398 (stuck-day sweep). A confirmed+paid meal-plan
// day whose customer has no default address can never generate an order
// (generateDueDayOrders silently skips it), so it sits confirmed forever, strands its
// escrow, and blocks completeFinishedPlans (allDaysTerminal). sweepStuckDays voids
// such a day (auto-refund via RefundDay — gated) once it is well past its date, so the
// escrow is returned and the plan can finish.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupStuckDayDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT DEFAULT '',
		customer_id TEXT, chef_id TEXT, status TEXT, subtotal REAL DEFAULT 0, tax REAL DEFAULT 0,
		total REAL DEFAULT 0, currency TEXT DEFAULT 'INR', escrow_payment_id TEXT DEFAULT '',
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT,
		status TEXT, payout_transfer_id TEXT DEFAULT '', price REAL DEFAULT 0,
		payout_hold_status TEXT DEFAULT '', refund_txn_id TEXT, date DATETIME,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, event_type TEXT,
		user_id TEXT, aggregate_type TEXT, aggregate_id TEXT, payload TEXT, status TEXT, attempts INT,
		next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedStuckPlanDay(t *testing.T, db *gorm.DB, dayStatus models.MealPlanDayStatus, orderID *string, dayDate time.Time) (planID, dayID uuid.UUID) {
	t.Helper()
	planID, dayID = uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, subtotal, tax, total)
		VALUES (?,?,?,?,?,?,?,?)`, planID.String(), "MP-"+planID.String()[:8], uuid.NewString(), uuid.NewString(),
		string(models.MealPlanActive), 200.0, 20.0, 240.0).Error)
	var ord any
	if orderID != nil {
		ord = *orderID
	}
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, price, date)
		VALUES (?,?,?,?,?,?)`, dayID.String(), planID.String(), ord, string(dayStatus), 120.0, dayDate).Error)
	return planID, dayID
}

func dayStatusOf(t *testing.T, db *gorm.DB, id uuid.UUID) models.MealPlanDayStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.MealPlanDayStatus(s)
}

func escrowFlag(t *testing.T, on bool) {
	t.Helper()
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{MealPlanEscrowEnabled: on}
}

// Escrow OFF: no per-day money is held, so RefundDay is a no-op — the sweep still
// voids the stranded day to a terminal state so the plan can complete.
func TestSweepStuckDays_VoidsAddresslessDay(t *testing.T) {
	escrowFlag(t, false)
	db := setupStuckDayDB(t)
	// confirmed, no order, well past its date → stuck
	_, dayID := seedStuckPlanDay(t, db, models.MealPlanDayConfirmed, nil, time.Now().Add(-48*time.Hour))

	sweepStuckDays()

	require.Equal(t, models.MealPlanDayRefunded, dayStatusOf(t, db, dayID), "stuck day voided to terminal")
	// Idempotent: a refunded day is no longer stuck.
	sweepStuckDays()
	require.Equal(t, models.MealPlanDayRefunded, dayStatusOf(t, db, dayID))
}

func TestSweepStuckDays_SkipsNonTargets(t *testing.T) {
	escrowFlag(t, false)
	db := setupStuckDayDB(t)
	oid := uuid.NewString()
	// has an order → not stuck
	_, withOrder := seedStuckPlanDay(t, db, models.MealPlanDayConfirmed, &oid, time.Now().Add(-48*time.Hour))
	// confirmed but within grace (date only just passed) → wait
	_, recent := seedStuckPlanDay(t, db, models.MealPlanDayConfirmed, nil, time.Now().Add(-1*time.Hour))
	// not confirmed (still requested) → not a due day
	_, requested := seedStuckPlanDay(t, db, models.MealPlanDayRequested, nil, time.Now().Add(-48*time.Hour))

	sweepStuckDays()

	require.Equal(t, models.MealPlanDayConfirmed, dayStatusOf(t, db, withOrder))
	require.Equal(t, models.MealPlanDayConfirmed, dayStatusOf(t, db, recent))
	require.Equal(t, models.MealPlanDayRequested, dayStatusOf(t, db, requested))
}

// A fully address-stuck plan never flips Confirmed→Active (generateDueDayOrders only
// does that when it generates a day). Once the sweep terminates all its days,
// completeFinishedPlans must still mark the plan Completed — it can't stay Confirmed
// forever (#398). Regression for the verifier's point-4 gap.
func TestCompleteFinishedPlans_CompletesFullyStuckConfirmedPlan(t *testing.T) {
	db := setupStuckDayDB(t)
	// A Confirmed plan whose only day is already terminal (refunded by the sweep).
	planID, _ := seedStuckPlanDay(t, db, models.MealPlanDayRefunded, nil, time.Now().Add(-48*time.Hour))
	require.NoError(t, db.Exec(`UPDATE meal_plans SET status = ? WHERE id = ?`,
		string(models.MealPlanConfirmed), planID.String()).Error)

	completeFinishedPlans()

	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plans WHERE id = ?`, planID.String()).Scan(&s).Error)
	require.Equal(t, string(models.MealPlanCompleted), s, "fully-stuck confirmed plan completes once all days terminal")
}

// Escrow ON with no gateway configured: RefundDay errors, so the tx rolls back and
// the day stays confirmed — it is NEVER marked refunded without the money actually
// being returned. The next sweep retries.
func TestSweepStuckDays_RefundFailureDoesNotFalselyVoid(t *testing.T) {
	escrowFlag(t, true) // MealPlanEscrowActive → RefundDay runs; GetRazorpay() is nil → it errors
	db := setupStuckDayDB(t)
	_, dayID := seedStuckPlanDay(t, db, models.MealPlanDayConfirmed, nil, time.Now().Add(-48*time.Hour))

	sweepStuckDays()

	require.Equal(t, models.MealPlanDayConfirmed, dayStatusOf(t, db, dayID),
		"day must stay confirmed when the refund could not be made (no false 'refunded')")
}
