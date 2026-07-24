package services

// meal_plan_refund_nocapture_test.go — capture-at-approval money-leak guard.
//
// Payment now happens AFTER the customer approves the chef's response, so a plan can
// reach reject / cancel-before-start / expiry while still pending_chef or
// awaiting_customer with NO advance captured (escrow_payment_id blank — nothing ever
// reached the platform). RefundUndeliveredDays is the whole-plan refund those three
// exit paths call; escrow being globally ON must NOT make it credit the wallet for
// money the customer never paid. It gates on the capture stamp — symmetric to
// HoldChefPayouts, which already refuses to hold a payout for a blank escrow id.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A plan whose advance was NEVER captured (escrow_payment_id blank) must be a clean
// no-op on a full-plan refund — no wallet credit, no per-day refund stamp. This is the
// reject / cancel / expiry state of an awaiting_customer plan.
func TestRefundUndeliveredDays_NoCapture_NoCredit(t *testing.T) {
	escrowFlag(t, true) // escrow globally ON — the leak only exists when the money flow is live
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)

	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 320, 32)
	d1, d2 := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{d1, d2} {
		require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
			(id, meal_plan_id, status, price, payout_hold_status)
			VALUES (?,?,?,?,?)`, id.String(), planID.String(),
			string(models.MealPlanDayAccepted), 160.0, "").Error)
	}
	// Uncaptured plan: escrow_payment_id blank (never verified) — the awaiting_customer
	// state a reject/cancel/expiry can reach with zero money collected.
	plan := &models.MealPlan{ID: planID, CustomerID: cust, MealPlanNumber: "MP-nocap",
		Subtotal: 320, Total: 320, EscrowPaymentID: "",
		Days: []models.MealPlanDay{
			{ID: d1, MealPlanID: planID, Price: 160, Status: models.MealPlanDayAccepted},
			{ID: d2, MealPlanID: planID, Price: 160, Status: models.MealPlanDayAccepted},
		}}

	require.NoError(t, RefundUndeliveredDays(db, plan, "customer rejected the revised plan"),
		"an uncaptured plan refund must be a clean no-op, not an error")

	require.Equal(t, 0.0, walletBalance(t, db, cust),
		"no advance was ever captured — the wallet must not be credited (free-money leak)")
	require.Nil(t, plan.Days[0].RefundTxnID, "no refund stamped on an uncaptured day")
	require.Nil(t, plan.Days[1].RefundTxnID, "no refund stamped on an uncaptured day")
}

// The complement: a CAPTURED plan (escrow_payment_id set — the customer paid, then
// cancels before start) must still be fully refunded. Proves the no-capture guard does
// not over-block a genuine paid-plan refund.
func TestRefundUndeliveredDays_Captured_Refunds(t *testing.T) {
	escrowFlag(t, true)
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)

	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 160, 16)
	d1 := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, status, price, payout_hold_status)
		VALUES (?,?,?,?,?)`, d1.String(), planID.String(),
		string(models.MealPlanDayConfirmed), 160.0, "").Error)
	// Captured plan: escrow_payment_id set (payment verified) — a paid plan cancelled
	// before start must make the customer whole.
	plan := &models.MealPlan{ID: planID, CustomerID: cust, MealPlanNumber: "MP-cap",
		Subtotal: 160, Total: 160, EscrowPaymentID: "pay_captured123",
		Days: []models.MealPlanDay{
			{ID: d1, MealPlanID: planID, Price: 160, Status: models.MealPlanDayConfirmed},
		}}

	require.NoError(t, RefundUndeliveredDays(db, plan, "cancelled by customer before start"))

	require.Greater(t, walletBalance(t, db, cust), 0.0,
		"a captured plan must still refund — the no-capture guard must not over-block")
	require.NotNil(t, plan.Days[0].RefundTxnID, "a captured-day refund is stamped")
}
