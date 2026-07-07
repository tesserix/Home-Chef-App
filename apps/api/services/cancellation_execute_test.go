package services

// cancellation_execute_test.go — #636. ExecuteCancellationRefund persisted refund_amount via the
// caller's STALE in-memory `order.RefundAmount + refund`. Its payment_status claim excludes the
// ReserveRefund family, but NOT RefundIssueToWallet (which never flips payment_status) — so a
// partial issue refund that atomically incremented refund_amount between the caller's read and
// this write was clobbered. The fix increments in-SQL (COALESCE(refund_amount,0) + refund).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// addCancellationRequestsTable + a seeded wallet let setupCancelRefundDB drive
// ExecuteCancellationRefund's wallet-destination path.
func seedCancelExecFixtures(t *testing.T, db *gorm.DB, o *models.Order, refundTotalPaise int) *models.CancellationRequest {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS cancellation_requests (
		id TEXT PRIMARY KEY, order_id TEXT, customer_id TEXT, refund_destination TEXT DEFAULT 'wallet',
		refund_total_paise INTEGER DEFAULT 0, refund_executed BOOLEAN DEFAULT 0, refund_ref TEXT DEFAULT '',
		resolved_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance) VALUES (?,?,0)`,
		uuid.NewString(), o.CustomerID.String()).Error)
	crID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?, 'wallet', ?, 0)`, crID.String(), o.ID.String(), o.CustomerID.String(), refundTotalPaise).Error)
	return &models.CancellationRequest{
		ID: crID, OrderID: o.ID, CustomerID: o.CustomerID,
		RefundTotalPaise: refundTotalPaise, RefundDestination: "wallet",
	}
}

// The clobber: a concurrent partial issue refund committed refund_amount=100 (in the DB) AFTER the
// caller loaded `order` (so the caller's snapshot still reads 0). ExecuteCancellationRefund must
// increment the CURRENT value, not overwrite it with stale+refund.
func TestExecuteCancellationRefund_AtomicIncrement_NoClobber(t *testing.T) {
	db := setupCancelRefundDB(t)
	// DB reflects the concurrent issue refund: refund_amount=100, still completed / not refunded.
	o := seedWalletOrder(t, db, models.PaymentCompleted, 1000, 100)
	cr := seedCancelExecFixtures(t, db, o, 50000) // cancellation refunds ₹500

	// The caller's STALE snapshot — loaded before the issue refund committed → RefundAmount 0.
	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, PaymentProvider: "wallet", RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr))

	_, refundAmount, _, _ := loadRefund(t, db, o.ID)
	require.InDelta(t, 600, refundAmount, 0.01,
		"#636: refund_amount = 100 (concurrent issue) + 500 (cancellation), not the stale 0 + 500")
}

// Happy path (no prior refund): refund_amount = the cancellation amount.
func TestExecuteCancellationRefund_NoPriorRefund(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 1000, 0)
	cr := seedCancelExecFixtures(t, db, o, 50000)

	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, PaymentProvider: "wallet", RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr))

	status, refundAmount, _, refundedAt := loadRefund(t, db, o.ID)
	require.InDelta(t, 500, refundAmount, 0.01)
	require.Equal(t, string(models.PaymentRefunded), status)
	require.NotNil(t, refundedAt)
}

// Idempotent: a second run (sweep retry) loses the payment_status claim and must NOT increment
// refund_amount again.
func TestExecuteCancellationRefund_Idempotent_NoDoubleIncrement(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 1000, 0)
	cr := seedCancelExecFixtures(t, db, o, 50000)

	stale := &models.Order{ID: o.ID, CustomerID: o.CustomerID, PaymentProvider: "wallet", RefundAmount: 0}
	require.NoError(t, ExecuteCancellationRefund(stale, cr))
	_, after1, _, _ := loadRefund(t, db, o.ID)

	// Re-run with a fresh CR struct (the sweep re-drives) — claim is lost (already refunded).
	cr2 := &models.CancellationRequest{ID: cr.ID, OrderID: o.ID, CustomerID: o.CustomerID,
		RefundTotalPaise: 50000, RefundDestination: "wallet"}
	require.NoError(t, ExecuteCancellationRefund(stale, cr2))
	_, after2, _, _ := loadRefund(t, db, o.ID)

	require.InDelta(t, after1, after2, 0.001, "a retry after the claim is lost must not double-increment refund_amount")
	require.InDelta(t, 500, after2, 0.01)
}
