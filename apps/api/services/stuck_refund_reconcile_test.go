package services

// stuck_refund_reconcile_test.go — #602. reconcileStuckRefunds finalizes an order left mid-
// refund by a persist failure (payment_status=refunded AND refunded_at IS NULL). The ledger
// (refund_amount) is already correct and MUST be left untouched; the reconcile only recovers the
// terminal write — revert-to-completed for a partial, refunded_at+status for a full. Escrow
// flags are OFF in tests, so the payout cross-guard is a state-only no-op on a held-none order.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupStuckRefundDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT, chef_id TEXT,
			status TEXT, payment_status TEXT, payment_provider TEXT DEFAULT 'razorpay', total REAL DEFAULT 0,
			refund_amount REAL DEFAULT 0, refund_id TEXT DEFAULT '', refund_reason TEXT, refund_initiated_by TEXT,
			refunded_at DATETIME, payout_hold_status TEXT DEFAULT '', razorpay_order_id TEXT DEFAULT '',
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE order_items (id TEXT PRIMARY KEY, order_id TEXT, is_cancelled BOOLEAN DEFAULT 0, refund_amount REAL DEFAULT 0)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, order_id TEXT)`,
		`CREATE TABLE group_orders (id TEXT PRIMARY KEY, order_id TEXT)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedStuckOrder(t *testing.T, db *gorm.DB, total, refundAmount float64, updatedAt time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		total, refund_amount, refunded_at, updated_at) VALUES (?,?,?,?,?,?,?,?,NULL,?)`,
		id.String(), "ORD-S", uuid.NewString(), uuid.NewString(), "delivered", string(models.PaymentRefunded),
		total, refundAmount, updatedAt).Error)
	return id
}

func stuckRow(t *testing.T, db *gorm.DB, id uuid.UUID) (ps, status string, refundAmount float64, refundedAt bool) {
	t.Helper()
	var r struct {
		PaymentStatus string
		Status        string
		RefundAmount  float64
		N             int64
	}
	require.NoError(t, db.Raw(`SELECT payment_status, status, refund_amount, (refunded_at IS NOT NULL) AS n
		FROM orders WHERE id = ?`, id.String()).Scan(&r).Error)
	return r.PaymentStatus, r.Status, r.RefundAmount, r.N == 1
}

// A stuck PARTIAL (refund_amount < total) finalizes by reverting to completed — refund_amount
// left CORRECT (never decremented), so the chef's hold stays releasable for the remainder.
func TestReconcileStuckRefunds_PartialFinalizesToCompleted(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedStuckOrder(t, db, 500, 100, time.Now().Add(-time.Hour))

	require.Equal(t, 1, reconcileStuckRefunds())

	ps, _, ra, at := stuckRow(t, db, id)
	require.Equal(t, string(models.PaymentCompleted), ps, "a stuck PARTIAL reverts to completed")
	require.Equal(t, 100.0, ra, "refund_amount left CORRECT — never decremented")
	require.False(t, at, "a partial does not stamp refunded_at")
}

// A stuck FULL (refund_amount == total) finalizes by stamping refunded_at + status=refunded.
func TestReconcileStuckRefunds_FullFinalizesTerminal(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedStuckOrder(t, db, 500, 500, time.Now().Add(-time.Hour))

	require.Equal(t, 1, reconcileStuckRefunds())

	ps, status, ra, at := stuckRow(t, db, id)
	require.Equal(t, string(models.PaymentRefunded), ps, "a full stays refunded")
	require.Equal(t, string(models.OrderStatusRefunded), status)
	require.Equal(t, 500.0, ra)
	require.True(t, at, "a full stamps refunded_at")
}

// A fresh (not-yet-stale) mid-refund is NOT touched — the grace prevents racing an in-flight
// refund whose reserve→persist window transiently shows the same signal.
func TestReconcileStuckRefunds_RespectsGrace(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedStuckOrder(t, db, 500, 100, time.Now()) // updated just now

	require.Equal(t, 0, reconcileStuckRefunds())
	ps, _, _, _ := stuckRow(t, db, id)
	require.Equal(t, string(models.PaymentRefunded), ps, "an in-flight refund inside the grace window is left alone")
}

// A legit fully-refunded order (refunded_at already set) is never matched — it's not stuck.
func TestReconcileStuckRefunds_SkipsAlreadyFinalized(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedStuckOrder(t, db, 500, 500, time.Now().Add(-time.Hour))
	require.NoError(t, db.Exec(`UPDATE orders SET refunded_at = CURRENT_TIMESTAMP WHERE id = ?`, id.String()).Error)

	require.Equal(t, 0, reconcileStuckRefunds(), "refunded_at set → not a stuck mid-refund")
}

// A typed-escrow shell (meal-plan-day / group) is skipped — those refund via their own flow.
func TestReconcileStuckRefunds_SkipsTypedShell(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedStuckOrder(t, db, 500, 100, time.Now().Add(-time.Hour))
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id) VALUES (?,?)`, uuid.NewString(), id.String()).Error)

	require.Equal(t, 0, reconcileStuckRefunds(), "typed-escrow shells are never finalized here")
	ps, _, _, _ := stuckRow(t, db, id)
	require.Equal(t, string(models.PaymentRefunded), ps, "left untouched")
}

// The per-line refund total is added back when deciding full-vs-partial (mirrors
// RemainingRefundable): an order whose Total was reduced by a per-line cancel must not be
// mis-finalized as FULL just because refund_amount reached the reduced Total.
func TestReconcileStuckRefunds_PerLineAddBack_StaysPartial(t *testing.T) {
	db := setupStuckRefundDB(t)
	// Total 300 (already reduced from 500 by a 200 per-line cancel); refund_amount 300.
	id := seedStuckOrder(t, db, 300, 300, time.Now().Add(-time.Hour))
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		uuid.NewString(), id.String(), true, 200.0).Error)
	// remaining = 300 − 300 + 200 = 200 > 0 → PARTIAL (without the add-back it would look FULL).
	require.Equal(t, 1, reconcileStuckRefunds())
	ps, _, _, at := stuckRow(t, db, id)
	require.Equal(t, string(models.PaymentCompleted), ps, "per-line add-back keeps it PARTIAL → reverts to completed")
	require.False(t, at, "not mis-stamped as full")
}

// finalizeStuckRefund is single-winner: a second call on an already-finalized row is a no-op
// (the WHERE guard is the serialization point — a concurrent refund/reconcile can't double it).
func TestFinalizeStuckRefund_SecondCallNoOp(t *testing.T) {
	db := setupStuckRefundDB(t)
	id := seedStuckOrder(t, db, 500, 100, time.Now().Add(-time.Hour))

	full1, healed1, paise1 := finalizeStuckRefund(id)
	require.True(t, healed1)
	require.False(t, full1)
	require.Equal(t, 10000, paise1, "returns the cumulative refunded amount read UNDER THE LOCK (₹100 → 10000 paise) for the chef claw")

	_, healed2, _ := finalizeStuckRefund(id) // already reverted to completed → not stuck
	require.False(t, healed2, "a second finalize is a no-op")
}
