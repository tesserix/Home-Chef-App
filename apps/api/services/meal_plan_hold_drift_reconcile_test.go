package services

// meal_plan_hold_drift_reconcile_test.go — #398/#1086. A meal-plan day whose hold
// transitioned but whose settle stamp never landed (crash between the two) is drift:
// invisible to the admin queue, unsettled forever. reconcileMealPlanDays re-drives it.
//
// The gateway claw-back this file was originally written for (a FAILED ReverseTransfer
// deliberately left as re-drivable drift) went with the Route rail in #1086 — the day
// never holds a transfer now, so reverseRefundedDayHold is state-only and its two
// terminal transitions are asserted in payout_disputed_fanout_test.go Part C. What
// survives is the crash window, which no removal can close.

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

func TestReconcileMealPlanDays_ReDrivesUnsettledReversedDay(t *testing.T) {
	escrowOn(t) // MealPlanEscrowActive() → the scan runs
	db := setupCrossguardDB(t)

	dayID := seedCrossDay(t, db, models.PayoutHoldReversed, nil)
	require.False(t, dayIsSettled(t, db, dayID.String()), "precondition: drift is unsettled")

	require.Equal(t, 1, reconcileMealPlanDays(models.PayoutHoldReversed, settleReverse),
		"the reversed-but-unsettled day is re-driven")
	require.True(t, dayIsSettled(t, db, dayID.String()), "reconcile settles the drift")
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, dayID))

	// Idempotent: a settled row drops out of the next scan.
	require.Equal(t, 0, reconcileMealPlanDays(models.PayoutHoldReversed, settleReverse))
}

// The sweep used to require a transfer id, because the seam had a transfer to move.
// It no longer does, and no day is written one — scoping on it would make the sweep
// dead by construction and leave every crash-window day unsettled forever.
func TestReconcileMealPlanDays_ReDrivesDayWithNoTransferID(t *testing.T) {
	escrowOn(t)
	db := setupCrossguardDB(t)

	dayID := seedCrossDay(t, db, models.PayoutHoldReversed, nil)
	require.NoError(t, db.Exec(`UPDATE meal_plan_days SET payout_transfer_id = '' WHERE id = ?`, dayID.String()).Error)

	require.Equal(t, 1, reconcileMealPlanDays(models.PayoutHoldReversed, settleReverse))
	require.True(t, dayIsSettled(t, db, dayID.String()))
}
