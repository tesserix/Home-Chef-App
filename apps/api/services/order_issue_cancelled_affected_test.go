package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// order_issue_cancelled_affected_test.go — #624. An item a customer named in a still-PENDING
// issue can be cancelled by the chef (CancelOrderItem) BEFORE the issue is resolved. That cancel
// already refunded the line AND added it back into RemainingRefundable's per-line headroom, so a
// later RefundIssueToWallet must NOT refund the same line a second time. This is the resolve-time
// mirror of the #622 ReportIssue is_cancelled exclusion, enforced under the order lock.

// Partial exclusion: an issue named A + B; A is cancelled after the report. Only B (still live)
// may be refunded through the issue — A's value must not be returned twice.
func TestRefundIssueToWallet_ExcludesAffectedLineCancelledAfterReport(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	itemA, itemB := uuid.New(), uuid.New()

	// Order originally A(200)+B(200)+C(200)+D(200)=800. After the issue was filed the chef
	// cancels A → recomputeOrderTotals leaves total=600, refund_amount=200, and A carries its own
	// 200 per-line refund. C+D stay live, so RemainingRefundable = 600−200+200 = 600 has headroom
	// that the naive order-level cap would let the issue consume for the already-refunded A.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 600, 200)`, orderID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		itemA.String(), orderID.String(), true, 200).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		itemB.String(), orderID.String(), false, 0).Error)

	// Issue filed while A + B were both live: requested = A+B = 400, names both lines, still pending.
	iss := &models.OrderIssue{
		OrderID: orderID, ChefID: uuid.New(), CustomerID: customer,
		Reason: models.IssueMissingItem, Status: models.IssuePending,
		AffectedItemIDs: pq.StringArray{itemA.String(), itemB.String()},
		RequestedAmount: 400,
	}
	require.NoError(t, db.Create(iss).Error)

	// Admin resolves the still-pending issue at its requested 400.
	require.NoError(t, RefundIssueToWallet(db, iss, 400, "admin", nil))

	assert.Equal(t, 200.0, balanceOfUser(t, db, customer),
		"only the still-live affected line (B) is refunded via the issue; A is not refunded twice")
	assert.Equal(t, 200.0, iss.RefundAmount)

	var orderRefund float64
	db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefund)
	assert.Equal(t, 400.0, orderRefund, "order refund = 200 (cancel) + 200 (issue), never the doubled 600")
}

// Full exclusion: an issue named only A; A is cancelled after the report while another line (C)
// stays live and gives RemainingRefundable headroom. Everything this issue could claim was
// already refunded via the cancel → nothing left to refund through the issue.
func TestRefundIssueToWallet_AllAffectedCancelled_NothingLeft(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	itemA, itemC := uuid.New(), uuid.New()

	// A(200)+C(200)=400; A cancelled → total=200, refund_amount=200, A per-line refund 200. C live.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 200, 200)`, orderID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		itemA.String(), orderID.String(), true, 200).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		itemC.String(), orderID.String(), false, 0).Error)

	iss := &models.OrderIssue{
		OrderID: orderID, ChefID: uuid.New(), CustomerID: customer,
		Reason: models.IssueMissingItem, Status: models.IssuePending,
		AffectedItemIDs: pq.StringArray{itemA.String()},
		RequestedAmount: 200,
	}
	require.NoError(t, db.Create(iss).Error)

	err := RefundIssueToWallet(db, iss, 200, "admin", nil)
	require.ErrorIs(t, err, ErrNothingToRefund)
	assert.Equal(t, 0.0, balanceOfUser(t, db, customer), "the only affected line was already refunded via the cancel — no second refund")

	var status string
	db.Raw(`SELECT status FROM order_issues WHERE id = ?`, iss.ID.String()).Scan(&status)
	assert.Equal(t, "pending", status, "issue stays pending for admin reject; nothing resolved")
}

// No affected line cancelled → the #624 cap does not bite: admin discretion above the requested
// amount is preserved (still bounded only by the order's remaining refundable).
func TestRefundIssueToWallet_NoCapWhenNoAffectedLineCancelled(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	itemA := uuid.New()

	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		itemA.String(), orderID.String(), false, 0).Error)

	iss := &models.OrderIssue{
		OrderID: orderID, ChefID: uuid.New(), CustomerID: customer,
		Reason: models.IssueMissingItem, Status: models.IssuePending,
		AffectedItemIDs: pq.StringArray{itemA.String()},
		RequestedAmount: 100,
	}
	require.NoError(t, db.Create(iss).Error)

	// Admin goodwill of 250 (above the requested 100) with nothing cancelled → not capped by #624.
	require.NoError(t, RefundIssueToWallet(db, iss, 250, "admin", nil))
	assert.Equal(t, 250.0, balanceOfUser(t, db, customer),
		"admin discretion above the requested amount is preserved when no affected line was cancelled")
}
