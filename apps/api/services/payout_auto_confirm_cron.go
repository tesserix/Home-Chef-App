package services

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// payout_auto_confirm_cron.go — the auto-confirm sweep (#387 follow-up). A
// delivered, dispute-free payout hold that has sat in awaiting_customer_confirmation
// past the confirm window (GetCustomerConfirmWindowHours, default 24h) is advanced
// to release_eligible with no customer tap; a hold whose order has an open issue
// lands in disputed. The transition + NATS event live in ConfirmOrderHold /
// ConfirmMealPlanDayHold, so the sweep just selects stale awaiting rows and calls
// them — the guarded UPDATE + RowsAffected>0 emit gate make repeated/concurrent
// runs safe. Behaviour is flag-gated downstream: this only moves DB state, #388
// drives the actual money off release_eligible.
const payoutAutoConfirmInterval = 15 * time.Minute

// sweepBatchLimit bounds one scan so a large backlog cannot run unboundedly.
const sweepBatchLimit = 500

// StartPayoutAutoConfirmCron is the legacy in-process fallback (used when Temporal
// is off): run once, then on a ticker until ctx is cancelled.
func StartPayoutAutoConfirmCron(ctx context.Context) {
	go func() {
		runPayoutAutoConfirmScan(ctx)
		ticker := time.NewTicker(payoutAutoConfirmInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("payout-auto-confirm-sweep: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runPayoutAutoConfirmScan(ctx)
			}
		}
	}()
	log.Println("payout-auto-confirm-sweep: cron started (interval=15m)")
}

// runPayoutAutoConfirmScan advances every past-window awaiting hold once.
func runPayoutAutoConfirmScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("payout-auto-confirm-sweep: panic recovered: %v", r)
		}
	}()

	hours := GetCustomerConfirmWindowHours(database.DB)
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	orders := sweepOrders(cutoff)
	days := sweepMealPlanDays(cutoff)
	if orders > 0 || days > 0 {
		log.Printf("payout-auto-confirm-sweep: advanced %d order(s), %d meal-plan day(s) (window=%dh)", orders, days, hours)
	}
}

// sweepOrders confirms every stale awaiting regular-order hold; returns the count
// scanned. Dispute-vs-release is decided inside ConfirmOrderHold. Log-and-continue
// per row so one bad row cannot abort the batch.
func sweepOrders(cutoff time.Time) int {
	var orders []models.Order
	if err := database.DB.
		Where("payout_hold_status = ? AND delivered_at IS NOT NULL AND delivered_at <= ? AND customer_confirmed_at IS NULL",
			models.PayoutHoldAwaitingConfirmation, cutoff).
		Limit(sweepBatchLimit).Find(&orders).Error; err != nil {
		log.Printf("payout-auto-confirm-sweep: query orders failed: %v", err)
		return 0
	}
	for i := range orders {
		if _, err := ConfirmOrderHold(database.DB, &orders[i]); err != nil {
			log.Printf("payout-auto-confirm-sweep: confirm order %s failed: %v", orders[i].ID, err)
		}
	}
	return len(orders)
}

// sweepMealPlanDays confirms every stale awaiting meal-plan-day hold; returns the
// count scanned.
func sweepMealPlanDays(cutoff time.Time) int {
	var days []models.MealPlanDay
	if err := database.DB.
		Where("payout_hold_status = ? AND delivered_at IS NOT NULL AND delivered_at <= ? AND customer_confirmed_at IS NULL",
			models.PayoutHoldAwaitingConfirmation, cutoff).
		Limit(sweepBatchLimit).Find(&days).Error; err != nil {
		log.Printf("payout-auto-confirm-sweep: query meal-plan days failed: %v", err)
		return 0
	}
	for i := range days {
		if _, err := ConfirmMealPlanDayHold(database.DB, &days[i]); err != nil {
			log.Printf("payout-auto-confirm-sweep: confirm meal-plan day %s failed: %v", days[i].ID, err)
		}
	}
	return len(days)
}
