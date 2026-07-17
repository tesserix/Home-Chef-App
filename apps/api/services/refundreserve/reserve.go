package refundreserve

// refundreserve — the atomic full/partial refund reservation shared by every path that
// moves refund money. Extracted verbatim from services/refund_reserve.go + the db-taking
// half of services/refundable.go (#690); `services` now re-exports thin wrappers, so the
// five existing call sites and their tests are unchanged.
//
// WHY IT MOVED (#690): the reservation is not a services-internal detail — it is the
// CROSS-PATH MUTEX. Every refund path must claim `payment_status = 'completed'` on the
// order row before touching a gateway; that shared claim is the only thing making two
// concurrent refunds mutually exclusive. services/orderrefund's coordinator has to take
// the SAME claim (a migrated path and an un-migrated sibling are concurrent for the whole
// of #690's migration, and a coordinator that reserved differently would let both refund
// the same order — proven by TestInterop_LegacyFullRefund_CannotRaceACoordinatorRefundInFlight).
// orderrefund cannot import services (services imports orderrefund as call sites migrate),
// so the protocol lives here, below both. Re-implementing it inside orderrefund would
// recreate exactly the duplication #687 exists to remove.
//
// ── #609, from the original refund_reserve.go ────────────────────────────────────────
//
// THE BUG IT FIXES: the full-refund paths read the refund amount (RemainingRefundable /
// order.Total) with an UNLOCKED snapshot, then SEPARATELY claimed `payment_status='completed'
// AND refunded_at IS NULL` and refunded that (possibly stale) amount, then persisted
// `refund_amount = order.RefundAmount + amount` (a stale read-modify-write). Meanwhile the
// PARTIAL path (RefundIssueToWallet) locks the order FOR UPDATE, caps at RemainingRefundable,
// and increments refund_amount atomically. Since #549/#586 a partial leaves refunded_at NULL,
// so a partial racing a full could collectively over-refund past the order total, and the
// full path's read-modify-write clobbered the partial's atomic increment.
//
// THE FIX: give the full paths the SAME discipline as the partial path — under a row lock,
// compute the remaining and RESERVE it (claim payment_status/refunded_at AND increment
// refund_amount) in ONE transaction. A concurrent partial then observes the reserved
// refund_amount and caps itself to 0; a concurrent full loses the payment_status claim. The
// reserved amount is what the caller refunds at the gateway; on gateway failure the caller
// releases the reservation so a retry can re-refund.

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// PerLineRefundedTotalTxErr sums refunds already issued per-line (recorded on cancelled
// order_items) via the provided db/tx, PROPAGATING the query error rather than failing open
// to 0 (#640: understating per-line understates the reconstructed captured amount, which can
// misclassify a still-live, partially-refunded order as fully refunded).
//
// This is exactly the amount recomputeOrderTotals removed from Order.Total, so callers add it
// back to undo the Total-vs-RefundAmount double-subtraction (#527).
func PerLineRefundedTotalTxErr(db *gorm.DB, orderID uuid.UUID) (float64, error) {
	var sum float64
	if err := db.Model(&models.OrderItem{}).
		Where("order_id = ? AND is_cancelled = ?", orderID, true).
		Select("COALESCE(SUM(refund_amount), 0)").Scan(&sum).Error; err != nil {
		return 0, err
	}
	return sum, nil
}

// PerLineRefundedTotalTx is PerLineRefundedTotalTxErr failing safe to 0 — a query error can
// never OVER-refund. Use it INSIDE a transaction so the sum is read on the SAME connection as
// the enclosing work (a cross-connection read is a separate DB under sqlite :memory: and a
// separate snapshot under a Postgres tx).
func PerLineRefundedTotalTx(db *gorm.DB, orderID uuid.UUID) float64 {
	sum, err := PerLineRefundedTotalTxErr(db, orderID)
	if err != nil {
		log.Printf("refundable: sum per-line refunds for order %s failed (treating as 0): %v", orderID, err)
		return 0
	}
	return sum
}

// lockOrders applies FOR UPDATE on Postgres; a behavior-preserving no-op on sqlite (the
// deterministic tests still pin the claim + cap contract).
func lockOrders(tx *gorm.DB) *gorm.DB {
	if tx.Dialector.Name() == "postgres" {
		return tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	return tx
}

// remainingUnderLock is the fresh amount still owed: Total − RefundAmount + per-line
// refunds (#527/#560). Callers must already hold the row lock.
func remainingUnderLock(tx *gorm.DB, o *models.Order, orderID uuid.UUID) float64 {
	return models.RoundAmount(o.Total - o.RefundAmount + PerLineRefundedTotalTx(tx, orderID))
}

// ReserveFullRefund atomically claims + reserves the full remaining refundable amount for an
// order, under a row lock. Returns (amount, true, nil) to the SINGLE winner — payment_status
// is flipped completed→refunded, refunded_at stamped, and refund_amount incremented by the
// reserved amount, all committed together. Returns (0, false, nil) when the order is not
// payment_status=completed, has nothing left to refund, or a sibling full-refund path already
// claimed it (its refunded_at is set). The caller performs the gateway refund for `amount` and,
// on failure, calls ReleaseFullRefundReservation to undo the reservation.
func ReserveFullRefund(db *gorm.DB, orderID uuid.UUID) (amount float64, won bool, err error) {
	var r Reservation
	err = db.Transaction(func(tx *gorm.DB) error {
		var e error
		r, e = ReserveInTx(tx, orderID, 0, IntentFull)
		return e
	})
	if err != nil {
		return 0, false, err
	}
	return r.Amount, r.Won, nil
}

// Outcome says why a reservation did or didn't win. The legacy callers only ever needed a
// bool (any loss is a no-op for them), but a caller that reports errors to a user — the #689
// coordinator — has to tell "this order was never charged" from "a sibling refund is already
// in flight", and those are very different things to show or retry on.
type Outcome string

const (
	// OutcomeWon — this caller owns the reservation and must drive the gateway.
	OutcomeWon Outcome = "won"
	// OutcomeNotPayable — payment_status is pending/failed: this order was never charged,
	// so there is nothing to refund. Terminal.
	OutcomeNotPayable Outcome = "not_payable"
	// OutcomeAlreadyRefunded — payment_status is already 'refunded'. Ambiguous ON ITS OWN
	// and deliberately reported as its own outcome rather than lumped in with
	// OutcomeNotPayable: the flag is BOTH the terminal "fully refunded" state AND the
	// short-lived mutex a partial refund holds mid-gateway before reverting it. Only the
	// caller can tell which — the coordinator checks its ledger for a pending row.
	OutcomeAlreadyRefunded Outcome = "already_refunded"
	// OutcomeNothingLeft — a prior refund already covered the order; remaining <= 0.
	OutcomeNothingLeft Outcome = "nothing_left"
	// OutcomeLostClaim — a sibling refund path holds the claim right now. Retryable once
	// that sibling finishes, unlike the other two.
	OutcomeLostClaim Outcome = "lost_claim"
)

// Reservation is what a ReserveRefund call produced. When Won, it is everything the caller
// needs to drive the gateway and to release the claim on failure.
type Reservation struct {
	// Outcome is why this reservation won or lost. Won is Outcome == OutcomeWon.
	Outcome Outcome

	// Amount reserved = min(requested-or-full, remaining). What the caller refunds.
	Amount float64
	// PriorRefunded is refund_amount observed UNDER THE LOCK before this reservation — the
	// exact prior-cumulative-refunded basis for the wallet / gateway idempotency keys
	// (replaces the caller's stale in-memory order.RefundAmount).
	PriorRefunded float64
	// FullRefund is true when the reservation exhausted the remaining, so the caller stamps
	// the terminal marker (status/payment_status → refunded + refunded_at). A partial-intent
	// request that a concurrent refund shrank the remainder below is reserved smaller AND
	// reclassified FULL — it does exhaust the order.
	FullRefund bool
	// Won is true for the SINGLE winner. False when the order is not payment_status=completed,
	// nothing is left, or a sibling refund path claimed it first.
	Won bool
	// Order carries only the columns the reservation itself read under the lock (id, total,
	// refund_amount, payment_status). Callers needing provider/payment ids read them
	// themselves, inside the same transaction — see the note on the Select in ReserveInTx.
	Order models.Order
}

// ReserveRefund is the PARTIAL-aware sibling of ReserveFullRefund (#611). Callers that support
// an arbitrary requested amount (InitiateRefund, the chef goodwill RefundOrder, the #689
// coordinator) don't fit ReserveFullRefund, which reserves the WHOLE remaining. This gives them
// the same discipline as the partial path: under a row lock, compute the remaining, reserve
// min(requested, remaining) atomically — flip payment_status completed→refunded as the
// concurrency mutex AND increment refund_amount together.
//
// requested <= 0 means a FULL refund of whatever is still refundable.
//
// UNLIKE ReserveFullRefund, ReserveRefund does NOT stamp refunded_at: a PARTIAL must leave it
// NULL (every release-side payout guard blocks the whole chef hold on refunded_at IS NOT NULL),
// so the terminal marker is the caller's to stamp only on FullRefund. On a downstream gateway/
// wallet failure the caller calls ReleaseRefundReservation to undo the reservation for retry.
func ReserveRefund(db *gorm.DB, orderID uuid.UUID, requested float64) (Reservation, error) {
	var r Reservation
	err := db.Transaction(func(tx *gorm.DB) error {
		var e error
		r, e = ReserveInTx(tx, orderID, requested, IntentPartial)
		return e
	})
	if err != nil {
		return Reservation{}, err
	}
	return r, nil
}

// Intent says which of the two refund claims to take. They are NOT interchangeable, and the
// difference is a money guard rather than a detail.
type Intent int

const (
	// IntentPartial claims `payment_status = 'completed'` only, and leaves refunded_at NULL.
	// A partial MUST leave it NULL — every release-side payout guard blocks the WHOLE chef
	// hold on refunded_at IS NOT NULL — so payment_status here is a short-lived mutex the
	// caller reverts once the money has moved.
	IntentPartial Intent = iota
	// IntentFull claims `payment_status = 'completed' AND refunded_at IS NULL`, and stamps
	// refunded_at as part of the claim.
	//
	// The refunded_at half is NOT redundant. It is the mutex against the full-refund paths
	// that stamp refunded_at WITHOUT flipping payment_status (cancellation_execute.go:35).
	// A full refund reserved on payment_status alone would sail straight past a sibling that
	// had already refunded the order and refund it a SECOND time — which is exactly what
	// happened when #690 first routed this path through the partial-aware claim, and what
	// TestRefundOrderForCancellation_AlreadyRefundedNoOp caught.
	IntentFull
)

// ReserveInTx is the reservation core, run inside a transaction the CALLER owns, for callers
// that must commit the claim together with their own bookkeeping. The #689 coordinator needs
// this: it writes a pending ledger row for the attempt, and if that row committed separately
// from the reservation then a crash in between would leave refund_amount incremented with NO
// ledger row — the order reads as refunded, the customer never gets the money, and nothing
// records that a refund was ever owed. One transaction makes that state unreachable.
//
// `requested` <= 0 means the full remaining. With IntentFull it is ignored — a full refund is
// by definition everything that is left.
//
// Prefer ReserveRefund / ReserveFullRefund unless you have work to commit atomically with the
// claim.
func ReserveInTx(tx *gorm.DB, orderID uuid.UUID, requested float64, intent Intent) (Reservation, error) {
	var r Reservation
	// Lock the order row so remaining is computed on — and reserved against — the same
	// committed state, serialized with every other refund path's FOR UPDATE.
	// Exactly the columns the RESERVATION needs — no more. Widening this to carry
	// provider/payment ids for one caller's convenience couples every OTHER caller to
	// columns it never asked for (it broke CompensateOrderRefund, whose orders table is
	// narrower). A caller that needs more reads it itself, under this same lock.
	var o models.Order
	if e := lockOrders(tx).Select("id", "total", "refund_amount", "payment_status").
		First(&o, "id = ?", orderID).Error; e != nil {
		return Reservation{}, e
	}
	if o.PaymentStatus != models.PaymentCompleted {
		r.Outcome = OutcomeNotPayable
		if o.PaymentStatus == models.PaymentRefunded {
			// Either fully refunded, or a sibling is mid-gateway holding the mutex.
			r.Outcome = OutcomeAlreadyRefunded
		}
		return r, nil // Won stays false either way
	}
	remaining := remainingUnderLock(tx, &o, orderID)
	if remaining <= 0 {
		r.Outcome = OutcomeNothingLeft
		return r, nil // nothing left — a prior refund already covered the order
	}

	reserve := remaining
	claim := tx.Model(&models.Order{}).
		Where("id = ? AND payment_status = ?", orderID, models.PaymentCompleted)
	updates := map[string]any{
		"payment_status": models.PaymentRefunded,
		// COALESCE so a NULL refund_amount increments from 0 rather than collapsing to NULL
		// under SQL NULL-arithmetic.
	}
	switch intent {
	case IntentFull:
		claim = claim.Where("refunded_at IS NULL")
		updates["refunded_at"] = time.Now()
	default:
		if requested > 0 && requested < remaining {
			reserve = models.RoundAmount(requested) // partial: reserve just the requested slice
		}
	}
	updates["refund_amount"] = gorm.Expr("COALESCE(refund_amount, 0) + ?", reserve)

	res := claim.Updates(updates)
	if res.Error != nil {
		return Reservation{}, res.Error
	}
	if res.RowsAffected == 0 {
		// Lost the claim to a sibling — including, under IntentFull, a sibling that stamped
		// refunded_at without touching payment_status.
		r.Outcome = OutcomeLostClaim
		return r, nil
	}
	return Reservation{
		Outcome: OutcomeWon, Amount: reserve, PriorRefunded: o.RefundAmount,
		FullRefund: reserve >= remaining, Won: true, Order: o,
	}, nil
}

// ReleaseRefundReservation undoes a winning ReserveRefund when the downstream gateway / wallet
// refund fails, so a retry can re-reserve: payment_status → completed and refund_amount
// decremented by the reserved amount. It does NOT touch refunded_at (ReserveRefund never
// stamped it). Best-effort — logged, never surfaced (the caller is already returning the
// failure, and the reconcile refund_mismatch check is the backstop). Callers MUST pass the
// exact amount ReserveRefund reserved.
func ReleaseRefundReservation(db *gorm.DB, orderID uuid.UUID, amount float64) {
	if err := db.Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]any{
		"payment_status": models.PaymentCompleted,
		"refund_amount":  gorm.Expr("COALESCE(refund_amount, 0) - ?", amount),
	}).Error; err != nil {
		log.Printf("refund-reserve: release partial reservation for order %s (amount %.2f) failed: %v", orderID, amount, err)
	}
}

// ReleaseFullRefundReservation undoes a winning ReserveFullRefund when the downstream gateway
// refund fails, so a retry can re-refund: payment_status→completed, refunded_at→NULL, and
// refund_amount decremented by the reserved amount. Best-effort — logged, never surfaced (the
// caller is already returning the gateway error, and the reconcile refund_mismatch check is the
// backstop). Callers MUST pass the exact amount ReserveFullRefund returned.
func ReleaseFullRefundReservation(db *gorm.DB, orderID uuid.UUID, amount float64) {
	if err := db.Model(&models.Order{}).Where("id = ?", orderID).Updates(map[string]any{
		"payment_status": models.PaymentCompleted,
		"refunded_at":    gorm.Expr("NULL"),
		"refund_amount":  gorm.Expr("COALESCE(refund_amount, 0) - ?", amount),
	}).Error; err != nil {
		log.Printf("refund-reserve: release reservation for order %s (amount %.2f) failed: %v", orderID, amount, err)
	}
}
