package services

// temporal_order_test.go — the money-critical saga compensation (#122). The
// workflow tests stub the activity Funcs, so here we pin the real
// CompensateOrderRefund: it credits the wallet once, marks the order refunded,
// and a retry (the activity re-running after a crash) never double-credits.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// addOrdersTable adds a minimal orders table to a wallet test DB so the refund
// compensation (which loads + updates the order) is exercisable.
func addOrdersTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id text PRIMARY KEY, order_number text, customer_id text, total real,
		status text, payment_status text, refunded_at datetime, refund_amount real,
		refund_reason text, created_at datetime, updated_at datetime, deleted_at datetime
	)`).Error)
	// #544: CompensateOrderRefund now checks TypedRefundOrderKind, which Counts these by order_id.
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id text PRIMARY KEY, order_id text, status text, deleted_at datetime)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_orders (id text PRIMARY KEY, order_id text, status text, deleted_at datetime)`).Error)
}

// #544: the saga compensation must NOT generic-refund a typed escrow order (meal-plan day /
// group) — its typed flow owns the refund on a disjoint keyspace. Skipping here prevents a
// double credit (saga-refund:<orderID> alongside the typed RefundDay/participant credit).
func TestCompensateOrderRefund_SkipsTypedOrder(t *testing.T) {
	db := setupWalletDB(t)
	addOrdersTable(t, db)
	oid, cid := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, total, status, payment_status)
		VALUES (?,?,?,?,?,?)`, oid.String(), "ORD-MP", cid.String(), 500.0,
		string(models.OrderStatusAccepted), string(models.PaymentCompleted)).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), oid.String(), "active").Error)

	require.NoError(t, CompensateOrderRefund(context.Background(), oid, "chef rejected"))

	// No wallet credit and the order is untouched — the typed flow owns this refund.
	w, _ := WalletBalance(db, cid)
	require.Equal(t, 0.0, w.Balance, "no generic wallet credit on a typed order")
	var o models.Order
	require.NoError(t, db.First(&o, "id = ?", oid).Error)
	require.Equal(t, models.PaymentCompleted, o.PaymentStatus, "payment_status stays completed — no generic refund claim")
	require.Equal(t, 0.0, o.RefundAmount)
}

func TestCompensateOrderRefund_CreditsWalletAndMarksRefunded(t *testing.T) {
	db := setupWalletDB(t) // creates wallets + wallet_txns, sets database.DB
	addOrdersTable(t, db)
	oid, cid := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, total, status, payment_status)
		VALUES (?,?,?,?,?,?)`, oid.String(), "ORD-1", cid.String(), 500.0,
		string(models.OrderStatusAccepted), string(models.PaymentCompleted)).Error)

	require.NoError(t, CompensateOrderRefund(context.Background(), oid, "chef rejected"))

	// Wallet credited once.
	w, _ := WalletBalance(db, cid)
	require.Equal(t, 500.0, w.Balance)
	// Order marked refunded.
	var o models.Order
	require.NoError(t, db.First(&o, "id = ?", oid).Error)
	require.Equal(t, models.OrderStatusRefunded, o.Status)
	require.NotNil(t, o.RefundedAt)
	require.Equal(t, 500.0, o.RefundAmount)
}

func TestCompensateOrderRefund_IdempotentOnRetry(t *testing.T) {
	db := setupWalletDB(t)
	addOrdersTable(t, db)
	oid, cid := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, total, status, payment_status)
		VALUES (?,?,?,?,?,?)`, oid.String(), "ORD-2", cid.String(), 300.0,
		string(models.OrderStatusAccepted), string(models.PaymentCompleted)).Error)

	// Run it three times (an activity retried after a crash).
	for i := 0; i < 3; i++ {
		require.NoError(t, CompensateOrderRefund(context.Background(), oid, "timeout"))
	}

	// Exactly one wallet credit — no double refund.
	var n int64
	db.Model(&models.WalletTxn{}).Where("user_id = ? AND source = ?", cid, models.WalletSourceRefund).Count(&n)
	require.Equal(t, int64(1), n)
	w, _ := WalletBalance(db, cid)
	require.Equal(t, 300.0, w.Balance)
}

// #609: CompensateOrderRefund must refund only the REMAINING refundable, not the full
// order.Total, when a prior partial refund already returned part of the money — else it
// double-refunds the partial amount. It must also INCREMENT refund_amount, not overwrite it.
func TestCompensateOrderRefund_PriorPartial_RefundsRemainderOnly(t *testing.T) {
	db := setupWalletDB(t)
	addOrdersTable(t, db)
	oid, cid := uuid.New(), uuid.New()
	// Total ₹1000, ₹400 already refunded via a partial (refunded_at NULL per #549/#586).
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, total, status, payment_status, refund_amount)
		VALUES (?,?,?,?,?,?,?)`, oid.String(), "ORD-P", cid.String(), 1000.0,
		string(models.OrderStatusAccepted), string(models.PaymentCompleted), 400.0).Error)

	require.NoError(t, CompensateOrderRefund(context.Background(), oid, "chef rejected"))

	w, _ := WalletBalance(db, cid)
	require.Equal(t, 600.0, w.Balance, "only the remaining ₹600 is refunded, not the full ₹1000")
	var o models.Order
	require.NoError(t, db.First(&o, "id = ?", oid).Error)
	require.Equal(t, 1000.0, o.RefundAmount, "refund_amount is the cumulative 400 + 600, not an overwrite")
	require.Equal(t, models.OrderStatusRefunded, o.Status)
}

func TestCompensateOrderRefund_SkipsAlreadyRefunded(t *testing.T) {
	db := setupWalletDB(t)
	addOrdersTable(t, db)
	oid, cid := uuid.New(), uuid.New()
	// Pre-refunded order → compensation must be a no-op (no wallet credit).
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, total, status, payment_status, refunded_at)
		VALUES (?,?,?,?,?,?, CURRENT_TIMESTAMP)`, oid.String(), "ORD-3", cid.String(), 200.0,
		string(models.OrderStatusRefunded), string(models.PaymentRefunded)).Error)

	require.NoError(t, CompensateOrderRefund(context.Background(), oid, "dup"))

	w, _ := WalletBalance(db, cid)
	require.Equal(t, 0.0, w.Balance) // nothing credited
}
