package services

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

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
// wallet and resolves the issue — exactly once. The wallet credit is idempotent
// on `issue:<id>`, and the order-refund increment + issue-status flip are guarded
// by a conditional `WHERE status='pending'` so a duplicate call (retry / race /
// auto+admin both firing) never double-refunds the order total. `by` is "system"
// (auto) or "admin" (assisted).
func RefundIssueToWallet(db *gorm.DB, issue *models.OrderIssue, amount float64, by string, resolvedBy *uuid.UUID) error {
	if amount <= 0 {
		return fmt.Errorf("refund amount must be positive, got %v", amount)
	}

	// Credit first — idempotent on the key, so at most one real credit ever.
	txn, err := CreditWallet(db, issue.CustomerID, amount, models.WalletSourceRefund, &issue.OrderID,
		"Refund for reported order issue", "issue:"+issue.ID.String(), resolvedBy)
	if err != nil {
		return err // issue not yet resolved; a retry is safe
	}

	status := models.IssueResolved
	if by == "system" {
		status = models.IssueAutoRefunded
	}
	now := time.Now()

	// Claim the resolution exactly once.
	res := db.Model(&models.OrderIssue{}).
		Where("id = ? AND status = ?", issue.ID, models.IssuePending).
		Updates(map[string]any{
			"status":        status,
			"refund_amount": amount,
			"resolved_at":   now,
			"resolved_by":   resolvedBy,
			"refund_txn_id": txn.ID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 1 {
		// Only the winner applies the order-level refund increment.
		db.Model(&models.Order{}).Where("id = ?", issue.OrderID).Updates(map[string]any{
			"refund_amount":       gorm.Expr("refund_amount + ?", amount),
			"refund_reason":       "Order issue: " + string(issue.Reason),
			"refund_initiated_by": by,
			"refunded_at":         now,
		})
	}

	issue.Status = status
	issue.RefundAmount = amount
	return nil
}
