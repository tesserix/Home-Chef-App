package services

import (
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// order_payout.go — settle / claw-back of a regular order's held chef/rider Route
// transfers (#217, #123). Regular-order payouts are payment-linked transfers
// created OnHold at checkout (orderSettlements):
//
//   - ReleaseOrderPayouts clears the hold once the food is delivered (the saga's
//     settle step), closing the gap where they were never auto-released.
//   - ReverseOrderPayouts claws the held/settled transfers back to the platform
//     when an order is refunded (the saga's compensation), so the chef/rider are
//     not paid for an order the customer was refunded.
//
// Both are gated by ORDER_PAYOUT_AUTO_RELEASE_ENABLED (default OFF) because they
// move live settlement — verify capture→hold→release/reverse in the Razorpay
// sandbox (#218) first. Both are no-ops for orders without a Razorpay order id
// (meal-plan/group consolidated orders settle through their own paths) and both
// return an error so the Temporal activity that wraps them retries on a transient
// gateway failure.

// payoutMovementEnabled reports whether live transfer release/reversal is on.
func payoutMovementEnabled() bool {
	return config.AppConfig != nil && config.AppConfig.OrderPayoutAutoReleaseEnabled
}

// orderRazorpayID loads the order's Razorpay order id, or "" if it isn't a
// gateway-charged regular order.
func orderRazorpayID(orderID uuid.UUID) (string, error) {
	var order models.Order
	if err := database.DB.Select("id", "razorpay_order_id").First(&order, "id = ?", orderID).Error; err != nil {
		return "", err
	}
	return order.RazorpayOrderID, nil
}

// ReleaseOrderPayouts releases any held Route transfers on a delivered order.
func ReleaseOrderPayouts(orderID uuid.UUID) error {
	if !payoutMovementEnabled() {
		return nil
	}
	rzOrderID, err := orderRazorpayID(orderID)
	if err != nil {
		return err
	}
	if rzOrderID == "" {
		return nil // not a gateway-charged regular order
	}
	rz := GetRazorpay()
	if rz == nil {
		return nil
	}
	transfers, err := rz.FetchOrderTransfers(rzOrderID)
	if err != nil {
		return fmt.Errorf("order-payout: fetch transfers for order %s: %w", orderID, err)
	}
	for _, t := range transfers {
		if t.OnHold && t.ID != "" {
			if _, err := rz.ReleaseTransfer(t.ID); err != nil {
				return fmt.Errorf("order-payout: release transfer %s (order %s): %w", t.ID, orderID, err)
			}
		}
	}
	return nil
}

// ReverseOrderPayouts reverses (claws back) the order's Route transfers to the
// platform balance — the compensation when a paid order is refunded. Best-effort
// per transfer reversal is logged; a fetch failure is returned so the activity
// retries. Order-level idempotency comes from the caller (the saga only
// compensates once, guarded on RefundedAt).
func ReverseOrderPayouts(orderID uuid.UUID) error {
	if !payoutMovementEnabled() {
		return nil
	}
	rzOrderID, err := orderRazorpayID(orderID)
	if err != nil {
		return err
	}
	if rzOrderID == "" {
		return nil
	}
	rz := GetRazorpay()
	if rz == nil {
		return nil
	}
	transfers, err := rz.FetchOrderTransfers(rzOrderID)
	if err != nil {
		return fmt.Errorf("order-payout: fetch transfers for order %s: %w", orderID, err)
	}
	for _, t := range transfers {
		if t.ID == "" {
			continue
		}
		// amountPaise 0 = full reversal. A transfer already reversed errors on
		// Razorpay; logged (not fatal) so one bad transfer doesn't block the rest.
		if _, err := rz.ReverseTransfer(t.ID, 0); err != nil {
			log.Printf("order-payout: reverse transfer %s (order %s) failed: %v", t.ID, orderID, err)
		}
	}
	return nil
}
