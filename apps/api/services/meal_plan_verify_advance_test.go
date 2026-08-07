package services

// meal_plan_verify_advance_test.go — #200 (tiffin E2E). Covers VerifyMealPlanAdvance, the
// meal-plan payment-capture entry point. It is the anti-under-payment gate: it binds the
// gateway capture to THIS plan's advance order + amount before stamping EscrowPaymentID —
// without which a ₹1 payment could mark a large plan "paid" out of the platform escrow.
// Drives Cashfree's order-scoped payments list through the shared httptest seam.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupAdvanceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plans (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id TEXT PRIMARY KEY, meal_plan_number TEXT,
		razorpay_order_id TEXT, total REAL, escrow_payment_id TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedAdvancePlan inserts a plan row (escrow_payment_id blank) and returns the matching struct the
// verify acts on (the handler passes a freshly-loaded plan; the row lets us read the stamp back).
func seedAdvancePlan(t *testing.T, db *gorm.DB, rzOrderID string, total float64) models.MealPlan {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO meal_plans (id, meal_plan_number, razorpay_order_id, total, escrow_payment_id)
		 VALUES (?,?,?,?,?)`,
		id.String(), "MP-"+id.String()[:8], rzOrderID, total, "").Error)
	return models.MealPlan{ID: id, RazorpayOrderID: rzOrderID, Total: total}
}

func escrowPaymentIDOf(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT escrow_payment_id FROM meal_plans WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}

// Escrow OFF → pure no-op: no gateway call, no stamp.
func TestVerifyMealPlanAdvance_EscrowOff_NoOp(t *testing.T) {
	escrowFlag(t, false)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	require.NoError(t, VerifyMealPlanAdvance(db, &plan))
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID), "escrow off → not stamped")
}

// A successful payment on the plan's own order for the full amount → stamps EscrowPaymentID.
func TestVerifyMealPlanAdvance_HappyPath_Stamps(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	withCashfreeOrderPayments(t, "order_adv1", 24000, CashfreePaymentSuccess) // 240.00, exact
	require.NoError(t, VerifyMealPlanAdvance(db, &plan))
	require.Equal(t, "4242", escrowPaymentIDOf(t, db, plan.ID), "persisted")
	require.Equal(t, "4242", plan.EscrowPaymentID, "struct updated")
}

// No advance order on the plan → reject before any gateway trust.
func TestVerifyMealPlanAdvance_NoAdvanceOrder_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "", 240)
	withCashfreeOrderPayments(t, "order_adv1", 24000, CashfreePaymentSuccess)
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan), "no advance order")
}

// The order carries an attempt that never succeeded (card abandoned at the OTP page) → reject.
func TestVerifyMealPlanAdvance_NotCaptured_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	withCashfreeOrderPayments(t, "order_adv1", 24000, CashfreePaymentPending)
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan), "not captured")
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID))
}

// Nothing was ever paid on the plan's order → reject. This is the surviving half of the
// cross-plan-reuse guard: the lookup is scoped to the plan's OWN order id, so a capture
// belonging to another plan can no longer be presented at all.
func TestVerifyMealPlanAdvance_NoPaymentOnPlansOrder_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	withCashfreeOrderPayments(t, "order_adv1", 0, "")
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan), "not captured")
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID))
}

// Captured amount below the plan total → reject (the anti-under-payment gate: a ₹1 short here
// must never mark the plan paid out of platform escrow).
func TestVerifyMealPlanAdvance_AmountTooLow_Errors(t *testing.T) {
	escrowFlag(t, true)
	db := setupAdvanceDB(t)
	plan := seedAdvancePlan(t, db, "order_adv1", 240)
	withCashfreeOrderPayments(t, "order_adv1", 23999, CashfreePaymentSuccess) // 1 paise short
	require.ErrorContains(t, VerifyMealPlanAdvance(db, &plan), "amount does not match")
	require.Empty(t, escrowPaymentIDOf(t, db, plan.ID))
}
