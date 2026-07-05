package services

import (
	"github.com/google/uuid"
)

// payout_audit.go — #397. An immutable money-movement audit trail for the escrow
// transfer seams. Every time a chef/rider Route transfer is actually HELD, RELEASED,
// or REVERSED at the gateway, we write one append-only AuditLog row recording which
// transfer moved, for which aggregate, how much, and why. For an escrow model where
// the platform holds customer funds and an admin approves releases, this "where did
// the money go" chain is a compliance requirement (ties into #19 legal sign-off).
//
// System-actor (no human on the seam — the mover is the delivery saga, the admin
// payout queue via the settle machinery, or the reconcile cron). Best-effort: audit
// failures never break the money path (LogSystemAudit swallows + logs its own error).

// Money-movement audit actions. Stable strings — an investigator greps audit_logs
// by these to reconstruct a transfer's lifecycle.
const (
	auditTransferHold    = "payout.transfer.hold"
	auditTransferRelease = "payout.transfer.release"
	auditTransferReverse = "payout.transfer.reverse"
)

// auditTransferMovement records ONE real gateway transfer movement. Call it only
// AFTER the gateway op succeeded (not on an idempotent already-done no-op, which
// moved no new money), so the trail reflects actual settlement, not intent.
//
//	action        one of auditTransfer{Hold,Release,Reverse}
//	aggType       the owning aggregate: aggTypeOrder / aggTypeMealPlanDay / aggTypeGroupOrder
//	aggID         the aggregate's id (the entity the money belongs to)
//	transferID    the Razorpay transfer id that moved
//	amountPaise   the amount moved in paise (0 when not known at the call site, e.g.
//	              a full release/reverse where the seam only holds the transfer id)
//	reason        short human cause ("delivered", "order refunded", "cancelled", …)
func auditTransferMovement(action, aggType string, aggID uuid.UUID, transferID string, amountPaise int, reason string) {
	LogSystemAudit(nil, action, aggType, aggID.String(), nil, map[string]any{
		"transfer_id":  transferID,
		"amount_paise": amountPaise,
		"reason":       reason,
	})
}
