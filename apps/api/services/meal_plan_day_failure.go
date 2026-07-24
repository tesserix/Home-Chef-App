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

// terminalOrFailedDayStatuses is the set of day statuses on which MarkMealPlanDayFailed is
// a no-op: either already frozen (`failed`) or terminally resolved. It is the single source
// of truth shared with the delivery-failure reconcile sweep — the sweep selects only days
// OUTSIDE this set, so a selected strand is guaranteed to freeze (never a re-driven
// froze=false no-op). Keep the freeze guard and the sweep predicate on this one list so
// they cannot drift apart.
var terminalOrFailedDayStatuses = []models.MealPlanDayStatus{
	models.MealPlanDayFailed, models.MealPlanDayDelivered, models.MealPlanDayCancelled,
	models.MealPlanDaySkipped, models.MealPlanDayDeclined, models.MealPlanDayRefunded,
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
		Where("id = ? AND status NOT IN ?", day.ID, terminalOrFailedDayStatuses).
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
	// Notify the CUSTOMER their tiffin failed. Resolve their user id from the plan
	// (the event previously passed day.MealPlanID — a plan id — in the user-id
	// slot, so it never reached a real recipient). Load through the struct (not a
	// bare Scan into uuid.UUID) so GORM handles the uuid column on both Postgres and
	// the sqlite test harness. Best-effort: if the plan can't be loaded, emit with a
	// zero recipient — the handler no-ops on a nil user, so the failure is still
	// recorded and the day stays frozen rather than the whole terminalize failing.
	var customerID uuid.UUID
	var plan models.MealPlan
	if err := tx.Select("customer_id").First(&plan, "id = ?", day.MealPlanID).Error; err == nil {
		customerID = plan.CustomerID
	}
	if err := EnqueueEvent(tx, SubjectMealPlanDayFailed, "meal_plans.day_failed", customerID, map[string]any{
		"meal_plan_id": day.MealPlanID.String(), "day_id": day.ID.String(), "dishName": day.DishName,
	}); err != nil {
		return false, err
	}
	return true, nil
}
