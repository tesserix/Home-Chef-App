package services

// group_order_resolve.go — #594 slice B. Execute the admin-confirmed money policy for a
// delivery-FAILED group order (frozen `disputed` by slice A). The group analog of
// ResolveMealPlanDayFailure. Hybrid model (owner policy):
//   - customer-fault → NO refund; the chef is paid: claim status failed→delivered (the
//     paid-terminal group state) + drive the hold disputed→release_eligible for the admin
//     payout queue.
//   - platform-fault / chef-fault → FULL refund: claim failed→cancelled, refund every paid
//     participant to their wallet (RefundGroupParticipant), cancel the consolidated order,
//     and reverse the chef's held DIRECT transfer via ReverseGroupHoldForCancel — exactly
//     the group cancel-refund flow (that reverse MUST run post-commit so its guarded
//     disputed→reversed transition also triggers settleReverse's transfer claw-back).
//   - ambiguous / unknown → ErrAmbiguousFault.

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ResolveGroupOrderFailure executes the admin-confirmed money policy for a failed group.
// Idempotent via a claim on the `failed` status (a concurrent/duplicate resolution loses
// and returns ErrIssueAlreadyHandled). `group` MUST be loaded with its Participants (the
// refund targets on the platform/chef-fault path).
func ResolveGroupOrderFailure(db *gorm.DB, group *models.GroupOrder, fault models.DeliveryFaultClass, adminID uuid.UUID) error {
	if group.Status != models.GroupOrderFailed {
		return ErrNotDeliveryFailure
	}
	switch fault {
	case models.FaultCustomer:
		return db.Transaction(func(tx *gorm.DB) error {
			ok, err := claimFailedGroup(tx, group.ID, models.GroupOrderDelivered, true)
			if err != nil {
				return err
			}
			if !ok {
				return ErrIssueAlreadyHandled
			}
			// Chef legitimately earned the payout → move the disputed hold into the pay queue.
			return releaseDisputedGroupHoldForCustomerFault(tx, group.ID)
		})
	case models.FaultPlatform, models.FaultChef:
		var claimed bool
		if err := db.Transaction(func(tx *gorm.DB) error {
			ok, err := claimFailedGroup(tx, group.ID, models.GroupOrderCancelled, false)
			if err != nil {
				return err
			}
			if !ok {
				return ErrIssueAlreadyHandled
			}
			claimed = true
			// Refund every paid participant to their wallet (NOT flag-gated — the real
			// customer money always comes back) and notify them.
			for i := range group.Participants {
				if err := RefundGroupParticipant(tx, &group.Participants[i], "group delivery failed — "+string(fault)+"-fault refund"); err != nil {
					return err
				}
				if err := EnqueueEvent(tx, SubjectGroupOrderCancelled, "group_order.cancelled", group.Participants[i].UserID, map[string]any{
					"group_order_id": group.ID.String(), "reason": "delivery failed",
				}); err != nil {
					return err
				}
			}
			// Cancel the consolidated order too (blocks any stray payout release; #457).
			if group.OrderID != nil {
				if err := tx.Model(&models.Order{}).Where("id = ?", *group.OrderID).
					Update("status", models.OrderStatusCancelled).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		// POST-commit (mirrors the cancel handler): drive the chef payout hold
		// disputed→reversed and claw back the held direct transfer. This MUST run outside
		// the tx and be the sole hold-transition on this path — ReverseGroupHoldForCancel
		// only claws back the transfer when its own guarded transition fires, so a pre-tx
		// hold flip would strand the transfer. Best-effort: the payout-reconcile cron
		// re-drives a reversed-but-unsettled group; releaseBlockedForAgg blocks release
		// meanwhile (cancelled consolidated order).
		if claimed {
			if err := ReverseGroupHoldForCancel(db, group.ID, "group delivery failed — refund"); err != nil {
				log.Printf("group-order resolve: reverse chef payout for %s failed (reconcile will re-drive): %v", group.ID, err)
			}
		}
		return nil
	default: // FaultAmbiguous or anything unrecognized
		return ErrAmbiguousFault
	}
}

// claimFailedGroup is the concurrency/idempotency gate: a guarded UPDATE flipping the group
// out of `failed` to the terminal `to` status; ok=false (no error) when it is no longer
// `failed` (already resolved). stampDelivered/cancelled set the matching timestamp.
func claimFailedGroup(tx *gorm.DB, groupID uuid.UUID, to models.GroupOrderStatus, stampDelivered bool) (bool, error) {
	updates := map[string]any{"status": to}
	if stampDelivered {
		updates["delivered_at"] = time.Now()
	} else {
		updates["cancelled_at"] = time.Now()
		updates["cancel_reason"] = "group delivery failed — refunded"
	}
	res := tx.Model(&models.GroupOrder{}).
		Where("id = ? AND status = ?", groupID, models.GroupOrderFailed).
		Updates(updates)
	if res.Error != nil {
		return false, fmt.Errorf("group-order resolve: claim group %s: %w", groupID, res.Error)
	}
	return res.RowsAffected == 1, nil
}

// releaseDisputedGroupHoldForCustomerFault drives a customer-fault group's hold
// disputed→release_eligible (the explicit admin authorization to pay the chef), emitting
// the release-eligible event for the #388 admin payout queue. Guarded to disputed only.
func releaseDisputedGroupHoldForCustomerFault(tx *gorm.DB, groupID uuid.UUID) error {
	res := tx.Model(&models.GroupOrder{}).
		Where("id = ? AND payout_hold_status = ?", groupID, models.PayoutHoldDisputed).
		Update("payout_hold_status", models.PayoutHoldReleaseEligible)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: release customer-fault group %s: %w", groupID, res.Error)
	}
	if res.RowsAffected == 0 {
		// Slice A always freezes the hold to disputed atomically with the failed status, so
		// a zero-match means the group was marked paid-terminal but its hold is NOT queued —
		// surface it rather than silently succeed.
		log.Printf("group-order resolve: customer-fault group %s had no disputed hold to release — hold NOT queued for payout", groupID)
		return nil
	}
	return emitHoldEvent(tx, models.PayoutHoldReleaseEligible, aggTypeGroupOrder, groupID)
}
