package services

// delivery_failure.go — #393 slice 1. Terminalize a failed delivery's MONEY state
// without moving money: open a pending `delivery_failed` OrderIssue (the dispute signal
// the admin queue surfaces) and freeze the order's payout hold to `disputed`, so the
// chef is not paid until an admin confirms fault. The actual money outcome (refund vs
// release, per the confirmed fault class) is executed by the admin-confirm path in a
// later slice — this slice only freezes.

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ErrAmbiguousFault is returned when the admin has not confirmed a concrete fault class
// for a delivery-failure resolution — an ambiguous case must not auto-execute a money
// outcome. ErrNotDeliveryFailure guards the resolver to delivery_failed issues only.
// ErrIssueAlreadyHandled signals the pending claim was lost to a concurrent resolution.
var (
	ErrAmbiguousFault      = errors.New("ambiguous fault requires a concrete admin decision")
	ErrNotDeliveryFailure  = errors.New("issue is not a delivery-failure issue")
	ErrIssueAlreadyHandled = errors.New("issue has already been handled")
)

// SetOrderHoldDisputed freezes a regular (gateway) order's payout hold at `disputed`
// from a PRE-TERMINAL state (none / awaiting_customer_confirmation). A terminally-failed
// delivery never reaches the customer-confirmation path that normally sets disputed, so
// this is the explicit freeze primitive. Guarded conditional UPDATE: a hold already
// release_eligible / released / reversed / withheld is untouched (the #458 invariant —
// disputed must never un-settle real money movement). Emits payout.hold_disputed onto
// the outbox on a genuine transition. Plain DB state — runs regardless of the escrow
// flags (no money moves; the chef's held transfer simply stays held).
func SetOrderHoldDisputed(tx *gorm.DB, orderID uuid.UUID) error {
	res := tx.Model(&models.Order{}).
		Where("id = ? AND payout_hold_status IN ?", orderID,
			[]models.PayoutHoldStatus{models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation}).
		Update("payout_hold_status", models.PayoutHoldDisputed)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: dispute order %s: %w", orderID, res.Error)
	}
	if res.RowsAffected == 0 {
		return nil // already disputed/settled/eligible — nothing to freeze, nothing to emit
	}
	return emitHoldEvent(tx, models.PayoutHoldDisputed, "order", orderID)
}

// RecordDeliveryFailure terminalizes a failed delivery's money state WITHOUT moving
// money: it opens a pending `delivery_failed` OrderIssue (recording the reported reason
// + suggested fault class for the admin-confirm step) and freezes the order's payout
// hold to `disputed`, atomically within tx. Returns froze=true only on the FIRST
// terminalization of the order, so the caller emits the failure notification exactly
// once. Idempotent — a re-fired failure finds the existing pending issue and returns
// froze=false without opening a second issue or re-disputing. Regular gateway orders
// only; meal-plan/group consolidated orders (no razorpay_order_id) settle through their
// own paths and are terminalized by a later slice (returns froze=false).
func RecordDeliveryFailure(tx *gorm.DB, order *models.Order, reason models.DeliveryFailureReason) (bool, error) {
	if order.RazorpayOrderID == "" {
		return false, nil
	}
	var pending int64
	if err := tx.Model(&models.OrderIssue{}).
		Where("order_id = ? AND reason = ? AND status = ?", order.ID, models.IssueDeliveryFailed, models.IssuePending).
		Count(&pending).Error; err != nil {
		return false, fmt.Errorf("delivery-failure: check existing issue for order %s: %w", order.ID, err)
	}
	if pending > 0 {
		return false, nil // already terminalized — idempotent no-op
	}
	fault := models.SuggestedFaultClass(reason)
	issue := models.OrderIssue{
		OrderID:     order.ID,
		ChefID:      order.ChefID,
		CustomerID:  order.CustomerID,
		Reason:      models.IssueDeliveryFailed,
		Status:      models.IssuePending,
		Description: fmt.Sprintf("delivery failed: reason=%s suggested_fault=%s", reason, fault),
	}
	if err := tx.Create(&issue).Error; err != nil {
		return false, fmt.Errorf("delivery-failure: open issue for order %s: %w", order.ID, err)
	}
	if err := SetOrderHoldDisputed(tx, order.ID); err != nil {
		return false, err
	}
	return true, nil
}

// ResolveDeliveryFailure executes the admin-confirmed money policy for a pending
// `delivery_failed` OrderIssue (#393 slice 3, hybrid model). The driver's reported reason
// only SUGGESTED a fault; the admin CONFIRMS a concrete class here and the matching
// outcome executes, reusing the existing money seams:
//   - customer-fault → NO customer refund; the chef is paid (food was made): the issue is
//     rejected and the disputed hold driven back to release_eligible. The delivery fee is
//     retained automatically (no refund happens).
//   - platform-fault / chef-fault → FULL customer refund + reverse the chef's hold:
//     RefundIssueToWallet credits the order's remaining refundable to the wallet and
//     cross-guards the hold to reversed (WithholdOrReverseOrderHoldForRefund).
//   - ambiguous / unknown → ErrAmbiguousFault; the admin must confirm a concrete fault.
//
// Idempotent via the pending-issue claim (a concurrent resolution loses with
// ErrIssueAlreadyHandled). The confirmed fault is appended to the issue description for
// audit.
func ResolveDeliveryFailure(db *gorm.DB, issue *models.OrderIssue, fault models.DeliveryFaultClass, adminID uuid.UUID) error {
	if issue.Reason != models.IssueDeliveryFailed {
		return ErrNotDeliveryFailure
	}
	switch fault {
	case models.FaultCustomer:
		now := time.Now()
		return db.Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&models.OrderIssue{}).
				Where("id = ? AND status = ?", issue.ID, models.IssuePending).
				Updates(map[string]any{
					"status":      models.IssueRejected,
					"resolved_at": now,
					"resolved_by": adminID,
					"description": confirmedFaultDescription(issue.Description, fault),
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return ErrIssueAlreadyHandled
			}
			// No refund; the chef legitimately earned the payout → clear the disputed hold.
			return ReleaseDisputedHoldsForOrderIfCleared(tx, issue.OrderID)
		})
	case models.FaultPlatform, models.FaultChef:
		// Full customer refund + reverse the chef's hold. RefundIssueToWallet caps the
		// credit at the order's remaining refundable, so passing the order total yields a
		// full refund, and its built-in cross-guard reverses the hold. Record the fault.
		var order models.Order
		if err := db.Select("total", "refund_amount").First(&order, "id = ?", issue.OrderID).Error; err != nil {
			return fmt.Errorf("delivery-failure resolve: load order %s: %w", issue.OrderID, err)
		}
		if err := RefundIssueToWallet(db, issue, order.Total, "admin", &adminID); err != nil {
			return err
		}
		// Stamp the confirmed fault ONLY if THIS call actually resolved the issue.
		// RefundIssueToWallet sets issue.Status on the winning claim; a lost cross-path race
		// (a concurrent resolution already handled it) leaves issue.Status unchanged, so we
		// must not clobber the winner's audit trail with our (losing) fault decision.
		if issue.Status != models.IssueResolved {
			return nil
		}
		// Best-effort audit stamp — the refund + hold change already committed, so a failed
		// description write must NOT surface as a failure (which would 500 a succeeded refund).
		if err := db.Model(&models.OrderIssue{}).Where("id = ?", issue.ID).
			Update("description", confirmedFaultDescription(issue.Description, fault)).Error; err != nil {
			log.Printf("delivery-failure resolve: audit-stamp fault for issue %s: %v", issue.ID, err)
		}
		return nil
	default: // FaultAmbiguous or anything unrecognized
		return ErrAmbiguousFault
	}
}

// confirmedFaultDescription appends the admin-confirmed fault class to the issue
// description (audit trail alongside the driver's originally-suggested fault).
func confirmedFaultDescription(existing string, fault models.DeliveryFaultClass) string {
	return existing + " | admin_confirmed_fault=" + string(fault)
}
