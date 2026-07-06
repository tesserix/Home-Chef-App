package services

// delivery_failure.go — #393 slice 1. Terminalize a failed delivery's MONEY state
// without moving money: open a pending `delivery_failed` OrderIssue (the dispute signal
// the admin queue surfaces) and freeze the order's payout hold to `disputed`, so the
// chef is not paid until an admin confirms fault. The actual money outcome (refund vs
// release, per the confirmed fault class) is executed by the admin-confirm path in a
// later slice — this slice only freezes.

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
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
