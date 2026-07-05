package services

// payout_disputed_fanout_test.go — #498. The disputed-hold drive (reject) and the
// refund crossguard both fan out from the order aggregate to any meal-plan-day /
// group-order whose order_id matches, and RefundDay drives its own day hold out of
// the releasable set. Reuses the setupCrossguardDB harness (flags OFF ⇒ pure DB
// state advance; money seams hit GetRazorpay()==nil).

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// loadGroupHold reads a group order's current payout_hold_status.
func loadGroupHold(t *testing.T, db *gorm.DB, id uuid.UUID) models.PayoutHoldStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM group_orders WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.PayoutHoldStatus(s)
}

// setDayRefundTxn stamps a day's refund_txn_id (the "already refunded" shape).
func setDayRefundTxn(t *testing.T, db *gorm.DB, dayID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE meal_plan_days SET refund_txn_id = ? WHERE id = ?`,
		uuid.NewString(), dayID.String()).Error)
}

// escrowOn turns the meal-plan escrow flag ON for a test and restores it after.
func escrowOn(t *testing.T) {
	t.Helper()
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{MealPlanEscrowEnabled: true}
}

// seedRefundablePlanDay builds a plan struct (the in-memory shape RefundDay reads
// for perDayGross) + a delivered day row with NO payout_transfer_id (so RefundDay
// takes the wallet-credit path without a gateway ReverseTransfer call) and the
// given hold status. Returns the structs RefundDay operates on.
func seedRefundablePlanDay(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus) (*models.MealPlan, *models.MealPlanDay) {
	t.Helper()
	planID, dayID, customer, chef := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status)
		VALUES (?,?,?,?,?)`, planID.String(), "MP-"+planID.String()[:8], customer.String(), chef.String(), "active").Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, order_id, status, payout_transfer_id, price, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), nil, "delivered", "", 120.0, string(hold),
		time.Now().Add(-10*time.Hour)).Error)
	day := &models.MealPlanDay{ID: dayID, Price: 120.0, PayoutHoldStatus: hold}
	plan := &models.MealPlan{
		ID: planID, CustomerID: customer, MealPlanNumber: "MP-" + planID.String()[:8],
		Subtotal: 120.0, Tax: 0, Total: 120.0, Days: []models.MealPlanDay{*day},
	}
	return plan, day
}

// ── Part A: reject drive fans out to linked day / group ─────────────────────

func TestReleaseDisputedHoldsForOrder_DrivesLinkedDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldDisputed, &orderID)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, dayID))
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible), "day transition emits once")
}

func TestReleaseDisputedHoldsForOrder_DrivesLinkedGroup(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	groupID := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadGroupHold(t, db, groupID))
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))
}

func TestReleaseDisputedHoldsForOrder_PendingIssueBlocksDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, customer := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldDisputed, &orderID)
	seedPendingIssue(t, db, orderID, customer) // a second, still-open issue on the order

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID), "a pending issue keeps the day disputed")
	require.Equal(t, 0, countOutbox(t, db, SubjectHoldReleaseEligible))
}

func TestReleaseDisputedHoldsForOrder_RefundedDayBlocked(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldDisputed, &orderID)
	setDayRefundTxn(t, db, dayID) // the day itself was refunded

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID), "a refunded day never releases")
}

func TestReleaseDisputedHoldsForOrder_CancelledGroupBlocked(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	groupID := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID)
	require.NoError(t, db.Exec(`UPDATE group_orders SET status = 'cancelled' WHERE id = ?`, groupID.String()).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldDisputed, loadGroupHold(t, db, groupID), "a cancelled group never releases")
}

func TestReleaseDisputedHoldsForOrder_RefundedOrderBlocksDay(t *testing.T) {
	db := setupCrossguardDB(t)
	now := time.Now()
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", &now) // order refunded_at set
	dayID := seedCrossDay(t, db, models.PayoutHoldDisputed, &orderID)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID), "a refunded linked order blocks the day")
}

// Idempotent: a second reject-drive on the same cleared order does not re-transition
// nor re-emit (the day is already release_eligible, not disputed).
func TestReleaseDisputedHoldsForOrder_Idempotent(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldDisputed, &orderID)

	for i := 0; i < 2; i++ {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
		}))
	}
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, dayID))
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible), "no double-emit on the second run")
}

// ── Part B: refund crossguard fans out to linked day / group ────────────────

func TestWithholdOrReverseOrderHoldForRefund_FansOutToDayGroup(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldAwaitingConfirmation, &orderID)
	groupID := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID)

	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, orderID, "order refund"))
	require.Equal(t, models.PayoutHoldWithheld, loadOrderHold(t, db, orderID))
	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID), "linked day withheld")
	require.Equal(t, models.PayoutHoldWithheld, loadGroupHold(t, db, groupID), "linked group withheld")
}

func TestWithholdOrReverseOrderHoldForRefund_ReleasedDayReversed(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldReleased, &orderID)

	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, orderID, "chargeback"))
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, dayID), "released linked day reversed")
}

// Isolation: an order with no linked day/group, and a day linked to a DIFFERENT
// order, are both untouched — proving the WHERE order_id = X scoping.
func TestWithholdOrReverseOrderHoldForRefund_IsolatesUnlinked(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)
	otherOrder, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	otherDay := seedCrossDay(t, db, models.PayoutHoldAwaitingConfirmation, &otherOrder)

	require.NoError(t, WithholdOrReverseOrderHoldForRefund(db, orderID, "refund"))
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadDayHold(t, db, otherDay),
		"a day linked to a different order is untouched")
}

// ── Part C: the refunded-day hold-state drive (state-only; money already reversed
// by RefundDay, so this NEVER re-runs the money seam) ───────────────────────

func TestMarkRefundedDayHold_Withheld(t *testing.T) {
	db := setupCrossguardDB(t)
	for _, hold := range []models.PayoutHoldStatus{
		models.PayoutHoldReleaseEligible, models.PayoutHoldAwaitingConfirmation, models.PayoutHoldDisputed,
	} {
		dayID := seedCrossDay(t, db, hold, nil)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			return markRefundedDayHold(tx, dayID)
		}))
		require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID), "%s → withheld", hold)
	}
}

// A released day → reversed AND payout_settled_at stamped, so the reconcile cron
// (which re-drives released/reversed-but-unsettled holds) never double-reverses a
// transfer RefundDay already reversed.
func TestMarkRefundedDayHold_ReleasedReversedAndSettled(t *testing.T) {
	db := setupCrossguardDB(t)
	dayID := seedCrossDay(t, db, models.PayoutHoldReleased, nil)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return markRefundedDayHold(tx, dayID)
	}))
	require.Equal(t, models.PayoutHoldReversed, loadDayHold(t, db, dayID))
	var settled *time.Time
	require.NoError(t, db.Raw(`SELECT payout_settled_at FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&settled).Error)
	require.NotNil(t, settled, "reversed refunded day is stamped settled (no reconcile re-reverse)")
}

// Integration: RefundDay (escrow ON) drives the day hold out of the releasable set.
func TestRefundDay_DrivesDayHoldWithheld(t *testing.T) {
	escrowOn(t)
	withRazorpayClient(t, &RazorpayClient{fetchedAt: time.Now()})
	db := setupCrossguardDB(t)
	plan, day := seedRefundablePlanDay(t, db, models.PayoutHoldReleaseEligible)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return RefundDay(tx, plan, day, "customer skipped this day")
	}))
	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, day.ID),
		"a refunded day is withheld so the admin can't release it")
}

// Flag-OFF: RefundDay no-ops entirely (documents the escrow-flag coupling), so the
// hold is left as-is — Part C sits behind that same precondition.
func TestRefundDay_FlagOffNoDrive(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	plan, day := seedRefundablePlanDay(t, db, models.PayoutHoldReleaseEligible)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return RefundDay(tx, plan, day, "skip")
	}))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, day.ID),
		"escrow OFF ⇒ RefundDay is a no-op, hold untouched")
}
