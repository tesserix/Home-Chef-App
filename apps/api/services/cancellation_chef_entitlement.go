package services

// cancellation_chef_entitlement.go — paying the chef the share of a cancelled
// order they were always entitled to (#947).
//
// Epic #475 specifies the cancellation money split as
//
//	grand == RefundTotalPaise + VendorKeptPaise + PlatformKeptPaise
//
// and, under Money integration, "the refunded % is reversed, THE WITHHELD %
// BECOMES THE VENDOR'S PAYOUT — never both". VendorKeptPaise was computed,
// persisted, and shown to the customer as the chef's retained share from the
// day the epic shipped, but nothing ever paid it: the weekly settlement
// statement (services/statement.go) aggregates `orders WHERE status =
// 'delivered'`, and a cancelled order is by definition not delivered. ₹710.53
// across six cancellations accrued to nobody.
//
// ── Why this does NOT touch the payout-hold guards ───────────────────────────
//
// The obvious fix — let a cancelled order through the payout-release path —
// means relaxing three guards (payout_release.go:167 and :391, payout_hold.go:165)
// that exist for exactly one reason: to stop a refunded order paying a chef out
// of money that went back to the customer. Those guards stay untouched. Instead
// the entitlement is raised as a ChefBonus, the existing settlement-credit
// mechanism already used for chef referrals and loyalty cashback: a pending
// rupee credit, unique on SourceKey, admin-voidable, that
// ApplyChefBonusesToStatement folds onto the next weekly statement. A cancelled
// order therefore still cannot reach the transfer-release path at all.
//
// ── The solvency cap is the safety property ──────────────────────────────────
//
// The entitlement is NOT simply VendorKeptPaise. It is
//
//	min(VendorKeptPaise, capture − alreadyRefunded)
//
// re-derived LIVE from the order at raise time. The snapshot alone cannot be
// trusted: ComputeCancellationRefund computes against
// subtotal+delivery+fee+tax, which EXCEEDS order.Total on a discounted order
// (it does not model order.Discount), and CappedAt then attributes the phantom
// gap to the vendor. Production carries exactly such a row — a 100%-refunded
// order whose snapshot still claims VendorKept ₹40.53. Paying that figure would
// credit a chef out of a fully-refunded capture: precisely the failure the
// payout guards exist to prevent. The cap makes it structurally impossible —
// a chef can only ever be credited money the platform still holds for that
// order. The underlying discount defect is tracked separately; this cap is
// correct regardless of whether it is fixed.

import (
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// cancellationEntitlementKeyPrefix namespaces the ChefBonus SourceKey. Keyed by
// ORDER id (cancellation_requests.order_id is unique) so the inline raise and
// the sweep converge on the same row and the unique index makes a double-raise
// impossible.
const cancellationEntitlementKeyPrefix = "cancelkept:"

// CancellationEntitlementSourceKey is the natural key of a chef's retained-share
// credit for one cancelled order.
func CancellationEntitlementSourceKey(orderID uuid.UUID) string {
	return cancellationEntitlementKeyPrefix + orderID.String()
}

// terminalCancellationStatuses are the request states in which the vendor's tier
// is SETTLED and the retained share is genuinely owed.
//
// `disputed` and `admin_review` are deliberately excluded: the customer has
// contested the tier (or it timed out to an admin) and the outcome can still
// move money BACK to the customer, shrinking VendorKeptPaise. Crediting the chef
// mid-dispute would need a claw-back that does not exist. Once an admin settles
// the request the status becomes `resolved` and the sweep picks it up on its
// next pass — held, not lost.
func terminalCancellationStatuses() []models.CancellationRequestStatus {
	return []models.CancellationRequestStatus{
		models.CancelReqApproved,
		models.CancelReqAutoRefunded,
		models.CancelReqResolved,
	}
}

func isTerminalCancellation(s models.CancellationRequestStatus) bool {
	for _, t := range terminalCancellationStatuses() {
		if s == t {
			return true
		}
	}
	return false
}

// CancellationEntitlement is the outcome of asking "what, if anything, does this
// cancellation owe the chef?". Reason is always populated when Payable is false,
// so a skip is explainable in a log or an admin screen rather than silent.
type CancellationEntitlement struct {
	Paise   int
	Payable bool
	Reason  string
}

// ComputeCancellationEntitlement is the pure decision: eligibility, then the
// solvency cap. No DB, no side effects — the money rule is exhaustively
// unit-testable in isolation, the same way ComputeCancellationRefund is.
//
// capturePaise is what the customer actually paid for the order (order.Total)
// and refundedPaise what has actually been returned to them (order.RefundAmount)
// — both read live, not from the snapshot.
func ComputeCancellationEntitlement(
	status models.CancellationRequestStatus,
	refundExecuted bool,
	vendorKeptPaise int,
	orderCancelled bool,
	capturePaise int,
	refundedPaise int,
) CancellationEntitlement {
	switch {
	case !refundExecuted:
		return CancellationEntitlement{Reason: "refund has not been executed yet"}
	case !isTerminalCancellation(status):
		return CancellationEntitlement{Reason: fmt.Sprintf("cancellation is %s, not settled", status)}
	case !orderCancelled:
		return CancellationEntitlement{Reason: "order is not cancelled/refunded"}
	case vendorKeptPaise <= 0:
		return CancellationEntitlement{Reason: "nothing was retained for the vendor"}
	}
	// The solvency test: never credit more than the platform still holds for this
	// order. On a well-formed snapshot this is a no-op (conservation already
	// guarantees vendorKept <= capture − refunded); it only bites on a row whose
	// snapshot over-attributes to the vendor.
	unrefunded := capturePaise - refundedPaise
	if unrefunded <= 0 {
		return CancellationEntitlement{
			Reason: "the capture was fully refunded — no money is held for this order",
		}
	}
	paise := vendorKeptPaise
	capped := ""
	if paise > unrefunded {
		paise = unrefunded
		capped = " (capped at the unrefunded capture)"
	}
	return CancellationEntitlement{Paise: paise, Payable: true, Reason: "vendor retained share" + capped}
}

// entitlementFor resolves an order + request pair into the decision above.
func entitlementFor(order *models.Order, cr *models.CancellationRequest) CancellationEntitlement {
	orderSettled := order.Status == models.OrderStatusCancelled ||
		order.Status == models.OrderStatusRefunded ||
		order.RefundedAt != nil
	return ComputeCancellationEntitlement(
		cr.Status, cr.RefundExecuted, cr.VendorKeptPaise, orderSettled,
		ToPaise(order.Total), ToPaise(order.RefundAmount),
	)
}

// RaiseCancellationRetainedBonus records the chef's retained share of a cancelled
// order as a pending settlement credit, so it rides their next weekly statement.
//
// Idempotent on the SourceKey unique index — the inline call and the sweep can
// both run for the same order and the chef is credited exactly once. Returns nil
// (not an error) when nothing is owed; a skip is a normal outcome, not a failure.
func RaiseCancellationRetainedBonus(db *gorm.DB, order *models.Order, cr *models.CancellationRequest) error {
	e := entitlementFor(order, cr)
	if !e.Payable {
		return nil
	}
	// The settlement join is by chef user, so resolve it once here (mirrors
	// ChefPenalty/ChefBonus, both of which denormalize user_id for that reason).
	var chef models.ChefProfile
	if err := db.Select("id", "user_id").First(&chef, "id = ?", cr.ChefID).Error; err != nil {
		return fmt.Errorf("cancellation-entitlement: load chef %s: %w", cr.ChefID, err)
	}
	if chef.UserID == uuid.Nil {
		return fmt.Errorf("cancellation-entitlement: chef %s has no user", cr.ChefID)
	}
	reason := fmt.Sprintf("Retained share of cancelled order %s (%s)", order.OrderNumber, cr.VendorReason)
	return raiseChefBonus(db, cr.ChefID, chef.UserID, models.ChefBonusCancellationRetained,
		CancellationEntitlementSourceKey(order.ID), FromPaise(e.Paise), nil, reason)
}

// SweepCancellationChefEntitlements raises the retained-share credit for every
// settled cancellation that does not have one yet.
//
// It is the backstop for an inline raise that failed (the inline call is
// best-effort — a payable must never fail a refund that already moved money),
// the path that picks up a request only settled later by an admin, AND the
// backfill for the cancellations that resolved before this existed. One
// idempotent code path covers all three, so the backfill is the same tested
// mechanism as the steady state rather than a one-off script.
func SweepCancellationChefEntitlements() {
	var owed []models.CancellationRequest
	if err := database.DB.
		Where("status IN ? AND refund_executed = ? AND vendor_kept_paise > 0",
			terminalCancellationStatuses(), true).
		Limit(200).Find(&owed).Error; err != nil {
		log.Printf("cancellation-entitlement: query owed entitlements failed: %v", err)
		return
	}
	// Filter already-credited rows in Go rather than with a `prefix || order_id`
	// NOT EXISTS: that concatenation needs a ::text cast on Postgres (uuid column)
	// and none on the sqlite-backed unit tests, so the SQL cannot be written once
	// for both. The candidate set is small and bounded by the LIMIT above.
	keys := make([]string, 0, len(owed))
	for i := range owed {
		keys = append(keys, CancellationEntitlementSourceKey(owed[i].OrderID))
	}
	credited := map[string]bool{}
	if len(keys) > 0 {
		var existing []string
		if err := database.DB.Model(&models.ChefBonus{}).
			Where("source_key IN ?", keys).Pluck("source_key", &existing).Error; err != nil {
			log.Printf("cancellation-entitlement: load existing credits failed: %v", err)
			return
		}
		for _, k := range existing {
			credited[k] = true
		}
	}
	for i := range owed {
		if credited[CancellationEntitlementSourceKey(owed[i].OrderID)] {
			continue
		}
		var order models.Order
		if err := database.DB.First(&order, "id = ?", owed[i].OrderID).Error; err != nil {
			continue
		}
		if err := RaiseCancellationRetainedBonus(database.DB, &order, &owed[i]); err != nil {
			log.Printf("cancellation-entitlement: raise for order %s failed (will retry): %v",
				owed[i].OrderID, err)
		}
	}
}
