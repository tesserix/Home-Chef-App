package services

// stuck_order_cron_test.go — an order the chef ACCEPTED and abandoned. The nudges are
// cosmetic; the refund is money, so the boundaries are what these pin.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedAbandonedOrder inserts a PAID order sitting in an in-flight status since `created`.
func seedAbandonedOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, status models.OrderStatus, created time.Time) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-STUCK", CustomerID: uuid.New(), ChefID: chefID,
		Status: status, PaymentStatus: models.PaymentCompleted,
		PaymentProvider: "razorpay", RazorpayPaymentID: "pay_stuck", Total: 250,
	}
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, payment_provider,
		 razorpay_payment_id, total, refund_amount, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,0,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), chefID.String(),
		string(status), string(models.PaymentCompleted), "razorpay",
		"pay_stuck", 250.0, created, created).Error)
	return o
}

func staleReminderCount(t *testing.T, db *gorm.DB, id uuid.UUID) int {
	t.Helper()
	var n int
	require.NoError(t, db.Raw(`SELECT COALESCE(stale_reminder_count,0) FROM orders WHERE id = ?`, id.String()).Scan(&n).Error)
	return n
}

// THE hole this closes: paid, accepted, then abandoned. Before this the customer had no
// way out and no money back.
func TestStuckSweep_RefundsAnAbandonedOrderInFull(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	now := ist(2026, 7, 20, 12, 0)
	o := seedAbandonedOrder(t, db, chefID, models.OrderStatusPreparing, now.AddDate(0, 0, -31))

	_, refunded := sweepStuckOrders(context.Background(), db, now)
	require.Equal(t, 1, refunded)

	status, payment, refund := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusCancelled), status)
	require.Equal(t, string(models.PaymentRefunded), payment)
	require.Equal(t, 250.0, refund, "nothing was delivered, so the customer is made whole")
}

// Inside the refund deadline the order is nudged, never refunded — the chef may still finish it.
func TestStuckSweep_NudgesBeforeTheDeadline(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	now := ist(2026, 7, 20, 12, 0)
	o := seedAbandonedOrder(t, db, chefID, models.OrderStatusPreparing, now.AddDate(0, 0, -5))

	nudged, refunded := sweepStuckOrders(context.Background(), db, now)
	require.Equal(t, 1, nudged)
	require.Equal(t, 0, refunded, "five days in, the chef can still cook this")
	require.Equal(t, 1, staleReminderCount(t, db, o.ID))

	status, _, _ := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPreparing), status)
}

// A fresh order is neither nudged nor refunded — being mid-cook is not being stuck.
func TestStuckSweep_LeavesFreshOrdersAlone(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	now := ist(2026, 7, 20, 12, 0)
	o := seedAbandonedOrder(t, db, chefID, models.OrderStatusPreparing, now.Add(-2*time.Hour))

	nudged, refunded := sweepStuckOrders(context.Background(), db, now)
	require.Equal(t, 0, nudged)
	require.Equal(t, 0, refunded)
	require.Equal(t, 0, staleReminderCount(t, db, o.ID))
}

// A delivered order is finished, however old — the sweep must never reopen completed work.
func TestStuckSweep_IgnoresDeliveredOrders(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	now := ist(2026, 7, 20, 12, 0)
	o := seedAbandonedOrder(t, db, chefID, models.OrderStatusDelivered, now.AddDate(0, 0, -60))

	_, refunded := sweepStuckOrders(context.Background(), db, now)
	require.Equal(t, 0, refunded)

	status, payment, _ := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusDelivered), status)
	require.Equal(t, string(models.PaymentCompleted), payment, "a delivered order keeps its money")
}

// A pending order belongs to the unaccepted-order sweep, which prices it off the chef's
// closing time. Two sweeps refunding one order is the double-refund this must not cause.
func TestStuckSweep_LeavesPendingOrdersToTheVoidSweep(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	now := ist(2026, 7, 20, 12, 0)
	o := seedAbandonedOrder(t, db, chefID, models.OrderStatusPending, now.AddDate(0, 0, -31))

	_, refunded := sweepStuckOrders(context.Background(), db, now)
	require.Equal(t, 0, refunded)

	status, _, _ := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status)
}

// Reminders are rate-limited: a second pass the same day must not re-nudge.
func TestStuckSweep_DoesNotRenudgeWithinTheWindow(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	now := ist(2026, 7, 20, 12, 0)
	o := seedAbandonedOrder(t, db, chefID, models.OrderStatusPreparing, now.AddDate(0, 0, -5))

	require.NoError(t, db.Exec(`UPDATE orders SET stale_reminder_count = 1, last_stale_reminder_at = ? WHERE id = ?`,
		now.Add(-1*time.Hour), o.ID.String()).Error)

	nudged, _ := sweepStuckOrders(context.Background(), db, now)
	require.Equal(t, 0, nudged)
	require.Equal(t, 1, staleReminderCount(t, db, o.ID), "one nudge per window, not one per pass")
}
