package services

// refund_reserve_partial_test.go — #611. ReserveRefund is the partial-aware sibling of
// ReserveFullRefund (#609): it reserves an ARBITRARY requested amount (0 = full remaining)
// under a row lock, so InitiateRefund / RefundOrder no longer read `remaining` +
// `order.RefundAmount` UNLOCKED before their claim (a concurrent partial committing in the
// gap → over-refund of live customer money). sqlite :memory: can't run true concurrent
// txns, so these pin the deterministic contract (cap vs a pre-existing refund_amount;
// single-winner claim; partial leaves refunded_at NULL; full vs partial classification;
// reservation release).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// A partial reservation reserves EXACTLY the requested amount, increments refund_amount
// atomically, flips payment_status → refunded as the mutex, but must NOT stamp refunded_at
// (only a FULL refund is terminal — the caller owns that marker).
func TestReserveRefund_PartialReservesRequested_NotTerminal(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 1000, 0, models.PaymentCompleted)

	amount, prior, full, won, err := ReserveRefund(db, id, 300)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, 300.0, amount, "reserves exactly the requested partial amount")
	require.Equal(t, 0.0, prior, "prior refunded read under the lock")
	require.False(t, full, "300 of 1000 is a PARTIAL refund")

	ps, ra, at := reserveRow(t, db, id)
	require.Equal(t, string(models.PaymentRefunded), ps, "payment_status claimed as the mutex")
	require.Equal(t, 300.0, ra, "refund_amount incremented by the reserved amount")
	require.False(t, at, "a PARTIAL reservation must NOT stamp refunded_at")
}

// A requested amount of 0 means full: it reserves the whole remaining and is terminal.
func TestReserveRefund_ZeroRequested_IsFull(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 500, 0, models.PaymentCompleted)

	amount, _, full, won, err := ReserveRefund(db, id, 0)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, 500.0, amount, "0 requested = full remaining")
	require.True(t, full, "exhausting the remaining is a FULL refund")
}

// The reservation caps at what's STILL refundable: with ₹400 already refunded on a ₹1000
// order, a request for the full ₹1000 reserves only the remaining ₹600 — never over-refunds.
func TestReserveRefund_CapsAtRemaining(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 1000, 400, models.PaymentCompleted)

	amount, prior, full, won, err := ReserveRefund(db, id, 1000)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, 600.0, amount, "capped at remaining (1000 − 400)")
	require.Equal(t, 400.0, prior)
	require.True(t, full, "a request that exhausts the remainder is FULL, even if it was < the original request")

	_, ra, at := reserveRow(t, db, id)
	require.Equal(t, 1000.0, ra, "refund_amount reaches the total, never past it")
	require.False(t, at, "ReserveRefund never stamps refunded_at — the caller does on FULL")
}

// A partial-intent request that a concurrent refund has shrunk the remainder below is
// reserved at the (smaller) remaining and reclassified as FULL — it exhausts the order, so
// the caller must stamp the terminal marker.
func TestReserveRefund_PartialExhaustingRemainder_BecomesFull(t *testing.T) {
	db := setupCancelRefundDB(t)
	// A concurrent partial already took 950 of 1000 → only 50 left.
	id := seedReserveOrder(t, db, 1000, 950, models.PaymentCompleted)

	amount, _, full, won, err := ReserveRefund(db, id, 200) // asked 200, only 50 left
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, 50.0, amount, "capped to the 50 that remained")
	require.True(t, full, "reserving the last 50 exhausts the order → FULL")
}

// Two back-to-back reservations (a racing double-submit): the second loses the claim —
// ReserveRefund leaves payment_status=refunded, so the second's guarded UPDATE matches nothing.
func TestReserveRefund_SecondCallLoses(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 500, 0, models.PaymentCompleted)

	a1, _, _, won1, err := ReserveRefund(db, id, 200)
	require.NoError(t, err)
	require.True(t, won1)
	require.Equal(t, 200.0, a1)

	a2, _, _, won2, err := ReserveRefund(db, id, 200)
	require.NoError(t, err)
	require.False(t, won2, "a second reservation loses while the first still holds the payment_status claim")
	require.Equal(t, 0.0, a2)

	_, ra, _ := reserveRow(t, db, id)
	require.Equal(t, 200.0, ra, "refund_amount reserved exactly once")
}

func TestReserveRefund_NotCompleted_NoReserve(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 500, 0, models.PaymentPending)

	amount, _, _, won, err := ReserveRefund(db, id, 100)
	require.NoError(t, err)
	require.False(t, won)
	require.Equal(t, 0.0, amount)
	ps, ra, at := reserveRow(t, db, id)
	require.Equal(t, string(models.PaymentPending), ps, "untouched")
	require.Equal(t, 0.0, ra)
	require.False(t, at)
}

func TestReserveRefund_NothingLeft_NoReserve(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 250, 250, models.PaymentCompleted) // already fully refunded

	amount, _, _, won, err := ReserveRefund(db, id, 100)
	require.NoError(t, err)
	require.False(t, won)
	require.Equal(t, 0.0, amount)
}

// After a partial reservation the caller reverts payment_status → completed (so the hold
// stays releasable + sequential partials work). A follow-up reservation then wins on the
// smaller remainder — proving the reservation is re-entrant across the caller's revert.
func TestReserveRefund_AfterCallerRevert_SecondPartialReservesRemainder(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 1000, 0, models.PaymentCompleted)

	_, _, _, won1, err := ReserveRefund(db, id, 300)
	require.NoError(t, err)
	require.True(t, won1)
	// Caller reverts the mutex for a PARTIAL (mirrors the handler).
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = ? WHERE id = ?`,
		string(models.PaymentCompleted), id.String()).Error)

	a2, prior2, _, won2, err := ReserveRefund(db, id, 400)
	require.NoError(t, err)
	require.True(t, won2, "a second partial reserves on the reduced remainder")
	require.Equal(t, 400.0, a2)
	require.Equal(t, 300.0, prior2, "prior refunded reflects the first partial")

	_, ra, _ := reserveRow(t, db, id)
	require.Equal(t, 700.0, ra, "300 + 400 accumulated, never over the total")
}

// ReleaseRefundReservation undoes a reservation on gateway/wallet failure: payment_status →
// completed and refund_amount decremented back, so a retry can re-reserve.
func TestReleaseRefundReservation_RevertsClaimAndDecrements(t *testing.T) {
	db := setupCancelRefundDB(t)
	id := seedReserveOrder(t, db, 500, 100, models.PaymentCompleted)

	amount, _, _, won, err := ReserveRefund(db, id, 250)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, 250.0, amount)

	ReleaseRefundReservation(db, id, amount)

	ps, ra, at := reserveRow(t, db, id)
	require.Equal(t, string(models.PaymentCompleted), ps, "payment_status reverted → completed")
	require.Equal(t, 100.0, ra, "refund_amount decremented back to the prior 100")
	require.False(t, at, "refunded_at untouched (never stamped by ReserveRefund)")
}

// The per-line refund total is added back under the lock (mirrors RemainingRefundable /
// ReserveFullRefund): a per-line cancel that reduced Total must not strand the remaining
// live items' money.
func TestReserveRefund_AddsBackPerLineRefunds(t *testing.T) {
	db := setupCancelRefundDB(t)
	// Total already reduced to 700 by a 300 per-line cancel; refund_amount recorded that 300.
	id := seedReserveOrder(t, db, 700, 300, models.PaymentCompleted)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount)
		VALUES (?,?,?,?)`, uuid.NewString(), id.String(), true, 300.0).Error)

	// remaining = 700 − 300 + 300 = 700 still owed on the live items.
	amount, _, full, won, err := ReserveRefund(db, id, 0)
	require.NoError(t, err)
	require.True(t, won)
	require.Equal(t, 700.0, amount, "per-line refund added back → full remaining is the live 700, not 400")
	require.True(t, full)
}
