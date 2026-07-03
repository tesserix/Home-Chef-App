package services

import (
	"fmt"
	"log"

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
		return fmt.Errorf("razorpay not configured")
	}
	if _, err := rz.ReleaseTransfer(g.PayoutTransferID); err != nil {
		return fmt.Errorf("release group payout %s: %w", g.PayoutTransferID, err)
	}
	return nil
}

// ReverseGroupChefPayout claws the held transfer back to the platform on cancel.
// Flag-gated on payoutMovementEnabled() (#456) — OFF ⇒ no money moves.
func ReverseGroupChefPayout(g *models.GroupOrder) {
	if !payoutMovementEnabled() {
		return
	}
	if g.PayoutTransferID == "" {
		return
	}
	rz := GetRazorpay()
	if rz == nil {
		return
	}
	if _, err := rz.ReverseTransfer(g.PayoutTransferID, 0); err != nil {
		log.Printf("group-order: reverse payout %s failed: %v", g.PayoutTransferID, err)
	}
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
// order is delivered, mark the group delivered + release the chef payout. Safe +
// idempotent on any order (no-op if not a group order or already delivered).
func MarkGroupOrderDelivered(orderID uuid.UUID) {
	var g models.GroupOrder
	if err := database.DB.Where("order_id = ?", orderID).First(&g).Error; err != nil {
		return // not a group order
	}
	if g.Status == models.GroupOrderDelivered {
		return
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.GroupOrder{}).
			Where("id = ? AND status <> ?", g.ID, models.GroupOrderDelivered).
			Update("status", models.GroupOrderDelivered)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		return ReleaseGroupChefPayout(&g)
	})
	if err != nil {
		log.Printf("group-order: mark delivered for order %s failed: %v", orderID, err)
	}
}
