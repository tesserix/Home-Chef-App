package services

// meal_plan_refund_v2_transitions.go — the refund state machine (docs/refund-policy-v3-spec.md).
// Per RBI the CUSTOMER — never the admin — chooses the refund medium, so every agreed refund
// lands in `pending_customer` and the customer picks:
//   AgreeMealPlanDayRefundAuto            — top tier: auto-approved at the tier percentage,
//                                           no chef step → pending_customer.
//   ChefDecideMealPlanRefund              — lower tiers: the chef sets any percentage from the
//                                           day's pinned FLOOR to 100 (→ pending_customer), or
//                                           declines (→ day served).
//   CustomerChooseMealPlanRefundMedium    — wallet → instant ledger credit (resolved); original →
//                                           pending_admin (the admin only EXECUTES the refund).
//   AdminExecuteMealPlanRefund            — run the customer-chosen ORIGINAL (gateway) refund.
// All are idempotent via the executor.
//
// v3 (#834): the CHEF chooses the amount and the FLOOR IS ENFORCED HERE — server-side, on the
// day's PINNED floor. Both web and mobile call the same endpoint, so a client-side constraint
// would be no constraint at all; and the floor is read from the day rather than recomputed off
// the current clock, so a chef cannot shrink their own obligation by sitting on the decision
// until the lead time falls into a lower band.

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

var (
	// ErrRefundStageMismatch — the day is not in the stage this transition expects.
	ErrRefundStageMismatch = errors.New("meal-plan day is not in the expected refund stage")
	// ErrInvalidRefundChoice — the requested percentage is outside 0–100 (or a legacy choice is
	// not full/half/none).
	ErrInvalidRefundChoice = errors.New("invalid refund choice")
	// ErrRefundBelowFloor — the chef tried to refund less than this day's lead-time floor.
	ErrRefundBelowFloor = errors.New("refund is below the minimum for this cancellation window")
	// ErrInvalidRefundMedium — the customer's medium is not wallet/source.
	ErrInvalidRefundMedium = errors.New("invalid refund medium")
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

// MealPlanDayRefundFloor is the floor the chef must honour for a day: the value PINNED when the
// request was raised. A day with no pinned floor (a pre-v3 row, or a request raised before the
// pin existed) falls back to 0 — the chef keeps the pre-v3 freedom to refund nothing rather
// than being retroactively bound by a floor nobody told them about.
func MealPlanDayRefundFloor(day *models.MealPlanDay) int {
	if day == nil || day.RefundFloorPercent == nil {
		return 0
	}
	return ClampRefundPercent(*day.RefundFloorPercent)
}

// notifyCustomerRefundReady best-effort pushes the customer to choose their refund medium.
func notifyCustomerRefundReady(plan *models.MealPlan, day *models.MealPlanDay) {
	amount := MealPlanRefundAmountForDay(plan, day)
	_ = SendPushNotification(plan.CustomerID,
		"Choose where your refund goes",
		fmt.Sprintf("Your %s refund for %s is ready. Send it to your Fe3dr wallet (instant) or back to your original payment method (5–7 days).",
			FormatMoney(amount), day.Date.In(scheduleIST).Format("Mon 2 Jan")),
		map[string]string{"type": "refund_choice", "day_id": day.ID.String(), "meal_plan_id": plan.ID.String()},
	)
}

// AgreeMealPlanDayRefundAuto is the top-tier path: the amount is agreed automatically at
// `percent` with NO chef step (prep has not started, so there is nothing to compensate). Per
// RBI the CUSTOMER still chooses the medium, so the day moves to pending_customer. Runs in the
// caller's tx; the caller notifies the customer after commit.
func AgreeMealPlanDayRefundAuto(tx *gorm.DB, _ *models.MealPlan, day *models.MealPlanDay, percent int) error {
	percent = ClampRefundPercent(percent)
	day.RefundPercent = &percent
	day.RefundFloorPercent = &percent
	day.ChefRefundChoice = models.RefundProportionLabel(percent)
	return tx.Model(&models.MealPlanDay{}).Where("id = ?", day.ID).Updates(map[string]any{
		"refund_percent":       percent,
		"refund_floor_percent": percent,
		"chef_refund_choice":   day.ChefRefundChoice,
		"refund_stage":         models.MPRefundPendingCustomer,
	}).Error
}

// RouteMealPlanDayToChef parks a day for the chef's decision, PINNING the tier floor that applied
// when the request was raised. Runs in the caller's tx.
func RouteMealPlanDayToChef(tx *gorm.DB, day *models.MealPlanDay, floorPercent int) error {
	floorPercent = ClampRefundPercent(floorPercent)
	day.RefundFloorPercent = &floorPercent
	day.RefundStage = models.MPRefundPendingChef
	update := map[string]any{
		"refund_stage":         models.MPRefundPendingChef,
		"refund_floor_percent": floorPercent,
	}
	// Start the chef's clock. Without a deadline the day sits pending_chef forever and the
	// customer's money with it; the sweep resolves it at 100% once this passes.
	if by, ok := ChefRefundDecisionDeadline(time.Now()); ok {
		day.RefundDecisionBy = &by
		update["refund_decision_by"] = by
	}
	return tx.Model(&models.MealPlanDay{}).Where("id = ?", day.ID).Updates(update).Error
}

// ChefRefundDecisionDeadline is when a refund raised now stops being the chef's to price.
// Reports false when policy disables the sweep (a negative window).
func ChefRefundDecisionDeadline(now time.Time) (time.Time, bool) {
	mins := GetPlatformPolicy().MealPlanChefRefundDecisionMinutes
	if mins < 0 {
		return time.Time{}, false
	}
	if mins == 0 {
		mins = DefaultPlatformPolicy().MealPlanChefRefundDecisionMinutes
	}
	return now.Add(time.Duration(mins) * time.Minute), true
}

// ChefDecideMealPlanRefund applies the chef's decision to a day awaiting them (pending_chef).
// A percentage at or above the day's floor moves the day to pending_customer (the customer then
// picks the medium); 0 (only reachable when the floor is 0) resolves via the executor — no
// refund, the chef keeps their payout, the day is skipped. Decline → the day returns to
// `confirmed` (it is cooked and the customer charged) and its frozen payout is restored.
//
// Rejects anything below the floor with ErrRefundBelowFloor. This is the ONLY place the floor is
// enforced, and it is enforced against the day's PINNED value, not the current clock.
func ChefDecideMealPlanRefund(db *gorm.DB, dayID uuid.UUID, percent int, decline bool) error {
	var notifyPlan *models.MealPlan
	var notifyDay *models.MealPlanDay
	err := db.Transaction(func(tx *gorm.DB) error {
		plan, day, err := loadV2PlanDay(tx, dayID)
		if err != nil {
			return err
		}
		if day.RefundStage != models.MPRefundPendingChef {
			return ErrRefundStageMismatch
		}

		if decline {
			if err := tx.Model(&models.MealPlanDay{}).Where("id = ? AND refund_stage = ?", dayID, models.MPRefundPendingChef).
				Updates(map[string]any{"status": models.MealPlanDayConfirmed, "refund_stage": ""}).Error; err != nil {
				return err
			}
			return restoreDisputedDayHoldToNone(tx, dayID)
		}
		if percent < 0 || percent > 100 {
			return ErrInvalidRefundChoice
		}
		if floor := MealPlanDayRefundFloor(day); percent < floor {
			return fmt.Errorf("%w: minimum %d%%, requested %d%%", ErrRefundBelowFloor, floor, percent)
		}
		if percent == 0 {
			return ExecuteMealPlanV2Refund(tx, plan, day, 0, "")
		}
		// Any positive percentage → the customer now chooses the medium.
		label := models.RefundProportionLabel(percent)
		res := tx.Model(&models.MealPlanDay{}).Where("id = ? AND refund_stage = ?", dayID, models.MPRefundPendingChef).
			Updates(map[string]any{
				"refund_percent":     percent,
				"chef_refund_choice": label,
				"refund_stage":       models.MPRefundPendingCustomer,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrRefundStageMismatch
		}
		day.RefundPercent = &percent
		day.ChefRefundChoice = label
		notifyPlan, notifyDay = plan, day
		return nil
	})
	if err == nil && notifyPlan != nil {
		notifyCustomerRefundReady(notifyPlan, notifyDay)
	}
	return err
}

// CustomerChooseMealPlanRefundMedium records the customer's RBI-required medium choice for a day
// awaiting them (pending_customer). Wallet → instant ledger credit (resolved). Original → the day
// moves to pending_admin for the admin to EXECUTE the gateway refund (the customer chose it).
// customerID authorizes: the day's plan must belong to the customer.
func CustomerChooseMealPlanRefundMedium(db *gorm.DB, dayID, customerID uuid.UUID, medium models.RefundDestination) error {
	if medium != models.RefundDestinationWallet && medium != models.RefundDestinationSource {
		return ErrInvalidRefundMedium
	}
	return db.Transaction(func(tx *gorm.DB) error {
		plan, day, err := loadV2PlanDay(tx, dayID)
		if err != nil {
			return err
		}
		if plan.CustomerID != customerID {
			return gorm.ErrRecordNotFound
		}
		if day.RefundStage != models.MPRefundPendingCustomer {
			return ErrRefundStageMismatch
		}
		if medium == models.RefundDestinationWallet {
			// Instant: credit the wallet/ledger now — no admin, no external money.
			return ExecuteMealPlanV2Refund(tx, plan, day, day.RefundPercentOf(), models.RefundDestinationWallet)
		}
		// Original method: park for the admin to execute the gateway refund.
		res := tx.Model(&models.MealPlanDay{}).Where("id = ? AND refund_stage = ?", dayID, models.MPRefundPendingCustomer).
			Updates(map[string]any{"refund_destination": models.RefundDestinationSource, "refund_stage": models.MPRefundPendingAdmin})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrRefundStageMismatch
		}
		return nil
	})
}

// AdminExecuteMealPlanRefund runs the customer-chosen ORIGINAL (gateway) refund for a day awaiting
// the admin (pending_admin). The admin only executes — the customer already chose the medium.
func AdminExecuteMealPlanRefund(db *gorm.DB, dayID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		plan, day, err := loadV2PlanDay(tx, dayID)
		if err != nil {
			return err
		}
		if day.RefundStage != models.MPRefundPendingAdmin {
			return ErrRefundStageMismatch
		}
		return ExecuteMealPlanV2Refund(tx, plan, day, day.RefundPercentOf(), models.RefundDestinationSource)
	})
}
