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

// orderRazorpayID loads the order's Razorpay order id, or "" if the order has no
// Route transfers to act on.
//
// The provider check is load-bearing, not defensive. razorpay_order_id is the
// shared gateway-order-id column (see models.GatewayOrderIDColumn), so a Cashfree
// order has a NON-EMPTY value here — one that means nothing to Razorpay. Testing
// only for emptiness, as this did when Razorpay was the sole INR gateway, would
// hand a Cashfree order id to FetchOrderTransfers and get a 400 back on every
// release, reversal and reconcile sweep for that order.
//
// Returning "" for a Cashfree order is the CORRECT answer, not a workaround:
// nothing was split at the gateway, so there is genuinely no transfer to release
// or reverse. Its chef/rider money settles through the statement/payout path.
func orderRazorpayID(orderID uuid.UUID) (string, error) {
	var order models.Order
	if err := database.DB.Select("id", "razorpay_order_id", "payment_provider").
		First(&order, "id = ?", orderID).Error; err != nil {
		return "", err
	}
	if !models.ProviderSupportsGatewaySplit(order.PaymentProvider) {
		return "", nil
	}
	return order.RazorpayOrderID, nil
}

// ReleaseOrderPayouts releases any held Route transfers on a delivered order.
// Since #387 no delivery path calls this directly (delivery parks a hold instead);
// it remains the seam the admin payout queue (#388) will drive off release_eligible.
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
	rz := GetRazorpayFor(PaymentModeForOrder(orderID))
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
			auditTransferMovement(auditTransferRelease, aggTypeOrder, orderID, t.ID, t.Amount, "order delivered — payout released")
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
	rz := GetRazorpayFor(PaymentModeForOrder(orderID))
	if rz == nil {
		return nil
	}
	transfers, err := rz.FetchOrderTransfers(rzOrderID)
	if err != nil {
		return fmt.Errorf("order-payout: fetch transfers for order %s: %w", orderID, err)
	}
	// amountPaise 0 = full reversal. An already-reversed transfer errors on Razorpay
	// and is tolerated (isAlreadyReversedErr) so a re-drive is a no-op. A GENUINE
	// failure is captured and returned so settlePayout does NOT stamp payout_settled_at
	// on an incomplete claw-back — the reconcile cron then re-drives it (#508). We still
	// attempt every transfer so one bad one doesn't block the rest.
	var firstErr error
	for _, t := range transfers {
		if t.ID == "" {
			continue
		}
		if _, err := rz.ReverseTransfer(t.ID, 0); err != nil {
			if isAlreadyReversedErr(err) {
				continue // idempotent re-drive — no new money moved, don't audit
			}
			log.Printf("order-payout: reverse transfer %s (order %s) failed: %v", t.ID, orderID, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("order-payout: reverse transfer %s: %w", t.ID, err)
			}
			continue // failed claw-back moved no money — don't audit it as reversed
		}
		auditTransferMovement(auditTransferReverse, aggTypeOrder, orderID, t.ID, t.Amount, "order refunded — payout clawed back")
	}
	return firstErr
}
