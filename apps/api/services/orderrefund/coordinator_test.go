package orderrefund

// coordinator_test.go — #689, part of #687. Written BEFORE the coordinator
// (TDD): this file is the contract.
//
// The coordinator exists to make refunds correct BY CONSTRUCTION rather than by
// five independent call sites each remembering the same rules. The tests below
// are the rules. They are deliberately about the SAGA — the ledger, the cap, the
// reservation lifecycle — not about any one caller's business logic, which stays
// with the callers and is already covered by their own suites plus the
// characterisation suite (#688).

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// ── The kill switch ──────────────────────────────────────────────────────────

// Disabled must refuse BEFORE touching the gateway or the ledger. A kill switch
// that still reserves would leave phantom pending rows against the cap.
func TestCoordinator_Disabled_RefusesWithoutTouchingAnything(t *testing.T) {
	f := newFixture(t, withEnabled(false))
	o := f.seedPaidOrder(300)

	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "test", Actor: "admin", ScopeID: "cancel",
	})

	require.ErrorIs(t, err, ErrRefundsDisabled)
	require.Zero(t, f.gateway.calls, "must not call the gateway when disabled")
	require.Empty(t, f.ledgerRows(o.ID), "must not reserve a ledger row when disabled")
}

// ── ScopeID validation ───────────────────────────────────────────────────────

// HomeChef HASHES the logical key before it reaches the gateway
// (gateway_idempotency.go normalizeIdempotencyKey → 32-char hex), so the key is
// charset-safe by construction — which is why the existing
// RefundFullIdempotencyKey ("refund:<uuid>:full") works today despite the colons.
//
// So unlike mark8ly (which passes the raw ScopeID through and therefore must
// validate [A-Za-z0-9_-]), we only reject what would break OUR identity: an
// empty scope, which would silently collapse two different refunds onto one key
// and dedup a legitimate second refund away.
func TestCoordinator_RejectsEmptyScopeID(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "test", Actor: "admin", ScopeID: "",
	})

	require.Error(t, err, "an empty scope would collapse distinct refunds onto one key")
	require.Zero(t, f.gateway.calls)
}

// Colons are legitimate — "issue:<id>", "line:<id>" — and must NOT be rejected.
// This is the counterpart to the test above: over-validating would break the
// scope shapes the callers actually use.
func TestCoordinator_AcceptsColonScopes_TheyAreHashedForTheGateway(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	amount := 50.0
	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Amount: &amount, Reason: "test", Actor: "admin", ScopeID: "issue:abc-123",
	})

	require.NoError(t, err, "issue:<id> is a real scope shape — the key is hashed, so the colon is safe")
	require.Equal(t, 1, f.gateway.calls)
}

// ── The happy path ───────────────────────────────────────────────────────────

func TestCoordinator_Refund_ReservesThenCallsGatewayThenFinalizes(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	res, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "customer cancelled", Actor: "customer", ScopeID: "cancel",
	})

	require.NoError(t, err)
	require.Equal(t, 300.0, res.Amount, "nil Amount means the full remaining")
	require.True(t, res.FullRefund, "₹300 of ₹300 is a full refund")
	require.Equal(t, 1, f.gateway.calls)
	require.Equal(t, 30000, f.gateway.lastAmountPaise, "the gateway is called in PAISE")

	rows := f.ledgerRows(o.ID)
	require.Len(t, rows, 1, "exactly one ledger row per attempt")
	require.Equal(t, models.RefundTxnSucceeded, rows[0].Status)
	require.Equal(t, "rfnd_test", rows[0].ProviderRefundID)
	require.NotNil(t, rows[0].CompletedAt)
}

// The ledger row must exist BEFORE the gateway call, not after. If we only wrote
// it on success, a crash mid-call would leave money moved at Razorpay and no
// record of it — unreconcilable.
func TestCoordinator_LedgerRowIsPendingDuringTheGatewayCall(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	var sawStatus models.RefundTxnStatus
	var sawCount int
	f.gateway.before = func() {
		rows := f.ledgerRows(o.ID)
		sawCount = len(rows)
		if len(rows) == 1 {
			sawStatus = rows[0].Status
		}
	}

	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "r", Actor: "admin", ScopeID: "cancel",
	})

	require.NoError(t, err)
	require.Equal(t, 1, sawCount, "the row must be committed before the gateway is called")
	require.Equal(t, models.RefundTxnPending, sawStatus,
		"it must be PENDING during the call — a crash here leaves a claim a sweep can reconcile")
}

// ── The cap ──────────────────────────────────────────────────────────────────

func TestCoordinator_RefusesAmountAboveRemaining(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	amount := 400.0
	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Amount: &amount, Reason: "r", Actor: "admin", ScopeID: "admin-1",
	})

	require.Error(t, err, "must never refund more than the order")
	require.Zero(t, f.gateway.calls, "must fail BEFORE the gateway — money out is unrecoverable")
}

// Two refunds with DIFFERENT scopes must never each clear the cap against a stale
// balance and both move real money.
//
// The MECHANISM changed in #690 and this test changed with it. It used to seed a
// bare pending ledger row and assert the coordinator's own pending-SUM refused the
// second refund. That cap was replaced by the shared reservation: an in-flight
// refund has already incremented refund_amount and holds the payment_status claim,
// so every path — coordinator or legacy — sees the money gone. The second refund is
// now refused because it LOSES THE CLAIM, not because a sum said so.
//
// The old setup is also unreachable now: the reservation and the ledger row commit
// in one transaction, so a pending row without a matching reservation cannot exist.
// Seeding one tested a state production can't produce.
//
// What did NOT change is the invariant, so that is what this still pins: money in
// flight for one scope must stop a second scope from refunding it too.
func TestCoordinator_InFlightRefund_BlocksASecondScope(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	// A real in-flight refund from another scope: reservation taken, gateway not
	// yet returned — exactly the window a sibling would race.
	f.seedInFlightRefund(o.ID, 250, "issue:abc")

	amount := 100.0
	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Amount: &amount, Reason: "r", Actor: "admin", ScopeID: "cancel",
	})

	require.ErrorIs(t, err, ErrRefundInFlight,
		"₹250 is in flight, so ₹100 must be refused while it settles — and refused as "+
			"TRANSIENT, since it becomes legitimate if that refund fails and releases")
	require.Zero(t, f.gateway.calls, "must fail BEFORE the gateway — money out is unrecoverable")
	require.Equal(t, 250.0, f.orderRefundAmount(o.ID), "the in-flight reservation is untouched")
}

// A FAILED row must NOT hold the cap — it released its money.
func TestCoordinator_FailedLedgerRowDoesNotHoldTheCap(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)
	f.seedFailedLedger(o.ID, 250, "issue:abc")

	amount := 300.0
	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Amount: &amount, Reason: "r", Actor: "admin", ScopeID: "cancel",
	})

	require.NoError(t, err, "a failed attempt moved no money — it must not block a later refund")
	require.Equal(t, 1, f.gateway.calls)
}

// ── Gateway failure ──────────────────────────────────────────────────────────

// A gateway failure must leave the ledger `failed` and the order's balance
// untouched, so a retry can re-refund. The existing paths do this via
// ReleaseRefundReservation; the coordinator must not regress it.
func TestCoordinator_GatewayFailure_MarksFailedAndReleasesTheReservation(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)
	f.gateway.err = errors.New("razorpay: card network down")

	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "r", Actor: "admin", ScopeID: "cancel",
	})

	require.Error(t, err)
	rows := f.ledgerRows(o.ID)
	require.Len(t, rows, 1)
	require.Equal(t, models.RefundTxnFailed, rows[0].Status)
	require.Contains(t, rows[0].FailureReason, "card network down",
		"record WHY — this is what you need when reconciling a failed refund")

	require.Equal(t, 0.0, f.orderRefundAmount(o.ID),
		"a failed refund must not leave the order's refund_amount advanced, or the money is stranded")
}

// ── Idempotency ──────────────────────────────────────────────────────────────

// Same ScopeID twice = one logical refund. The second must not move money again.
func TestCoordinator_SameScopeTwice_DoesNotDoubleRefund(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	amount := 100.0
	cmd := RefundCommand{OrderID: o.ID, Amount: &amount, Reason: "r", Actor: "admin", ScopeID: "issue:abc"}

	_, err := f.coord.Refund(f.ctx, cmd)
	require.NoError(t, err)
	require.Equal(t, 1, f.gateway.calls)

	_, err = f.coord.Refund(f.ctx, cmd)
	require.NoError(t, err, "a replay must be a no-op, not an error — callers retry")
	require.Equal(t, 1, f.gateway.calls, "the gateway must NOT be called twice for one logical refund")
	require.Len(t, f.ledgerRows(o.ID), 1, "one logical refund = one ledger row")
}

// Different scopes on the same order are different refunds and DO both move
// money — provided the cap allows it. This is the counterpart to the
// in-flight-cap test: the ledger must not over-dedup either.
func TestCoordinator_DifferentScopes_AreDistinctRefunds(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	a, b := 100.0, 150.0
	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Amount: &a, Reason: "r", Actor: "admin", ScopeID: "issue:one",
	})
	require.NoError(t, err)

	_, err = f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Amount: &b, Reason: "r", Actor: "admin", ScopeID: "issue:two",
	})
	require.NoError(t, err)

	require.Equal(t, 2, f.gateway.calls, "two distinct refunds")
	require.Len(t, f.ledgerRows(o.ID), 2)
	require.Equal(t, 250.0, f.orderRefundAmount(o.ID), "₹100 + ₹150, both recorded")
}

// ── Preconditions ────────────────────────────────────────────────────────────

func TestCoordinator_RefusesOrderWithNoCapturedPayment(t *testing.T) {
	f := newFixture(t)
	o := f.seedUnpaidOrder(300)

	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "r", Actor: "admin", ScopeID: "cancel",
	})

	require.Error(t, err, "nothing was captured — there is nothing to refund")
	require.Zero(t, f.gateway.calls)
}

func TestCoordinator_RefusesNonPositiveAmount(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	for _, bad := range []float64{0, -1} {
		amt := bad
		_, err := f.coord.Refund(f.ctx, RefundCommand{
			OrderID: o.ID, Amount: &amt, Reason: "r", Actor: "admin", ScopeID: "cancel",
		})
		require.Error(t, err, "amount %v must be refused", bad)
	}
	require.Zero(t, f.gateway.calls)
}

func TestCoordinator_UnknownOrder_Errors(t *testing.T) {
	f := newFixture(t)

	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: uuid.New(), Reason: "r", Actor: "admin", ScopeID: "cancel",
	})

	require.Error(t, err)
	require.Zero(t, f.gateway.calls)
}

// A caller retrying the SAME scope after a gateway FAILURE must be able to
// re-refund — that is the whole point of releasing the reservation. But the
// ledger's idempotency_key is UNIQUE, so a naive "fall through and reserve
// again" would hit the constraint and the refund would be permanently stuck.
func TestCoordinator_RetryAfterGatewayFailure_SameScope_Succeeds(t *testing.T) {
	f := newFixture(t)
	o := f.seedPaidOrder(300)

	// First attempt: the gateway is down.
	f.gateway.err = errors.New("razorpay: timeout")
	_, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "r", Actor: "admin", ScopeID: "cancel",
	})
	require.Error(t, err)
	require.Equal(t, models.RefundTxnFailed, f.ledgerRows(o.ID)[0].Status)

	// Gateway recovers; the caller retries the same logical refund.
	f.gateway.err = nil
	res, err := f.coord.Refund(f.ctx, RefundCommand{
		OrderID: o.ID, Reason: "r", Actor: "admin", ScopeID: "cancel",
	})

	require.NoError(t, err, "a retry after a FAILED attempt must be able to refund — otherwise the money is stranded forever")
	require.Equal(t, 300.0, res.Amount)
	require.Equal(t, 300.0, f.orderRefundAmount(o.ID), "the money is recorded exactly once")
}
