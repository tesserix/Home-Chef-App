package services

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// meal_plan_refund_deadline_cron.go — the chef's refund clock (docs/refund-policy-v3-spec.md).
//
// A customer cancellation parks each unserved day on the CHEF, who prices it. That is the
// right authority — the kitchen carries the cost — but it gave an unresponsive chef an
// indefinite hold over someone else's money. Past RefundDecisionBy the day resolves at 100%
// of the refund base, which already excludes the platform's commission, so the customer is
// made whole on everything the kitchen and platform did not spend.
//
// 100%, not the pinned floor: the floor is the chef's protected MINIMUM, earned by answering.
// Reading it as the price of silence would pay a chef for not replying.
const refundDeadlineSweepInterval = 5 * time.Minute

// StartMealPlanRefundDeadlineCron is the legacy in-process fallback (used when Temporal is off).
func StartMealPlanRefundDeadlineCron(ctx context.Context) {
	go func() {
		runMealPlanRefundDeadlineSweep(ctx)
		ticker := time.NewTicker(refundDeadlineSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("meal-plan-refund-deadline: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runMealPlanRefundDeadlineSweep(ctx)
			}
		}
	}()
	log.Println("meal-plan-refund-deadline: cron started (interval=5m)")
}

func runMealPlanRefundDeadlineSweep(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("meal-plan-refund-deadline: panic recovered: %v", r)
		}
	}()
	if database.DB == nil {
		return
	}

	var due []models.MealPlanDay
	if err := database.DB.
		Where("refund_stage = ? AND refund_decision_by IS NOT NULL AND refund_decision_by <= ?",
			models.MPRefundPendingChef, time.Now()).
		Limit(200).
		Find(&due).Error; err != nil {
		log.Printf("meal-plan-refund-deadline: scan failed: %v", err)
		return
	}

	for i := range due {
		day := &due[i]
		if err := resolveLapsedRefund(day); err != nil {
			// Per-day: one bad row must not strand the rest of the queue.
			log.Printf("meal-plan-refund-deadline: day %s failed: %v", day.ID, err)
		}
	}
}

// resolveLapsedRefund agrees 100% for one day whose chef window has passed.
func resolveLapsedRefund(day *models.MealPlanDay) error {
	return database.DB.Transaction(func(tx *gorm.DB) error {
		// Re-read under the tx and re-check the stage: the chef may have answered between
		// the scan and here, and their decision must win over the deadline.
		var fresh models.MealPlanDay
		if err := tx.First(&fresh, "id = ? AND refund_stage = ?", day.ID, models.MPRefundPendingChef).Error; err != nil {
			return nil //nolint:nilerr // already decided — not an error
		}
		var plan models.MealPlan
		if err := tx.First(&plan, "id = ?", fresh.MealPlanID).Error; err != nil {
			return err
		}
		if err := AgreeMealPlanDayRefundAuto(tx, &plan, &fresh, 100); err != nil {
			return err
		}
		// Clear the clock so a re-run can't re-resolve a day that already moved on.
		if err := tx.Model(&models.MealPlanDay{}).Where("id = ?", fresh.ID).
			Update("refund_decision_by", nil).Error; err != nil {
			return err
		}

		amount := MealPlanRefundAmount(&plan, &fresh, 100)
		if err := EnqueueEvent(tx, SubjectMealPlanDayRefunded, "meal_plan.refund_auto_agreed", plan.CustomerID, map[string]any{
			"meal_plan_id": plan.ID.String(), "meal_plan_no": plan.MealPlanNumber,
			"day_id": fresh.ID.String(), "amount": amount, "percent": 100,
			"reason": "chef did not respond in time",
		}); err != nil {
			return err
		}
		// Pluck into a slice: GORM leaves a scalar uuid.UUID zeroed, which would
		// silently drop the chef's copy of the notification.
		var ids []uuid.UUID
		tx.Model(&models.ChefProfile{}).Where("id = ?", plan.ChefID).Pluck("user_id", &ids)
		if len(ids) == 0 {
			return nil
		}
		return EnqueueEvent(tx, SubjectMealPlanDayRefunded, "meal_plan.refund_auto_agreed", ids[0], map[string]any{
			"meal_plan_id": plan.ID.String(), "meal_plan_no": plan.MealPlanNumber,
			"day_id": fresh.ID.String(), "amount": amount, "percent": 100,
			"reason": "refund window elapsed",
		})
	})
}
