package services

// meal_plan_refund_v2_transitions.go — the v2 refund state machine
// (docs/meal-plan-refund-flow-design.md). Per RBI the CUSTOMER — never the admin — chooses the
// refund medium, so every agreed refund lands in `pending_customer` and the customer picks:
//   AgreeMealPlanDayRefundFull            — >12h path: amount = full, → pending_customer.
//   ChefDecideMealPlanRefund              — ≤12h: chef Full/Half (→ pending_customer) | None
//                                           (→ resolved, no refund) | Decline (→ day served).
//   CustomerChooseMealPlanRefundMedium    — wallet → instant ledger credit (resolved); original →
//                                           pending_admin (the admin only EXECUTES the gateway refund).
//   AdminExecuteMealPlanRefund            — run the customer-chosen ORIGINAL (gateway) refund.
// All are idempotent via the executor.

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

var (
	// ErrRefundStageMismatch — the day is not in the stage this transition expects.
	ErrRefundStageMismatch = errors.New("meal-plan day is not in the expected refund stage")
	// ErrInvalidRefundChoice — the chef choice is not full/half/none.
	ErrInvalidRefundChoice = errors.New("invalid refund choice")
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

// notifyCustomerRefundReady best-effort pushes the customer to choose their refund medium.
func notifyCustomerRefundReady(plan *models.MealPlan, day *models.MealPlanDay) {
	amount := MealPlanRefundAmount(plan, day, day.ChefRefundChoice)
	_ = SendPushNotification(plan.CustomerID,
		"Choose where your refund goes",
		fmt.Sprintf("Your ₹%.0f refund for %s is ready. Send it to your HomeChef wallet (instant) or back to your original payment method (5–7 days).",
			amount, day.Date.Format("Mon 2 Jan")),
		map[string]string{"type": "refund_choice", "day_id": day.ID.String(), "meal_plan_id": plan.ID.String()},
	)
}

// AgreeMealPlanDayRefundFull is the >12h path: the refund amount is FULL (the chef has not started
// prep), but per RBI the CUSTOMER still chooses the medium — so the day moves to pending_customer.
// Runs in the caller's tx; the caller notifies the customer after commit.
func AgreeMealPlanDayRefundFull(tx *gorm.DB, _ *models.MealPlan, day *models.MealPlanDay) error {
	return tx.Model(&models.MealPlanDay{}).Where("id = ?", day.ID).Updates(map[string]any{
		"chef_refund_choice": models.RefundProportionFull,
		"refund_stage":       models.MPRefundPendingCustomer,
	}).Error
}

// ChefDecideMealPlanRefund applies the chef's decision to a day awaiting them (pending_chef).
// Full/Half → pending_customer (records the amount; the customer then picks the medium). None →
// resolved via the executor (no refund, chef keeps payout, day skipped). Decline → the day returns
// to `confirmed` (served; customer charged) and its frozen payout is restored.
func ChefDecideMealPlanRefund(db *gorm.DB, dayID uuid.UUID, choice models.RefundProportion, decline bool) error {
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
		if !ValidRefundProportion(choice) {
			return ErrInvalidRefundChoice
		}
		if choice == models.RefundProportionNone {
			return ExecuteMealPlanV2Refund(tx, plan, day, models.RefundProportionNone, "")
		}
		// Full/Half → the customer now chooses the medium.
		res := tx.Model(&models.MealPlanDay{}).Where("id = ? AND refund_stage = ?", dayID, models.MPRefundPendingChef).
			Updates(map[string]any{"chef_refund_choice": choice, "refund_stage": models.MPRefundPendingCustomer})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrRefundStageMismatch
		}
		day.ChefRefundChoice = choice
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
			return ExecuteMealPlanV2Refund(tx, plan, day, day.ChefRefundChoice, models.RefundDestinationWallet)
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
		return ExecuteMealPlanV2Refund(tx, plan, day, day.ChefRefundChoice, models.RefundDestinationSource)
	})
}
