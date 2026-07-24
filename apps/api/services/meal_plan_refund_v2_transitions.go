package services

// meal_plan_refund_v2_transitions.go — the state-machine transitions over the v2 executor
// (docs/meal-plan-refund-flow-design.md):
//   AutoApproveMealPlanDayRefund — >12h path: Full refund → wallet, no chef/admin.
//   ChefDecideMealPlanRefund     — ≤12h path: chef Full/Half (→ pending admin) | None (→ resolved,
//                                  no refund) | Decline (→ day back to confirmed, will be served).
//   AdminPayMealPlanRefund       — pay a pending-admin day at the chef's choice to a destination.
// All are gated by the caller (MealPlanRefundFlowV2Active) and idempotent via the executor.

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

var (
	// ErrRefundStageMismatch — the day is not in the stage this transition expects (lost a race
	// or already resolved).
	ErrRefundStageMismatch = errors.New("meal-plan day is not in the expected refund stage")
	// ErrInvalidRefundChoice — the chef choice is not full/half/none.
	ErrInvalidRefundChoice = errors.New("invalid refund choice")
)

// loadV2PlanDay loads a day + its plan for a transition.
func loadV2PlanDay(tx *gorm.DB, dayID uuid.UUID) (*models.MealPlan, *models.MealPlanDay, error) {
	var day models.MealPlanDay
	if err := tx.First(&day, "id = ?", dayID).Error; err != nil {
		return nil, nil, err
	}
	var plan models.MealPlan
	if err := tx.First(&plan, "id = ?", day.MealPlanID).Error; err != nil {
		return nil, nil, err
	}
	return &plan, &day, nil
}

// AutoApproveMealPlanDayRefund is the >12h path: the chef has not started prep, so the customer is
// auto-refunded the FULL fee/GST-excluded base to their wallet with no chef or admin step. Runs in
// the caller's tx.
func AutoApproveMealPlanDayRefund(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay) error {
	return ExecuteMealPlanV2Refund(tx, plan, day, models.RefundProportionFull, models.RefundDestinationWallet)
}

// ChefDecideMealPlanRefund applies the chef's decision to a day awaiting them (RefundStage
// pending_chef). Full/Half → pending_admin (records the choice; no money yet). None → resolved via
// the executor (no customer refund, chef keeps payout, day skipped). Decline → the day returns to
// `confirmed` (it will be cooked and delivered; customer charged) and its frozen payout is restored.
func ChefDecideMealPlanRefund(db *gorm.DB, dayID uuid.UUID, choice models.RefundProportion, decline bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		plan, day, err := loadV2PlanDay(tx, dayID)
		if err != nil {
			return err
		}
		if day.RefundStage != models.MPRefundPendingChef {
			return ErrRefundStageMismatch
		}

		if decline {
			// The chef will serve the day after all → back to confirmed, clear the v2 stage.
			if err := tx.Model(&models.MealPlanDay{}).Where("id = ? AND refund_stage = ?", dayID, models.MPRefundPendingChef).
				Updates(map[string]any{
					"status":       models.MealPlanDayConfirmed,
					"refund_stage": "",
				}).Error; err != nil {
				return err
			}
			// Restore the payout hold the skip froze (disputed → none; a confirmed un-delivered
			// day is not yet payable) — the same restore the reject-skip path uses.
			return restoreDisputedDayHoldToNone(tx, dayID)
		}

		if !ValidRefundProportion(choice) {
			return ErrInvalidRefundChoice
		}
		if choice == models.RefundProportionNone {
			// No customer refund; chef keeps full payout; day skipped — terminal now.
			return ExecuteMealPlanV2Refund(tx, plan, day, models.RefundProportionNone, "")
		}
		// Full/Half → await admin pay. Record the chef's choice; no money moves yet.
		res := tx.Model(&models.MealPlanDay{}).Where("id = ? AND refund_stage = ?", dayID, models.MPRefundPendingChef).
			Updates(map[string]any{
				"chef_refund_choice": choice,
				"refund_stage":       models.MPRefundPendingAdmin,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrRefundStageMismatch
		}
		return nil
	})
}

// AdminPayMealPlanRefund pays a day awaiting admin (RefundStage pending_admin) at the chef's
// recorded choice, to the given destination (wallet instant, or source RBI). Runs its own tx.
func AdminPayMealPlanRefund(db *gorm.DB, dayID uuid.UUID, dest models.RefundDestination) error {
	return db.Transaction(func(tx *gorm.DB) error {
		plan, day, err := loadV2PlanDay(tx, dayID)
		if err != nil {
			return err
		}
		if day.RefundStage != models.MPRefundPendingAdmin {
			return ErrRefundStageMismatch
		}
		if dest != models.RefundDestinationWallet && dest != models.RefundDestinationSource {
			dest = models.RefundDestinationWallet // default to instant wallet
		}
		return ExecuteMealPlanV2Refund(tx, plan, day, day.ChefRefundChoice, dest)
	})
}
