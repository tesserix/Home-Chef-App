package handlers

// cancel_item_issue_guard_test.go — #622. A per-line cancel (CancelOrderItem) must NOT refund a
// line whose money was already returned via the customer-issue path (services.RefundIssueToWallet
// credits the wallet + bumps order.refund_amount / order_issues.refund_amount, but — unlike the
// ReserveRefund family — never flips payment_status, so the #620 mutex can't see it). Two blocks,
// both enforced under the order lock inside claimOrderItemForCancel:
//   (a) refunded_at set  → the WHOLE order was refunded (incl. a FULL issue refund) → block all lines.
//   (b) a resolved/auto_refunded issue with refund_amount>0 naming this line → block that line.
// A rejected/pending issue, or one not naming the line, does NOT block.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// addIssueGuardTables gives setupPayDB the order_issues table + the per-line cancel bookkeeping
// columns claimOrderItemForCancel writes.
// addIssueGuardTables adds the per-line cancel bookkeeping columns claimOrderItemForCancel
// writes (order_issues now lives in setupPayDB).
func addIssueGuardTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, col := range []string{"cancelled_reason TEXT DEFAULT ''", "cancelled_at DATETIME", "refund_id TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE order_items ADD COLUMN `+col).Error)
	}
}

func seedGuardOrderItem(t *testing.T, db *gorm.DB, orderID uuid.UUID) uuid.UUID {
	t.Helper()
	itemID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,0,0)`,
		itemID.String(), orderID.String()).Error)
	return itemID
}

// seedIssue inserts an order_issue. affectedItemIDs is stored in Postgres array-literal form so
// pq.StringArray scans it back (same as prod text[]).
func seedIssue(t *testing.T, db *gorm.DB, orderID uuid.UUID, status string, refund float64, affected string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason,
		affected_item_ids, refund_amount, status) VALUES (?,?,?,?,?,?,?,?)`,
		uuid.NewString(), orderID.String(), uuid.NewString(), uuid.NewString(), "damaged",
		affected, refund, status).Error)
}

func claimGuard(t *testing.T, db *gorm.DB, orderID, itemID uuid.UUID) (bool, error) {
	t.Helper()
	var won bool
	err := db.Transaction(func(tx *gorm.DB) error {
		w, e := claimOrderItemForCancel(tx, orderID, itemID, "customer_request", time.Now().UTC())
		won = w
		return e
	})
	return won, err
}

// (b) A resolved issue that refunded this line blocks the per-line cancel.
func TestClaimOrderItemForCancel_BlocksLineRefundedViaIssue(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 1000, "rzp_o", "pay_x")
	itemID := seedGuardOrderItem(t, db, orderID)
	seedIssue(t, db, orderID, "resolved", 500, "{"+itemID.String()+"}")

	won, err := claimGuard(t, db, orderID, itemID)
	require.ErrorIs(t, err, errLineAlreadyRefunded, "a line already refunded via a resolved issue can't be per-line-refunded again")
	require.False(t, won)

	var cancelled bool
	require.NoError(t, db.Raw(`SELECT is_cancelled FROM order_items WHERE id = ?`, itemID.String()).Row().Scan(&cancelled))
	require.False(t, cancelled, "no second refund; the line stays live")
}

// (b) auto_refunded (the instant-refund path) also counts.
func TestClaimOrderItemForCancel_BlocksLineAutoRefundedViaIssue(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 1000, "rzp_o", "pay_x")
	itemID := seedGuardOrderItem(t, db, orderID)
	seedIssue(t, db, orderID, "auto_refunded", 500, "{"+itemID.String()+"}")

	won, err := claimGuard(t, db, orderID, itemID)
	require.ErrorIs(t, err, errLineAlreadyRefunded)
	require.False(t, won)
}

// (a) A fully-refunded order (refunded_at set — e.g. a FULL issue refund, which keeps
// payment_status=completed) blocks every line.
func TestClaimOrderItemForCancel_BlocksFullyRefundedOrder(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 1000, "rzp_o", "pay_x")
	itemID := seedGuardOrderItem(t, db, orderID)
	require.NoError(t, db.Exec(`UPDATE orders SET refunded_at = CURRENT_TIMESTAMP WHERE id = ?`, orderID.String()).Error)

	won, err := claimGuard(t, db, orderID, itemID)
	require.ErrorIs(t, err, errLineAlreadyRefunded, "a fully-refunded order blocks all per-line cancels even with payment_status=completed")
	require.False(t, won)
}

// An issue that refunded a DIFFERENT line does not block this line.
func TestClaimOrderItemForCancel_OtherLineIssue_DoesNotBlock(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 1000, "rzp_o", "pay_x")
	itemA := seedGuardOrderItem(t, db, orderID)
	itemB := seedGuardOrderItem(t, db, orderID)
	seedIssue(t, db, orderID, "resolved", 500, "{"+itemA.String()+"}") // names A only

	won, err := claimGuard(t, db, orderID, itemB) // cancelling B
	require.NoError(t, err)
	require.True(t, won, "an issue on item A must not block cancelling item B")
}

// A rejected issue (no money credited) does not block; nor does a pending one.
func TestClaimOrderItemForCancel_RejectedOrPendingIssue_DoesNotBlock(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 1000, "rzp_o", "pay_x")
	itemID := seedGuardOrderItem(t, db, orderID)
	seedIssue(t, db, orderID, "rejected", 0, "{"+itemID.String()+"}")
	seedIssue(t, db, orderID, "pending", 0, "{"+itemID.String()+"}")

	won, err := claimGuard(t, db, orderID, itemID)
	require.NoError(t, err)
	require.True(t, won, "a rejected/pending issue returned no money, so the line is still refundable on cancel")
}
