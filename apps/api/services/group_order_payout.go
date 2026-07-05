package services

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// group_order_payout.go — chef payout for group/office orders (#46). Participants
// pay their shares into the platform balance; on consolidation the chef's slice
// (items + tax — delivery/service stay with the platform, the driver is paid via
// the normal delivery flow) is paid as ONE on-hold Route transfer, released on
// delivery and reversed on cancellation. Mirrors the meal-plan escrow primitives.

// GroupChefPayout is the chef's slice of a group order (subtotal + tax).
func GroupChefPayout(g *models.GroupOrder) float64 { return g.Subtotal + g.Tax }

// HoldGroupChefPayout creates the single on-hold Route transfer to the chef and
// stamps PayoutTransferID. Idempotent (skips when already held or no account).
// Flag-gated on payoutMovementEnabled() (#456): with escrow OFF at launch this
// moves no live money — the group hold is driven purely as DB state, exactly like
// order/meal-plan-day, until the flag flips.
func HoldGroupChefPayout(tx *gorm.DB, g *models.GroupOrder, chefAccount string) error {
	if !payoutMovementEnabled() {
		return nil
	}
	if g.PayoutTransferID != "" || chefAccount == "" {
		return nil
	}
	amt := GroupChefPayout(g)
	if amt <= 0 {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return fmt.Errorf("razorpay not configured")
	}
	tr, err := rz.CreateTransfer(&DirectTransferRequest{
		Account: chefAccount, Amount: ToPaise(amt), Currency: g.Currency, OnHold: true,
		Notes: map[string]string{"group_order_id": g.ID.String()},
	})
	if err != nil {
		return fmt.Errorf("hold group payout: %w", err)
	}
	g.PayoutTransferID = tr.ID
	return tx.Model(&models.GroupOrder{}).Where("id = ?", g.ID).
		Update("payout_transfer_id", tr.ID).Error
}

// ReleaseGroupChefPayout releases the held transfer. Since #456 no delivery path
// calls this directly (delivery parks a hold instead); it is the seam the admin
// payout queue drives off release_eligible. Flag-gated on payoutMovementEnabled()
// (#456 P0 stop-the-bleed) — OFF ⇒ no money moves.
func ReleaseGroupChefPayout(g *models.GroupOrder) error {
	if !payoutMovementEnabled() {
		return nil
	}
	if g.PayoutTransferID == "" {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return nil // gateway unconfigured — no-op like ReleaseOrderPayouts
	}
	if _, err := rz.ReleaseTransfer(g.PayoutTransferID); err != nil && !isAlreadyReleasedErr(err) {
		return fmt.Errorf("release group payout %s: %w", g.PayoutTransferID, err)
	}
	return nil
}

// ReverseGroupChefPayout claws the held transfer back to the platform on cancel.
// Flag-gated on payoutMovementEnabled() (#456) — OFF ⇒ no money moves. Returns a real
// gateway error (tolerating an already-reversed transfer) so settlePayout does NOT
// stamp payout_settled_at on a failed claw-back — the reconcile cron then re-drives it
// (#508). Previously it swallowed the error, silently stranding a chef net-paid.
func ReverseGroupChefPayout(g *models.GroupOrder) error {
	if !payoutMovementEnabled() {
		return nil
	}
	if g.PayoutTransferID == "" {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return nil
	}
	if _, err := rz.ReverseTransfer(g.PayoutTransferID, 0); err != nil && !isAlreadyReversedErr(err) {
		return fmt.Errorf("group-order: reverse payout %s: %w", g.PayoutTransferID, err)
	}
	return nil
}

// ReverseGroupHoldForCancel drives a CANCELLED group order's chef payout hold to
// reversed and claws back the held direct transfer (#456 W-A — replaces the old
// unconditional pre-tx ReverseGroupChefPayout in the cancel handler). Self-guards on
// status==cancelled, so it is safe to call on BOTH the cancel success path AND the
// already-cancelled conflict/retry path (crash-window recovery) and NEVER reverses a
// delivered group. The status transition is a guarded conditional UPDATE (idempotent:
// a second call no-ops once the hold is terminal), and settleReverse runs the
// flag-gated reverse seam (ReverseGroupChefPayout, keyed on PayoutTransferID) +
// stamps payout_settled_at. The group chef transfer is a DIRECT transfer that the
// participant wallet refunds do NOT auto-reverse, so this explicit reverse is required.
func ReverseGroupHoldForCancel(db *gorm.DB, groupID uuid.UUID, reason string) error {
	var g models.GroupOrder
	if err := db.Select("status", "payout_hold_status").First(&g, "id = ?", groupID).Error; err != nil {
		return fmt.Errorf("group-order: load %s for cancel reverse: %w", groupID, err)
	}
	if g.Status != models.GroupOrderCancelled {
		return nil // only a cancelled group's payout is clawed back — never a delivered one
	}
	ok, err := transitionHold(db, aggTypeGroupOrder, groupID,
		[]models.PayoutHoldStatus{
			models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation,
			models.PayoutHoldReleaseEligible, models.PayoutHoldReleased, models.PayoutHoldDisputed,
		}, models.PayoutHoldReversed, false)
	if err != nil {
		return fmt.Errorf("group-order: reverse hold %s: %w", groupID, err)
	}
	if !ok {
		return nil // already withheld/reversed — idempotent
	}
	return settleReverse(db, aggTypeGroupOrder, groupID)
}

// RefundGroupParticipant refunds one paid participant to wallet (idempotent on the
// participant id) and flips them to refunded.
func RefundGroupParticipant(tx *gorm.DB, p *models.GroupOrderParticipant, reason string) error {
	if p.RefundTxnID != nil || p.PaymentStatus != models.GroupPayCompleted || p.ShareAmount <= 0 {
		return nil
	}
	txn, err := CreditWallet(tx, p.UserID, p.ShareAmount, models.WalletSourceRefund, nil,
		"Group order — "+reason, "grouporder-refund:"+p.ID.String(), nil)
	if err != nil {
		return fmt.Errorf("refund participant %s: %w", p.ID, err)
	}
	p.RefundTxnID = &txn.ID
	return tx.Model(&models.GroupOrderParticipant{}).Where("id = ?", p.ID).
		Updates(map[string]any{"refund_txn_id": txn.ID, "payment_status": models.GroupPayRefunded}).Error
}

// MarkGroupOrderDelivered is the delivery-pipeline hook: when the consolidated
// order is delivered, mark the group delivered and PARK its chef payout in a
// customer-confirmation hold — it no longer releases money on delivery (#456).
// Delivered no longer implies paid: the host confirming advances the hold to
// release_eligible, which the admin payout queue drives (flag-gated). Safe +
// idempotent on any order (no-op if not a group order or already delivered).
func MarkGroupOrderDelivered(orderID uuid.UUID) {
	var g models.GroupOrder
	if err := database.DB.Where("order_id = ?", orderID).First(&g).Error; err != nil {
		return // not a group order
	}
	if g.Status == models.GroupOrderDelivered {
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return parkGroupOrderOnDelivery(tx, g.ID)
	}); err != nil {
		log.Printf("group-order: mark delivered for order %s failed: %v", orderID, err)
	}
}

// parkGroupOrderOnDelivery runs the guarded delivered transition and, only when it
// genuinely advances the row, stamps delivered_at + parks the payout hold in the
// same tx. The WHERE guard makes a replayed delivered event a no-op.
func parkGroupOrderOnDelivery(tx *gorm.DB, groupID uuid.UUID) error {
	// Exclude cancelled too (#534): a late/duplicate delivered event must not flip a
	// just-cancelled group back to delivered and park its hold to awaiting, which
	// would defeat ReverseGroupHoldForCancel's status==cancelled self-guard and
	// abandon the claw-back. RowsAffected==0 on a cancelled group → no-op.
	res := tx.Model(&models.GroupOrder{}).
		Where("id = ? AND status NOT IN ?", groupID,
			[]models.GroupOrderStatus{models.GroupOrderDelivered, models.GroupOrderCancelled}).
		Updates(map[string]any{"status": models.GroupOrderDelivered, "delivered_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return SetGroupOrderHoldAwaitingConfirmation(tx, groupID)
}
