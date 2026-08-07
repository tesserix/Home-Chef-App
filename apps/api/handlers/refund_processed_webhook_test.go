package handlers

// refund_processed_webhook_test.go — #635. The refund webhook used to stamp refunded_at (the
// whole-order-refunded marker) for ANY refund event. But per-line cancels and goodwill
// PARTIAL refunds fire a refund webhook against the same order — so a single partial
// wrongly stamped refunded_at, which (a) freezes the ENTIRE chef payout (payout_release blocks on
// refunded_at IS NOT NULL) and (b) trips claimOrderItemForCancel's "whole order refunded" guard.
// Now refunded_at is stamped only for a FULL refund (amount ≥ the captured Total−WalletApplied).
//
// Driven on the Cashfree refund webhook since #1086 — the retired gateway one is gone, and this is
// where the rule lives now.

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func refundProcessedPayload(cfOrderID, refundID string, amountPaise int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"refund":{"cf_refund_id":1,"refund_id":%q,"order_id":%q,"refund_amount":%.2f,"refund_status":"SUCCESS"}}`,
		refundID, cfOrderID, float64(amountPaise)/100))
}

func refundedAtSet(t *testing.T, db *gorm.DB, orderID string) (refundID string, refundedAtSet bool) {
	t.Helper()
	var row struct {
		RefundID   string
		RefundedAt *string
	}
	require.NoError(t, db.Raw(`SELECT refund_id, refunded_at FROM orders WHERE id = ?`, orderID).Scan(&row).Error)
	return row.RefundID, row.RefundedAt != nil
}

// A PARTIAL refund (amount < captured Total) records the refund_id but must NOT stamp refunded_at.
func TestHandleRefundProcessed_Partial_DoesNotStampRefundedAt(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	// total 500 → captured 50000 paise (no wallet). A ₹100 per-line/goodwill refund = 10000 paise.
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf_o")

	err := NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_partial", 10000), models.ChefModeLive)
	require.NoError(t, err)

	refundID, hasRefundedAt := refundedAtSet(t, db, orderID.String())
	require.Equal(t, "rfnd_partial", refundID, "the gateway refund id is still recorded")
	require.False(t, hasRefundedAt, "#635: a PARTIAL refund must not stamp the whole-order refunded_at marker")
}

// A FULL refund (amount == captured Total) stamps refunded_at.
func TestHandleRefundProcessed_Full_StampsRefundedAt(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf_o") // captured 50000 paise

	err := NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_full", 50000), models.ChefModeLive)
	require.NoError(t, err)

	refundID, hasRefundedAt := refundedAtSet(t, db, orderID.String())
	require.Equal(t, "rfnd_full", refundID)
	require.True(t, hasRefundedAt, "a full refund of the captured amount stamps refunded_at")
}

// A wallet-funded order: captured = Total − WalletApplied. A gateway refund covering the captured
// portion is a full refund of what was charged → stamps refunded_at.
func TestHandleRefundProcessed_Full_RespectsWalletApplied(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf_o")
	require.NoError(t, db.Exec(`UPDATE orders SET wallet_applied = 100 WHERE id = ?`, orderID.String()).Error)
	// captured = (500 − 100) = 400 → 40000 paise.

	// A 10000-paise refund is still partial.
	require.NoError(t, NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_p", 10000), models.ChefModeLive))
	_, hasRA := refundedAtSet(t, db, orderID.String())
	require.False(t, hasRA, "partial vs the captured (wallet-reduced) amount → no stamp")

	// A 40000-paise refund covers the full captured amount → stamp.
	require.NoError(t, NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_f", 40000), models.ChefModeLive))
	_, hasRA2 := refundedAtSet(t, db, orderID.String())
	require.True(t, hasRA2, "full refund of the captured (Total−WalletApplied) amount → stamp")
}

// #635 Finding-1 regression: a per-line cancel reduces order.Total (in reserveOrderItemForCancel,
// #628) by the SAME amount its refund webhook then reports — so comparing the refund
// against the reduced Total would double-count it and wrongly stamp refunded_at. The
// PerLineRefundedTotal add-back reconstructs the original captured amount so a per-line partial
// stays partial (even when cancelling the last line drives Total to 0).
func TestHandleRefundProcessed_PerLineCancel_NotStampedAsFull(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "completed", 1000, "cf_o") // captured 100000 paise

	// Simulate the committed state after cancelling a ₹600 line: Total reduced to 400, the
	// cancelled line's refund recorded (PerLineRefundedTotal = 600).
	require.NoError(t, db.Exec(`UPDATE orders SET total = 400 WHERE id = ?`, orderID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, subtotal, is_cancelled, refund_amount) VALUES (?,?,600,1,600)`,
		uuid.NewString(), orderID.String()).Error)

	// That line's refund webhook (₹600 = 60000 paise) must NOT be seen as a full refund:
	// captured = ToPaise(400 + 600 − 0) = 100000, and 60000 < 100000.
	require.NoError(t, NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_line", 60000), models.ChefModeLive))
	_, hasRA := refundedAtSet(t, db, orderID.String())
	require.False(t, hasRA, "#635: a per-line cancel must not stamp refunded_at as if the whole order were refunded")

	// Cancelling the LAST line drives Total to 0 — still must not stamp (PerLineRefundedTotal now 1000).
	require.NoError(t, db.Exec(`UPDATE orders SET total = 0 WHERE id = ?`, orderID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, subtotal, is_cancelled, refund_amount) VALUES (?,?,400,1,400)`,
		uuid.NewString(), orderID.String()).Error)
	require.NoError(t, NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_line2", 40000), models.ChefModeLive))
	_, hasRA2 := refundedAtSet(t, db, orderID.String())
	require.False(t, hasRA2, "#635: cancelling the last line (Total→0) must not look like a full refund")
}

// Idempotent: a redelivered refund webhook with the SAME id is a no-op.
func TestHandleRefundProcessed_Idempotent(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf_o")

	require.NoError(t, NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_full", 50000), models.ChefModeLive))
	require.NoError(t, NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_o", "rfnd_full", 50000), models.ChefModeLive))
	refundID, hasRA := refundedAtSet(t, db, orderID.String())
	require.Equal(t, "rfnd_full", refundID)
	require.True(t, hasRA)
}

// No matching order for the gateway order id → clean no-op (e.g. a tip refund).
func TestHandleRefundProcessed_NoOrder_NoOp(t *testing.T) {
	db := setupPayDB(t)
	require.NoError(t, NewPaymentHandler().handleCashfreeRefundStatus(refundProcessedPayload("cf_absent", "rfnd_x", 10000), models.ChefModeLive))
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM orders WHERE refund_id = 'rfnd_x'`).Scan(&n).Error)
	require.Equal(t, int64(0), n)
}
