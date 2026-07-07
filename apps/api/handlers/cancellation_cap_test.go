package handlers

// cancellation_cap_test.go — #642. snapshotFor must cap the tiered cancellation refund at the
// order's still-refundable balance (Total − already-refunded), so a cancellation AFTER a prior
// partial refund (e.g. a customer-issue RefundIssueToWallet) can't return more than what's owed.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestSnapshotFor_CapsAgainstPriorRefund(t *testing.T) {
	db := setupPayDB(t)
	oid := uuid.New()
	// ₹1100 order, but ₹600 already refunded via a prior partial issue refund → ₹500 remaining.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, total, refund_amount, status) VALUES (?, 1100, 600, 'accepted')`,
		oid.String()).Error)
	order := &models.Order{
		ID: oid, Subtotal: 800, DeliveryFee: 100, ServiceFee: 100, Tax: 100, Total: 1100,
		RefundAmount: 600, Status: models.OrderStatusAccepted,
	}

	// not_started (90% food) on the original amounts would refund > ₹500; snapshotFor caps it.
	s := snapshotFor(order, 90)
	require.Equal(t, 50000, s.Total, "#642: capped at the ₹500 (50000 paise) remaining balance")
	require.Equal(t, s.Total, s.FoodRefund+s.DeliveryRefund+s.TaxRefund, "breakdown sums to the capped total")

	// Sanity: with no prior refund the same order is NOT capped (refund < Total).
	order.RefundAmount = 0
	require.NoError(t, db.Exec(`UPDATE orders SET refund_amount = 0 WHERE id = ?`, oid.String()).Error)
	full := snapshotFor(order, 90)
	require.Greater(t, full.Total, 50000, "without a prior refund the tier refund is not capped")
}
