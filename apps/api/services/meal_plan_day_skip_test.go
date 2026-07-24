package services

// meal_plan_day_skip_test.go — #422 policy change (admin-reviewed day skip). A customer's
// skip no longer auto-credits: SkipMealPlanDay freezes the day `skip_req` + disputes its
// hold, and the admin resolves it here. Approve → day `skipped` + a PARTIAL refund (food
// minus the platform commission) + the chef's hold fully reversed. Reject → day back to
// `confirmed` + hold restored. Reuses the crossguard harness (meal_plan_days + wallets +
// outbox); the flags-on approve test proves the perDaySkipRefund wallet-credit wiring.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// ── perDaySkipRefund ─────────────────────────────────────────────────────────

func TestPerDaySkipRefund(t *testing.T) {
	day := &models.MealPlanDay{Price: 200}
	// price 200, rate 6% → 200 − 12 = 188.
	require.Equal(t, 188.0, perDaySkipRefund(nil, day, 0.06))
	// rate unset (0) falls back to DefaultCommissionRate (0.06) → 188.
	require.Equal(t, 188.0, perDaySkipRefund(nil, day, 0))
	require.Equal(t, perDaySkipRefund(nil, day, DefaultCommissionRate), perDaySkipRefund(nil, day, 0))
	// out-of-range rate (>=1) also falls back to the default.
	require.Equal(t, 188.0, perDaySkipRefund(nil, day, 1.5))
}

// ── ResolveMealPlanDaySkip: approve ──────────────────────────────────────────

func TestResolveMealPlanDaySkip_Approve_CreditsPartialAndWithholds(t *testing.T) {
	escrowFlag(t, true)
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 200, 20)
	// A skip_req day frozen `disputed` (the state SkipMealPlanDay leaves), NO held transfer
	// → refundDayAmount credits without a gateway ReverseTransfer call.
	dayID := seedResolveDay(t, db, planID, models.MealPlanDaySkipRequested, models.PayoutHoldDisputed, "", 200)
	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 200, Tax: 20, Total: 250,
		Days: []models.MealPlanDay{{ID: dayID, Price: 200}}}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 200, Status: models.MealPlanDaySkipRequested}

	require.NoError(t, ResolveMealPlanDaySkip(db, plan, day, true, uuid.New()))

	require.Equal(t, models.MealPlanDaySkipped, loadDayStatus(t, db, dayID), "approved → terminal skipped")
	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID), "hold reversed out of the pay queue")
	require.True(t, dayRefundTxnSet(t, db, dayID), "refund txn stamped")
	require.Equal(t, 188.0, walletBalance(t, db, cust), "food (200) minus 6% commission (12) — GST/delivery forfeited")
	require.Equal(t, models.MealPlanCompleted, loadPlanStatus(t, db, planID), "last day terminal → plan completes")
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayRefunded))
}

func TestResolveMealPlanDaySkip_Approve_PlanStaysOpenWithPendingDay(t *testing.T) {
	flagsOff(t) // state transitions only; money-safe
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 400, 40)
	skipDay := seedResolveDay(t, db, planID, models.MealPlanDaySkipRequested, models.PayoutHoldDisputed, "", 200)
	seedResolveDay(t, db, planID, models.MealPlanDayConfirmed, models.PayoutHoldNone, "", 200) // still pending
	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 400, Tax: 40, Total: 500,
		Days: []models.MealPlanDay{{ID: skipDay, Price: 200}}}
	day := &models.MealPlanDay{ID: skipDay, MealPlanID: planID, Price: 200, Status: models.MealPlanDaySkipRequested}

	require.NoError(t, ResolveMealPlanDaySkip(db, plan, day, true, uuid.New()))

	require.Equal(t, models.MealPlanDaySkipped, loadDayStatus(t, db, skipDay))
	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, skipDay), "flags-off backstop withholds the disputed hold")
	require.Equal(t, models.MealPlanActive, loadPlanStatus(t, db, planID), "plan stays open — a day is still pending")
}

// ── ResolveMealPlanDaySkip: reject ───────────────────────────────────────────

func TestResolveMealPlanDaySkip_Reject_ReturnsToConfirmed(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 200, 20)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDaySkipRequested, models.PayoutHoldDisputed, "", 200)
	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 200, Tax: 20, Total: 250,
		Days: []models.MealPlanDay{{ID: dayID, Price: 200}}}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 200, Status: models.MealPlanDaySkipRequested}

	require.NoError(t, ResolveMealPlanDaySkip(db, plan, day, false, uuid.New()))

	require.Equal(t, models.MealPlanDayConfirmed, loadDayStatus(t, db, dayID), "rejected → day stands")
	require.Equal(t, models.PayoutHoldNone, loadDayHold(t, db, dayID), "frozen hold restored to none")
	require.False(t, dayRefundTxnSet(t, db, dayID), "no refund on reject")
	require.Equal(t, 0.0, walletBalance(t, db, cust))
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDaySkipDeclined), "customer told the skip was declined")
	require.Equal(t, 0, countOutbox(t, db, SubjectMealPlanDayRefunded))
}

// ── guards ───────────────────────────────────────────────────────────────────

func TestResolveMealPlanDaySkip_NotSkipRequest(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 200, 20)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayConfirmed, models.PayoutHoldNone, "", 200)
	plan := &models.MealPlan{ID: planID, CustomerID: cust}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Status: models.MealPlanDayConfirmed}

	err := ResolveMealPlanDaySkip(db, plan, day, true, uuid.New())
	require.ErrorIs(t, err, ErrNotSkipRequest)
	require.Equal(t, models.MealPlanDayConfirmed, loadDayStatus(t, db, dayID), "untouched")
}

func TestResolveMealPlanDaySkip_Idempotent(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 200, 20)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDaySkipRequested, models.PayoutHoldDisputed, "", 200)
	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 200, Tax: 20, Total: 250,
		Days: []models.MealPlanDay{{ID: dayID, Price: 200}}}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 200, Status: models.MealPlanDaySkipRequested}

	require.NoError(t, ResolveMealPlanDaySkip(db, plan, day, true, uuid.New()))
	// Second call: the day struct is still `skip_req` (stale) but the DB row is now `skipped`
	// → the guarded claim loses.
	err := ResolveMealPlanDaySkip(db, plan, day, true, uuid.New())
	require.ErrorIs(t, err, ErrIssueAlreadyHandled)
	require.Equal(t, models.MealPlanDaySkipped, loadDayStatus(t, db, dayID), "first resolution wins")
}
