package services

// refund_reserve_test.go — #609. ReserveFullRefund gives the full-refund paths the SAME
// atomic discipline the partial path (RefundIssueToWallet) already uses: under a row lock,
// compute the remaining refundable and RESERVE it (claim payment_status/refunded_at AND
// increment refund_amount) in one transaction, so a full refund and a concurrent partial can
// never collectively over-refund. sqlite :memory: can't run true concurrent txns, so these
// pin the deterministic contract (cap vs a pre-existing refund_amount; single-winner claim;
// non-payable states; reservation release).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func seedReserveOrder(t *testing.T, db *gorm.DB, total, refundAmount float64, ps models.PaymentStatus) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		payment_provider, total, refund_amount) VALUES (?,?,?,?,?,?,?,?,?)`,
		id.String(), "ORD-R", uuid.NewString(), uuid.NewString(), "cancelled", string(ps), "razorpay",
		total, refundAmount).Error)
	return id
}

func reserveRow(t *testing.T, db *gorm.DB, id uuid.UUID) (ps string, refundAmount float64, refundedAtSet bool) {
	t.Helper()
	row := struct {
		PaymentStatus string
		RefundAmount  float64
		N             int64
	}{}
	require.NoError(t, db.Raw(`SELECT payment_status, refund_amount,
		(refunded_at IS NOT NULL) AS n FROM orders WHERE id = ?`, id.String()).Scan(&row).Error)
	return row.PaymentStatus, row.RefundAmount, row.N == 1
}

func TestReserveFullRefund_ReservesRemainingUnderLock(t *testing.T) {
	db := setupCancelRefundDB(t)
	// ₹400 already refunded on a ₹1000 order → the reservation is the remaining ₹600.
	id := seedReserveOrder(t, db, 1000, 400, models.PaymentCompleted)

	amount, won, err := ReserveFullRefund(db, id)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, 600.0, amount, "reserves only the remaining refundable")

	ps, ra, at := reserveRow(t, db, id)
	require.Equal(t, string(models.PaymentRefunded), ps, "claimed")
	require.True(t, at, "refunded_at stamped")
	require.Equal(t, 1000.0, ra, "refund_amount INCREMENTED by the reserved amount (400 + 600)")
}

func TestReserveFullRefund_SecondCallLoses(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 250, 0, models.PaymentCompleted)

	amount1, won1, err := ReserveFullRefund(db, id)
	require.NoError(t, err)
	require.True(t, won1)
	require.Equal(t, 250.0, amount1)

	// Second reservation (a racing full refund) must lose — the claim is single-winner.
	amount2, won2, err := ReserveFullRefund(db, id)
	require.NoError(t, err)
	require.False(t, won2, "a second full-refund claim loses")
	require.Equal(t, 0.0, amount2)

	_, ra, _ := reserveRow(t, db, id)
	require.Equal(t, 250.0, ra, "refund_amount reserved exactly once")
}

func TestReserveFullRefund_NotCompleted_NoReserve(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 250, 0, models.PaymentPending)

	amount, won, err := ReserveFullRefund(db, id)
	require.NoError(t, err)
	require.False(t, won)
	require.Equal(t, 0.0, amount)
	ps, ra, at := reserveRow(t, db, id)
	require.Equal(t, string(models.PaymentPending), ps, "untouched")
	require.Equal(t, 0.0, ra)
	require.False(t, at)
}

func TestReserveFullRefund_NothingLeft_NoReserve(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 250, 250, models.PaymentCompleted) // fully refunded already

	amount, won, err := ReserveFullRefund(db, id)
	require.NoError(t, err)
	require.False(t, won)
	require.Equal(t, 0.0, amount)
}

func TestReserveFullRefund_RefundedAtSet_NoReserve(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 250, 0, models.PaymentCompleted)
	require.NoError(t, db.Exec(`UPDATE orders SET refunded_at = CURRENT_TIMESTAMP WHERE id = ?`, id.String()).Error)

	amount, won, err := ReserveFullRefund(db, id)
	require.NoError(t, err)
	require.False(t, won, "a stamped refunded_at (sibling mid-flight) blocks the reservation")
	require.Equal(t, 0.0, amount)
}

func TestReleaseFullRefundReservation_RevertsAllThreeColumns(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 250, 0, models.PaymentCompleted)

	amount, won, err := ReserveFullRefund(db, id)
	require.NoError(t, err)
	require.True(t, won)

	ReleaseFullRefundReservation(db, id, amount)

	ps, ra, at := reserveRow(t, db, id)
	require.Equal(t, string(models.PaymentCompleted), ps, "payment_status reverted → completed (retry can re-refund)")
	require.False(t, at, "refunded_at cleared")
	require.Equal(t, 0.0, ra, "refund_amount decremented back")
}
