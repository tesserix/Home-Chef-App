package services

// order_payment_settle_test.go — #872 step 2, Task 1. Relocated from
// handlers/payment_complete_race_test.go (#395 item 2) and
// handlers/payment_complete_generic_test.go (#555), unchanged assertions, now
// exercising the moved CompleteOrderPaymentTx / CompleteRazorpayOrderTx
// directly in `services` (their new, single home).
//
// #395 item 2 background: the Razorpay verify path computed `wasUnpaid` from
// the in-memory order and then ran an UNCONDITIONAL completion update, so a
// payment.captured webhook winning the race (it IS conditional) left the
// concurrent verify still in the notify branch → duplicate chef "new order"
// push + duplicate order.paid event. CompleteRazorpayOrderTx makes the
// completion a single conditional transition (WHERE payment_status <>
// 'completed') and gates the chef notify + event on RowsAffected, mirroring
// the webhook, so exactly one fires.
//
// #555 background: verifyStripePayment and settleFullWalletOrder used to run
// an UNCONDITIONAL completion UPDATE + an UNCONDITIONAL order.paid emit, so a
// re-verify or a verify/webhook race double-emitted order.paid (and
// double-pushed the chef "new order") — the same defect #553 fixed for
// Razorpay. Both now route through the provider-generic
// CompleteOrderPaymentTx, whose guarded UPDATE fires the notify + event on
// exactly one transition. These tests exercise that shared core with the
// Stripe and wallet payloads — no live gateway needed.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// setupCompleteTxDB is a minimal in-memory schema for CompleteOrderPaymentTx /
// CompleteRazorpayOrderTx: just the `orders` columns the guarded UPDATE reads
// or writes, plus `outbox_events` (the transactional-outbox destination for
// the chef push + order.paid emit). `deleted_at` is required — GORM's
// soft-delete default scope adds `WHERE deleted_at IS NULL` to the guarded
// UPDATE.
func setupCompleteTxDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id TEXT PRIMARY KEY, order_number TEXT, payment_status TEXT DEFAULT 'pending',
		payment_method TEXT DEFAULT '', payment_provider TEXT DEFAULT 'razorpay',
		razorpay_payment_id TEXT DEFAULT '', wallet_applied REAL DEFAULT 0, total REAL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT,
		aggregate_type TEXT, aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
		next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`).Error)
	return db
}

// seedTxOrder inserts a minimal order row with the given payment_status.
func seedTxOrder(t *testing.T, db *gorm.DB, paymentStatus string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, payment_status, total, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id.String(), "HC-"+id.String()[:8], paymentStatus, 500.0, time.Now(), time.Now()).Error)
	return id
}

func loadTxOrder(t *testing.T, db *gorm.DB, id uuid.UUID) *models.Order {
	t.Helper()
	var o models.Order
	require.NoError(t, db.First(&o, "id = ?", id).Error)
	return &o
}

func stripeTxEvent(order *models.Order) map[string]interface{} {
	return map[string]interface{}{
		"order_id": order.ID.String(), "order_number": order.OrderNumber,
		"amount": 500.0, "method": "card", "provider": "stripe", "currency": "INR",
	}
}

func walletTxEvent(order *models.Order) map[string]interface{} {
	return map[string]interface{}{
		"order_id": order.ID.String(), "order_number": order.OrderNumber,
		"amount": order.Total, "method": "wallet", "provider": "wallet",
	}
}

// Stripe: the first completion flips the order + emits order.paid and the chef push once.
func TestCompleteOrderPaymentTx_Stripe_FirstCompletionEmitsOnce(t *testing.T) {
	db := setupCompleteTxDB(t)
	orderID := seedTxOrder(t, db, "pending")
	order := loadTxOrder(t, db, orderID)

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = CompleteOrderPaymentTx(tx, order, map[string]interface{}{"payment_method": "card"}, stripeTxEvent(order))
		return err
	}))
	require.True(t, ok)
	require.Equal(t, "completed", paymentStatusOfTx(t, db, orderID))
	require.Equal(t, int64(1), countOutboxTx(t, db, "orders.paid"), "one order.paid")
	require.Equal(t, int64(1), countOutboxTx(t, db, SubjectChefNewOrder), "one chef push")
}

// Stripe: a re-verify (or verify/webhook race) on an already-completed order is a no-op —
// the #555 fix. Previously this second call emitted a duplicate order.paid.
func TestCompleteOrderPaymentTx_Stripe_ReVerifyDoesNotDoubleEmit(t *testing.T) {
	db := setupCompleteTxDB(t)
	orderID := seedTxOrder(t, db, "pending")

	fresh := loadTxOrder(t, db, orderID)
	stale := loadTxOrder(t, db, orderID) // still pending in memory — the racing verify's snapshot

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := CompleteOrderPaymentTx(tx, fresh, map[string]interface{}{"payment_method": "card"}, stripeTxEvent(fresh))
		return err
	}))

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = CompleteOrderPaymentTx(tx, stale, map[string]interface{}{"payment_method": "card"}, stripeTxEvent(stale))
		return err
	}))
	require.False(t, ok, "second completion performs no transition")
	require.Equal(t, int64(1), countOutboxTx(t, db, "orders.paid"), "still exactly one order.paid")
	require.Equal(t, int64(1), countOutboxTx(t, db, SubjectChefNewOrder), "still exactly one chef push")
}

// #563: a REFUNDED order must never be re-completed — that would silently re-enable the
// chef payout on money already returned. The guard excludes refunded, so it's a no-op.
func TestCompleteOrderPaymentTx_RefundedOrderNotReCompleted(t *testing.T) {
	db := setupCompleteTxDB(t)
	orderID := seedTxOrder(t, db, "refunded")
	order := loadTxOrder(t, db, orderID)

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = CompleteOrderPaymentTx(tx, order, map[string]interface{}{"payment_method": "card"}, stripeTxEvent(order))
		return err
	}))
	require.False(t, ok, "a refunded order is not re-completed")
	require.Equal(t, string(models.PaymentRefunded), paymentStatusOfTx(t, db, orderID))
	require.Equal(t, int64(0), countOutboxTx(t, db, "orders.paid"), "no order.paid re-emit on a refunded order")
}

// #563: a FAILED order (a prior card decline) MUST still complete on retry — `failed` is
// intentionally NOT in the blocked set. Retry-after-decline is preserved.
func TestCompleteOrderPaymentTx_FailedOrderCompletesOnRetry(t *testing.T) {
	db := setupCompleteTxDB(t)
	orderID := seedTxOrder(t, db, "failed")
	order := loadTxOrder(t, db, orderID)

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = CompleteOrderPaymentTx(tx, order, map[string]interface{}{"payment_method": "card"}, stripeTxEvent(order))
		return err
	}))
	require.True(t, ok, "a failed order completes on retry")
	require.Equal(t, string(models.PaymentCompleted), paymentStatusOfTx(t, db, orderID))
	require.Equal(t, int64(1), countOutboxTx(t, db, "orders.paid"))
}

// Wallet (full store-credit order): a retried settle stamps the wallet columns once and
// never double-emits order.paid.
func TestCompleteOrderPaymentTx_Wallet_RetryDoesNotDoubleEmit(t *testing.T) {
	db := setupCompleteTxDB(t)
	orderID := seedTxOrder(t, db, "pending")

	fresh := loadTxOrder(t, db, orderID)
	stale := loadTxOrder(t, db, orderID)
	walletUpdates := map[string]interface{}{"payment_method": "wallet", "payment_provider": "wallet", "wallet_applied": 300.0}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := CompleteOrderPaymentTx(tx, fresh, walletUpdates, walletTxEvent(fresh))
		return err
	}))

	var ok bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		ok, err = CompleteOrderPaymentTx(tx, stale, walletUpdates, walletTxEvent(stale))
		return err
	}))
	require.False(t, ok)
	require.Equal(t, int64(1), countOutboxTx(t, db, "orders.paid"), "still exactly one order.paid")
	require.Equal(t, int64(1), countOutboxTx(t, db, SubjectChefNewOrder), "still exactly one chef push")
}

// First completion notifies the chef + emits order.paid exactly once.
func TestCompleteRazorpayOrderTx_FirstCompletionNotifiesOnce(t *testing.T) {
	db := setupCompleteTxDB(t)
	orderID := seedTxOrder(t, db, "pending")

	order := loadTxOrder(t, db, orderID) // loaded before the tx, mirroring the handler
	var justCompleted bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		ok, err := CompleteRazorpayOrderTx(tx, order, "card", "pay_x", 50000)
		justCompleted = ok
		return err
	}))

	require.True(t, justCompleted, "the pending→completed transition happened")
	require.Equal(t, "completed", paymentStatusOfTx(t, db, orderID))
	require.Equal(t, int64(1), countOutboxTx(t, db, SubjectChefNewOrder), "one chef new-order push")
	require.Equal(t, int64(1), countOutboxTx(t, db, "orders.paid"), "one order.paid event")
}

// A second completion (webhook/verify race, or re-verify) must NOT re-notify or
// re-emit — the conditional update finds nothing to flip.
func TestCompleteRazorpayOrderTx_SecondCompletionIsNoop(t *testing.T) {
	db := setupCompleteTxDB(t)
	orderID := seedTxOrder(t, db, "pending")

	// The verify path loads its order (still pending) — then a webhook completes the
	// order underneath it. `stale` keeps that pending in-memory snapshot, exactly the
	// state the old `wasUnpaid` check misread as "notify the chef".
	order1 := loadTxOrder(t, db, orderID)
	stale := loadTxOrder(t, db, orderID)

	// Webhook completes it first.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := CompleteRazorpayOrderTx(tx, order1, "card", "pay_x", 50000)
		return err
	}))

	// The racing verify path runs with its STALE pending order — must be a no-op.
	var justCompleted bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		ok, err := CompleteRazorpayOrderTx(tx, stale, "card", "pay_x", 50000)
		justCompleted = ok
		return err
	}))

	require.False(t, justCompleted, "second call performs no transition")
	require.Equal(t, int64(1), countOutboxTx(t, db, SubjectChefNewOrder), "still exactly one chef push")
	require.Equal(t, int64(1), countOutboxTx(t, db, "orders.paid"), "still exactly one order.paid event")
}

func countOutboxTx(t *testing.T, db *gorm.DB, subject string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM outbox_events WHERE subject = ?`, subject).Scan(&n).Error)
	return n
}

func paymentStatusOfTx(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}
