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

	// The delivery-failure FREEZE safety net runs FIRST and UNCONDITIONALLY: disputing a
	// hold is plain DB state (no money moves), so a stranded failed/returned order must be
	// frozen even with both escrow flags OFF (the pre-launch money-safety invariant).
	if df := reconcileStrandedDeliveryFailures(); df > 0 {
		log.Printf("payout-reconcile: froze %d stranded delivery-failure order(s)", df)
	}

	if !payoutMovementEnabled() && !MealPlanEscrowActive() {
		return // both escrow flags off → the money seam is a no-op, nothing to settle
	}

	n := 0
	n += reconcileOrders(models.PayoutHoldReleased, settleRelease)
	n += reconcileOrders(models.PayoutHoldReversed, settleReverse)
	n += reconcileMealPlanDays(models.PayoutHoldReleased, settleRelease)
	n += reconcileMealPlanDays(models.PayoutHoldReversed, settleReverse)
	n += reconcileGroupOrders(models.PayoutHoldReleased, settleRelease)
	n += reconcileGroupOrders(models.PayoutHoldReversed, settleReverse)
	// A cancelled group whose reverse never ran sits at a non-terminal hold with a
	// held transfer — invisible to the status-keyed sweeps above (#534).
	n += reconcileCancelledGroups()
	// Same class for orders + meal-plan days: a refund cross-guard that crashed before
	// flipping the hold terminal leaves a cancelled/refunded aggregate at a parked hold
	// with a held transfer, unseen by the released/reversed sweeps (#542).
	n += reconcileCancelledOrders()
	n += reconcileRefundedDays()
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

// reconcileGroupOrders re-drives drift group/office order holds in the given status.
// Guards on a present transfer id (the seam has nothing to release otherwise).
func reconcileGroupOrders(status models.PayoutHoldStatus, settle settleFn) int {
	var ids []string
	if err := database.DB.Model(&models.GroupOrder{}).
		Where("payout_hold_status = ? AND payout_settled_at IS NULL AND payout_transfer_id <> '' AND payout_settle_attempts < ?",
			status, payoutReconcileMaxAttempts).
		Limit(sweepBatchLimit).Pluck("id", &ids).Error; err != nil {
		log.Printf("payout-reconcile: query %s group orders failed: %v", status, err)
		return 0
	}
	return driveSettles(aggTypeGroupOrder, ids, settle)
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

// reconcileCancelledGroups claws back a stranded held transfer on a CANCELLED group
// whose reverse never ran (#534). The released/reversed sweeps above only catch holds
// already flipped to a terminal-in-flight status; a cancelled group left at
// none/awaiting/… with a held transfer (crash after the cancel tx committed, no client
// retry) is invisible to them, so the chef's held payout is never clawed back.
//
// Uses ReverseGroupHoldForCancel (NOT the generic settleReverse): the gap rows haven't
// transitioned yet, so they need the guarded transition (→ reversed) + settle, not just
// a seam re-drive. It self-guards on status==cancelled and is idempotent, and stamps
// hold=reversed + settled_at so a driven row drops out of the next scan on both the
// `NOT IN (reversed)` and `settled_at IS NULL` predicates.
func reconcileCancelledGroups() int {
	var ids []string
	if err := database.DB.Model(&models.GroupOrder{}).
		Where(`status = ? AND payout_hold_status NOT IN ? AND payout_settled_at IS NULL
		       AND payout_transfer_id <> '' AND payout_settle_attempts < ?`,
			models.GroupOrderCancelled,
			[]models.PayoutHoldStatus{models.PayoutHoldWithheld, models.PayoutHoldReversed},
			payoutReconcileMaxAttempts).
		Limit(sweepBatchLimit).Pluck("id", &ids).Error; err != nil {
		log.Printf("payout-reconcile: query cancelled group holds failed: %v", err)
		return 0
	}
	driven := 0
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			log.Printf("payout-reconcile: bad group id %q: %v", raw, err)
			continue
		}
		if err := ReverseGroupHoldForCancel(database.DB, id, "reconcile: stranded cancelled group"); err != nil {
			bumpSettleAttempt(aggTypeGroupOrder, id, err)
			continue
		}
		driven++
	}
	return driven
}

// parkedActionableHolds is the set of non-terminal holds a stranded refund/cancel
// can still act on: withheld/reversed are terminal (already handled) and none has no
// hold to drive. withholdOrReverseHoldForRefund moves these → withheld (parked) or
// → reversed (released), so after a drive the row drops out of the next scan.
var parkedActionableHolds = []models.PayoutHoldStatus{
	models.PayoutHoldAwaitingConfirmation,
	models.PayoutHoldReleaseEligible,
	models.PayoutHoldDisputed,
	models.PayoutHoldReleased,
}

// reconcileCancelledOrders drives the refund cross-guard on an order that was
// refunded/cancelled but whose hold never reached a terminal state — the crash
// window #542 closes (mirror of reconcileCancelledGroups for the order aggregate).
// An order refunded via the issue path keeps status='delivered' but stamps
// refunded_at, so both are matched. Scoped to gateway-charged orders (a wallet-only
// order has no Route transfer to strand). Idempotent: withholdOrReverseHoldForRefund
// moves the hold out of parkedActionableHolds, so a driven row leaves the scan.
func reconcileCancelledOrders() int {
	var ids []string
	if err := database.DB.Model(&models.Order{}).
		Where(`(status IN ? OR refunded_at IS NOT NULL) AND payout_hold_status IN ?
		       AND payout_settled_at IS NULL AND razorpay_order_id <> '' AND payout_settle_attempts < ?`,
			[]models.OrderStatus{models.OrderStatusCancelled, models.OrderStatusRefunded},
			parkedActionableHolds, payoutReconcileMaxAttempts).
		Limit(sweepBatchLimit).Pluck("id", &ids).Error; err != nil {
		log.Printf("payout-reconcile: query stranded refunded/cancelled orders failed: %v", err)
		return 0
	}
	driven := 0
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			log.Printf("payout-reconcile: bad order id %q: %v", raw, err)
			continue
		}
		if err := withholdOrReverseHoldForRefund(database.DB, aggTypeOrder, id, "reconcile: stranded refunded/cancelled order"); err != nil {
			bumpSettleAttempt(aggTypeOrder, id, err)
			continue
		}
		driven++
	}
	return driven
}

// reconcileRefundedDays is reconcileCancelledOrders for the meal-plan-day aggregate:
// a day refunded/cancelled but left at a parked hold with a held transfer (its
// RefundDay cross-guard crashed before reverseRefundedDayHold). Scoped to days that
// actually hold a transfer (payout_transfer_id <> ”). NOTE: a day whose reverse
// FAILED (not crashed) is left at hold=reversed+unsettled by reverseRefundedDayHold
// (#398) — that drift is re-driven by reconcileMealPlanDays(reversed), not here.
func reconcileRefundedDays() int {
	var ids []string
	if err := database.DB.Model(&models.MealPlanDay{}).
		Where(`status IN ? AND payout_hold_status IN ? AND payout_settled_at IS NULL
		       AND payout_transfer_id <> '' AND payout_settle_attempts < ?`,
			[]models.MealPlanDayStatus{models.MealPlanDayRefunded, models.MealPlanDayCancelled},
			parkedActionableHolds, payoutReconcileMaxAttempts).
		Limit(sweepBatchLimit).Pluck("id", &ids).Error; err != nil {
		log.Printf("payout-reconcile: query stranded refunded/cancelled days failed: %v", err)
		return 0
	}
	driven := 0
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			log.Printf("payout-reconcile: bad meal-plan-day id %q: %v", raw, err)
			continue
		}
		if err := withholdOrReverseHoldForRefund(database.DB, aggTypeMealPlanDay, id, "reconcile: stranded refunded/cancelled day"); err != nil {
			bumpSettleAttempt(aggTypeMealPlanDay, id, err)
			continue
		}
		driven++
	}
	return driven
}
