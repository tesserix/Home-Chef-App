package services

// payout_hold.go — the payout hold state machine seam (#387). Delivery parks a
// hold instead of releasing funds; an explicit customer confirmation advances it
// to release_eligible (consumed later by the admin payout queue #388). Every
// transition is a conditional UPDATE guarded on the source status, so replayed
// webhooks / retried saga activities can never double-advance or re-open a hold.
//
// KEY INVARIANT: a disputed or unconfirmed hold must NEVER reach
// release_eligible. The only UPDATE that produces release_eligible is gated on
// payout_hold_status = 'awaiting_customer_confirmation'; disputed/released rows
// fail the WHERE and are untouched.
//
// Setting the hold is plain DB state and runs regardless of the escrow flags;
// the flags only gate real MONEY movement (ReleaseOrderPayouts / ReleaseDayPayout),
// which #388 will drive off release_eligible.

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// SetOrderHoldAwaitingConfirmation parks a delivered regular order's payout in a
// customer-confirmation hold. No-op for orders without a Razorpay order id
// (meal-plan/group consolidated orders settle through their own paths — mirrors
// the order_payout.go scoping). Idempotent: the conditional update only fires
// from the empty (pre-delivery) state, so a replayed delivered event is a no-op.
func SetOrderHoldAwaitingConfirmation(db *gorm.DB, orderID uuid.UUID) error {
	var order models.Order
	if err := db.Select("id", "razorpay_order_id", "payout_hold_status").
		First(&order, "id = ?", orderID).Error; err != nil {
		return fmt.Errorf("payout-hold: load order %s: %w", orderID, err)
	}
	if order.RazorpayOrderID == "" {
		return nil // not a gateway-charged regular order
	}
	res := db.Model(&models.Order{}).
		Where("id = ? AND payout_hold_status = ?", orderID, models.PayoutHoldNone).
		Update("payout_hold_status", models.PayoutHoldAwaitingConfirmation)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: park order %s: %w", orderID, res.Error)
	}
	return nil
}

// SetMealPlanDayHoldAwaitingConfirmation parks a delivered meal-plan day's payout
// in a customer-confirmation hold. Called inside the delivery transaction.
// Idempotent (only advances from the empty state).
func SetMealPlanDayHoldAwaitingConfirmation(tx *gorm.DB, dayID uuid.UUID) error {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status = ?", dayID, models.PayoutHoldNone).
		Update("payout_hold_status", models.PayoutHoldAwaitingConfirmation)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: park meal-plan day %s: %w", dayID, res.Error)
	}
	return nil
}
