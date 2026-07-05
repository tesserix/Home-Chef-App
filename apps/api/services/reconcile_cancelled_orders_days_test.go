package services

// reconcile_cancelled_orders_days_test.go — #542. Mirrors reconcile_cancelled_group_test.go
// (#534) for the ORDER and MEAL-PLAN-DAY aggregates. An order/day whose refund
// cross-guard crashed before flipping the hold terminal sits cancelled/refunded at a
// non-terminal hold with a held transfer — invisible to the released/reversed sweeps.
// Flags OFF ⇒ the money seam is a no-op; assertions are on the DB state advance.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func setOrderStatus(t *testing.T, db *gorm.DB, id uuid.UUID, status models.OrderStatus) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE orders SET status = ? WHERE id = ?`, string(status), id.String()).Error)
}

func clearOrderRazorpay(t *testing.T, db *gorm.DB, id uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE orders SET razorpay_order_id = '' WHERE id = ?`, id.String()).Error)
}

func setDayStatus(t *testing.T, db *gorm.DB, id uuid.UUID, status models.MealPlanDayStatus) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE meal_plan_days SET status = ? WHERE id = ?`, string(status), id.String()).Error)
}

func clearDayTransfer(t *testing.T, db *gorm.DB, id uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE meal_plan_days SET payout_transfer_id = '' WHERE id = ?`, id.String()).Error)
}

func orderSettledAt(t *testing.T, db *gorm.DB, id uuid.UUID) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM orders WHERE id = ? AND payout_settled_at IS NOT NULL`, id.String()).Scan(&n).Error)
	return n == 1
}

// ── Orders ───────────────────────────────────────────────────────────────────

// A refunded order stuck at a non-terminal (parked) hold with a gateway charge is
// withheld by the reconcile — the released/reversed sweeps never see it.
func TestReconcileCancelledOrders_DrivesStrandedRefundHold(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	id, _ := seedCrossOrder(t, db, models.PayoutHoldAwaitingConfirmation, string(models.OrderStatusRefunded), nil)

	require.Equal(t, 1, reconcileCancelledOrders(), "the stranded refunded order hold is driven")
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, id))

	// Idempotent: withheld is terminal → dropped from the next scan.
	require.Equal(t, 0, reconcileCancelledOrders())
}

// A released (already-paid) hold on a refunded order is reversed + settled — the
// chef's transfer is clawed back, not left paid.
func TestReconcileCancelledOrders_ReleasedIsReversed(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleased, string(models.OrderStatusCancelled), nil)

	require.Equal(t, 1, reconcileCancelledOrders())
	require.Equal(t, models.PayoutHoldReversed, loadOrderHold(t, db, id))
	require.True(t, orderSettledAt(t, db, id), "settled stamped after claw-back")
	require.Equal(t, 0, reconcileCancelledOrders())
}

func TestReconcileCancelledOrders_SkipsNonTargets(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	// already withheld → terminal, skip
	wh, _ := seedCrossOrder(t, db, models.PayoutHoldWithheld, string(models.OrderStatusRefunded), nil)
	// delivered, not refunded/cancelled → skip
	del, _ := seedCrossOrder(t, db, models.PayoutHoldAwaitingConfirmation, string(models.OrderStatusDelivered), nil)
	// refunded but no gateway charge (wallet-only) → no Route transfer stranded, skip
	noRzp, _ := seedCrossOrder(t, db, models.PayoutHoldAwaitingConfirmation, string(models.OrderStatusRefunded), nil)
	clearOrderRazorpay(t, db, noRzp)

	require.Equal(t, 0, reconcileCancelledOrders())
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, wh))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadOrderHold(t, db, del))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadOrderHold(t, db, noRzp))
}

// An order refunded via the issue path leaves status='delivered' but stamps
// refunded_at — it must still be caught.
func TestReconcileCancelledOrders_CatchesRefundedAtOnDeliveredStatus(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	now := time.Now()

	id, _ := seedCrossOrder(t, db, models.PayoutHoldAwaitingConfirmation, string(models.OrderStatusDelivered), &now)

	require.Equal(t, 1, reconcileCancelledOrders())
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, id))
}

// ── Meal-plan days ───────────────────────────────────────────────────────────

func TestReconcileRefundedDays_DrivesStrandedHold(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	id := seedCrossDay(t, db, models.PayoutHoldAwaitingConfirmation, nil)
	setDayStatus(t, db, id, models.MealPlanDayRefunded)

	require.Equal(t, 1, reconcileRefundedDays(), "the stranded refunded day hold is driven")
	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, id))
	require.Equal(t, 0, reconcileRefundedDays())
}

func TestReconcileRefundedDays_SkipsNonTargets(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	// reversed → terminal, skip
	rev := seedCrossDay(t, db, models.PayoutHoldReversed, nil)
	setDayStatus(t, db, rev, models.MealPlanDayRefunded)
	// delivered (not refunded/cancelled) → skip
	del := seedCrossDay(t, db, models.PayoutHoldAwaitingConfirmation, nil) // seed leaves status=delivered
	// refunded but no held transfer → nothing stranded, skip
	noTrf := seedCrossDay(t, db, models.PayoutHoldAwaitingConfirmation, nil)
	setDayStatus(t, db, noTrf, models.MealPlanDayRefunded)
	clearDayTransfer(t, db, noTrf)

	require.Equal(t, 0, reconcileRefundedDays())
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, rev))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadDayHold(t, db, del))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadDayHold(t, db, noTrf))
}
