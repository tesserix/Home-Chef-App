package services

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// ErrNothingToRefund means the order has no remaining refundable amount (it was
// already fully refunded, e.g. by a concurrent issue on the same order). The
// issue is left pending so an admin can reject it; no money moves.
var ErrNothingToRefund = errors.New("order has nothing left to refund")

// order_issue.go — the order-issue refund engine (#37). Reuses the wallet
// (idempotent CreditWallet, source=refund) as the refund sink. The reward grant
// is a plain in-tx credit — no Temporal.

// LineRefundAmount is the refundable amount for one order line: its subtotal
// plus its proportional share of the order tax (computed against the ORIGINAL
// order subtotal/tax so concurrent partial refunds can't drift the split). This
// is the single source of truth shared by chef per-line cancels and issue refunds.
func LineRefundAmount(lineSubtotal, orderSubtotal, orderTax float64) float64 {
	refund := lineSubtotal
	if orderSubtotal > 0 {
		refund += orderTax * (lineSubtotal / orderSubtotal)
	}
	return refund
}

// ComputeIssueRefund sums the per-line refunds for the affected items, capped at
// the order's remaining refundable amount (Total − alreadyRefunded). With no
// affected items it returns 0 — the report then goes to assisted review where an
// admin sets the amount.
func ComputeIssueRefund(orderSubtotal, orderTax, orderTotal, alreadyRefunded float64, affectedSubtotals []float64) float64 {
	var total float64
	for _, s := range affectedSubtotals {
		total += LineRefundAmount(s, orderSubtotal, orderTax)
	}
	remaining := orderTotal - alreadyRefunded
	if remaining < 0 {
		remaining = 0
	}
	if total > remaining {
		total = remaining
	}
	if total < 0 {
		total = 0
	}
	return models.RoundAmount(total)
}

// IssueConfig is the admin-tunable issue-refund policy (#37).
type IssueConfig struct {
	Enabled        bool    `json:"enabled"`
	AutoApproveCap float64 `json:"autoApproveCap"` // auto-refund when computed amount ≤ this; else assisted
}

// GetIssueConfig reads the policy from PlatformSettings `order_issue.*` keys.
func GetIssueConfig(db *gorm.DB) IssueConfig {
	cfg := IssueConfig{Enabled: true, AutoApproveCap: 300}
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "order_issue.%").Find(&settings)
	for _, s := range settings {
		switch s.Key {
		case "order_issue.enabled":
			cfg.Enabled = s.Value == "true" || s.Value == "1"
		case "order_issue.auto_approve_cap":
			if v, err := strconv.ParseFloat(s.Value, 64); err == nil {
				cfg.AutoApproveCap = v
			}
		}
	}
	return cfg
}

// ShouldAutoRefund decides whether a computed refund is small/clear enough to
// settle instantly (vs. routed to assisted review).
func ShouldAutoRefund(cfg IssueConfig, amount float64) bool {
	return cfg.Enabled && amount > 0 && amount <= cfg.AutoApproveCap
}

// RefundIssueToWallet credits a partial refund for an issue to the customer's
// wallet and resolves the issue — exactly once, atomically, and never beyond the
// order total. Everything runs in ONE transaction:
//   - the order row is locked (FOR UPDATE on Postgres) and re-read, so the
//     requested amount is capped against the *current* remaining refundable
//     (Total − RefundAmount). This is what stops N separate issues on one order
//     from collectively over-refunding past the total when they race.
//   - the wallet credit (idempotent on `issue:<id>`), the issue-status flip
//     (guarded by `WHERE status='pending'`), and the order-refund increment
//     commit together — a partial failure rolls all three back, so the ledger,
//     the issue and the order can never drift.
//
// `by` is "system" (auto) or "admin" (assisted). Returns ErrNothingToRefund when
// the order has no remaining refundable amount (issue left pending, no money moves).
func RefundIssueToWallet(db *gorm.DB, issue *models.OrderIssue, amount float64, by string, resolvedBy *uuid.UUID) error {
	if amount <= 0 {
		return fmt.Errorf("refund amount must be positive, got %v", amount)
	}

	status := models.IssueResolved
	if by == "system" {
		status = models.IssueAutoRefunded
	}
	now := time.Now()

	// won is set true only when THIS call wins the pending-issue claim and actually
	// credits — so the payout cross-guard below runs ONLY for the real refunder. A lost
	// claim (idempotent retry, or a non-wallet path like AdminRejectIssue / the #393
	// customer-fault resolver already resolved the issue) must NOT drive the hold: doing
	// so would flip a hold the winner legitimately released/left alone to `withheld` with
	// no refund behind it (#582 cross-path race).
	won := false
	// #586: is this refund FULL (exhausts the remaining refundable) or PARTIAL? Decided
	// under the SAME locked read that caps the credit, and read post-tx to pick the
	// terminal marker + the cross-guard flavour. Mirrors chef_order_cancel RefundOrder /
	// crossGuardRefundHold (#549) — the one refund path that split #549 never covered.
	var fullRefund bool
	var creditApplied float64
	err := db.Transaction(func(tx *gorm.DB) error {
		// Lock + re-read the order so the cap reflects any concurrent refund.
		readTx := tx
		if tx.Dialector.Name() == "postgres" {
			readTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var order models.Order
		if err := readTx.Select("total", "refund_amount").First(&order, "id = ?", issue.OrderID).Error; err != nil {
			return err
		}

		// Cap the credit at what's actually left to refund on the order.
		// #560/#527: RemainingRefundable = Total − RefundAmount + per-line refunds. The
		// locked read above serializes concurrent refunds on the order row; the per-line
		// sum (cancelled order_items, committed + immutable) adds back what recomputeOrder-
		// Totals removed from Total so a legitimate issue refund isn't short-changed after
		// a per-line cancel.
		credit := models.RoundAmount(amount)
		remaining := models.RoundAmount(order.Total - order.RefundAmount + PerLineRefundedTotalTx(tx, issue.OrderID))
		if credit > remaining {
			credit = remaining
		}
		if credit <= 0 {
			return ErrNothingToRefund
		}
		fullRefund = credit >= remaining
		creditApplied = credit

		// CLAIM the resolution BEFORE crediting (#581). The order-row lock above serializes
		// concurrent refunds on this order; this conditional UPDATE is the single authority
		// on "who credits". A lost claim (RowsAffected != 1) means the issue is no longer
		// pending — either a prior RefundIssueToWallet already refunded it, OR a NON-wallet
		// path (AdminRejectIssue / the #393 delivery-failure customer-fault resolver)
		// resolved it. In BOTH cases NO wallet credit must happen here, so we return before
		// CreditWallet. (Previously the credit ran first and a lost claim silently kept it,
		// committing a real credit with order.refund_amount left un-incremented → a later
		// double-refund window.)
		res := tx.Model(&models.OrderIssue{}).
			Where("id = ? AND status = ?", issue.ID, models.IssuePending).
			Updates(map[string]any{
				"status":      status,
				"resolved_at": now,
				"resolved_by": resolvedBy,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return nil // lost the claim → do NOT credit (and do NOT drive the hold below)
		}
		won = true

		// Winner: credit the wallet (idempotent on the key — defence in depth), record the
		// money fields on the issue, and apply the order-level increment. Under the lock
		// with a freshly-read RefundAmount and a credit capped at remaining, this can never
		// push refund_amount past total.
		txn, err := CreditWallet(tx, issue.CustomerID, credit, models.WalletSourceRefund, &issue.OrderID,
			"Refund for reported order issue", "issue:"+issue.ID.String(), resolvedBy)
		if err != nil {
			return err
		}
		if err := tx.Model(&models.OrderIssue{}).Where("id = ?", issue.ID).
			Updates(map[string]any{"refund_amount": credit, "refund_txn_id": txn.ID}).Error; err != nil {
			return err
		}
		orderUpdates := map[string]any{
			"refund_amount":       gorm.Expr("refund_amount + ?", credit),
			"refund_reason":       "Order issue: " + string(issue.Reason),
			"refund_initiated_by": by,
		}
		// #586: only a FULL refund stamps refunded_at (the terminal never-pay marker). A
		// PARTIAL must leave it NULL — every release-side guard blocks the WHOLE chef
		// payout on `refunded_at IS NOT NULL`, so stamping it on a partial would forfeit
		// the chef's remaining payout and silently strand a later customer-fault delivery
		// ruling (#549 over-withholding, for this path).
		if fullRefund {
			orderUpdates["refunded_at"] = now
		}
		if err := tx.Model(&models.Order{}).Where("id = ?", issue.OrderID).Updates(orderUpdates).Error; err != nil {
			return err
		}

		issue.Status = status
		issue.RefundAmount = credit
		return nil
	})
	if err != nil {
		return err
	}
	if !won {
		return nil // lost the claim → no credit happened → must not touch the payout hold
	}
	// Cross-guard the payout hold (#457/#549/#586): the customer just got money back, so the
	// chef must not keep it. A FULL refund drives the WHOLE hold to withheld/reversed; a
	// PARTIAL claws back ONLY the refunded portion from the chef's transfer and LEAVES the
	// hold releasable (the chef keeps the remainder — the food was made). Best-effort — a
	// hold-drive failure must never fail the committed refund (the release-side guard +
	// reconcile are the backstop). Single choke point for both the auto handler and admin path.
	//
	// Reaching here means the refund tx COMMITTED, and the issue-claim above guarantees a
	// retry loses the claim (won=false) and never re-reaches this point — so the partial
	// claw, which is NOT re-fire idempotent (#568), runs at most once per real refund.
	reason := "order issue refund: " + string(issue.Reason)
	if fullRefund {
		if hErr := WithholdOrReverseOrderHoldForRefund(db, issue.OrderID, reason); hErr != nil {
			log.Printf("payout cross-guard failed for order %s (issue %s): %v", issue.OrderID, issue.ID, hErr)
		}
		return nil
	}
	// Partial: claw back only the refunded portion; the hold stays releasable.
	clawErr := WithholdOrReverseOrderHoldForPartialRefund(db, issue.OrderID, ToPaise(creditApplied), reason)
	if clawErr != nil {
		log.Printf("payout partial cross-guard failed for order %s (issue %s): %v", issue.OrderID, issue.ID, clawErr)
	}
	// #586: a partial refund RESOLVES the dispute without terminally blocking the payout. If
	// this now-resolved issue had frozen the hold to `disputed` (the customer confirmed
	// delivery while it was still pending — applyHoldConfirm), the dispute is settled and the
	// chef's REMAINDER must become releasable. Mirror AdminRejectIssue: advance
	// disputed → release_eligible once no pending issue remains. The just-resolved issue is
	// committed above, so the guard's NOT EXISTS(pending) observes it. Guarded (a no-op for a
	// non-disputed hold or while another issue is pending).
	//
	// GATED ON THE CLAW SUCCEEDING: this is the only path that moves a hold disputed →
	// release_eligible off the back of a best-effort claw. If the claw genuinely failed (a
	// real gateway error — the benign flags-OFF / no-transfer / already-reversed cases all
	// return nil), the refunded portion may still be un-clawed on the chef's transfer, so we
	// must NOT make the remainder releasable (once flags are ON that would over-pay the chef
	// by the un-clawed portion). Leaving it `disputed` fails safe (chef under-paid, never
	// over-paid; recoverable). No-op today since the claw always returns nil while flags OFF.
	if clawErr == nil {
		if cErr := db.Transaction(func(tx *gorm.DB) error {
			return ReleaseDisputedHoldsForOrderIfCleared(tx, issue.OrderID)
		}); cErr != nil {
			log.Printf("payout dispute-clear after partial refund failed for order %s (issue %s): %v", issue.OrderID, issue.ID, cErr)
		}
	}
	return nil
}
