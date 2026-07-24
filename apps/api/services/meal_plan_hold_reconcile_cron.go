package services

// meal_plan_hold_reconcile_cron.go — #395·3 GAP 2 backstop. ConfirmMealPlanAdvance now
// commits the confirm in a fast local tx and holds the chef payouts OUTSIDE it (external
// Route transfers kept out of the DB tx). That leaves a small window: confirm committed,
// then the process crashed / the hold step errored → a confirmed, captured plan whose
// per-day payouts were never held (so the release sweep would pay the chef nothing).
// This sweep finds those plans — live + captured (escrow_payment_id set) with at least
// one payable day lacking a payout_transfer_id — and completes the hold. Idempotent
// (the per-day PayoutTransferID guard + the escrow gate make an already-held day a
// no-op), so it is safe to run alongside the inline hold. No-op when escrow is off.

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const mealPlanHoldReconcileInterval = 10 * time.Minute

func runMealPlanHoldReconcileScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mealplan-hold-reconcile: panic recovered: %v", r)
		}
	}()
	if n := reconcileUnheldMealPlans(database.DB, time.Now()); n > 0 {
		log.Printf("mealplan-hold-reconcile: completed payout holds for %d confirmed plan(s)", n)
	}
}

// reconcileUnheldMealPlans holds the chef payouts for live + captured plans whose payable
// days aren't yet held. Returns how many plans it (re)held.
func reconcileUnheldMealPlans(db *gorm.DB, _ time.Time) int {
	if !MealPlanEscrowActive() || GetRazorpay() == nil {
		return 0
	}
	liveStatuses := []models.MealPlanStatus{models.MealPlanConfirmed, models.MealPlanActive}
	payableDay := []models.MealPlanDayStatus{
		models.MealPlanDayAccepted, models.MealPlanDayConfirmed,
		models.MealPlanDayPrepared, models.MealPlanDayDelivered,
	}
	var planIDs []uuid.UUID
	if err := db.Raw(`
		SELECT DISTINCT d.meal_plan_id
		FROM meal_plan_days d
		JOIN meal_plans mp ON mp.id = d.meal_plan_id
		WHERE mp.status IN (?) AND mp.escrow_payment_id <> ''
		  AND d.status IN (?)
		  AND (d.payout_transfer_id IS NULL OR d.payout_transfer_id = '')`,
		liveStatuses, payableDay).Scan(&planIDs).Error; err != nil {
		log.Printf("mealplan-hold-reconcile: query failed: %v", err)
		return 0
	}
	held := 0
	for _, id := range planIDs {
		var plan models.MealPlan
		if err := db.Preload("Days").First(&plan, "id = ?", id).Error; err != nil {
			log.Printf("mealplan-hold-reconcile: load plan %s failed: %v", id, err)
			continue
		}
		if err := HoldMealPlanPayouts(db, &plan); err != nil {
			log.Printf("mealplan-hold-reconcile: hold plan %s failed: %v", id, err)
			continue
		}
		held++
		log.Printf("mealplan-hold-reconcile: completed payout hold for plan %s", id)
	}
	return held
}

func StartMealPlanHoldReconcileCron(ctx context.Context) {
	go func() {
		t := time.NewTicker(mealPlanHoldReconcileInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runMealPlanHoldReconcileScan(ctx)
			}
		}
	}()
}
