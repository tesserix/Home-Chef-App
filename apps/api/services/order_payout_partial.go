package services

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// order_payout_partial.go — #549. A PARTIAL goodwill refund must NOT forfeit the
// chef's whole held payout. Policy (2026-07-05): the chef eats the refunded amount
// — claw back ONLY the refunded portion from the chef's held Route transfer and
// leave the payout hold releasable, so the chef is still paid the remainder.
//
// Contrast WithholdOrReverseOrderHoldForRefund (a FULL refund), which drives the
// entire hold to withheld/reversed. The full-vs-partial split is decided at the
// caller (refundAmount < remaining ⇒ partial). Critically, a partial refund must
// also NOT stamp order.refunded_at nor flip status/payment_status → Refunded:
// every release-side guard keys the whole-payout block on `refunded_at IS NOT
// NULL`, so a partial that stamped it would re-block the entire chef payout — the
// exact over-withholding this fixes. That terminal-marker restraint lives in the
// caller; here we only move (a fraction of) the money.

// WithholdOrReverseOrderHoldForPartialRefund is the partial-refund cross-guard. It
// deliberately does NOT touch the order's payout hold status — the chef is still
// owed the remainder, so the hold stays releasable — and claws back exactly
// refundedPaise from the chef's held transfer via ReverseOrderChefTransferPartial.
// Best-effort like the full-refund guard: callers invoke it so a claw-back failure
// never fails the refund (the DB state is already correct — hold releasable, no
// terminal markers — and the reconcile cron is the transfer-side backstop).
func WithholdOrReverseOrderHoldForPartialRefund(db *gorm.DB, orderID uuid.UUID, refundedPaise int, reason string) error {
	return ReverseOrderChefTransferPartial(db, orderID, refundedPaise, reason)
}

// ReverseOrderChefTransferPartial claws back min(refundedPaise, un-reversed chef
// transfer amount) from the CHEF's Route transfer on a partially-refunded order —
// never the rider's (the driver still delivered) and never more than the chef was
// paid (the chef floors at zero; the platform absorbs any excess). Flag-gated by
// ORDER_PAYOUT_AUTO_RELEASE_ENABLED: a pure no-op (no live money) while OFF, so
// until the gateway is live-verified (#218) the DB state — hold left releasable,
// no refunded_at — is the only effect. A real movement is audited; an idempotent
// already-reversed transfer is tolerated and not re-audited.
func ReverseOrderChefTransferPartial(db *gorm.DB, orderID uuid.UUID, refundedPaise int, reason string) error {
	if !payoutMovementEnabled() || refundedPaise <= 0 {
		return nil
	}
	// The chef's linked account is the reversal target — load it alongside the
	// order's gateway id. Either missing (non-gateway order, or a chef with no Route
	// account) ⇒ nothing to claw back.
	var row struct {
		RazorpayOrderID   string
		RazorpayAccountID string
	}
	if err := db.Raw(`SELECT o.razorpay_order_id AS razorpay_order_id, c.razorpay_account_id AS razorpay_account_id
		FROM orders o JOIN chef_profiles c ON c.id = o.chef_id WHERE o.id = ?`, orderID.String()).
		Scan(&row).Error; err != nil {
		return fmt.Errorf("order-payout: load order/chef %s for partial reverse: %w", orderID, err)
	}
	if row.RazorpayOrderID == "" || row.RazorpayAccountID == "" {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return nil
	}
	transfers, err := rz.FetchOrderTransfers(row.RazorpayOrderID)
	if err != nil {
		return fmt.Errorf("order-payout: fetch transfers for order %s: %w", orderID, err)
	}
	for _, t := range transfers {
		if t.ID == "" || t.Account != row.RazorpayAccountID {
			continue // rider transfer / unlinked — the chef eats the refund, not the rider
		}
		// Cap at what remains un-reversed so a repeated partial refund never
		// over-claws this transfer (the chef floors at zero; platform eats the rest).
		available := t.Amount - t.AmountReversed
		amt := refundedPaise
		if amt > available {
			amt = available
		}
		if amt <= 0 {
			return nil // already fully clawed back — no new money to move
		}
		if _, err := rz.ReverseTransfer(t.ID, amt); err != nil {
			if isAlreadyReversedErr(err) {
				return nil // idempotent re-drive — no new money moved, don't audit
			}
			return fmt.Errorf("order-payout: partial reverse transfer %s (order %s): %w", t.ID, orderID, err)
		}
		auditTransferMovement(auditTransferReverse, aggTypeOrder, orderID, t.ID, amt,
			"partial goodwill refund — "+reason)
		return nil // exactly one chef transfer per order
	}
	return nil
}
