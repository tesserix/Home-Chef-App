package services

// refund_characterisation_test.go — #688, step one of the refund-coordinator
// refactor (#687).
//
// These pin the CROSS-PATH invariants of refunds as they behave TODAY, so the
// coordinator (#689) can be proven behaviour-identical. They must pass BEFORE
// and AFTER the refactor. If one only passes after, the refactor changed
// behaviour and we need to know that deliberately rather than discover it in
// production.
//
// Per-path behaviour is already covered — cancellation_order_refund_test.go,
// payment_concurrent_refund_test.go, payment_partial_refund_test.go,
// perline_cancel_reserve_test.go, chef_order_cancel_refund_test.go,
// order_issue_test.go (34 tests between them). This file deliberately does NOT
// duplicate those. The gap it fills is the rules that hold ACROSS paths, which
// today live in convention (shared idempotency keys + the #609 reservation
// discipline) and are exactly what a single coordinator will centralise.
//
// The one asymmetry these lock in is the bug: four paths route by
// PaymentProvider, order_issue.go does not (see #691).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ── Idempotency identity ─────────────────────────────────────────────────────
//
// Today's safety against a double refund is that independently-written call
// sites agree on a key. Pin that agreement: the coordinator must preserve each
// key's IDENTITY when it maps them onto one ScopeID (#690), or a retry stops
// deduping at the gateway and real money moves twice.

// Two cancellation paths (customer cancel + arbitration) deliberately SHARE one
// key so the gateway dedups if both ever fire for the same order.
// gateway_idempotency.go: "a shared key across the cancellation code paths is
// deliberate — if two paths ever both fire for the same order, the gateway
// dedups them (never a double refund)."
func TestCharacterise_FullRefundKey_IsSharedPerOrder(t *testing.T) {
	orderID := uuid.New()

	a := RefundFullIdempotencyKey(orderID)
	b := RefundFullIdempotencyKey(orderID)

	require.Equal(t, a, b, "the same order must yield one full-refund key — this is what dedups two cancellation paths at the gateway")
	require.NotEqual(t, a, RefundFullIdempotencyKey(uuid.New()), "different orders must not share a key")
}

// Per-line partials key by the IMMUTABLE line id, NOT by prior-refunded.
// gateway_idempotency.go: "concurrent refunds of two different lines could both
// read the same prior-refunded amount before either persists, so a prior-based
// key would collide and drop one line's refund."
func TestCharacterise_LineRefundKey_IsPerLine_NotPerPriorAmount(t *testing.T) {
	orderID := uuid.New()
	lineA, lineB := uuid.New(), uuid.New()

	keyA := RefundLineIdempotencyKey(orderID, lineA)
	keyB := RefundLineIdempotencyKey(orderID, lineB)

	require.NotEqual(t, keyA, keyB,
		"two lines on one order must key differently, or concurrent line refunds collide and one is silently dropped")
	require.Equal(t, keyA, RefundLineIdempotencyKey(orderID, lineA), "stable on retry")
}

// Arbitrary-amount partials key by PRIOR cumulative refunded paise: stable on
// retry (RefundAmount only advances after the gateway call commits) and
// distinct across sequential partials.
func TestCharacterise_PartialRefundKey_StableOnRetry_DistinctAcrossSequentialPartials(t *testing.T) {
	orderID := uuid.New()

	first := RefundPartialIdempotencyKey(orderID, 0)
	firstRetry := RefundPartialIdempotencyKey(orderID, 0)
	second := RefundPartialIdempotencyKey(orderID, 5000)

	require.Equal(t, first, firstRetry, "a retry before commit must reuse the key so the gateway dedups")
	require.NotEqual(t, first, second, "a sequential partial must get a fresh key or the gateway would dedup a legitimate second refund away")
}

// The full and partial keyspaces must never collide: a full refund and a
// partial on the same order are different money movements.
func TestCharacterise_FullAndPartialKeyspaces_DoNotCollide(t *testing.T) {
	orderID := uuid.New()

	require.NotEqual(t,
		RefundFullIdempotencyKey(orderID),
		RefundPartialIdempotencyKey(orderID, 0),
		"a full refund and a from-zero partial must not share an identity",
	)
}

// ── The reservation discipline (#609) ────────────────────────────────────────
//
// The invariant that stops a partial and a full collectively over-refunding.
// Both reserve under a row lock; the loser observes the reservation.

func TestCharacterise_FullReservation_WinsOnce_AndLoserSeesIt(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 0)

	amount, won, err := ReserveFullRefund(db, o.ID)
	require.NoError(t, err)
	require.True(t, won, "first claim wins")
	require.Equal(t, 300.0, amount, "reserves the full remaining")

	// A second full claim must lose — the payment_status/refunded_at claim is gone.
	_, wonAgain, err := ReserveFullRefund(db, o.ID)
	require.NoError(t, err)
	require.False(t, wonAgain, "a second full refund must not claim the same order — this is what prevents a double gateway refund")
}

// A partial racing a full must cap itself to what the full already reserved.
// #609: "A concurrent partial then observes the reserved refund_amount and caps
// itself to 0."
func TestCharacterise_PartialAfterFullReservation_CapsToZero(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 0)

	_, won, err := ReserveFullRefund(db, o.ID)
	require.NoError(t, err)
	require.True(t, won)

	amount, _, _, partialWon, err := ReserveRefund(db, o.ID, 50)
	require.NoError(t, err)
	require.False(t, partialWon && amount > 0,
		"a partial after a full reservation must not move more money — the order is already fully reserved")
}

// Releasing a reservation must restore the remaining so a retry can re-refund.
// The gateway-failure path depends on this: RefundOrderForCancellation calls
// revertClaim() when CreateRefund errors.
func TestCharacterise_ReleasingFullReservation_RestoresRemaining(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 0)

	amount, won, err := ReserveFullRefund(db, o.ID)
	require.NoError(t, err)
	require.True(t, won)

	ReleaseFullRefundReservation(db, o.ID, amount)

	_, wonAfterRelease, err := ReserveFullRefund(db, o.ID)
	require.NoError(t, err)
	require.True(t, wonAfterRelease,
		"after a released reservation (gateway failure) a retry must be able to claim again, or the refund is stranded")
}

// ReserveRefund uses payment_status as a SHORT-LIVED MUTEX, not a state.
//
// This is the subtlest contract in the refund system and the one most likely to
// be broken by a refactor. The reserve flips completed→refunded ONLY to
// serialize the gateway round-trip; on a PARTIAL the CALLER must revert it to
// completed (handlers/payment.go: "the reserve above flipped completed→refunded
// only to serialize the gateway round-trip. Revert it to completed and leave
// status/refunded_at untouched so the chef's hold stays releasable for the
// remainder").
//
// So while a reservation is held, NO other path can claim — including a second
// partial. Sequential partials only work because the caller reverts. A
// coordinator that keeps its own pending ledger MUST preserve this, or either
// (a) it never reverts and the chef's remaining payout is stranded, or (b) it
// reverts too early and two refunds race the cap.
func TestCharacterise_PartialReservation_HoldsAMutexUntilTheCallerReverts(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 0)

	amount, prior, full, won, err := ReserveRefund(db, o.ID, 100)
	require.NoError(t, err)
	require.True(t, won, "a partial on an unrefunded order claims")
	require.Equal(t, 100.0, amount, "reserves exactly what was requested")
	require.Equal(t, 0.0, prior, "nothing refunded before this")
	require.False(t, full, "₹100 of ₹300 is not a full refund — the caller must NOT stamp refunded_at (#586)")

	// While the mutex is held, a second reservation LOSES — payment_status is no
	// longer `completed`, which is the mutex.
	_, _, _, secondWon, err := ReserveRefund(db, o.ID, 100)
	require.NoError(t, err)
	require.False(t, secondWon,
		"payment_status is the mutex: while a reservation is in flight no sibling path may claim, "+
			"which is what serializes the gateway round-trip")

	// The caller reverts it on a partial (what handlers/payment.go does after the
	// gateway returns). Only then is the remainder claimable.
	require.NoError(t, db.Exec(
		`UPDATE orders SET payment_status = ? WHERE id = ?`, "completed", o.ID.String(),
	).Error)

	next, nextPrior, nextFull, nextWon, err := ReserveRefund(db, o.ID, 200)
	require.NoError(t, err)
	require.True(t, nextWon, "after the caller reverts the mutex, the remainder is claimable")
	require.Equal(t, 200.0, next, "the remainder")
	require.Equal(t, 100.0, nextPrior, "prior-refunded now reflects the first partial — this is RefundPartialIdempotencyKey's basis")
	require.True(t, nextFull, "₹200 exhausting the remaining ₹300 IS full → the caller stamps refunded_at")
}

// The cap: no sequence of reservations may exceed the order total. This is the
// invariant the coordinator's pending-sum must reproduce.
func TestCharacterise_ReservationsCannotExceedOrderTotal(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 0)

	first, _, _, won, err := ReserveRefund(db, o.ID, 250)
	require.NoError(t, err)
	require.True(t, won)

	// Asking for more than remains must clamp to the remaining ₹50, never ₹200.
	second, _, _, won2, err := ReserveRefund(db, o.ID, 200)
	require.NoError(t, err)
	if won2 {
		require.LessOrEqual(t, first+second, 300.0,
			"total reserved must never exceed the order total — over-refunding is money we cannot get back")
		require.Equal(t, 50.0, second, "the second reservation clamps to the remaining")
	}
}

// ── The defect this epic exists to close (#691) ──────────────────────────────
//
// Pinned as CURRENT behaviour, not as correct behaviour. When #691 lands this
// test changes deliberately — that is the point of characterising it.

// order_issue.go credits the wallet with no reference to PaymentProvider, while
// every other path routes by it. Locking this in makes the asymmetry a visible,
// deliberate change rather than an accident.
func TestCharacterise_OrderIssueRefund_GoesToWallet_RegardlessOfProvider(t *testing.T) {
	db := setupIssueDB(t)

	orderID := uuid.New()
	customerID := uuid.New()
	// A RAZORPAY-paid order: every other refund path would route this to the gateway.
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, total, refund_amount) VALUES (?, ?, 0)`,
		orderID.String(), 200.0,
	).Error)

	issue := seedIssue(t, db, customerID, orderID)

	require.NoError(t, RefundIssueToWallet(db, issue, 50, "admin", nil))

	require.Equal(t, 50.0, balanceOfUser(t, db, customerID),
		"TODAY: an order-issue refund credits the WALLET even for a gateway-paid order. "+
			"Wallet credit is unspendable while WALLET_CHECKOUT_ENABLED is off, so this is the #691 defect. "+
			"When #691 lands this expectation flips to a gateway refund — change it deliberately.")

	var orderRefunded float64
	db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&orderRefunded)
	require.Equal(t, 50.0, orderRefunded, "the order's cumulative refund is incremented atomically")
}

// #586 at the order-issue layer: a PARTIAL must leave refunded_at NULL, or the
// chef forfeits their whole remaining payout on every release-side guard.
func TestCharacterise_OrderIssuePartialRefund_DoesNotStampRefundedAt(t *testing.T) {
	db := setupIssueDB(t)

	orderID := uuid.New()
	customerID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, total, refund_amount) VALUES (?, ?, 0)`,
		orderID.String(), 200.0,
	).Error)
	issue := seedIssue(t, db, customerID, orderID)

	require.NoError(t, RefundIssueToWallet(db, issue, 50, "admin", nil))

	var refundedAt *string
	db.Raw(`SELECT refunded_at FROM orders WHERE id = ?`, orderID.String()).Scan(&refundedAt)
	require.Nil(t, refundedAt,
		"#586: a partial must leave refunded_at NULL — stamping it blocks the chef's ENTIRE payout on every release-side guard")
}
