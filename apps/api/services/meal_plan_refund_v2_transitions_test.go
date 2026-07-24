package services

// meal_plan_refund_v2_transitions_test.go — the v2 state machine with the RBI customer-choice:
// chef Full/Half → pending_customer (no money); the CUSTOMER then picks wallet (instant credit) or
// original (→ pending_admin for the admin to execute). None → skipped; Decline → confirmed.

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

// Chef "Full" records the amount and moves to pending_customer — NO money, and the customer now
// picks the medium (RBI).
func TestChefDecide_Full_ToPendingCustomer(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionFull, false))

	require.Equal(t, 0.0, v2WalletBalance(t, db, u), "chef decision moves no money")
	status, stage, choice, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDaySkipRequested), status)
	require.Equal(t, string(models.MPRefundPendingCustomer), stage)
	require.Equal(t, "full", choice)
}

// The customer chooses WALLET → instant credit, day refunded.
func TestCustomerChoose_Wallet_Refunds(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)
	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionFull, false))

	require.NoError(t, CustomerChooseMealPlanRefundMedium(db, dayID, u, models.RefundDestinationWallet))
	require.Equal(t, 136.0, v2WalletBalance(t, db, u))
	status, stage, _, dest := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDayRefunded), status)
	require.Equal(t, string(models.MPRefundResolved), stage)
	require.Equal(t, "wallet", dest)
}

// The customer chooses ORIGINAL → pending_admin, no money yet (the admin executes the gateway refund).
func TestCustomerChoose_Source_ToPendingAdmin(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)
	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionFull, false))

	require.NoError(t, CustomerChooseMealPlanRefundMedium(db, dayID, u, models.RefundDestinationSource))
	require.Equal(t, 0.0, v2WalletBalance(t, db, u), "original method moves no wallet money")
	_, stage, _, dest := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MPRefundPendingAdmin), stage)
	require.Equal(t, "source", dest)
}

// A different customer cannot choose someone else's refund medium.
func TestCustomerChoose_WrongOwner(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)
	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionFull, false))
	require.ErrorIs(t, CustomerChooseMealPlanRefundMedium(db, dayID, uuid.New(), models.RefundDestinationWallet), gorm.ErrRecordNotFound)
}

// An invalid medium is rejected.
func TestCustomerChoose_InvalidMedium(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u)
	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, models.RefundProportionFull, false))
	require.ErrorIs(t, CustomerChooseMealPlanRefundMedium(db, dayID, u, "bank"), ErrInvalidRefundMedium)
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

// The >12h auto path agrees FULL and hands the medium choice to the customer (pending_customer).
func TestAgreeFull_ToPendingCustomer(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AgreeMealPlanDayRefundFull(tx, plan, day)
	}))
	require.Equal(t, 0.0, v2WalletBalance(t, db, u), "no money until the customer picks a medium")
	_, stage, choice, _ := v2DayRow(t, db, day.ID)
	require.Equal(t, string(models.MPRefundPendingCustomer), stage)
	require.Equal(t, "full", choice)
}
