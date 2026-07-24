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
	if err := database.DB.Preload("Days").
		Where("status = ? AND "+cutoffWhere, status, now).Find(&plans).Error; err != nil {
		log.Printf("meal-plan-sweep: query (%s) failed: %v", status, err)
		return
	}
	for i := range plans {
		p := plans[i]
		// GAP 1 interlock (#395·3): never expire a plan whose advance was actually
		// captured. The client verify + webhook may both have been lost; expiring here
		// would terminally strand the capture — RefundUndeliveredDays no-ops on a blank
		// escrow_payment_id, and once `expired` the advance-reconcile can't re-select it
		// (money taken, chef unpaid, no refund). Ask the gateway first; if paid, confirm
		// instead of expiring; if the gateway is unreachable, DEFER (never expire a plan
		// we couldn't verify — the reconcile will handle it).
		if status == models.MealPlanAwaitingCustomer && p.RazorpayOrderID != "" && p.EscrowPaymentID == "" &&
			rescueCapturedBeforeExpiry(&p) {
			continue
		}
		// When an awaiting_customer plan expires, the chef cherry-picked and was
		// waiting too — notify both parties (not just the customer).
		var chefUserID uuid.UUID
		if status == models.MealPlanAwaitingCustomer {
			// Pluck must target a slice — GORM does not scan a single column into a
			// scalar uuid.UUID (it silently leaves it zero), which previously dropped
			// the chef notification below. Take the first row (chef id is unique).
			var ids []uuid.UUID
			database.DB.Model(&models.ChefProfile{}).
				Where("id = ?", p.ChefID).Pluck("user_id", &ids)
			if len(ids) > 0 {
				chefUserID = ids[0]
			}
		}
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
			if err := EnqueueEvent(tx, SubjectMealPlanCancelled, "meal_plan.expired", p.CustomerID, map[string]any{
				"meal_plan_id": p.ID.String(), "reason": reason,
			}); err != nil {
				return err
			}
			if chefUserID != uuid.Nil {
				if err := EnqueueEvent(tx, SubjectMealPlanCancelled, "meal_plan.expired", chefUserID, map[string]any{
					"meal_plan_id": p.ID.String(), "reason": reason,
				}); err != nil {
					return err
				}
			}
			// Escrow (gated): a lapsed plan fully refunds any captured advance to
			// the customer's wallet (idempotent per day).
			return RefundUndeliveredDays(tx, &p, "plan expired — "+reason)
		})
		if err != nil {
			log.Printf("meal-plan-sweep: expire %s failed: %v", p.ID, err)
		}
	}
}

// rescueCapturedBeforeExpiry asks Razorpay whether an about-to-expire plan's advance was
// actually captured; if so it confirms the plan (via the shared ConfirmMealPlanAdvance)
// and returns true so the sweep skips the expiry. It ALSO returns true (defer expiry)
// when the gateway can't be reached or the confirm errors — expiring a plan we couldn't
// verify is exactly the money hole this closes, and the advance-reconcile will retry.
// Returns false only when the gateway confirms there is NO captured payment.
func rescueCapturedBeforeExpiry(p *models.MealPlan) bool {
	if !MealPlanEscrowActive() {
		return false
	}
	rz := GetRazorpay()
	if rz == nil {
		return false
	}
	pays, err := rz.FetchOrderPayments(p.RazorpayOrderID)
	if err != nil {
		log.Printf("meal-plan-sweep: gateway check for %s failed — deferring expiry: %v", p.ID, err)
		return true
	}
	captured := capturedPaymentFor(pays, p.RazorpayOrderID)
	if captured == "" {
		return false // gateway confirms unpaid → safe to expire
	}
	if _, err := ConfirmMealPlanAdvance(database.DB, p, captured, ""); err != nil {
		log.Printf("meal-plan-sweep: confirm captured plan %s failed — deferring expiry: %v", p.ID, err)
		return true
	}
	log.Printf("meal-plan-sweep: plan %s was captured — confirmed instead of expiring", p.ID)
	return true
}
