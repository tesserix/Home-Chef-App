package services

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// stuck_refund_reconcile.go — #602. A refund whose money leg (gateway CreateRefund /
// CreditWallet) SUCCEEDED but whose terminal persist tx FAILED leaves the order STUCK at
// payment_status=refunded with refunded_at NULL: services.ReserveRefund already committed
// `refund_amount += reserved` in its OWN tx, so the ledger is CORRECT — only the terminal write
// (revert-to-completed for a partial / refunded_at+status for a full) didn't land. The refund
// handlers no longer "unstick" this by decrementing refund_amount (that erased a real refund →
// over-refund on the next distinct refund + amount-key collision). This reconcile finalizes the
// stuck row instead, money-safely.

// stuckRefundGrace is how long a mid-refund may sit refunded+refunded_at-NULL before the
// reconcile finalizes it. A live refund passes through this exact state only for the few seconds
// between ReserveRefund and its persist, so the grace guarantees we never race one in flight.
const stuckRefundGrace = 15 * time.Minute

// reconcileStuckRefunds finalizes orders left mid-refund by a persist failure. The signal
// `payment_status='refunded' AND refunded_at IS NULL` is UNIQUE to a stuck mid-refund — a
// finalized FULL stamps refunded_at; a finalized PARTIAL reverts to completed. Under a row lock
// it recomputes the remaining refundable: <=0 ⇒ the stuck refund was FULL (stamp refunded_at +
// status=refunded); else PARTIAL (revert payment_status→completed). refund_amount is left
// untouched (already correct). It then drives the payout cross-guard (flag-gated) to the
// terminal state the failed persist's cross-guard would have reached. Ledger finalization is
// plain DB state, so this runs with escrow flags OFF; only the cross-guard is gated. Returns the
// count healed.
func reconcileStuckRefunds() int {
	cutoff := time.Now().Add(-stuckRefundGrace)
	var stuck []models.Order
	if err := database.DB.
		Where("payment_status = ? AND refunded_at IS NULL AND updated_at < ?", models.PaymentRefunded, cutoff).
		Limit(sweepBatchLimit).
		Find(&stuck).Error; err != nil {
		log.Printf("stuck-refund-reconcile: load failed: %v", err)
		return 0
	}
	healed := 0
	for i := range stuck {
		o := &stuck[i]
		// Defensive: never finalize a typed-escrow shell (meal-plan-day / group). Those refund
		// through their own flows (RefundDay / participant refunds) and never reach this generic
		// stuck state — but the guard stops the reconcile from ever acting on a false signal.
		if kind, err := TypedRefundOrderKind(database.DB, o.ID); err != nil || kind != "" {
			continue
		}
		full, ok, refundedPaise := finalizeStuckRefund(o.ID)
		if !ok {
			continue
		}
		healed++
		// Drive the payout cross-guard to the terminal state the failed persist would have
		// (flag-gated → no-op while escrow flags are off). FULL → withhold/reverse the whole
		// hold; PARTIAL → claw the chef transfer down to the cumulative refunded amount (capped
		// by AmountReversed → idempotent, converges with any request-time partial claws). Use
		// refundedPaise from finalizeStuckRefund's LOCKED read, NOT the batch-load snapshot — a
		// concurrent CancelOrderItem can bump refund_amount between the two, and the stale
		// snapshot would under-claw the chef permanently once the row leaves the stuck state.
		var cgErr error
		if full {
			cgErr = WithholdOrReverseOrderHoldForRefund(database.DB, o.ID, "reconcile: finalize stuck full refund")
		} else {
			cgErr = WithholdOrReverseOrderHoldForPartialRefund(database.DB, o.ID, refundedPaise, "reconcile: finalize stuck partial refund")
		}
		if cgErr != nil {
			log.Printf("stuck-refund-reconcile: cross-guard for order %s failed: %v", o.ID, cgErr)
		}
	}
	if healed > 0 {
		log.Printf("stuck-refund-reconcile: finalized %d stuck refund(s)", healed)
	}
	return healed
}

// finalizeStuckRefund atomically finalizes ONE stuck order under a row lock. Returns
// (wasFull, healed, refundedPaise) — refundedPaise is the cumulative refund_amount observed
// UNDER THE LOCK (the caller must use this for the payout claw, never a pre-lock snapshot).
// healed=false when the row is no longer stuck (a concurrent refund/reconcile finalized it
// first) — the guarded UPDATE's WHERE is the single serialization point.
func finalizeStuckRefund(orderID uuid.UUID) (full, healed bool, refundedPaise int) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		lockTx := tx
		if tx.Dialector.Name() == "postgres" {
			lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var o models.Order
		if e := lockTx.Select("id", "total", "refund_amount", "refund_id", "payment_status", "refunded_at").
			First(&o, "id = ?", orderID).Error; e != nil {
			return e
		}
		// Re-check the stuck signal under the lock — a concurrent actor may have finalized it.
		if o.PaymentStatus != models.PaymentRefunded || o.RefundedAt != nil {
			return nil // no longer stuck → healed stays false
		}
		refundedPaise = ToPaise(o.RefundAmount) // cumulative refunded, read under the lock
		remaining := models.RoundAmount(o.Total - o.RefundAmount + PerLineRefundedTotalTx(tx, orderID))
		full = remaining <= 0

		updates := map[string]any{}
		// The failed persist never wrote refund_id / initiated_by; recover a marker for
		// traceability, but PRESERVE a non-empty value left by a PRIOR successful partial refund.
		if o.RefundID == "" {
			updates["refund_id"] = "reconciled:persist-recovered"
			updates["refund_initiated_by"] = "reconcile"
		}
		if full {
			updates["status"] = models.OrderStatusRefunded
			updates["refunded_at"] = time.Now()
			// payment_status is already refunded — leave it.
		} else {
			updates["payment_status"] = models.PaymentCompleted
		}
		res := tx.Model(&models.Order{}).
			Where("id = ? AND payment_status = ? AND refunded_at IS NULL", orderID, models.PaymentRefunded).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		healed = res.RowsAffected == 1
		return nil
	})
	if err != nil {
		log.Printf("stuck-refund-reconcile: finalize order %s failed: %v", orderID, err)
		return false, false, 0
	}
	return full, healed, refundedPaise
}
