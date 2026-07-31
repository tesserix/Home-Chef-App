package services

// meal_plan_refund_v2_flow_test.go — the v2 refund executor: Full/Half refund the fee/GST-excluded
// base to the wallet and drive the day → refunded; None refunds nothing and drives → skipped; and
// it is idempotent (a second call never double-credits).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

func setupV2RefundDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE meal_plan_days (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id TEXT PRIMARY KEY, meal_plan_id TEXT, status TEXT, price REAL,
			commission_rate REAL, payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			refund_txn_id TEXT, refund_stage TEXT DEFAULT '', chef_refund_choice TEXT DEFAULT '',
			refund_percent INTEGER, refund_floor_percent INTEGER,
			refund_destination TEXT DEFAULT '', refund_decision_by DATETIME, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plans (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, meal_plan_number TEXT,
			escrow_payment_id TEXT DEFAULT '', subtotal REAL, tax REAL, total REAL, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0,
			currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT, source TEXT,
			amount REAL, balance_after REAL, currency TEXT, order_id TEXT, reason TEXT, created_by TEXT,
			idempotency_key TEXT UNIQUE, created_at DATETIME)`,
		// v3 (#834): the executor issues a GST credit note inside the refund tx, so the
		// harness must carry the table or every refund rolls back.
		`CREATE TABLE credit_notes (id TEXT PRIMARY KEY, credit_note_number TEXT UNIQUE, source_key TEXT UNIQUE,
			customer_id TEXT, chef_id TEXT, meal_plan_id TEXT, meal_plan_day_id TEXT, order_id TEXT,
			reference TEXT, currency TEXT, taxable_value REAL, tax_amount REAL, total_amount REAL,
			refund_percent INTEGER, reason TEXT, issued_at DATETIME, created_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func v2EscrowOn(t *testing.T) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{MealPlanEscrowEnabled: true} // ledger shadow stays off
	t.Cleanup(func() { config.AppConfig = prev })
}

// seedV2Day inserts a skip_req day (no chef transfer) and returns the plan + day the executor runs on.
func seedV2Day(t *testing.T, db *gorm.DB, custID uuid.UUID) (*models.MealPlan, *models.MealPlanDay) {
	t.Helper()
	planID, dayID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, commission_rate, refund_stage)
		VALUES (?,?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDaySkipRequested), 160.0, 0.15, string(models.MPRefundPendingChef)).Error)
	plan := &models.MealPlan{ID: planID, CustomerID: custID, MealPlanNumber: "MP-TEST",
		EscrowPaymentID: "pay_test", Subtotal: 320, Tax: 32, Total: 372}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 160, CommissionRate: 0.15,
		Status: models.MealPlanDaySkipRequested}
	return plan, day
}

func v2WalletBalance(t *testing.T, db *gorm.DB, u uuid.UUID) float64 {
	t.Helper()
	var w models.Wallet
	if err := db.First(&w, "user_id = ?", u).Error; err != nil {
		return 0
	}
	return w.Balance
}

func v2DayRow(t *testing.T, db *gorm.DB, id uuid.UUID) (status, stage, choice, dest string) {
	t.Helper()
	row := db.Raw(`SELECT status, refund_stage, chef_refund_choice, refund_destination FROM meal_plan_days WHERE id = ?`, id.String()).Row()
	require.NoError(t, row.Scan(&status, &stage, &choice, &dest))
	return
}

// v3: a 100% refund returns food-minus-commission plus that day's GST and delivery to the
// wallet; the day is refunded and the percentage + wallet destination recorded.
func TestExecuteV2Refund_FullToWallet(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 100, models.RefundDestinationWallet))

	require.Equal(t, 162.0, v2WalletBalance(t, db, u),
		"v3 full = (160 food − 24 commission) + 16 GST + 10 delivery = 162")
	status, stage, choice, dest := v2DayRow(t, db, day.ID)
	require.Equal(t, string(models.MealPlanDayRefunded), status)
	require.Equal(t, string(models.MPRefundResolved), stage)
	require.Equal(t, "full", choice)
	require.Equal(t, "wallet", dest)
}

// 50% refund → half the gross.
func TestExecuteV2Refund_HalfToWallet(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 50, models.RefundDestinationWallet))
	require.Equal(t, 81.0, v2WalletBalance(t, db, u), "half = 162/2")
	status, _, choice, _ := v2DayRow(t, db, day.ID)
	require.Equal(t, string(models.MealPlanDayRefunded), status)
	require.Equal(t, "half", choice)
}

// 0% → no customer refund; day skipped (customer forfeits, chef keeps payout).
func TestExecuteV2Refund_NoneNoRefund(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 0, models.RefundDestinationWallet))
	require.Equal(t, 0.0, v2WalletBalance(t, db, u), "0% refunds nothing")
	status, stage, choice, _ := v2DayRow(t, db, day.ID)
	require.Equal(t, string(models.MealPlanDaySkipped), status)
	require.Equal(t, string(models.MPRefundResolved), stage)
	require.Equal(t, "none", choice)
}

// Idempotent: a second Full call never double-credits (refund_txn_id already stamped).
func TestExecuteV2Refund_Idempotent(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 100, models.RefundDestinationWallet))
	// Reload the day (refund_txn_id + stage now set) and re-run.
	day2 := &models.MealPlanDay{ID: day.ID, MealPlanID: plan.ID, Price: 160, CommissionRate: 0.15}
	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day2, 100, models.RefundDestinationWallet))
	require.Equal(t, 162.0, v2WalletBalance(t, db, u), "credited once, not twice")
}
