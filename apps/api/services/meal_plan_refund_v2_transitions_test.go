package services

// meal_plan_refund_v2_transitions_test.go — the v2 state machine: chef Full → pending_admin (no
// money yet), None → skipped (no refund), Decline → confirmed (hold restored); admin pay →
// refunded; and the >12h auto path → full wallet refund directly.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedV2FlowRow inserts a plan + a pending_chef, skip_req day (frozen hold) into the DB.
func seedV2FlowRow(t *testing.T, db *gorm.DB, custID uuid.UUID) (planID, dayID uuid.UUID) {
	t.Helper()
	planID, dayID = uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, customer_id, meal_plan_number, escrow_payment_id, subtotal, tax, total)
		VALUES (?,?,?,?,?,?,?)`, planID.String(), custID.String(), "MP-FLOW", "pay_test", 320.0, 32.0, 372.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, commission_rate, payout_hold_status, refund_stage)
		VALUES (?,?,?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDaySkipRequested), 160.0, 0.15,
		string(models.PayoutHoldDisputed), string(models.MPRefundPendingChef)).Error)
	return
}

func v2HoldStatus(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}

// Chef "Full" records the choice and moves to pending_admin — NO money moves yet.
func TestChefDecide_Full_ToPendingAdmin(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionFull, false))

	require.Equal(t, 0.0, v2WalletBalance(t, db, u), "chef decision moves no money")
	status, stage, choice, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDaySkipRequested), status, "day stays in the skip_req umbrella")
	require.Equal(t, string(models.MPRefundPendingAdmin), stage)
	require.Equal(t, "full", choice)
}

// Chef "None" resolves now: no customer refund, day skipped.
func TestChefDecide_None_Skipped(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionNone, false))
	require.Equal(t, 0.0, v2WalletBalance(t, db, u))
	status, stage, choice, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDaySkipped), status)
	require.Equal(t, string(models.MPRefundResolved), stage)
	require.Equal(t, "none", choice)
}

// Chef "Decline" returns the day to confirmed and restores the frozen hold (disputed → none).
func TestChefDecide_Decline_Confirmed(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, "", true))
	status, stage, _, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDayConfirmed), status)
	require.Equal(t, "", stage)
	require.Equal(t, string(models.PayoutHoldNone), v2HoldStatus(t, db, dayID), "frozen hold restored")
}

// Admin pays a pending_admin day → the chef's chosen amount lands in the wallet, day refunded.
func TestAdminPay_ToWallet(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)
	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionFull, false))

	require.NoError(t, AdminPayMealPlanRefund(db, dayID, models.RefundDestinationWallet))
	require.Equal(t, 136.0, v2WalletBalance(t, db, u))
	status, stage, _, dest := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDayRefunded), status)
	require.Equal(t, string(models.MPRefundResolved), stage)
	require.Equal(t, "wallet", dest)
}

// A stage mismatch (paying a day still awaiting the chef) is rejected.
func TestAdminPay_StageMismatch(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u) // still pending_chef, not pending_admin
	require.ErrorIs(t, AdminPayMealPlanRefund(db, dayID, models.RefundDestinationWallet), ErrRefundStageMismatch)
}

// The >12h auto path refunds the full base to the wallet directly.
func TestAutoApprove_FullWallet(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AutoApproveMealPlanDayRefund(tx, plan, day)
	}))
	require.Equal(t, 136.0, v2WalletBalance(t, db, u))
	status, _, choice, dest := v2DayRow(t, db, day.ID)
	require.Equal(t, string(models.MealPlanDayRefunded), status)
	require.Equal(t, "full", choice)
	require.Equal(t, "wallet", dest)
}
