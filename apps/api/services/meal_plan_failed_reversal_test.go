package services

// meal_plan_failed_reversal_test.go — #398 (P1). RefundDay's gateway claw-back
// (ReverseTransfer) can FAIL while the customer wallet refund still succeeds. The
// old markRefundedDayHold stamped the hold terminal (released→reversed+settled, or
// parked→withheld) REGARDLESS of whether the reverse landed, so a failed reversal
// stranded the still-live held transfer invisibly to every reconcile sweep — the
// chef kept money the customer got back (released = real double-pay; on-hold =
// stranded reserved funds).
//
// The fix threads a reverseOK bool into reverseRefundedDayHold: on a FAILED reverse
// the day is left as re-drivable drift (payout_hold_status=reversed, settled_at
// NULL) so reconcileMealPlanDays(reversed, settleReverse) retries the claw-back
// (attempt-capped + ALERT) instead of silently swallowing it. These tests exercise
// the DB-testable core directly — no live Razorpay needed (the #553/#554 technique).

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// dayIsSettled reports whether a meal-plan day has payout_settled_at stamped.
func dayIsSettled(t *testing.T, db *gorm.DB, id string) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM meal_plan_days WHERE id = ? AND payout_settled_at IS NOT NULL`, id).Scan(&n).Error)
	return n == 1
}

// reverseOK=true (claw-back landed): a released day → reversed AND settled, so the
// reconcile cron never re-reverses an already-clawed-back transfer. Unchanged
// behaviour, re-asserted under the new signature.
func TestReverseRefundedDayHold_ReverseOK_ReleasedReversedAndSettled(t *testing.T) {
	db := setupCrossguardDB(t)
	dayID := seedCrossDay(t, db, models.PayoutHoldReleased, nil)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return reverseRefundedDayHold(tx, dayID, true)
	}))
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, dayID))
	require.True(t, dayIsSettled(t, db, dayID.String()), "successful claw-back is settled — no reconcile re-reverse")
}

// reverseOK=true: a parked (eligible/awaiting/disputed) day → withheld, since the
// on-hold transfer was already freed by the successful reverse. Unchanged behaviour.
func TestReverseRefundedDayHold_ReverseOK_ParkedWithheld(t *testing.T) {
	db := setupCrossguardDB(t)
	for _, hold := range []models.PayoutHoldStatus{
		models.PayoutHoldReleaseEligible, models.PayoutHoldAwaitingConfirmation, models.PayoutHoldDisputed,
	} {
		dayID := seedCrossDay(t, db, hold, nil)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return reverseRefundedDayHold(tx, dayID, true)
		}))
		require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID), "%s → withheld", hold)
	}
}

// reverseOK=false (gateway claw-back FAILED): a released day must NOT be stamped
// settled — it stays reversed + settled_at NULL so the reconcile cron re-drives the
// ReverseTransfer. This is THE core defect: the old code stamped settled here and
// stranded the chef's transfer.
func TestReverseRefundedDayHold_ReverseFailed_ReleasedStaysUnsettledDrift(t *testing.T) {
	db := setupCrossguardDB(t)
	dayID := seedCrossDay(t, db, models.PayoutHoldReleased, nil)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return reverseRefundedDayHold(tx, dayID, false)
	}))
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, dayID))
	require.False(t, dayIsSettled(t, db, dayID.String()),
		"failed claw-back is left UNSETTLED so the reconcile cron re-drives the reverse")
}

// reverseOK=false: a parked day whose on-hold transfer reverse FAILED must not be
// marked withheld (invisible to reconcile). It becomes reversed + settled_at NULL so
// reconcileMealPlanDays(reversed) retries clawing back the still-live on-hold transfer.
func TestReverseRefundedDayHold_ReverseFailed_ParkedBecomesReversedDrift(t *testing.T) {
	db := setupCrossguardDB(t)
	for _, hold := range []models.PayoutHoldStatus{
		models.PayoutHoldReleaseEligible, models.PayoutHoldAwaitingConfirmation, models.PayoutHoldDisputed,
	} {
		dayID := seedCrossDay(t, db, hold, nil)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return reverseRefundedDayHold(tx, dayID, false)
		}))
		require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, dayID),
			"%s + failed reverse → reversed drift (not withheld)", hold)
		require.False(t, dayIsSettled(t, db, dayID.String()), "%s drift is unsettled", hold)
	}
}

// End-to-end: the reversed+unsettled drift the failed-reverse path now produces is
// visible to and settled by reconcileMealPlanDays(reversed, settleReverse). With
// escrow ON and GetRazorpay()==nil, reverseMoney is a no-op success → the row settles,
// proving the drift is genuinely re-drivable (not a new permanent strand).
func TestReconcileMealPlanDays_ReDrivesFailedReversalDrift(t *testing.T) {
	escrowOn(t) // MealPlanEscrowActive() → reconcile scan runs; GetRazorpay()==nil → seam no-op success
	db := setupCrossguardDB(t)

	// Shape the exact drift row reverseRefundedDayHold(_, false) leaves for a released day.
	dayID := seedCrossDay(t, db, models.PayoutHoldReleased, nil)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return reverseRefundedDayHold(tx, dayID, false)
	}))
	require.False(t, dayIsSettled(t, db, dayID.String()), "precondition: drift is unsettled")

	require.Equal(t, 1, reconcileMealPlanDays(models.PayoutHoldReversed, settleReverse),
		"the reversed-but-unsettled day is re-driven")
	require.True(t, dayIsSettled(t, db, dayID.String()), "reconcile settles the drift after the seam succeeds")
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, dayID))

	// Idempotent: a settled row drops out of the next scan.
	require.Equal(t, 0, reconcileMealPlanDays(models.PayoutHoldReversed, settleReverse))
}
