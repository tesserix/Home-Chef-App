package services

// refund_claim_before_credit_test.go — #581. RefundIssueToWallet must CLAIM the issue
// (pending → resolved) BEFORE crediting the wallet, so a credit only ever happens for the
// claim-winner. The bug: it credited first and treated a lost claim (RowsAffected != 1)
// as an idempotent no-op — safe only if the claim was lost to ANOTHER RefundIssueToWallet
// (same wallet key). When the issue is flipped out of pending by a NON-wallet path
// (AdminRejectIssue, the #393 delivery-failure customer-fault resolver), a REAL wallet
// credit committed with no matching order.refund_amount increment → a later double-refund
// window. Deterministic test (pre-flip the issue), per the sqlite-can't-test-true-
// concurrency guidance.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundIssueToWallet_NoCreditWhenIssueAlreadyResolvedElsewhere(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
	iss := seedIssue(t, db, customer, orderID)

	// A non-wallet path (e.g. AdminRejectIssue / delivery-failure customer-fault) flips the
	// issue out of pending WITHOUT touching the wallet.
	require.NoError(t, db.Exec(`UPDATE order_issues SET status = 'rejected' WHERE id = ?`, iss.ID.String()).Error)

	// RefundIssueToWallet must observe the lost claim and NOT move real money.
	require.NoError(t, RefundIssueToWallet(db, iss, 120, "admin", nil))

	assert.Equal(t, 0.0, balanceOfUser(t, db, customer),
		"no wallet credit once the issue was resolved via a non-wallet path")
	var orderRefund float64
	db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefund)
	assert.Equal(t, 0.0, orderRefund, "order refund ledger untouched")
	var status string
	db.Raw(`SELECT status FROM order_issues WHERE id = ?`, iss.ID.String()).Scan(&status)
	assert.Equal(t, "rejected", status, "the prior resolution stands")
}

// The winner still fully refunds; the loser (issue already resolved) does not credit —
// proving the claim, not the wallet key, is now the serialization point.
func TestRefundIssueToWallet_SecondDistinctCallOnResolvedIssueNoOps(t *testing.T) {
	db := setupIssueDB(t)
	customer, orderID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount) VALUES (?, 500, 0)`, orderID.String()).Error)
	iss := seedIssue(t, db, customer, orderID)

	require.NoError(t, RefundIssueToWallet(db, iss, 120, "admin", nil)) // winner credits 120
	assert.Equal(t, 120.0, balanceOfUser(t, db, customer))

	// Re-drive on the now-resolved issue → no additional credit, ledger unchanged.
	require.NoError(t, RefundIssueToWallet(db, iss, 120, "admin", nil))
	assert.Equal(t, 120.0, balanceOfUser(t, db, customer), "no second credit")
	var orderRefund float64
	db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefund)
	assert.Equal(t, 120.0, orderRefund)
}
