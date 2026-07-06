package services

// meal_plan_day_failure.go — #393 slice A. Terminalize a failed delivery's MONEY state
// for a meal-plan DAY without moving money. The per-day fulfilment order is a shell with
// no razorpay_order_id, so RecordDeliveryFailure (gateway-only) skips it and the day
// would otherwise stall the plan forever (allDaysTerminal never true). This slice marks
// the day `failed` — a NON-terminal status the plan waits on — and freezes the day's
// payout hold to `disputed`, so the chef is not paid until an admin resolves the day.
// The money outcome (RefundDay vs release, per the confirmed fault class) is executed by
// the day-resolution slice (B); this slice only freezes.

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// SetMealPlanDayHoldDisputed freezes a meal-plan day's payout hold at `disputed` from a
// PRE-TERMINAL state (none / awaiting_customer_confirmation), mirroring
// SetOrderHoldDisputed. A hold already release_eligible / released / reversed / withheld
// is untouched (the #458 invariant — disputed must never un-settle real money movement,
// and a customer-confirmed released hold must not be re-disputed). Emits
// payout.hold_disputed onto the outbox on a genuine transition. Plain DB state — runs
// regardless of the escrow flags (no money moves; the day's held transfer stays held).
func SetMealPlanDayHoldDisputed(tx *gorm.DB, dayID uuid.UUID) error {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status IN ?", dayID,
			[]models.PayoutHoldStatus{models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation}).
		Update("payout_hold_status", models.PayoutHoldDisputed)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: dispute meal-plan day %s: %w", dayID, res.Error)
	}
	if res.RowsAffected == 0 {
		return nil // already disputed/settled/eligible — nothing to freeze, nothing to emit
	}
	return emitHoldEvent(tx, models.PayoutHoldDisputed, aggTypeMealPlanDay, dayID)
}

// MarkMealPlanDayFailed is the failure-path mirror of MarkMealPlanDayDelivered: when a
// terminally-failed delivery's order is a meal-plan per-day shell, find the day by its
// fulfilment order id, mark it `failed` (NON-terminal — the plan waits for admin
// resolution) and freeze the day's payout hold to disputed. The guarded UPDATE excludes
// every terminal day state AND `failed` itself, so a re-fired failure (or a late failure
// on an already-resolved day) is a froze=false no-op that never resurrects a terminal
// day. Returns froze=false (no error) when the order isn't a meal-plan-day order. Emits
// meal_plans.day_failed only on a genuine transition. Called inside the terminalize
// transaction; no money moves.
func MarkMealPlanDayFailed(tx *gorm.DB, orderID uuid.UUID) (bool, error) {
	var day models.MealPlanDay
	if err := tx.Where("order_id = ?", orderID).First(&day).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // not a meal-plan-day order
		}
		return false, fmt.Errorf("meal-plan day failure: load day for order %s: %w", orderID, err)
	}
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND status NOT IN ?", day.ID, []models.MealPlanDayStatus{
			models.MealPlanDayFailed, models.MealPlanDayDelivered, models.MealPlanDayCancelled,
			models.MealPlanDaySkipped, models.MealPlanDayDeclined, models.MealPlanDayRefunded,
		}).
		Update("status", models.MealPlanDayFailed)
	if res.Error != nil {
		return false, fmt.Errorf("meal-plan day failure: mark day %s failed: %w", day.ID, res.Error)
	}
	if res.RowsAffected == 0 {
		return false, nil // already terminal/failed — idempotent no-op
	}
	if err := SetMealPlanDayHoldDisputed(tx, day.ID); err != nil {
		return false, err
	}
	if err := EnqueueEvent(tx, SubjectMealPlanDayFailed, "meal_plans.day_failed", day.MealPlanID, map[string]any{
		"meal_plan_id": day.MealPlanID.String(), "day_id": day.ID.String(),
	}); err != nil {
		return false, err
	}
	return true, nil
}
