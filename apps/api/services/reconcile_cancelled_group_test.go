package services

// reconcile_cancelled_group_test.go — #534. Two delivery-race / reconcile gaps for
// CANCELLED payout holds left after the W-A work (#532):
//   1. reconcileCancelledGroups claws back a stranded held transfer on a cancelled
//      group whose reverse never ran (the released/reversed sweeps miss it).
//   2. parkGroupOrderOnDelivery / MarkMealPlanDayDelivered must not resurrect a
//      cancelled/terminal aggregate back to delivered and park its hold.
// Flags OFF ⇒ the money seam is a no-op; assertions are on the DB state advance.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func setGroupTransfer(t *testing.T, db *gorm.DB, id uuid.UUID, transferID string) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE group_orders SET payout_transfer_id = ? WHERE id = ?`, transferID, id.String()).Error)
}

func groupStatusOf(t *testing.T, db *gorm.DB, id uuid.UUID) models.GroupOrderStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM group_orders WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.GroupOrderStatus(s)
}

// Fix 1 — a cancelled group with a held transfer stuck at a non-terminal hold is
// clawed back by the reconcile scan (invisible to the released/reversed sweeps).
func TestReconcileCancelledGroups_ClawsBackStrandedHold(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	id := seedCrossGroup(t, db, models.PayoutHoldAwaitingConfirmation, nil)
	setGroupStatus(t, db, id, models.GroupOrderCancelled)
	setGroupTransfer(t, db, id, "trf_stranded")

	require.Equal(t, 1, reconcileCancelledGroups(), "the stranded cancelled hold is driven")
	require.Equal(t, models.PayoutHoldReversed, loadGroupHold(t, db, id))
	require.True(t, loadGroupSettledAt(t, db, id), "settled stamped after claw-back")

	// Idempotent: a second pass no longer matches (hold=reversed + settled).
	require.Equal(t, 0, reconcileCancelledGroups())
}

// Fix 1 — the scan skips rows it must not touch: already terminal-for-cancel
// (withheld/reversed), non-cancelled groups, and cancelled groups with no held
// transfer (no money stranded).
func TestReconcileCancelledGroups_SkipsNonTargets(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	// already reversed → terminal, skip
	rev := seedCrossGroup(t, db, models.PayoutHoldReversed, nil)
	setGroupStatus(t, db, rev, models.GroupOrderCancelled)
	setGroupTransfer(t, db, rev, "t")
	// non-cancelled (seed leaves status=delivered) with a held transfer → skip
	del := seedCrossGroup(t, db, models.PayoutHoldAwaitingConfirmation, nil)
	setGroupTransfer(t, db, del, "t")
	// cancelled but NO transfer → not money-stranded, skip
	noTrf := seedCrossGroup(t, db, models.PayoutHoldAwaitingConfirmation, nil)
	setGroupStatus(t, db, noTrf, models.GroupOrderCancelled)

	require.Equal(t, 0, reconcileCancelledGroups())
	require.Equal(t, models.PayoutHoldReversed, loadGroupHold(t, db, rev))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadGroupHold(t, db, del))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadGroupHold(t, db, noTrf))
}

// Fix 2 — a delivered event on a cancelled group must NOT flip it to delivered or
// park its hold (defeating the cancel reverse).
func TestParkGroupOrderOnDelivery_SkipsCancelled(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	id := seedCrossGroup(t, db, models.PayoutHoldNone, nil)
	setGroupStatus(t, db, id, models.GroupOrderCancelled)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return parkGroupOrderOnDelivery(tx, id)
	}))

	require.Equal(t, models.GroupOrderCancelled, groupStatusOf(t, db, id), "cancelled group stays cancelled")
	require.Equal(t, models.PayoutHoldNone, loadGroupHold(t, db, id), "hold not parked on a cancelled group")
}

// Fix 2b — a delivered event on a terminal (cancelled/skipped/declined/refunded)
// meal-plan day must NOT resurrect it to delivered or park its hold (defeating the
// day refund). Mirrors Fix 2 for the meal-plan-day pipeline.
func TestMarkMealPlanDayDelivered_SkipsTerminalDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	orderID := uuid.New()
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, payout_hold_status, price)
		VALUES (?,?,?,?,?,?)`,
		dayID.String(), uuid.NewString(), orderID.String(),
		string(models.MealPlanDayCancelled), string(models.PayoutHoldNone), 150.0).Error)

	MarkMealPlanDayDelivered(orderID)

	var status, hold string
	require.NoError(t, db.Raw(`SELECT status, payout_hold_status FROM meal_plan_days WHERE id = ?`, dayID.String()).
		Row().Scan(&status, &hold))
	require.Equal(t, string(models.MealPlanDayCancelled), status, "terminal day stays cancelled")
	require.Equal(t, string(models.PayoutHoldNone), hold, "hold not parked on a terminal day")
}
