package services

// meal_plan_advance_reconcile_cron.go — #395·3 belt-and-suspenders for the tiffin
// payment-after-approval flow. The payment.captured webhook confirms a captured
// meal-plan advance durably (handlers/payment.go), and so does the client verify path;
// this sweep is the last line for the rare DOUBLE miss (webhook AND client both lost):
// it finds plans stuck in awaiting_customer WITH an advance order minted but no
// escrow_payment_id, asks the gateway whether that order was actually paid, and confirms
// via the SAME ConfirmMealPlanAdvance seam if so. No-op when escrow is off. A grace
// window keeps it from racing a just-approved plan's normal confirm path.

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const (
	mealPlanAdvanceReconcileInterval = 10 * time.Minute
	// Only sweep plans idle at least this long, so the client verify + the webhook get
	// first crack before the reconcile pings the gateway.
	mealPlanAdvanceReconcileGrace = 5 * time.Minute
)

func runMealPlanAdvanceReconcileScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mealplan-advance-reconcile: panic recovered: %v", r)
		}
	}()
	if n := reconcileMealPlanAdvances(database.DB, time.Now()); n > 0 {
		log.Printf("mealplan-advance-reconcile: confirmed %d captured-but-unconfirmed advance(s)", n)
	}
}

// reconcileMealPlanAdvances confirms captured-but-unconfirmed meal-plan advances and
// returns how many it confirmed. Belt-and-suspenders behind the webhook + client verify.
func reconcileMealPlanAdvances(db *gorm.DB, now time.Time) int {
	if !MealPlanEscrowActive() {
		return 0
	}
	// No top-level client: each plan is reconciled against its own mode's slot, so a
	// live plan and a test plan in the same sweep talk to different merchant
	// accounts. Bail only when neither slot is configured.
	if GetCashfreeFor(models.ChefModeLive) == nil && GetCashfreeFor(models.ChefModeTest) == nil {
		return 0
	}
	var plans []models.MealPlan
	cutoff := now.Add(-mealPlanAdvanceReconcileGrace)
	if err := db.Preload("Days").
		Where("status = ? AND razorpay_order_id <> '' AND (escrow_payment_id IS NULL OR escrow_payment_id = '') AND updated_at < ?",
			models.MealPlanAwaitingCustomer, cutoff).
		Find(&plans).Error; err != nil {
		log.Printf("mealplan-advance-reconcile: query failed: %v", err)
		return 0
	}
	confirmedN := 0
	for i := range plans {
		p := &plans[i]
		captured, err := capturedMealPlanAdvance(p)
		if err != nil {
			log.Printf("mealplan-advance-reconcile: gateway check for plan %s failed: %v", p.ID, err)
			continue
		}
		if captured == "" {
			continue // unpaid — leave it; the expiry sweep cancels an abandoned plan
		}
		confirmed, txErr := ConfirmMealPlanAdvance(db, p)
		if txErr != nil {
			log.Printf("mealplan-advance-reconcile: confirm plan %s failed: %v", p.ID, txErr)
			continue
		}
		if confirmed {
			confirmedN++
			log.Printf("mealplan-advance-reconcile: confirmed plan %s from captured payment %s (order %s)",
				p.ID, captured, p.RazorpayOrderID)
		}
	}
	return confirmedN
}

// capturedMealPlanAdvance returns the gateway payment id of the capture on this plan's
// own advance order, or "" when the gateway reports none. Shared with the expiry
// rescue, which asks the identical question a moment before expiring a plan.
func capturedMealPlanAdvance(p *models.MealPlan) (string, error) {
	cf := GetCashfreeFor(p.Mode)
	if cf == nil {
		return "", fmt.Errorf("no %s gateway configured", models.NormalizeMode(p.Mode))
	}
	pay, err := cf.SuccessfulPayment(p.RazorpayOrderID)
	if err != nil {
		return "", fmt.Errorf("fetch payments for order %s: %w", p.RazorpayOrderID, err)
	}
	if pay == nil {
		return "", nil
	}
	return pay.CFPaymentID.String(), nil
}

func StartMealPlanAdvanceReconcileCron(ctx context.Context) {
	go func() {
		t := time.NewTicker(mealPlanAdvanceReconcileInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runMealPlanAdvanceReconcileScan(ctx)
			}
		}
	}()
}
