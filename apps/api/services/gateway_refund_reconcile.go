package services

// gateway_refund_reconcile.go — #640. Remediation for an order that is fully refunded in
// AGGREGATE at the gateway (e.g. an in-app partial refund + an out-of-band Razorpay-dashboard
// refund) but never had refunded_at stamped by any single app event — so the payout release
// guard (refunded_at IS NOT NULL / status IN refunded,cancelled / payment_status=refunded)
// never blocks and the chef could be paid on a fully-refunded order once escrow flags flip.
//
// The daily reconciliation sweep detects this drift (DriftFullRefundUnstamped) by comparing the
// gateway's cumulative amount_refunded against the captured total, then calls this to finalize
// the order + drive the payout cross-guard — mirroring finalizeStuckRefund's FULL branch.
//
// refund_amount is deliberately LEFT UNTOUCHED. An out-of-band leg the app never observed can't
// be reconstructed into local per-channel numbers here, and the safety-critical effect is the
// payout BLOCK (refunded_at), not the ledger figure. Ops reconciles books from the drift alert.
// The stamp itself is plain DB state (runs with escrow flags OFF); only the payout drive is
// flag-gated.

import (
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// FinalizeGatewayFullRefund stamps a cumulative-full-refund order terminal and blocks its
// payout. gatewayRefundedPaise is the gateway's cumulative amount_refunded the detector already
// fetched (monotonic non-decreasing, so a slightly stale value only makes us MORE conservative —
// never a false stamp). Returns true only when THIS call performed the stamp (single-winner via
// the guarded UPDATE), so the caller can log/act exactly once. Idempotent: a second call on an
// already-stamped row returns false.
func FinalizeGatewayFullRefund(orderID uuid.UUID, gatewayRefundedPaise int) (healed bool) {
	// Never finalize a typed-escrow shell (meal-plan-day / group) — those refund via their own
	// flow and keep their hold on the typed row, not the shell order.
	if kind, err := TypedRefundOrderKind(database.DB, orderID); err != nil || kind != "" {
		return false
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		lockTx := tx
		if tx.Dialector.Name() == "postgres" {
			lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var o models.Order
		if e := lockTx.Select("id", "status", "total", "refund_amount", "wallet_applied",
			"refund_id", "payment_status", "refunded_at").
			First(&o, "id = ?", orderID).Error; e != nil {
			return e
		}
		// Re-check under the lock — a concurrent refund/reconcile may have finalized it first.
		if o.RefundedAt != nil {
			return nil // already stamped → healed stays false
		}
		// Re-verify the full-refund conclusion FRESH under the lock — never trust the detector's
		// pre-lock, possibly-stale classification (mirrors finalizeStuckRefund). The per-line
		// read PROPAGATES its error so a DB blip can't understate captured and false-stamp a
		// still-live, partially-refunded order.
		perLine, plErr := PerLineRefundedTotalTxErr(tx, orderID)
		if plErr != nil {
			return plErr
		}
		capturedPaise := ToPaise(o.Total + perLine - o.WalletApplied)
		if capturedPaise <= 0 || gatewayRefundedPaise < capturedPaise {
			return nil // not actually fully refunded under fresh state → skip (healed=false)
		}
		now := time.Now()
		updates := map[string]any{
			"refunded_at":    now,
			"payment_status": models.PaymentRefunded, // was 'completed' for an out-of-band refund
		}
		// Only a NON-terminal order transitions to refunded (a normal full refund's terminal
		// state). Preserve a legitimately cancelled order's status — refunded_at alone already
		// blocks its payout, and clobbering cancelled→refunded would destroy the RTO
		// cancelled-vs-refunded distinction downstream reporting relies on.
		if o.Status != models.OrderStatusCancelled && o.Status != models.OrderStatusRefunded {
			updates["status"] = models.OrderStatusRefunded
		}
		// Recover a traceability marker only if no prior refund left one.
		if o.RefundID == "" {
			updates["refund_id"] = "reconciled:gateway-full-refund"
			updates["refund_initiated_by"] = "reconcile"
		}
		res := tx.Model(&models.Order{}).
			Where("id = ? AND refunded_at IS NULL", orderID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		healed = res.RowsAffected == 1
		return nil
	})
	if err != nil {
		log.Printf("gateway-refund-reconcile: finalize order %s failed: %v", orderID, err)
		return false
	}
	if !healed {
		return false
	}

	// Drive the payout cross-guard to the terminal state a normal full refund would have
	// reached (flag-gated → a state-only no-op while escrow flags are OFF). A fully-refunded
	// order must never pay the chef: withhold a never-released hold, reverse a released one.
	if err := WithholdOrReverseOrderHoldForRefund(database.DB, orderID, "reconcile: gateway cumulative full refund"); err != nil {
		log.Printf("gateway-refund-reconcile: cross-guard for order %s failed: %v", orderID, err)
	}
	return true
}
