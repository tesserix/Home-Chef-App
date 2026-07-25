package handlers

// chef_order_cancel_penalty_test.go — the chef-cancel platform-fee penalty.
//
// When a chef cancels an in-progress order (their fault), the customer is
// made whole via the existing refund path (chef_order_cancel_deferred_test.go
// etc.), and — the piece this file pins — the chef is penalised the order's
// platform/service fee: a debit.penalty entry on the payout ledger, collected
// off their NEXT payout (services/payout_recovery.go, handlers/payment.go's
// applyChefRecoveryDeduction / dischargeChefRecoveryForOrder). Reuses the
// same setupPayDB/payOrder/callChefCancel harness as the other chef-cancel
// test files in this package.

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
	"github.com/homechef/api/payouts"
)

// setServiceFee stamps the order's service_fee column — payOrder doesn't set
// it, and CancelOrder's penalty gate reads it directly off the loaded order.
func setServiceFee(t *testing.T, orderID uuid.UUID, fee float64) {
	t.Helper()
	require.NoError(t, database.DB.Exec(`UPDATE orders SET service_fee = ? WHERE id = ?`, fee, orderID.String()).Error)
}

// penaltyRowsFor reads back every debit.penalty ledger row a cancel of this
// order raised against this chef.
func penaltyRowsFor(t *testing.T, chefID, orderID uuid.UUID) []payouts.LedgerEntry {
	t.Helper()
	var entries []payouts.LedgerEntry
	require.NoError(t, database.DB.Where(
		"payee_type = ? AND payee_id = ? AND kind = ? AND source_type = ? AND source_id = ?",
		payouts.PayeeChef, chefID, payouts.EntryDebitPenalty, "order", orderID.String(),
	).Find(&entries).Error)
	return entries
}

// TestCancelOrder_ServiceFeePenalty_RaisedOnChefCancel proves the happy path:
// a chef cancelling a mid-prep order with a nonzero service fee raises
// exactly one debit.penalty entry for the platform fee amount, in paise.
func TestCancelOrder_ServiceFeePenalty_RaisedOnChefCancel(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	_, refundCalls := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chefID := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chefID, "completed", 500, "rzp_o", "pay_x")
	markPreparing(t, orderID)
	setServiceFee(t, orderID, 18.96)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 1, *refundCalls, "the customer refund path is unaffected by the new penalty")

	rows := penaltyRowsFor(t, chefID, orderID)
	require.Len(t, rows, 1, "exactly one penalty entry must be raised")
	require.EqualValues(t, 1896, rows[0].AmountMinor, "the penalty must equal the order's service fee in paise (₹18.96)")
	require.Equal(t, payouts.CurrencyINR, rows[0].Currency)
	require.Contains(t, rows[0].Reason, "HC-", "the reason must identify the cancelled order")
}

// TestCancelOrder_ZeroServiceFee_NoPenalty guards the `order.ServiceFee > 0`
// gate: an order with no platform fee raises no penalty row at all.
func TestCancelOrder_ZeroServiceFee_NoPenalty(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chefID := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chefID, "completed", 500, "rzp_o", "pay_x")
	markPreparing(t, orderID) // service_fee is left at its DDL default of 0

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "equipment_failure"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Empty(t, penaltyRowsFor(t, chefID, orderID), "no service fee owed means no penalty")
}

// TestCancelOrder_ConcurrentDuplicateCancel_PenaltyRaisedOnce mirrors the
// concurrency shape RaiseChefRecoveryPenalty's dedupe protects: the loser of
// a concurrent ReserveFullRefund claim still runs the rest of CancelOrder's
// tail (cross-guard, penalty raise) with amountPaise=0 for the refund, but
// order.ServiceFee is unchanged — so without ledger-level dedup two racing
// requests would double-penalise. Simulated deterministically by invoking the
// handler twice in a row against an order the FIRST call already reserved and
// cancelled (payment_status flips away from completed after the first
// success, closely mirroring what a true second concurrent request would
// observe at the reservation step) — the assertion that matters is the
// ledger, not the HTTP outcome of the second call.
func TestCancelOrder_ConcurrentDuplicateCancel_PenaltyRaisedOnce(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chefID := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chefID, "completed", 500, "rzp_o", "pay_x")
	markPreparing(t, orderID)
	setServiceFee(t, orderID, 18.96)

	w1 := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w1.Code, w1.Body.String())

	// A second call against the now-cancelled, already-refunded order hits the
	// handler's own top-of-function idempotent short-circuit before it would
	// ever reach the penalty tail again — but RaiseChefRecoveryPenalty's own
	// dedupe is the belt-and-suspenders guard for the race where a genuine
	// concurrent request reaches the tail before the first commits. Either
	// way, the ledger must show exactly one row.
	w2 := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	rows := penaltyRowsFor(t, chefID, orderID)
	require.Len(t, rows, 1, "a re-cancel of the same order must never double-penalise the chef")
}
