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
// the normal delivery flow) is parked as ONE payout hold on delivery, released
// through the statement path and reversed on cancellation.

// GroupChefPayout is the chef's GROSS slice of a group order (subtotal + tax) — the
// customer-facing food value, used as the queue's context Amount. The chef is paid
// groupNetPayout, NOT this.
func GroupChefPayout(g *models.GroupOrder) float64 { return g.Subtotal + g.Tax }

// groupNetPayout is the chef's NET payout for a group order — the amount the hold
// must carry (#546). It mirrors ComputeOrderEarnings / perDayNetPayout
// so group orders settle the chef on the SAME basis as regular orders and meal-plan
// days, instead of paying the gross slice with no platform commission or TDS:
//
//	gross      = subtotal + tax          (chef food income; delivery/service fee are the platform's, excluded)
//	commission = rate × subtotal         (platform commission on the food subtotal only)
//	tds        = RateTDS × gross          (§194-O, on gross)
//	net        = gross − commission − tds
func groupNetPayout(g *models.GroupOrder, rate float64) float64 {
	if rate <= 0 || rate >= 1 {
		rate = DefaultCommissionRate
	}
	gross := g.Subtotal + g.Tax
	commission := rate * g.Subtotal
	tds := RateTDS * gross
	return Round2(gross - commission - tds)
}

// ReverseGroupHoldForCancel drives a CANCELLED group order's chef payout hold to
// reversed (#456 W-A). Self-guards on status==cancelled, so it is safe to call on
// BOTH the cancel success path AND the already-cancelled conflict/retry path
// (crash-window recovery) and NEVER reverses a delivered group. The transition is a
// guarded conditional UPDATE (idempotent: a second call no-ops once the hold is
// terminal), and settleReverse stamps payout_settled_at. The chef is paid for a
// group order on the statement path, which this reversed hold keeps it off.
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
			// Exclude `failed` too (#594): a late/duplicate delivered event must not flip a
			// delivery-FAILED group (hold frozen disputed) back to delivered and park its
			// hold to awaiting — that would orphan the disputed hold and hide the group from
			// admin resolution. A failed group is resolved only by the group-resolution path.
			[]models.GroupOrderStatus{models.GroupOrderDelivered, models.GroupOrderCancelled, models.GroupOrderFailed}).
		Updates(map[string]any{"status": models.GroupOrderDelivered, "delivered_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil
	}
	return SetGroupOrderHoldAwaitingConfirmation(tx, groupID)
}
