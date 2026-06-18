package services

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// meal_plan_cron.go — sweeps the negotiation cutoffs for the tiffin meal-plan
// (#197). Registered in cronJobs() so it runs as a durable Temporal Schedule
// (exactly-once, leader-elected) with the in-process ticker as the legacy
// fallback. A lapsed chef-respond-by or customer-approve-by expires the plan
// (and, once escrow is enabled (#194), fully refunds it).
const mealPlanSweepInterval = 5 * time.Minute

// StartMealPlanCron is the legacy in-process fallback (used when Temporal is off).
func StartMealPlanCron(ctx context.Context) {
	go func() {
		runMealPlanSweep(ctx)
		ticker := time.NewTicker(mealPlanSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("meal-plan-sweep: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runMealPlanSweep(ctx)
			}
		}
	}()
	log.Println("meal-plan-sweep: cron started (interval=5m)")
}

func runMealPlanSweep(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("meal-plan-sweep: panic recovered: %v", r)
		}
	}()

	now := time.Now()
	expireMealPlans(now, models.MealPlanPendingChef,
		"chef_respond_by IS NOT NULL AND chef_respond_by <= ?", "chef did not respond in time")
	expireMealPlans(now, models.MealPlanAwaitingCustomer,
		"customer_approve_by IS NOT NULL AND customer_approve_by <= ?", "customer did not approve in time")
}

func expireMealPlans(now time.Time, status models.MealPlanStatus, cutoffWhere, reason string) {
	var plans []models.MealPlan
	if err := database.DB.
		Where("status = ? AND "+cutoffWhere, status, now).Find(&plans).Error; err != nil {
		log.Printf("meal-plan-sweep: query (%s) failed: %v", status, err)
		return
	}
	for i := range plans {
		p := plans[i]
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			// Guard on the current status so a concurrent customer/chef action wins.
			res := tx.Model(&models.MealPlan{}).
				Where("id = ? AND status = ?", p.ID, status).
				Updates(map[string]any{
					"status":        models.MealPlanExpired,
					"cancelled_at":  now,
					"cancel_reason": reason,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return nil // already transitioned by a live action
			}
			return EnqueueEvent(tx, SubjectMealPlanCancelled, "meal_plan.expired", p.CustomerID, map[string]any{
				"meal_plan_id": p.ID.String(), "reason": reason,
			})
		})
		if err != nil {
			log.Printf("meal-plan-sweep: expire %s failed: %v", p.ID, err)
		}
		// TODO(#194): when escrow is enabled, fully refund the advance to the
		// customer's wallet here (CreditWallet, idempotent on the plan id).
	}
}
