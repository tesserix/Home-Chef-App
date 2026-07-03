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

// payout_reconcile_cron.go — the released/reversed-but-unsettled payout reconcile
// (#459, the #388 follow-up the payout_release.go header flags). ReleaseHold /
// ReverseHold flip payout_hold_status in a committed tx and only THEN run the money
// seam post-commit; if that seam fails the row is left released/reversed with money
// UNMOVED and payout_settled_at NULL, and nothing re-drives it (the auto-confirm
// sweep only touches awaiting_customer_confirmation). This cron re-drives exactly
// those drift rows (settled_at IS NULL) through the SAME settleRelease/settleReverse
// helpers, stamping settled_at on seam success.
//
// SAFETY:
//   - Flag-gated: the whole scan is a pure DB no-op unless payoutMovementEnabled()
//     || MealPlanEscrowActive(). Both default OFF at launch → nothing runs.
//   - Never invents a status transition — it only re-drives the seam for a status an
//     admin/sweep already committed. It cannot double-pay: stampPayoutSettled is a
//     conditional UPDATE on settled_at IS NULL, ReleaseOrderPayouts re-filters
//     OnHold==true, and ReleaseDayPayout tolerates an already-released transfer.
//   - Bounded (sweepBatchLimit per sweep), recover() at the top, log-and-continue per
//     row so one bad row can't abort the batch.
//   - Attempt-capped in the DB (payout_settle_attempts): a permanently-failing
//     transfer is retried at most payoutReconcileMaxAttempts times, then ALERT-logged
//     for ops. The column (not an in-memory counter) is deliberate — the cron may run
//     as a fresh Temporal activity process each tick, where an in-memory map would
//     never persist and so never cap. Tradeoff: one extra small INT column vs. a
//     counter that silently fails to cap under the per-activity Temporal model.

// payoutReconcileInterval is how often the reconcile sweep runs.
const payoutReconcileInterval = 10 * time.Minute

// payoutReconcileMaxAttempts caps re-drives of a single stranded row before it is
// ALERT-logged and left for ops (stops a permanently-bad transfer looping forever).
const payoutReconcileMaxAttempts = 5

// settleFn is the shared "run seam then stamp settled_at" signature — settleRelease
// / settleReverse both satisfy it, so the sweeps stay status-generic.
type settleFn func(db *gorm.DB, aggType string, id uuid.UUID) error

// StartPayoutReconcileCron is the legacy in-process fallback (used when Temporal is
// off): run once, then on a ticker until ctx is cancelled.
func StartPayoutReconcileCron(ctx context.Context) {
	go func() {
		runPayoutReconcileScan(ctx)
		ticker := time.NewTicker(payoutReconcileInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("payout-reconcile: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runPayoutReconcileScan(ctx)
			}
		}
	}()
	log.Println("payout-reconcile: cron started (interval=10m)")
}

// runPayoutReconcileScan re-drives every released/reversed-but-unsettled hold once.
// Pure no-op while both escrow flags are OFF.
func runPayoutReconcileScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("payout-reconcile: panic recovered: %v", r)
		}
	}()

	if !payoutMovementEnabled() && !MealPlanEscrowActive() {
		return // both escrow flags off → the seam is a no-op, nothing to reconcile
	}

	n := 0
	n += reconcileOrders(models.PayoutHoldReleased, settleRelease)
	n += reconcileOrders(models.PayoutHoldReversed, settleReverse)
	n += reconcileMealPlanDays(models.PayoutHoldReleased, settleRelease)
	n += reconcileMealPlanDays(models.PayoutHoldReversed, settleReverse)
	if n > 0 {
		log.Printf("payout-reconcile: re-drove %d stranded hold(s)", n)
	}
}

// reconcileOrders re-drives drift order holds in the given status. Bounded; returns
// the count actually settled.
func reconcileOrders(status models.PayoutHoldStatus, settle settleFn) int {
	var ids []string
	if err := database.DB.Model(&models.Order{}).
		Where("payout_hold_status = ? AND payout_settled_at IS NULL AND razorpay_order_id <> '' AND payout_settle_attempts < ?",
			status, payoutReconcileMaxAttempts).
		Limit(sweepBatchLimit).Pluck("id", &ids).Error; err != nil {
		log.Printf("payout-reconcile: query %s orders failed: %v", status, err)
		return 0
	}
	return driveSettles(aggTypeOrder, ids, settle)
}

// reconcileMealPlanDays re-drives drift meal-plan-day holds in the given status.
// Guards on a present transfer id (the seam has nothing to release otherwise).
func reconcileMealPlanDays(status models.PayoutHoldStatus, settle settleFn) int {
	var ids []string
	if err := database.DB.Model(&models.MealPlanDay{}).
		Where("payout_hold_status = ? AND payout_settled_at IS NULL AND payout_transfer_id <> '' AND payout_settle_attempts < ?",
			status, payoutReconcileMaxAttempts).
		Limit(sweepBatchLimit).Pluck("id", &ids).Error; err != nil {
		log.Printf("payout-reconcile: query %s meal-plan days failed: %v", status, err)
		return 0
	}
	return driveSettles(aggTypeMealPlanDay, ids, settle)
}

// driveSettles runs the settle seam per id, log-and-continue on failure (with an
// attempt bump), returning how many settled cleanly.
func driveSettles(aggType string, ids []string, settle settleFn) int {
	driven := 0
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			log.Printf("payout-reconcile: bad %s id %q: %v", aggType, raw, err)
			continue
		}
		if err := settle(database.DB, aggType, id); err != nil {
			bumpSettleAttempt(aggType, id, err)
			continue
		}
		driven++
	}
	return driven
}

// bumpSettleAttempt increments the DB-backed attempt counter for a failed re-drive
// and ALERT-logs when it reaches the cap so ops can act on a permanently-bad row.
func bumpSettleAttempt(aggType string, id uuid.UUID, cause error) {
	model, mErr := holdModel(aggType)
	if mErr != nil {
		log.Printf("payout-reconcile: %v", mErr)
		return
	}
	if err := database.DB.Model(model).Where("id = ?", id).
		UpdateColumn("payout_settle_attempts", gorm.Expr("payout_settle_attempts + 1")).Error; err != nil {
		log.Printf("payout-reconcile: bump attempts for %s %s failed: %v", aggType, id, err)
		return
	}
	var attempts int
	database.DB.Model(model).Where("id = ?", id).Pluck("payout_settle_attempts", &attempts)
	if attempts >= payoutReconcileMaxAttempts {
		log.Printf("payout-reconcile: ALERT %s %s reached %d settle attempts, giving up (last error: %v)", aggType, id, attempts, cause)
		return
	}
	log.Printf("payout-reconcile: %s %s settle failed (attempt %d): %v", aggType, id, attempts, cause)
}
