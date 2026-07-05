package handlers

// payment_complete_race_test.go — #395 item 2. The Razorpay verify path computed
// `wasUnpaid` from the in-memory order and then ran an UNCONDITIONAL completion
// update, so a payment.captured webhook winning the race (it IS conditional) left the
// concurrent verify still in the notify branch → duplicate chef "new order" push +
// duplicate order.paid event. completeRazorpayOrderTx makes the completion a single
// conditional transition (WHERE payment_status <> 'completed') and gates the chef
// notify + event on RowsAffected, mirroring the webhook, so exactly one fires.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func countOutbox(t *testing.T, db *gorm.DB, subject string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM outbox_events WHERE subject = ?`, subject).Scan(&n).Error)
	return n
}

func paymentStatusOf(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}

func loadPayOrder(t *testing.T, db *gorm.DB, id uuid.UUID) *models.Order {
	t.Helper()
	var o models.Order
	require.NoError(t, db.First(&o, "id = ?", id).Error)
	return &o
}

// First completion notifies the chef + emits order.paid exactly once.
func TestCompleteRazorpayOrderTx_FirstCompletionNotifiesOnce(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "rzp_order_x", "")

	order := loadPayOrder(t, db, orderID) // loaded before the tx, mirroring the handler
	var justCompleted bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		ok, err := completeRazorpayOrderTx(tx, order, "card", "pay_x", 50000)
		justCompleted = ok
		return err
	}))

	require.True(t, justCompleted, "the pending→completed transition happened")
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder), "one chef new-order push")
	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"), "one order.paid event")
}

// A second completion (webhook/verify race, or re-verify) must NOT re-notify or
// re-emit — the conditional update finds nothing to flip.
func TestCompleteRazorpayOrderTx_SecondCompletionIsNoop(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "rzp_order_x", "")

	// The verify path loads its order (still pending) — then a webhook completes the
	// order underneath it. `stale` keeps that pending in-memory snapshot, exactly the
	// state the old `wasUnpaid` check misread as "notify the chef".
	order1 := loadPayOrder(t, db, orderID)
	stale := loadPayOrder(t, db, orderID)

	// Webhook completes it first.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := completeRazorpayOrderTx(tx, order1, "card", "pay_x", 50000)
		return err
	}))

	// The racing verify path runs with its STALE pending order — must be a no-op.
	var justCompleted bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		ok, err := completeRazorpayOrderTx(tx, stale, "card", "pay_x", 50000)
		justCompleted = ok
		return err
	}))

	require.False(t, justCompleted, "second call performs no transition")
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder), "still exactly one chef push")
	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"), "still exactly one order.paid event")
}
