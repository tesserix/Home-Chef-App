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
		credit := models.RoundAmount(amount)
		remaining := models.RoundAmount(order.Total - order.RefundAmount)
		if credit > remaining {
			credit = remaining
		}
		if credit <= 0 {
			return ErrNothingToRefund
		}

		// Credit the wallet inside this txn (nested savepoint) — idempotent on
		// the key, so at most one real credit per issue ever.
		txn, err := CreditWallet(tx, issue.CustomerID, credit, models.WalletSourceRefund, &issue.OrderID,
			"Refund for reported order issue", "issue:"+issue.ID.String(), resolvedBy)
		if err != nil {
			return err
		}

		// Claim the resolution exactly once.
		res := tx.Model(&models.OrderIssue{}).
			Where("id = ? AND status = ?", issue.ID, models.IssuePending).
			Updates(map[string]any{
				"status":        status,
				"refund_amount": credit,
				"resolved_at":   now,
				"resolved_by":   resolvedBy,
				"refund_txn_id": txn.ID,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			// Already resolved (idempotent retry / race on the same issue). Because
			// the wallet credit keys on issue:<id>, the winner already used that key,
			// so our CreditWallet above was a guaranteed no-op — committing here
			// without bumping the order total is safe and never double-refunds.
			return nil
		}

		// Winner applies the order-level refund increment. Under the lock with a
		// freshly-read RefundAmount and a credit capped at remaining, this can
		// never push refund_amount past total.
		if err := tx.Model(&models.Order{}).Where("id = ?", issue.OrderID).Updates(map[string]any{
			"refund_amount":       gorm.Expr("refund_amount + ?", credit),
			"refund_reason":       "Order issue: " + string(issue.Reason),
			"refund_initiated_by": by,
			"refunded_at":         now,
		}).Error; err != nil {
			return err
		}

		issue.Status = status
		issue.RefundAmount = credit
		return nil
	})
	if err != nil {
		return err
	}
	// Cross-guard the payout hold (#457): the customer just got money back, so the
	// chef must not keep it. Best-effort — a hold-drive failure must never fail the
	// refund (the release-side guard + reconcile are the backstop). This single
	// choke point covers both the auto handler and the admin path.
	if hErr := WithholdOrReverseOrderHoldForRefund(db, issue.OrderID, "order issue refund: "+string(issue.Reason)); hErr != nil {
		log.Printf("payout cross-guard failed for order %s (issue %s): %v", issue.OrderID, issue.ID, hErr)
	}
	return nil
}
