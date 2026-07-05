package services

// refundable_test.go — #527. Proves RemainingRefundable = paid − already-refunded,
// correcting the Total − RefundAmount strand: after a per-line cancel, Total was
// reduced by the refunded line AND RefundAmount counts it, so the naive difference
// double-subtracts and a full cancel refunds ~0, stranding the live items' money.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedCancelledItem inserts a cancelled order line carrying a per-line refund_amount —
// the exact shape recomputeOrderTotals leaves behind after a per-line cancel.
func seedCancelledItem(t *testing.T, db *gorm.DB, orderID uuid.UUID, refund float64) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount, subtotal)
		VALUES (?,?,?,?,?)`, uuid.NewString(), orderID.String(), true, refund, refund).Error)
}

func walletBalance(t *testing.T, db *gorm.DB, userID uuid.UUID) float64 {
	t.Helper()
	var bal float64
	require.NoError(t, db.Raw(`SELECT COALESCE(balance,0) FROM wallets WHERE user_id = ?`, userID.String()).Scan(&bal).Error)
	return bal
}

// seedOrderRow inserts an order carrying total + refund_amount so RemainingRefundable
// (which reads a consistent DB snapshot, not the caller's struct) sees real state.
func seedOrderRow(t *testing.T, db *gorm.DB, total, refundAmount float64) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount, payment_status, payment_provider)
		VALUES (?,?,?,?,?)`, id.String(), total, refundAmount, "completed", "razorpay").Error)
	return id
}

func TestPerLineRefundedTotal_SumsOnlyCancelledLines(t *testing.T) {
	db := setupCancelRefundDB(t)
	orderID := uuid.New()
	seedCancelledItem(t, db, orderID, 110)
	seedCancelledItem(t, db, orderID, 40)
	// A live (not cancelled) line must NOT count.
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount, subtotal)
		VALUES (?,?,?,?,?)`, uuid.NewString(), orderID.String(), false, 0, 110).Error)

	require.Equal(t, 150.0, PerLineRefundedTotal(orderID))
	require.Equal(t, 0.0, PerLineRefundedTotal(uuid.New()), "an order with no cancelled lines → 0 (no-op)")
}

// The #527 scenario: 2×₹110 order, per-line cancel item 1 → Total 110, RefundAmount 110,
// one cancelled line refund 110. A full cancel must refund the remaining live ₹110, NOT
// the naive Total − RefundAmount = 0.
func TestRemainingRefundable_PerLineCancel_DoesNotStrandLiveItems(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedOrderRow(t, db, 110, 110)
	seedCancelledItem(t, db, id, 110)

	require.Equal(t, 110.0, RemainingRefundable(&models.Order{ID: id}),
		"remaining owed = paid(220) − refunded(110) = 110, not Total−RefundAmount = 0")
}

// Goodwill/full-cancel/issue refunds bump RefundAmount WITHOUT touching Total or items,
// so they are subtracted normally: a ₹20 goodwill on top of a per-line ₹110 cancel of a
// ₹220 order leaves ₹90 owed (paid 220 − refunded 130).
func TestRemainingRefundable_MixedPerLineAndGoodwill(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedOrderRow(t, db, 110, 130) // 110 per-line + 20 goodwill
	seedCancelledItem(t, db, id, 110)   // only the per-line refund is on an item

	require.Equal(t, 90.0, RemainingRefundable(&models.Order{ID: id}))
}

// No per-line cancel ⇒ strict no-op: RemainingRefundable == Total − RefundAmount.
func TestRemainingRefundable_NoPerLineCancel_IsNoOp(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedOrderRow(t, db, 220, 50) // e.g. a prior goodwill only
	require.Equal(t, 170.0, RemainingRefundable(&models.Order{ID: id}))
}

// Never negative (defensive) — a fully-refunded order with no live items → 0.
func TestRemainingRefundable_NeverNegative(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedOrderRow(t, db, 0, 50)
	require.Equal(t, 0.0, RemainingRefundable(&models.Order{ID: id}))
}

// Race guard (the verifier's #1): RemainingRefundable must IGNORE a stale caller
// snapshot and use the committed DB state — mixing a stale Total with the fresh
// per-line sum would over-count and over-refund.
func TestRemainingRefundable_IgnoresStaleCallerSnapshot(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedOrderRow(t, db, 110, 110) // real state after a per-line cancel
	seedCancelledItem(t, db, id, 110)

	// Caller's struct is STALE (pre-cancel: Total 220, RefundAmount 0). A naive
	// stale-Total + fresh-per-line would yield 220 − 0 + 110 = 330 (over-refund).
	stale := &models.Order{ID: id, Total: 220, RefundAmount: 0}
	require.Equal(t, 110.0, RemainingRefundable(stale),
		"uses committed DB state (110), not the stale snapshot (would be 330)")
}

// Fallback: an order not present in the DB (unexpected) falls back to the caller's
// snapshot minus refunded — never over-refunds.
func TestRemainingRefundable_MissingOrderFallsBackToSnapshot(t *testing.T) {
	setupCancelRefundDB(t)
	order := &models.Order{ID: uuid.New(), Total: 200, RefundAmount: 50} // no DB row
	require.Equal(t, 150.0, RemainingRefundable(order))
}

// End-to-end: RefundOrderForCancellation on the #527 shape refunds the remaining live
// ₹110 (wallet provider = no gateway), stamps cumulative refund_amount = 220, and the
// customer's wallet is actually credited ₹110 — the previously-stranded money.
func TestRefundOrderForCancellation_PerLineCancel_RefundsRemainingLive(t *testing.T) {
	db := setupCancelRefundDB(t)
	order := seedWalletOrder(t, db, models.PaymentCompleted, 110, 110) // post per-line-cancel shape
	seedCancelledItem(t, db, order.ID, 110)

	require.NoError(t, RefundOrderForCancellation(order, "customer", "customer cancelled remaining order"))

	status, refundAmount, refundID, refundedAt := loadRefund(t, db, order.ID)
	require.Equal(t, string(models.PaymentRefunded), status)
	require.Equal(t, 220.0, refundAmount, "cumulative refund = 110 per-line + 110 remaining = 220 (full paid)")
	require.NotEmpty(t, refundID)
	require.NotNil(t, refundedAt)
	require.Equal(t, 110.0, walletBalance(t, db, order.CustomerID), "the previously-stranded ₹110 is credited back")
}
