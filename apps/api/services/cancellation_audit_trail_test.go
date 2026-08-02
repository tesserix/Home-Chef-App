package services

// cancellation_audit_trail_test.go — the cancellation refund path must leave a
// complete audit trail behind it, not just correct money (#932, #940).
//
// Both gaps were found by cancelling a real order at each fulfilment stage against
// production: the money reconciled to the paise every time, but the resulting rows
// could not be tied back to when the order was cancelled or to the provider's
// refund.

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// #932: the arbitration path stamped refunded_at but never cancelled_at, leaving
// every order cancelled through it `cancelled` with no cancellation timestamp.
func TestExecuteCancellationRefund_StampsCancelledAt(t *testing.T) {
	db := setupSweepDB(t)
	cust, chef := uuid.New(), uuid.New()
	oid, reqID := uuid.New(), uuid.New()

	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, chef_id, status, payment_status, subtotal, total, refund_amount)
		VALUES (?,?,?,?,?,500,630,0)`, oid.String(), cust.String(), chef.String(), "preparing", "completed").Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?,?,?,?,?,0)`, reqID.String(), oid.String(), cust.String(), chef.String(), "approved", "wallet", 25200).Error)

	SweepCancellationRefunds()

	var status string
	db.Raw(`SELECT status FROM orders WHERE id = ?`, oid.String()).Scan(&status)
	require.Equal(t, "cancelled", status)

	var cancelledAt *string
	db.Raw(`SELECT cancelled_at FROM orders WHERE id = ?`, oid.String()).Scan(&cancelledAt)
	require.NotNil(t, cancelledAt, "a cancelled order must carry a cancelled_at timestamp")
	require.NotEmpty(t, *cancelledAt)
}

// An order cancelled twice must keep the FIRST cancellation timestamp — the
// COALESCE is what stops a re-drive from rewriting history.
func TestExecuteCancellationRefund_KeepsEarlierCancelledAt(t *testing.T) {
	db := setupSweepDB(t)
	cust, chef := uuid.New(), uuid.New()
	oid, reqID := uuid.New(), uuid.New()

	const earlier = "2020-01-02 03:04:05+00:00"
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, chef_id, status, payment_status, subtotal, total, refund_amount, cancelled_at)
		VALUES (?,?,?,?,?,500,630,0,?)`, oid.String(), cust.String(), chef.String(), "preparing", "completed", earlier).Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?,?,?,?,?,0)`, reqID.String(), oid.String(), cust.String(), chef.String(), "approved", "wallet", 25200).Error)

	SweepCancellationRefunds()

	// Compare the date only: sqlite/GORM normalise the stored literal on read-back
	// ("2020-01-02T03:04:05Z"), so an exact string match would assert the driver's
	// formatting rather than the COALESCE this test is about.
	var cancelledAt string
	db.Raw(`SELECT cancelled_at FROM orders WHERE id = ?`, oid.String()).Scan(&cancelledAt)
	require.True(t, strings.HasPrefix(cancelledAt, "2020-01-02"),
		"an existing cancelled_at must not be overwritten, got %q", cancelledAt)
}

// #940: when the shared refund claim is lost to a sibling path, this branch used to
// overwrite the only reference we hold with the literal string "already-refunded" —
// discarding the provider's real refund id. Observed live on order …14481790, whose
// ₹129.94 Cashfree refund had no id anywhere in the database afterwards.
func TestExecuteCancellationRefund_ClaimLost_PreservesGatewayRefundID(t *testing.T) {
	db := setupSweepDB(t)
	cust, chef := uuid.New(), uuid.New()
	oid, reqID := uuid.New(), uuid.New()

	// A sibling already refunded: payment_status is refunded and refunded_at is set,
	// so our conditional claim matches zero rows.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, chef_id, status, payment_status, subtotal, total, refund_amount, refunded_at, refund_id)
		VALUES (?,?,?,?,?,500,630,252,?,?)`,
		oid.String(), cust.String(), chef.String(), "cancelled", "refunded",
		"2026-08-02 14:52:00+00:00", "cfr_7bb02b1d133ea77a").Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?,?,?,?,?,0)`, reqID.String(), oid.String(), cust.String(), chef.String(), "approved", "wallet", 25200).Error)

	SweepCancellationRefunds()

	var executed bool
	db.Raw(`SELECT refund_executed FROM cancellation_requests WHERE id = ?`, reqID.String()).Scan(&executed)
	require.True(t, executed, "the sibling already covered the refund, so the request resolves")

	var ref string
	db.Raw(`SELECT refund_ref FROM cancellation_requests WHERE id = ?`, reqID.String()).Scan(&ref)
	require.Equal(t, "cfr_7bb02b1d133ea77a", ref,
		"the winning path's gateway refund id must be carried over, not replaced by a sentinel")
}

// With no id recorded anywhere there is nothing to carry over, so the sentinel is
// still the honest answer.
func TestExecuteCancellationRefund_ClaimLost_FallsBackToSentinel(t *testing.T) {
	db := setupSweepDB(t)
	cust, chef := uuid.New(), uuid.New()
	oid, reqID := uuid.New(), uuid.New()

	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, chef_id, status, payment_status, subtotal, total, refund_amount, refunded_at)
		VALUES (?,?,?,?,?,500,630,252,?)`,
		oid.String(), cust.String(), chef.String(), "cancelled", "refunded", "2026-08-02 14:52:00+00:00").Error)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, refund_total_paise, refund_executed)
		VALUES (?,?,?,?,?,?,?,0)`, reqID.String(), oid.String(), cust.String(), chef.String(), "approved", "wallet", 25200).Error)

	SweepCancellationRefunds()

	var ref string
	db.Raw(`SELECT refund_ref FROM cancellation_requests WHERE id = ?`, reqID.String()).Scan(&ref)
	require.Equal(t, "already-refunded", ref)
}
