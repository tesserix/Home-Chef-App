package services

// delivery_fee_adjust.go — the chef's delivery-fee decision at accept (#703).
//
// The customer is charged the recommended self-delivery fee (approx-max, by
// distance, capped at the chef's max) UPFRONT at checkout. At accept the chef can
// bring it DOWN — to a lower number, or to 0 when the drop is inside their free
// zone — but never UP (the customer already agreed to the ceiling). The
// difference is refunded to the customer's original payment method via the order
// refund coordinator, which is idempotent on the (order, scope) pair, so a
// re-accept never double-refunds.

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
)

// deliveryFeeAdjustScope is the once-per-order logical refund identity for the
// chef's delivery-fee reduction — the coordinator dedups retries on it.
const deliveryFeeAdjustScope = "delivery-fee-adjust"

// AdjustDeliveryFeeAtAccept applies the chef's chosen delivery fee. It clamps the
// fee to [0, charged], refunds any difference to the customer, records the final
// fee on the order, and returns the resolved fee + the amount refunded. A fee
// equal to (or above) the charged amount is a no-op "confirm as-is".
//
// Never fails the accept for a benign reason: an already-in-flight refund returns
// (chefFee, 0, nil). A real gateway error is returned so the caller can decide.
func AdjustDeliveryFeeAtAccept(ctx context.Context, order *models.Order, chefFee float64) (finalFee, refunded float64, err error) {
	charged := order.DeliveryFee
	if chefFee < 0 {
		chefFee = 0
	}
	if chefFee > charged {
		chefFee = charged // the chef can only bring it DOWN, never above the ceiling.
	}
	diff := models.RoundAmount(charged - chefFee)
	if diff < 0.01 {
		// Confirmed as-is (or negligible) — nothing to refund; still record the
		// chef's explicit choice so the order shows a settled figure.
		setDeliveryFeeFinal(order.ID, chefFee)
		return chefFee, 0, nil
	}

	res, rErr := NewOrderRefundCoordinator().Refund(ctx, orderrefund.RefundCommand{
		OrderID: order.ID,
		Amount:  &diff,
		Reason:  "Chef reduced the delivery fee",
		Actor:   "chef",
		ScopeID: deliveryFeeAdjustScope,
	})
	switch {
	case errors.Is(rErr, orderrefund.ErrRefundInFlight),
		errors.Is(rErr, orderrefund.ErrExceedsRemaining):
		// Already being processed / nothing left to refund — treat as done.
		return chefFee, 0, nil
	case errors.Is(rErr, orderrefund.ErrNoCapturedPayment):
		// No captured payment yet (e.g. order not paid) — just record the choice;
		// there is nothing to refund.
		setDeliveryFeeFinal(order.ID, chefFee)
		return chefFee, 0, nil
	case rErr != nil:
		return charged, 0, rErr
	}

	// Money moved. Record the chef's final fee for the invoice + settlement. The
	// coordinator already stamped refund_amount/payment_status; we only add the
	// final-fee figure (Total stays the frozen billed amount; effective paid =
	// Total − RefundAmount).
	setDeliveryFeeFinal(order.ID, chefFee)
	final := chefFee
	order.DeliveryFeeFinal = &final
	return chefFee, res.Amount, nil
}

// NotifyDeliveryFeeRefund tells the customer the chef lowered the delivery fee and
// the difference is on its way back. Best-effort push; the refund + reduced fee
// are also visible on the order. (#703)
func NotifyDeliveryFeeRefund(order models.Order, finalFee, refunded float64) {
	title := "Delivery fee reduced"
	body := fmt.Sprintf(
		"Your chef set delivery to ₹%.0f for order %s — ₹%.0f is being refunded to your original payment method.",
		finalFee, order.OrderNumber, refunded)
	if err := SendPushNotification(order.CustomerID, title, body, map[string]string{
		"type":     "delivery_fee_refund",
		"orderId":  order.ID.String(),
		"refunded": fmt.Sprintf("%.2f", refunded),
	}); err != nil {
		log.Printf("delivery-fee-adjust: push notify failed order=%s: %v", order.ID, err)
	}
}

func setDeliveryFeeFinal(orderID uuid.UUID, fee float64) {
	if err := database.DB.Model(&models.Order{}).Where("id = ?", orderID).
		Update("delivery_fee_final", fee).Error; err != nil {
		log.Printf("delivery-fee-adjust: persist final fee failed order=%s: %v", orderID.String(), err)
		CaptureBackgroundError(err)
	}
}
