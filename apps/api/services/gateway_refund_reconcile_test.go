package services

// gateway_refund_reconcile_test.go — #640. The daily reconciliation sweep must (a) detect an
// order that is fully refunded in AGGREGATE at the gateway (in-app partial + out-of-band
// dashboard refund) but never stamped refunded_at locally, and (b) NOT false-positive on an
// order whose per-line cancels are fully recorded. FinalizeGatewayFullRefund then stamps the
// terminal refund + drives the (flag-gated) payout block. Reuses the stuck-refund sqlite harness
// (orders + order_items + meal_plan_days + group_orders) and the httptest gateway seam.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// cannedPayment points the gateway at a single /payments/{id} response with the given
// captured amount + cumulative amount_refunded (paise).
func cannedPayment(t *testing.T, capturedPaise, refundedPaise int) {
	t.Helper()
	withRazorpayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/payments/") {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "pay_x", "status": "captured", "captured": true,
				"amount": capturedPaise, "amount_refunded": refundedPaise,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
}

// seedOrderStatus inserts a paid (refunded_at NULL) order in the given status the #640 sweep acts on.
func seedOrderStatus(t *testing.T, db *gorm.DB, total float64, status models.OrderStatus, ps models.PaymentStatus) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, status, payment_status, total,
		wallet_applied, refund_amount, refunded_at, payout_hold_status) VALUES (?,?,?,?,?,0,0,NULL,'')`,
		id.String(), "ORD-C", string(status), string(ps), total).Error)
	return id
}

// seedCompletedOrder is the common case: a delivered+completed order.
func seedCompletedOrder(t *testing.T, db *gorm.DB, total float64) uuid.UUID {
	return seedOrderStatus(t, db, total, models.OrderStatusDelivered, models.PaymentCompleted)
}

// capturedFullPaise is the gateway cumulative refund that equals captured for a wallet-free order.
func capturedFullPaise(total float64) int { return ToPaise(total) }

// A cumulative full refund the local ledger missed (gateway amount_refunded == captured, but
// refunded_at NULL) is flagged as DriftFullRefundUnstamped — the #640 signal.
func TestReconcileRazorpay_FullRefundUnstamped_Detected(t *testing.T) {
	setupStuckRefundDB(t) // provides order_items for PerLineRefundedTotal (0 here)
	cannedPayment(t, 100000, 100000)

	o := &models.Order{ID: uuid.New(), OrderNumber: "ORD-640", RazorpayPaymentID: "pay_x", Total: 1000}
	drifts := reconcileRazorpay(o)

	require.Len(t, drifts, 1)
	require.Equal(t, DriftFullRefundUnstamped, drifts[0].Kind)
}

// A per-line cancel that is fully recorded locally (gateway amount_refunded == per-line refund)
// must NOT be flagged — the old code compared the gateway total against Order.RefundAmount only
// (which excludes per-line) and false-positived on every partially-cancelled order.
func TestReconcileRazorpay_PerLineFullyRecorded_NoFalsePositive(t *testing.T) {
	db := setupStuckRefundDB(t)
	cannedPayment(t, 100000, 60000) // gateway refunded the ₹600 per-line only

	o := &models.Order{ID: uuid.New(), OrderNumber: "ORD-PL", RazorpayPaymentID: "pay_x", Total: 400}
	// ₹600 per-line cancel, fully recorded on order_items.
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		uuid.NewString(), o.ID.String(), true, 600.0).Error)

	drifts := reconcileRazorpay(o)
	require.Empty(t, drifts, "captured=1000, gateway refunded=600 (< captured) == local cumulative (0+600) → no drift")
}

// A genuine partial mismatch (gateway refunded > local cumulative, still < captured) is flagged.
func TestReconcileRazorpay_GenuineMismatch_Flagged(t *testing.T) {
	setupStuckRefundDB(t)
	cannedPayment(t, 100000, 70000) // gateway shows ₹700 refunded

	o := &models.Order{ID: uuid.New(), OrderNumber: "ORD-MM", RazorpayPaymentID: "pay_x", Total: 1000, RefundAmount: 600}
	drifts := reconcileRazorpay(o)

	require.Len(t, drifts, 1)
	require.Equal(t, DriftRefundMismatch, drifts[0].Kind, "gateway ₹700 vs local cumulative ₹600 (< captured ₹1000) → mismatch")
}

// FinalizeGatewayFullRefund stamps the terminal refund (status + refunded_at + payment_status)
// under the row-lock guard; the payout drive is a flag-gated no-op in tests.
func TestFinalizeGatewayFullRefund_StampsAndBlocks(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedCompletedOrder(t, db, 1000)

	require.True(t, FinalizeGatewayFullRefund(id, capturedFullPaise(1000)))

	ps, status, _, at := stuckRow(t, db, id)
	require.True(t, at, "refunded_at stamped → payout release guard now blocks")
	require.Equal(t, string(models.OrderStatusRefunded), status)
	require.Equal(t, string(models.PaymentRefunded), ps)
}

// The guarded UPDATE (WHERE refunded_at IS NULL) makes it single-shot — a second sweep no-ops.
func TestFinalizeGatewayFullRefund_Idempotent(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedCompletedOrder(t, db, 1000)

	require.True(t, FinalizeGatewayFullRefund(id, capturedFullPaise(1000)))
	require.False(t, FinalizeGatewayFullRefund(id, capturedFullPaise(1000)), "already stamped → not healed again")
}

// A typed-escrow shell (meal-plan-day / group) is never finalized here — it refunds via its flow.
func TestFinalizeGatewayFullRefund_SkipsTypedShell(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedCompletedOrder(t, db, 1000)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id) VALUES (?,?)`, uuid.NewString(), id.String()).Error)

	require.False(t, FinalizeGatewayFullRefund(id, capturedFullPaise(1000)), "typed shell skipped")
	_, _, _, at := stuckRow(t, db, id)
	require.False(t, at, "left untouched")
}

// Under-lock re-verification: a gateway refund BELOW the freshly-recomputed captured must NOT
// stamp — the finalize never trusts a stale pre-lock full-refund classification.
func TestFinalizeGatewayFullRefund_BelowCaptured_Skips(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedCompletedOrder(t, db, 1000)

	require.False(t, FinalizeGatewayFullRefund(id, 50000), "gateway ₹500 < captured ₹1000 → not full → skip")
	_, _, _, at := stuckRow(t, db, id)
	require.False(t, at, "not stamped")
}

// The finding-#1 live-order scenario: a genuine ₹900 per-line cancel (Total reduced to ₹100)
// with the chef still owed ₹100 must NOT be mis-stamped as fully refunded. Under the lock the
// per-line add-back reconstructs captured=₹1000, so gateway ₹900 < captured → skip.
func TestFinalizeGatewayFullRefund_PerLineLiveOrder_NotStamped(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedCompletedOrder(t, db, 100) // Total already reduced by a ₹900 per-line cancel
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		uuid.NewString(), id.String(), true, 900.0).Error)

	require.False(t, FinalizeGatewayFullRefund(id, 90000), "captured=100+900=1000; gateway ₹900 < captured → live order preserved")
	_, _, _, at := stuckRow(t, db, id)
	require.False(t, at, "chef's remaining ₹100 payout not blocked")
}

// A legitimately CANCELLED order keeps its status — refunded_at + payment_status block the payout,
// but cancelled→refunded clobbering would destroy the RTO cancelled-vs-refunded distinction.
func TestFinalizeGatewayFullRefund_PreservesCancelledStatus(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedOrderStatus(t, db, 1000, models.OrderStatusCancelled, models.PaymentCompleted)

	require.True(t, FinalizeGatewayFullRefund(id, capturedFullPaise(1000)))

	ps, status, _, at := stuckRow(t, db, id)
	require.True(t, at, "refunded_at stamped → payout blocked")
	require.Equal(t, string(models.PaymentRefunded), ps)
	require.Equal(t, string(models.OrderStatusCancelled), status, "cancelled status preserved, not clobbered to refunded")
}
