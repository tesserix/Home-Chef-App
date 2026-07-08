package services

// meal_plan_day_refund_atomic_test.go — #656. RefundDay guarded re-entry on the caller's
// in-memory struct (`day.RefundTxnID != nil`), which is loaded before the tx and can be
// STALE. The day's refund_txn_id is written by MULTIPLE independent paths on disjoint wallet
// keys — RefundDay (mealplan-refund:<dayID>) and the #618 order-issue path (issue:<id>) — so a
// stale nil let RefundDay credit the customer a SECOND time across keys (CreditWallet only
// dedups WITHIN a key). RefundDay must re-read refund_txn_id from the DB (under a row lock)
// and no-op when a concurrent/prior writer already refunded the day.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A day already stamped `refund_txn_id` in the DB (by another writer, on a DIFFERENT wallet
// key so CreditWallet's own idempotency can't mask it) must NOT be credited again when
// RefundDay is handed a stale in-memory struct whose RefundTxnID is still nil.
func TestRefundDay_AlreadyRefundedInDB_StaleStruct_NoDoubleCredit(t *testing.T) {
	escrowFlag(t, true)
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)

	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := uuid.New()
	priorTxn := uuid.New() // stamped by a prior/concurrent refunder (e.g. the issue:<id> path)
	// The day is ALREADY refunded in the DB (refund_txn_id set) but carries NO
	// mealplan-refund:<dayID> wallet txn — so nothing dedups RefundDay's own credit.
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, status, price, payout_hold_status, refund_txn_id)
		VALUES (?,?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDayRefunded),
		120.0, string(models.PayoutHoldWithheld), priorTxn.String()).Error)

	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 120, Tax: 12, Total: 150,
		Days: []models.MealPlanDay{{ID: dayID, Price: 120}}}
	// STALE struct: RefundTxnID zero-value (nil) even though the DB row is stamped.
	stale := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 120, OrderID: nil,
		Status: models.MealPlanDayRefunded}

	require.NoError(t, RefundDay(db, plan, stale, "cancel"))

	require.Equal(t, 0.0, walletBalance(t, db, cust),
		"an already-refunded day (stamped by another writer) must not be credited a second time")
	require.NotNil(t, stale.RefundTxnID, "the stale struct is reconciled to the DB stamp")
}

// A genuinely-unrefunded day (DB refund_txn_id NULL) still refunds normally — the atomic
// re-read must not block the happy path.
func TestRefundDay_UnrefundedInDB_RefundsNormally(t *testing.T) {
	escrowFlag(t, true)
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)

	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, payout_hold_status)
		VALUES (?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDayConfirmed), 120.0,
		string(models.PayoutHoldNone)).Error)
	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 120, Tax: 12, Total: 150,
		Days: []models.MealPlanDay{{ID: dayID, Price: 120}}}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 120, Status: models.MealPlanDayConfirmed}

	require.NoError(t, RefundDay(db, plan, day, "cancel"))
	require.Equal(t, 150.0, walletBalance(t, db, cust), "perDayGross credited")
	require.NotNil(t, day.RefundTxnID, "refund_txn_id stamped")
}
