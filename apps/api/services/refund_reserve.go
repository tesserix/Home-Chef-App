package services

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// refund_reserve.go — #609. The atomic full-refund reservation shared by every full-order
// refund path (customer cancel, chef reject, chef-app goodwill full cancel, saga compensation).
//
// THE BUG IT FIXES: those paths read the refund amount (RemainingRefundable / order.Total)
// with an UNLOCKED snapshot, then SEPARATELY claimed `payment_status='completed' AND
// refunded_at IS NULL` and refunded that (possibly stale) amount, then persisted
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

// ReserveFullRefund atomically claims + reserves the full remaining refundable amount for an
// order, under a row lock. Returns (amount, true, nil) to the SINGLE winner — payment_status
// is flipped completed→refunded, refunded_at stamped, and refund_amount incremented by the
// reserved amount, all committed together. Returns (0, false, nil) when the order is not
// payment_status=completed, has nothing left to refund, or a sibling full-refund path already
// claimed it (its refunded_at is set). The caller performs the gateway refund for `amount` and,
// on failure, calls ReleaseFullRefundReservation to undo the reservation.
func ReserveFullRefund(db *gorm.DB, orderID uuid.UUID) (amount float64, won bool, err error) {
	err = db.Transaction(func(tx *gorm.DB) error {
		// Lock the order row so the amount is computed on — and reserved against — the same
		// committed state, serialized with the partial path's own FOR UPDATE. Postgres-only;
		// on sqlite the clause is a behavior-preserving no-op (the deterministic tests still
		// pin the claim + cap contract).
		lockTx := tx
		if tx.Dialector.Name() == "postgres" {
			lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var o models.Order
		if e := lockTx.Select("id", "total", "refund_amount", "payment_status").
			First(&o, "id = ?", orderID).Error; e != nil {
			return e
		}
		if o.PaymentStatus != models.PaymentCompleted {
			return nil // not payable, or already refunded by a sibling — won stays false
		}
		// Fresh remaining under the lock: Total − RefundAmount + per-line refunds (#527/#560).
		remaining := models.RoundAmount(o.Total - o.RefundAmount + PerLineRefundedTotalTx(tx, orderID))
		if remaining <= 0 {
			return nil // nothing left — a prior refund already covered the order
		}
		// Claim + reserve in one guarded UPDATE. The `refunded_at IS NULL` predicate loses to a
		// sibling that claimed mid-gateway (matching every other full-refund path's mutex).
		res := tx.Model(&models.Order{}).
			Where("id = ? AND payment_status = ? AND refunded_at IS NULL", orderID, models.PaymentCompleted).
			Updates(map[string]any{
				"payment_status": models.PaymentRefunded,
				"refunded_at":    time.Now(),
				// COALESCE so a NULL refund_amount (never set) increments from 0 rather than
				// collapsing the column to NULL under SQL NULL-arithmetic.
				"refund_amount": gorm.Expr("COALESCE(refund_amount, 0) + ?", remaining),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // lost the claim to a sibling
		}
		amount = remaining
		won = true
		return nil
	})
	if err != nil {
		return 0, false, err
	}
	return amount, won, nil
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
