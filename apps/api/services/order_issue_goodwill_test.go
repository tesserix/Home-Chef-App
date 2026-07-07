package services

// order_issue_goodwill_test.go — #618. Platform-goodwill vs chef-clawback on a
// resolved order issue. Goodwill refunds the customer but does NOT claw back the
// chef's payout (the platform absorbs it) — and is honoured only for a PARTIAL
// refund, since a full refund stamps refunded_at, which blocks the chef payout.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// Goodwill on a FULL refund is a contradiction (refunded_at would block the chef
// payout) → rejected BEFORE any money moves.
func TestRefundIssueToWalletWithPolicy_GoodwillFullRejected(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
	iss := seedIssue(t, db, customer, orderID)

	err := RefundIssueToWalletWithPolicy(db, iss, 500, "admin", nil, false /* goodwill */)
	require.ErrorIs(t, err, ErrGoodwillFullRefund)

	assert.Equal(t, 0.0, balanceOfUser(t, db, customer), "no wallet credit on a rejected goodwill-full")
	var orderRefund float64
	db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefund)
	assert.Equal(t, 0.0, orderRefund, "order refund_amount untouched — no money moved")
	var status string
	db.Raw(`SELECT status FROM order_issues WHERE id = ?`, iss.ID.String()).Scan(&status)
	assert.Equal(t, "pending", status, "issue stays pending for a re-resolve")
}

// Goodwill on a PARTIAL refund: the customer is refunded, the issue resolves, but
// refunded_at stays NULL (the chef's payout stays releasable — chef keeps it) and
// the decision is audited.
func TestRefundIssueToWalletWithPolicy_GoodwillPartial(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
	iss := seedIssue(t, db, customer, orderID)

	require.NoError(t, RefundIssueToWalletWithPolicy(db, iss, 120, "admin", nil, false /* goodwill */))

	assert.Equal(t, 120.0, balanceOfUser(t, db, customer), "customer refunded the partial goodwill amount")
	assert.Equal(t, models.IssueResolved, iss.Status)

	var refundedAt *string
	db.Raw(`SELECT refunded_at FROM orders WHERE id = ?`, orderID.String()).Scan(&refundedAt)
	assert.Nil(t, refundedAt, "partial leaves refunded_at NULL → chef payout stays releasable (chef keeps it)")

	var auditCount int64
	db.Raw(`SELECT COUNT(*) FROM audit_logs WHERE action = 'payout.hold.refund_goodwill' AND entity_id = ?`, orderID.String()).Scan(&auditCount)
	assert.Equal(t, int64(1), auditCount, "goodwill decision is audited (chef payout NOT clawed back)")
}

// Clawback (the default) refunds the customer and does NOT write a goodwill audit
// row — it drives the payout cross-guard instead (unchanged behaviour).
func TestRefundIssueToWalletWithPolicy_ClawbackDoesNotAuditGoodwill(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
	iss := seedIssue(t, db, customer, orderID)

	require.NoError(t, RefundIssueToWalletWithPolicy(db, iss, 120, "admin", nil, true /* clawback */))

	assert.Equal(t, 120.0, balanceOfUser(t, db, customer))
	var auditCount int64
	db.Raw(`SELECT COUNT(*) FROM audit_logs WHERE action = 'payout.hold.refund_goodwill' AND entity_id = ?`, orderID.String()).Scan(&auditCount)
	assert.Equal(t, int64(0), auditCount, "clawback must not record a goodwill audit")
}
