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

// ErrGoodwillFullRefund is returned when a platform-goodwill resolution would fully
// refund the order (#618). Goodwill means "refund the customer, chef keeps their
// payout" — but a FULL refund stamps refunded_at, which every release-side guard
// treats as "block the whole chef payout", so goodwill can't preserve the payout on
// a full refund. Rejected BEFORE any money moves; the admin lowers the amount or uses
// clawback.
var ErrGoodwillFullRefund = errors.New("platform goodwill applies to partial refunds only")

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
	// DefaultFaultPolicy seeds the admin's clawback-vs-goodwill choice at resolve time
	// (#618). Defaults to chef clawback (today's behaviour) so nothing changes unless an
	// admin explicitly overrides to goodwill.
	DefaultFaultPolicy models.IssueFaultPolicy `json:"defaultFaultPolicy"`
}

// GetIssueConfig reads the policy from PlatformSettings `order_issue.*` keys.
func GetIssueConfig(db *gorm.DB) IssueConfig {
	cfg := IssueConfig{Enabled: true, AutoApproveCap: 300, DefaultFaultPolicy: models.FaultChefClawback}
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
		case "order_issue.default_fault_policy":
			if p := models.IssueFaultPolicy(s.Value); p == models.FaultChefClawback || p == models.FaultPlatformGoodwill {
				cfg.DefaultFaultPolicy = p
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
//
// This is the CLAWBACK wrapper (the default): the customer is refunded AND the chef's
// payout is clawed back. Every caller EXCEPT the admin resolver keeps this behaviour;
// the admin resolver calls RefundIssueToWalletWithPolicy to offer platform goodwill (#618).
func RefundIssueToWallet(db *gorm.DB, issue *models.OrderIssue, amount float64, by string, resolvedBy *uuid.UUID) error {
	return RefundIssueToWalletWithPolicy(db, issue, amount, by, resolvedBy, true)
}

// RefundIssueToWalletWithPolicy is RefundIssueToWallet with an explicit fault policy
// (#618). `clawback=true` is the default (refund + claw back the chef). `clawback=false`
// is platform GOODWILL: the customer is refunded but the chef KEEPS their payout and the
// platform absorbs the cost. Goodwill is honoured only for a PARTIAL refund — a full
// refund stamps refunded_at (which blocks the chef payout on every release-side guard), so
// a goodwill request that would fully refund is rejected with ErrGoodwillFullRefund BEFORE
// any money moves. All other semantics are identical to RefundIssueToWallet.
func RefundIssueToWalletWithPolicy(db *gorm.DB, issue *models.OrderIssue, amount float64, by string, resolvedBy *uuid.UUID, clawback bool) error {
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

		// #624: an item named in THIS issue that was CANCELLED after the report was already
		// refunded to the customer via CancelOrderItem — and PerLineRefundedTotalTx just added its
		// value BACK into `remaining` above (the per-line add-back the #527 headroom relies on), so
		// without this the issue would refund that line a SECOND time. Subtract the per-line refund
		// already issued for the affected lines that are now cancelled from the issue's OWN reported
		// value (RequestedAmount) and cap the credit to it. This is the resolve-time mirror of the
		// #622 ReportIssue is_cancelled exclusion, evaluated here under the same order lock so it
		// sees a per-line cancel that committed first. Only bites when an affected line was actually
		// cancelled between report and resolve → the auto path (nothing cancelled yet) and admin
		// discretion on an untouched issue are unchanged. A query error propagates → the tx aborts
		// and no money moves (never over-refunds).
		if len(issue.AffectedItemIDs) > 0 {
			cancelledAffected, cErr := cancelledAffectedRefundTx(tx, issue.OrderID, issue.AffectedItemIDs)
			if cErr != nil {
				return cErr
			}
			if cancelledAffected > 0 {
				issueCap := models.RoundAmount(issue.RequestedAmount - cancelledAffected)
				if issueCap < 0 {
					issueCap = 0
				}
				if credit > issueCap {
					credit = issueCap
				}
			}
		}

		if credit <= 0 {
			return ErrNothingToRefund
		}
		fullRefund = credit >= remaining
		creditApplied = credit

		// #618: goodwill can't preserve the chef payout on a FULL refund (refunded_at would
		// stamp → every release-side guard blocks the whole payout). Reject here — under the
		// lock, before the claim/credit — so NO money moves and the admin can lower the amount
		// or use clawback. Partial goodwill is fine (leaves refunded_at NULL → chef releasable).
		if !clawback && fullRefund {
			return ErrGoodwillFullRefund
		}

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
			// #618 slice 2: a fully-refunded meal-plan-day SHELL order must leave the weekly
			// statement (statement.go selects shell orders WHERE status='delivered'). The
			// shell isn't a customer-facing order, so terminalizing it to `refunded` is safe
			// and consistent with the refunded_at stamped alongside. Scoped to a day shell —
			// a normal order's status is owned by its own lifecycle.
			if issue.MealPlanDayID != nil {
				orderUpdates["status"] = models.OrderStatusRefunded
			}
		}
		if err := tx.Model(&models.Order{}).Where("id = ?", issue.OrderID).Updates(orderUpdates).Error; err != nil {
			return err
		}

		// #618 slice 2: reconcile the linked meal-plan DAY when this issue refund claws the
		// chef. The day carries its payout hold on meal_plan_days (the shell order's own hold
		// stays ''), so stamp the day's refund txn HERE — atomically with the refund — so the
		// day model reflects the money: refund_txn_id blocks a late delivered-day resurrect
		// (#631 MarkMealPlanDayDelivered) AND blocks releaseDisputedDayHoldIfCleared (guards
		// refund_txn_id IS NULL) from re-releasing the day even if the post-commit hold-drive
		// below fails. The day HOLD state is driven by the cross-guard after commit. Goodwill
		// (clawback=false) keeps the chef's day payout → no stamp.
		if clawback && issue.MealPlanDayID != nil {
			if err := tx.Model(&models.MealPlanDay{}).
				Where("id = ? AND refund_txn_id IS NULL", *issue.MealPlanDayID).
				Update("refund_txn_id", txn.ID).Error; err != nil {
				return err
			}
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
	// Partial: claw back only the refunded portion (the chef keeps the remainder) — UNLESS this
	// is a platform-GOODWILL resolution (#618), where the chef keeps their FULL payout and the
	// platform absorbs the refund. Goodwill SKIPS the claw and audits the decision; clawErr
	// stays nil so the disputed-clear below still runs (a resolved dispute must make the chef's
	// payout releasable). Reaching here means the refund was PARTIAL (goodwill-full was rejected
	// before any money moved), so refunded_at is NULL and the chef's hold is releasable.
	var clawErr error
	if clawback {
		clawErr = WithholdOrReverseOrderHoldForPartialRefund(db, issue.OrderID, ToPaise(creditApplied), reason)
		if clawErr != nil {
			log.Printf("payout partial cross-guard failed for order %s (issue %s): %v", issue.OrderID, issue.ID, clawErr)
		}
		// #618 slice 2: the order partial-claw above is a no-op for a meal-plan-day SHELL
		// order (no razorpay_order_id → ReverseOrderChefTransferPartial returns nil) and never
		// fans out to the day. A single-unit day forfeits its WHOLE payout on any chef-fault
		// refund, so withhold the day's hold via the full cross-guard (which fans
		// meal_plan_days.order_id → withheld/reversed). This also flips the day out of
		// `disputed` so the dispute-clear below cannot release it (the refund_txn_id stamped
		// in-tx is the durable backstop if this best-effort drive fails). No-op for a normal
		// order (issue.MealPlanDayID == nil).
		if issue.MealPlanDayID != nil {
			if hErr := WithholdOrReverseOrderHoldForRefund(db, issue.OrderID, reason); hErr != nil {
				log.Printf("payout day cross-guard failed for order %s (issue %s): %v", issue.OrderID, issue.ID, hErr)
			}
		}
	} else {
		LogSystemAudit(nil, "payout.hold.refund_goodwill", "order", issue.OrderID.String(), nil, map[string]any{
			"issueId": issue.ID.String(),
			"refund":  creditApplied,
			"reason":  string(issue.Reason),
			"policy":  string(models.FaultPlatformGoodwill),
		})
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

// cancelledAffectedRefundTx sums the per-line refunds already issued (OrderItem.RefundAmount) for
// the affected lines of an issue that are now CANCELLED — the portion of the issue's reported
// value that was already returned to the customer via a per-line cancel AFTER the report (#624).
// Read on the passed tx (under the order lock) so it reflects a cancel that committed first. An
// empty affected list is 0. Errors are returned (not swallowed) so the enclosing refund tx aborts
// rather than under-subtracting and risking a double refund.
func cancelledAffectedRefundTx(tx *gorm.DB, orderID uuid.UUID, affectedIDs []string) (float64, error) {
	if len(affectedIDs) == 0 {
		return 0, nil
	}
	// order_items.id is a uuid column on Postgres; binding []string in an `id IN ?` throws
	// "operator does not exist: uuid = text" (every other id IN ? site in this repo passes
	// []uuid.UUID). Parse to uuid.UUID — an unparseable id can't match a uuid PK, so skip it.
	ids := make([]uuid.UUID, 0, len(affectedIDs))
	for _, s := range affectedIDs {
		if id, err := uuid.Parse(s); err == nil {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	var sum float64
	if err := tx.Model(&models.OrderItem{}).
		Where("order_id = ? AND is_cancelled = ? AND id IN ?", orderID, true, ids).
		Select("COALESCE(SUM(refund_amount), 0)").Scan(&sum).Error; err != nil {
		return 0, err
	}
	return sum, nil
}
