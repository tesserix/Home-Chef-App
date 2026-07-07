package services

// meal_plan_day_issue_refund_test.go — #618 slice 2. A customer QUALITY issue on a
// DELIVERED meal-plan day is reported against the day's per-day fulfilment SHELL order
// (no razorpay_order_id; the day carries its payout hold on meal_plan_days, not the
// shell). RefundIssueToWalletWithPolicy must reconcile the DAY when it claws the chef:
//   - FULL clawback   → whole day withheld, day.refund_txn_id stamped, shell → refunded
//   - PARTIAL clawback → whole day withheld (a single-unit day forfeits its payout on any
//     chef-fault refund) + day.refund_txn_id stamped, NOT released
//   - PARTIAL goodwill → day left releasable (chef keeps the payout, platform absorbs)
//
// Uses setupCrossguardDB (orders + meal_plan_days + order_issues + wallets). The shell is
// seeded with NO razorpay_order_id — the faithful day-shell shape that makes the order
// partial-claw a no-op (the exact hole this slice plugs). Flags OFF, GetRazorpay()==nil,
// so every transition is a pure DB state advance.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedDayShellIssue inserts a delivered meal-plan-day SHELL order (razorpay_order_id
// EMPTY), its parent plan + linked day (in dayHold, with a held transfer), and a pending
// OrderIssue on the shell carrying MealPlanDayID. Returns the shell order id, day id,
// customer id and the issue.
func seedDayShellIssue(t *testing.T, db *gorm.DB, dayHold models.PayoutHoldStatus, total float64) (uuid.UUID, uuid.UUID, uuid.UUID, *models.OrderIssue) {
	t.Helper()
	orderID, customer, chef, planID, dayID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, razorpay_order_id, total, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		orderID.String(), "ORD-"+orderID.String()[:8], customer.String(), chef.String(),
		"delivered", "", total, "", time.Now().Add(-10*time.Hour)).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status)
		VALUES (?,?,?,?,?)`, planID.String(), "MP-"+planID.String()[:8], customer.String(), chef.String(), "active").Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, order_id, status, payout_transfer_id, price, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), orderID.String(), "delivered", "trf_day123", total,
		string(dayHold), time.Now().Add(-10*time.Hour)).Error)
	issue := &models.OrderIssue{
		OrderID: orderID, ChefID: chef, CustomerID: customer,
		Reason: models.IssueQualityIssue, Status: models.IssuePending, MealPlanDayID: &dayID,
	}
	require.NoError(t, db.Create(issue).Error)
	return orderID, dayID, customer, issue
}

// loadDayRefundTxn reads a day's refund_txn_id (nil when unrefunded).
func loadDayRefundTxn(t *testing.T, db *gorm.DB, id uuid.UUID) *string {
	t.Helper()
	var txn *string
	require.NoError(t, db.Raw(`SELECT refund_txn_id FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&txn).Error)
	return txn
}

// A FULL quality refund on a delivered day: the whole day payout is withheld, the day is
// stamped refunded (refund_txn_id) and the shell order → refunded (out of the statement).
func TestDayIssueRefund_FullClawback_WithholdsAndStampsDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, dayID, _, issue := seedDayShellIssue(t, db, models.PayoutHoldDisputed, 120.0)

	require.NoError(t, RefundIssueToWallet(db, issue, 120.0, "system", nil)) // full

	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID), "full day refund withholds the whole day payout")
	require.NotNil(t, loadDayRefundTxn(t, db, dayID), "day.refund_txn_id stamped so the day reflects the money")
	ord := loadOrder(t, db, orderID)
	require.Equal(t, string(models.OrderStatusRefunded), string(ord.Status), "shell order → refunded (out of the weekly statement)")
	require.NotNil(t, ord.RefundedAt, "a full refund stamps refunded_at")
}

// A PARTIAL quality refund with the default clawback: the day is a single unit, so a
// chef-fault refund forfeits the WHOLE day payout — the day is withheld (NOT released),
// and refund_txn_id is stamped. Without the fix the partial cross-guard no-ops on the
// shell and the dispute-clear RELEASES the day → chef paid in full (the money hole).
func TestDayIssueRefund_PartialClawback_WithholdsWholeDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, dayID, _, issue := seedDayShellIssue(t, db, models.PayoutHoldDisputed, 120.0)

	require.NoError(t, RefundIssueToWallet(db, issue, 60.0, "admin", nil)) // partial (60 of 120)

	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID),
		"a partial chef-fault refund withholds the whole single-unit day (NOT released)")
	require.NotNil(t, loadDayRefundTxn(t, db, dayID), "day.refund_txn_id stamped")
	require.Nil(t, loadOrder(t, db, orderID).RefundedAt, "a partial refund does not stamp refunded_at")
}

// A PARTIAL platform-GOODWILL refund on a delivered day: the customer is refunded but the
// chef KEEPS the day payout (Slice 1 parity). The frozen day hold is cleared back to
// release_eligible and the day is NOT stamped refunded.
func TestDayIssueRefund_PartialGoodwill_ReleasesDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	_, dayID, _, issue := seedDayShellIssue(t, db, models.PayoutHoldDisputed, 120.0)
	admin := uuid.New()

	require.NoError(t, RefundIssueToWalletWithPolicy(db, issue, 60.0, "admin", &admin, false)) // partial goodwill

	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, dayID),
		"goodwill keeps the chef's day payout — the dispute clears back to release_eligible")
	require.Nil(t, loadDayRefundTxn(t, db, dayID), "goodwill does not stamp the day refunded")
}
