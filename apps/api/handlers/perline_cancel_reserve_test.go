package handlers

// perline_cancel_reserve_test.go — #628. The per-line cancel (CancelOrderItem) must record its
// WHOLE refund ledger — line.refund_amount, the reduced order subtotal/tax/total, and the
// order-level refund_amount increment — atomically inside the order-locked claim, NOT after the
// (unlocked) gateway call. Otherwise a concurrent services.RefundIssueToWallet locking the order
// in the gateway window reads PerLineRefundedTotalTx=0 + an un-reduced Total, so its
// RemainingRefundable cap still counts the cancelled line as refundable and can refund it twice.
//
// Exercised deterministically on the extracted reserveOrderItemForCancel /
// releaseOrderItemCancelReservation helpers (the sqlite :memory: harness can't run true
// concurrent txns; the reservation-visible-immediately property is the concurrency-safety proof).

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// seedItem inserts an order line with a subtotal (the reserve reads it to compute the line refund
// and the order-total delta).
func seedItem(t *testing.T, db *gorm.DB, orderID uuid.UUID, subtotal float64) uuid.UUID {
	t.Helper()
	itemID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, subtotal, is_cancelled, refund_amount) VALUES (?,?,?,0,0)`,
		itemID.String(), orderID.String(), subtotal).Error)
	return itemID
}

// orderMoney reads the order's live subtotal/tax/total/refund_amount.
func orderMoney(t *testing.T, db *gorm.DB, orderID uuid.UUID) (subtotal, tax, total, refundAmount float64) {
	t.Helper()
	var row struct{ Subtotal, Tax, Total, RefundAmount float64 }
	require.NoError(t, db.Raw(`SELECT subtotal, tax, total, refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&row).Error)
	return row.Subtotal, row.Tax, row.Total, row.RefundAmount
}

func lineState(t *testing.T, db *gorm.DB, itemID uuid.UUID) (cancelled bool, refundAmount float64) {
	t.Helper()
	var row struct {
		IsCancelled  bool
		RefundAmount float64
	}
	require.NoError(t, db.Raw(`SELECT is_cancelled, refund_amount FROM order_items WHERE id = ?`, itemID.String()).Scan(&row).Error)
	return row.IsCancelled, row.RefundAmount
}

// reserveTwoLineOrder seeds a 1000-order (subtotal 900 + tax 100) with two live lines A=500, B=400
// (sum of subtotals == order subtotal, as a real order maintains).
func reserveTwoLineOrder(t *testing.T, db *gorm.DB) (orderID, itemA, itemB uuid.UUID) {
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID = payOrder(t, db, cust, chef, "completed", 1000, "rzp_o", "pay_x") // subtotal 900, tax 100
	itemA = seedItem(t, db, orderID, 500)
	itemB = seedItem(t, db, orderID, 400)
	return orderID, itemA, itemB
}

func doReserve(t *testing.T, db *gorm.DB, orderID, itemID uuid.UUID) (lineRefund float64, won bool) {
	t.Helper()
	err := db.Transaction(func(tx *gorm.DB) error {
		r, w, e := reserveOrderItemForCancel(tx, orderID, itemID, "customer_request", time.Now().UTC())
		lineRefund, won = r, w
		return e
	})
	require.NoError(t, err)
	return lineRefund, won
}

// The reservation writes the whole ledger under the order lock, so RemainingRefundable drops by
// the line refund IMMEDIATELY — a concurrent RefundIssueToWallet is capped to the surviving lines.
func TestReserveOrderItemForCancel_LedgerVisibleImmediately(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db)

	lineRefund, won := doReserve(t, db, orderID, itemA)
	require.True(t, won)
	// A's refund = 500 + 100*(500/900) = 555.555…
	require.InDelta(t, 555.5556, lineRefund, 0.01)

	cancelled, lineRA := lineState(t, db, itemA)
	require.True(t, cancelled, "the line is cancelled")
	require.InDelta(t, 555.5556, lineRA, 0.01, "the line refund_amount is written IN the claim (#628), not after the gateway")

	sub, tax, total, refundAmt := orderMoney(t, db, orderID)
	require.InDelta(t, 400, sub, 0.01, "order subtotal reduced by the line subtotal")
	require.InDelta(t, 44.4444, tax, 0.01, "order tax reduced by the line's proportional share")
	require.InDelta(t, 444.4444, total, 0.01, "order total reduced by the line refund")
	require.InDelta(t, 555.5556, refundAmt, 0.01, "order-level refund_amount incremented")

	// The concurrency-safety proof: a RefundIssueToWallet that locks the order AFTER this
	// reservation sees only the surviving line's value as refundable (not the full 1000).
	remaining := services.RemainingRefundable(&models.Order{ID: orderID})
	require.InDelta(t, 444.4444, remaining, 0.01, "the cancelled line is no longer counted refundable — no double-refund window")
}

// The delta-based reduction equals the canonical recomputeOrderTotals for the same scenario.
func TestReserveOrderItemForCancel_MatchesRecompute(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db)

	_, won := doReserve(t, db, orderID, itemA)
	require.True(t, won)

	// Canonical recompute with A cancelled, B live — no fees on this order.
	items := []models.OrderItem{{Subtotal: 500, IsCancelled: true}, {Subtotal: 400}}
	wantSub, wantTax, wantTotal := recomputeOrderTotals(items, 900, 100, 0, 0, 0, 0)

	sub, tax, total, _ := orderMoney(t, db, orderID)
	require.InDelta(t, wantSub, sub, 0.01)
	require.InDelta(t, wantTax, tax, 0.01)
	require.InDelta(t, wantTotal, total, 0.01)
}

// Releasing the reservation (gateway failure — no money moved) restores the full ledger.
func TestReleaseOrderItemCancelReservation_RestoresLedger(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db)

	lineRefund, won := doReserve(t, db, orderID, itemA)
	require.True(t, won)

	releaseOrderItemCancelReservation(db, orderID, itemA, lineRefund)

	cancelled, lineRA := lineState(t, db, itemA)
	require.False(t, cancelled, "the line is un-cancelled")
	require.InDelta(t, 0, lineRA, 0.001, "the line refund_amount is zeroed")

	sub, tax, total, refundAmt := orderMoney(t, db, orderID)
	require.InDelta(t, 900, sub, 0.01, "order subtotal restored")
	require.InDelta(t, 100, tax, 0.01, "order tax restored")
	require.InDelta(t, 1000, total, 0.01, "order total restored")
	require.InDelta(t, 0, refundAmt, 0.01, "order-level refund_amount restored")
}

// Two sequential per-line cancels compose: each reduces the order off the CURRENT (locked) state,
// so after both lines are cancelled the whole order is refundable-accounted with no drift.
func TestReserveOrderItemForCancel_SequentialLinesCompose(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, itemB := reserveTwoLineOrder(t, db)

	rA, wonA := doReserve(t, db, orderID, itemA)
	require.True(t, wonA)
	rB, wonB := doReserve(t, db, orderID, itemB)
	require.True(t, wonB)

	// A (555.56) + B: after A, order subtotal=400 tax=44.44; B's refund = 400 + 44.44 = 444.44.
	require.InDelta(t, 555.5556, rA, 0.01)
	require.InDelta(t, 444.4444, rB, 0.01)

	sub, tax, total, refundAmt := orderMoney(t, db, orderID)
	require.InDelta(t, 0, sub, 0.01, "both lines cancelled → subtotal 0")
	require.InDelta(t, 0, tax, 0.01, "tax fully removed")
	require.InDelta(t, 0, total, 0.01, "total fully removed")
	require.InDelta(t, 1000, refundAmt, 0.01, "the whole order value is accounted as refunded (555.56 + 444.44)")
	require.InDelta(t, 0, services.RemainingRefundable(&models.Order{ID: orderID}), 0.01, "nothing left refundable")
}

// #628 verify Finding-1: if a concurrent ORDER-LEVEL full refund consumed the reservation and
// moved the order terminal (payment_status=refunded / refunded_at set) during the per-line gateway
// window, a subsequent release must be a NO-OP — reverting would strand money on a terminal order.
func TestReleaseOrderItemCancelReservation_NoOpWhenOrderWentTerminal(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db)

	lineRefund, won := doReserve(t, db, orderID, itemA)
	require.True(t, won)
	subBefore, taxBefore, totalBefore, _ := orderMoney(t, db, orderID)

	// Simulate a concurrent ReserveFullRefund that ran in the gateway window: it consumed the
	// reservation (remaining = surviving lines), marked the order terminal.
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status='refunded', refunded_at=?, refund_amount=1000 WHERE id=?`,
		time.Now().UTC(), orderID.String()).Error)

	// Our per-line gateway then fails → release. It must NOT revert (order is terminal).
	releaseOrderItemCancelReservation(db, orderID, itemA, lineRefund)

	cancelled, lineRA := lineState(t, db, itemA)
	require.True(t, cancelled, "the reservation is LEFT stuck (not un-cancelled) on a terminal order")
	require.InDelta(t, 555.5556, lineRA, 0.01, "the line refund_amount is left intact for reconciliation")

	sub, tax, total, refundAmt := orderMoney(t, db, orderID)
	require.InDelta(t, subBefore, sub, 0.01, "subtotal not restored")
	require.InDelta(t, taxBefore, tax, 0.01, "tax not restored")
	require.InDelta(t, totalBefore, total, 0.01, "total not restored")
	require.InDelta(t, 1000, refundAmt, 0.01, "the concurrent full refund's refund_amount is not clobbered")
	// RemainingRefundable stays 0 (444.44 − 1000 + 555.56) — the gateway-vs-refund_amount drift
	// is what reconciliation flags, never a silent strand.
	require.InDelta(t, 0, services.RemainingRefundable(&models.Order{ID: orderID}), 0.01)
}

// A second reserve of an already-cancelled line is an idempotent no-op — no double reduction.
func TestReserveOrderItemForCancel_IdempotentSecondReserve(t *testing.T) {
	db := setupPayDB(t)
	addIssueGuardTables(t, db)
	orderID, itemA, _ := reserveTwoLineOrder(t, db)

	_, won1 := doReserve(t, db, orderID, itemA)
	require.True(t, won1)
	sub1, _, total1, refund1 := orderMoney(t, db, orderID)

	r2, won2 := doReserve(t, db, orderID, itemA)
	require.False(t, won2, "the line is already cancelled → idempotent, no second reservation")
	require.InDelta(t, 0, r2, 0.001)

	sub2, _, total2, refund2 := orderMoney(t, db, orderID)
	require.InDelta(t, sub1, sub2, 0.001, "subtotal not double-reduced")
	require.InDelta(t, total1, total2, 0.001, "total not double-reduced")
	require.InDelta(t, refund1, refund2, 0.001, "refund_amount not double-counted")
}
