package services

// meal_plan_refund_deadline_cron_test.go — the chef's refund clock. A day the kitchen never
// priced resolves at 100% of the base (which excludes the platform commission); a day still
// inside its window, or one the chef answered, is left alone.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// deadlineDB is setupV2RefundDB plus the global wiring the sweep needs: it runs on a schedule,
// so it reads database.DB rather than taking a handle.
func deadlineDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupV2RefundDB(t)
	// The sweep notifies both sides, so the outbox has to exist or the whole tx rolls back.
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT,
		aggregate_type TEXT, aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
		next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedLapsedRefundDay inserts a pending_chef day whose decision window closed `ago` in the past.
// A positive `ago` is lapsed; a negative one is still running.
func seedLapsedRefundDay(t *testing.T, db *gorm.DB, ago time.Duration, floor int) (planID, dayID uuid.UUID) {
	t.Helper()
	planID, dayID = uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, customer_id, chef_id, meal_plan_number, escrow_payment_id, subtotal, tax, total)
		VALUES (?,?,?,?,?,?,?,?)`, planID.String(), uuid.NewString(), uuid.NewString(), "MP-DEADLINE", "pay_test", 320.0, 32.0, 372.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, commission_rate, payout_hold_status, refund_stage, refund_floor_percent, refund_decision_by)
		VALUES (?,?,?,?,?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDaySkipRequested), 160.0, 0.15,
		string(models.PayoutHoldDisputed), string(models.MPRefundPendingChef), floor, time.Now().Add(-ago)).Error)
	return
}

func dayRefundState(t *testing.T, db *gorm.DB, id uuid.UUID) (stage string, percent *int) {
	t.Helper()
	var row struct {
		RefundStage   string
		RefundPercent *int
	}
	require.NoError(t, db.Raw(`SELECT refund_stage, refund_percent FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&row).Error)
	return row.RefundStage, row.RefundPercent
}

// A lapsed window resolves at 100% — NOT at the pinned floor. The floor is what the chef earns
// by answering; paying it for silence would reward not replying.
func TestRefundDeadline_LapsedResolvesAtFull(t *testing.T) {
	v2EscrowOn(t)
	db := deadlineDB(t)
	_, dayID := seedLapsedRefundDay(t, db, time.Hour, 75)

	runMealPlanRefundDeadlineSweep(t.Context())

	stage, percent := dayRefundState(t, db, dayID)
	require.Equal(t, string(models.MPRefundPendingCustomer), stage, "a lapsed refund moves to the customer's medium choice")
	require.NotNil(t, percent)
	require.Equal(t, 100, *percent, "silence resolves at full, not at the chef's floor")
}

// Still inside the window: the chef's to price, untouched.
func TestRefundDeadline_LeavesLiveWindowAlone(t *testing.T) {
	v2EscrowOn(t)
	db := deadlineDB(t)
	_, dayID := seedLapsedRefundDay(t, db, -30*time.Minute, 75)

	runMealPlanRefundDeadlineSweep(t.Context())

	stage, percent := dayRefundState(t, db, dayID)
	require.Equal(t, string(models.MPRefundPendingChef), stage)
	require.Nil(t, percent, "no amount is agreed while the chef still has time")
}

// The chef answering wins over the deadline, even at a lower percentage than the sweep
// would have granted — their decision is the authority, the sweep only covers silence.
func TestRefundDeadline_ChefDecisionWins(t *testing.T) {
	v2EscrowOn(t)
	db := deadlineDB(t)
	_, dayID := seedLapsedRefundDay(t, db, time.Hour, 75)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, 80, false))
	runMealPlanRefundDeadlineSweep(t.Context())

	_, percent := dayRefundState(t, db, dayID)
	require.NotNil(t, percent)
	require.Equal(t, 80, *percent, "the sweep must not overwrite a decision the chef already made")
}

// A day with no deadline recorded (raised before this shipped) is never swept — it has no
// window to have missed.
func TestRefundDeadline_IgnoresDaysWithoutADeadline(t *testing.T) {
	v2EscrowOn(t)
	db := deadlineDB(t)
	planID, dayID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, customer_id, chef_id, meal_plan_number, escrow_payment_id, subtotal, tax, total)
		VALUES (?,?,?,?,?,?,?,?)`, planID.String(), uuid.NewString(), uuid.NewString(), "MP-LEGACY", "pay_test", 320.0, 32.0, 372.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, commission_rate, payout_hold_status, refund_stage, refund_floor_percent)
		VALUES (?,?,?,?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDaySkipRequested), 160.0, 0.15,
		string(models.PayoutHoldDisputed), string(models.MPRefundPendingChef), 75).Error)

	runMealPlanRefundDeadlineSweep(t.Context())

	stage, _ := dayRefundState(t, db, dayID)
	require.Equal(t, string(models.MPRefundPendingChef), stage)
}

// Policy drives the window, and a negative value disables the clock entirely.
func TestChefRefundDecisionDeadline_PolicyDriven(t *testing.T) {
	now := time.Now()
	by, ok := ChefRefundDecisionDeadline(now)
	require.True(t, ok)
	require.Equal(t, now.Add(60*time.Minute), by, "defaults to the one-hour window")
}
