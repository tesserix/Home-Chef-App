package services

// group_order_failure.go — #393/#594 slice A. Terminalize a failed delivery's MONEY state
// for a GROUP order without moving money. The consolidated group Order is a shell with no
// razorpay_order_id, so RecordDeliveryFailure (gateway-only) skips it and no meal_plan_days
// row exists either — so a failed group delivery would otherwise strand unfrozen. This
// slice marks the group `failed` (NON-terminal) and freezes its payout hold to `disputed`,
// so the chef's held direct transfer is not released until an admin resolves the group.
// The money outcome (refund participants + reverse the chef transfer, per the confirmed
// fault class) is executed by the group-resolution slice (B); this slice only freezes.

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// SetGroupOrderHoldDisputed freezes a group order's payout hold at `disputed` from a
// PRE-TERMINAL state (none / awaiting_customer_confirmation), mirroring
// SetMealPlanDayHoldDisputed / SetOrderHoldDisputed. A hold already
// release_eligible / released / reversed / withheld is untouched (the #458 invariant —
// disputed must never un-settle real money movement). Emits payout.hold_disputed on a
// genuine transition. Plain DB state — runs regardless of the escrow flags.
func SetGroupOrderHoldDisputed(tx *gorm.DB, groupID uuid.UUID) error {
	res := tx.Model(&models.GroupOrder{}).
		Where("id = ? AND payout_hold_status IN ?", groupID,
			[]models.PayoutHoldStatus{models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation}).
		Update("payout_hold_status", models.PayoutHoldDisputed)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: dispute group order %s: %w", groupID, res.Error)
	}
	if res.RowsAffected == 0 {
		return nil // already disputed/settled/eligible — nothing to freeze, nothing to emit
	}
	return emitHoldEvent(tx, models.PayoutHoldDisputed, aggTypeGroupOrder, groupID)
}

// terminalOrFailedGroupStatuses is the set of group statuses on which MarkGroupOrderFailed
// is a no-op: either already frozen (`failed`) or terminally resolved. Single source of
// truth shared with the delivery-failure reconcile sweep (mirrors
// terminalOrFailedDayStatuses) — the sweep selects only groups OUTSIDE this set so a
// selected strand is guaranteed to freeze. Keep freeze guard + sweep predicate on this one
// list so they cannot drift apart.
var terminalOrFailedGroupStatuses = []models.GroupOrderStatus{
	models.GroupOrderFailed, models.GroupOrderDelivered,
	models.GroupOrderCancelled, models.GroupOrderExpired,
}

// MarkGroupOrderFailed is the failure-path mirror of MarkGroupOrderDelivered: when a
// terminally-failed delivery's order is a consolidated group shell, find the group by its
// consolidated order id, mark it `failed` (NON-terminal — awaits admin resolution) and
// freeze its payout hold to disputed. The guarded UPDATE excludes every terminal group
// state AND `failed` itself, so a re-fired failure (or a late failure on an already-
// resolved group) is a froze=false no-op that never resurrects a terminal group. Returns
// froze=false (no error) when the order isn't a group order. Emits group_orders.failed
// only on a genuine transition. Called inside the terminalize transaction; no money moves.
func MarkGroupOrderFailed(tx *gorm.DB, orderID uuid.UUID) (bool, error) {
	var g models.GroupOrder
	if err := tx.Where("order_id = ?", orderID).First(&g).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // not a group order
		}
		return false, fmt.Errorf("group-order failure: load group for order %s: %w", orderID, err)
	}
	res := tx.Model(&models.GroupOrder{}).
		Where("id = ? AND status NOT IN ?", g.ID, terminalOrFailedGroupStatuses).
		Update("status", models.GroupOrderFailed)
	if res.Error != nil {
		return false, fmt.Errorf("group-order failure: mark group %s failed: %w", g.ID, res.Error)
	}
	if res.RowsAffected == 0 {
		return false, nil // already terminal/failed — idempotent no-op
	}
	if err := SetGroupOrderHoldDisputed(tx, g.ID); err != nil {
		return false, err
	}
	if err := EnqueueEvent(tx, SubjectGroupOrderFailed, "group_orders.failed", g.ID, map[string]any{
		"group_order_id": g.ID.String(), "order_id": orderID.String(),
	}); err != nil {
		return false, err
	}
	return true, nil
}
