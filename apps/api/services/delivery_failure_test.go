package services

// delivery_failure_test.go — #393 slice 1. A terminally-failed delivery must not
// strand the order in limbo: it opens a pending `delivery_failed` OrderIssue (the
// dispute signal the admin queue surfaces) and FREEZES the order's payout hold to
// `disputed` so the chef is not paid until an admin confirms fault. Slice 1 moves NO
// money — it only freezes. Reuses the setupCrossguardDB harness (flags irrelevant —
// disputing a hold is plain DB state).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func countOrderIssues(t *testing.T, db *gorm.DB, orderID uuid.UUID, reason models.IssueReason, status models.IssueStatus) int {
	t.Helper()
	var n int
	require.NoError(t, db.Raw(`SELECT count(*) FROM order_issues WHERE order_id = ? AND reason = ? AND status = ?`,
		orderID.String(), string(reason), string(status)).Scan(&n).Error)
	return n
}

func loadOrderRefundAmount(t *testing.T, db *gorm.DB, orderID uuid.UUID) float64 {
	t.Helper()
	var v float64
	require.NoError(t, db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&v).Error)
	return v
}

// loadFailureOrder reads only the columns RecordDeliveryFailure needs.
func loadFailureOrder(t *testing.T, db *gorm.DB, orderID uuid.UUID) *models.Order {
	t.Helper()
	var o models.Order
	require.NoError(t, db.Raw(`SELECT id, chef_id, customer_id, razorpay_order_id FROM orders WHERE id = ?`,
		orderID.String()).Scan(&o).Error)
	return &o
}

func TestSuggestedFaultClass(t *testing.T) {
	require.Equal(t, models.FaultCustomer, models.SuggestedFaultClass(models.FailureCustomerUnavailable))
	require.Equal(t, models.FaultCustomer, models.SuggestedFaultClass(models.FailureCustomerRefused))
	require.Equal(t, models.FaultPlatform, models.SuggestedFaultClass(models.FailureDriverNoShow))
	require.Equal(t, models.FaultChef, models.SuggestedFaultClass(models.FailureFoodDamaged))
	// Ambiguous reasons never auto-resolve — they default to admin review.
	require.Equal(t, models.FaultAmbiguous, models.SuggestedFaultClass(models.FailureWrongAddress))
	require.Equal(t, models.FaultAmbiguous, models.SuggestedFaultClass(models.FailureOther))
	require.Equal(t, models.FaultAmbiguous, models.SuggestedFaultClass("unrecognized"))
}

func TestSetOrderHoldDisputed_FreezesFromPreTerminal(t *testing.T) {
	for _, start := range []models.PayoutHoldStatus{models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation} {
		db := setupCrossguardDB(t)
		orderID, _ := seedCrossOrder(t, db, start, "delivering", nil)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return SetOrderHoldDisputed(tx, orderID) }))
		require.Equal(t, models.PayoutHoldDisputed, loadOrderHold(t, db, orderID))
		require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed), "genuine transition emits hold_disputed once")
	}
}

func TestSetOrderHoldDisputed_NoOpOnSettledStates(t *testing.T) {
	// A hold already eligible/released/reversed must never be dragged back to disputed
	// (the #458 invariant + no un-settling real money movement).
	for _, start := range []models.PayoutHoldStatus{models.PayoutHoldReleaseEligible, models.PayoutHoldReleased, models.PayoutHoldReversed} {
		db := setupCrossguardDB(t)
		orderID, _ := seedCrossOrder(t, db, start, "delivered", nil)
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return SetOrderHoldDisputed(tx, orderID) }))
		require.Equal(t, start, loadOrderHold(t, db, orderID))
		require.Equal(t, 0, countOutbox(t, db, SubjectHoldDisputed))
	}
}

func TestRecordDeliveryFailure_OpensIssueAndFreezesNoMoney(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	order := loadFailureOrder(t, db, orderID)
	var froze bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		froze, err = RecordDeliveryFailure(tx, order, models.FailureCustomerUnavailable)
		return err
	}))
	require.True(t, froze, "first terminalization reports froze=true so the caller notifies once")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, models.PayoutHoldDisputed, loadOrderHold(t, db, orderID))
	// Slice 1 freezes money — it must not refund/release/reverse anything.
	require.Equal(t, 0.0, loadOrderRefundAmount(t, db, orderID))
}

func TestRecordDeliveryFailure_Idempotent(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	order := loadFailureOrder(t, db, orderID)
	var froze [2]bool
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			var err error
			froze[i], err = RecordDeliveryFailure(tx, order, models.FailureCustomerUnavailable)
			return err
		}))
	}
	require.True(t, froze[0], "first call froze")
	require.False(t, froze[1], "second call is an idempotent no-op → froze=false (no duplicate notification)")
	require.Equal(t, 1, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending),
		"a re-fired failure must not open a second issue")
	require.Equal(t, models.PayoutHoldDisputed, loadOrderHold(t, db, orderID))
}

func TestRecordDeliveryFailure_SkipsNonGatewayOrder(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	require.NoError(t, db.Exec(`UPDATE orders SET razorpay_order_id = '' WHERE id = ?`, orderID.String()).Error)
	order := loadFailureOrder(t, db, orderID)
	var froze bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		froze, err = RecordDeliveryFailure(tx, order, models.FailureCustomerUnavailable)
		return err
	}))
	require.False(t, froze, "non-gateway order is not frozen here (no misleading notification)")
	// Meal-plan/group consolidated orders settle their own path (slice 4) — no issue,
	// hold untouched.
	require.Equal(t, 0, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
	require.Equal(t, models.PayoutHoldNone, loadOrderHold(t, db, orderID))
}
