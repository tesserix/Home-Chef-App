package services

// meal_plan_day_resolve.go — #393 slice B. Execute the admin-confirmed money policy for
// a delivery-FAILED meal-plan day (frozen `disputed` by slice A), then re-check the plan
// for completion. The day-specific analog of ResolveDeliveryFailure (which handles the
// gateway-order aggregate): the #582 order resolver does NOT fit days — it would open an
// OrderIssue on the shell per-day order and route to RefundIssueToWallet(order.Total), a
// SECOND refund path racing RefundDay (double-refund) that also never advances the day
// status. So the day flow uses RefundDay (the day money seam) directly.
//
// Hybrid model (owner policy, [[project_rto_money_policy]]):
//   - customer-fault → NO refund; the chef is paid (food was made). The day is
//     terminalized `delivered` (the only paid-terminal day state — reusing it keeps the
//     day inside all the existing paid-day machinery: the admin payout queue, the
//     release cross-guards, the reconcile cron) and its hold driven disputed →
//     release_eligible for the admin payout queue (#388).
//   - platform-fault / chef-fault → FULL day refund (perDayGross to wallet) + reverse the
//     chef's held transfer via RefundDay, and the day is terminalized `refunded`.
//   - ambiguous / unknown → ErrAmbiguousFault.

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ResolveMealPlanDayFailure executes the admin-confirmed money policy for a failed
// meal-plan day and completes the plan if every day is now terminal. Idempotent via a
// claim on the `failed` status: a concurrent/duplicate resolution loses the guarded
// status UPDATE and returns ErrIssueAlreadyHandled. plan MUST be loaded with its
// snapshotted totals (Subtotal/Tax/Total) and Days for the RefundDay per-day-gross basis;
// day is the target failed day.
func ResolveMealPlanDayFailure(db *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, fault models.DeliveryFaultClass, adminID uuid.UUID) error {
	if day.Status != models.MealPlanDayFailed {
		return ErrNotDeliveryFailure
	}
	switch fault {
	case models.FaultCustomer:
		return db.Transaction(func(tx *gorm.DB) error {
			ok, err := claimFailedDay(tx, day.ID, models.MealPlanDayDelivered, true)
			if err != nil {
				return err
			}
			if !ok {
				return ErrIssueAlreadyHandled
			}
			// Chef legitimately earned the payout → move the disputed hold into the pay
			// queue (the admin still does the final #388 release). No customer refund; the
			// retained delivery fee follows automatically (no money moves out).
			if err := releaseDisputedDayHoldForCustomerFault(tx, day.ID); err != nil {
				return err
			}
			return completePlanIfAllDaysTerminal(tx, plan.ID)
		})
	case models.FaultPlatform, models.FaultChef:
		return db.Transaction(func(tx *gorm.DB) error {
			ok, err := claimFailedDay(tx, day.ID, models.MealPlanDayRefunded, false)
			if err != nil {
				return err
			}
			if !ok {
				return ErrIssueAlreadyHandled
			}
			// Full refund + reverse the chef's held transfer (flag-gated; a no-op when
			// escrow is OFF). RefundDay stamps refund_txn_id and drives the hold out of the
			// releasable set (disputed → withheld/reversed) via reverseRefundedDayHold.
			if err := RefundDay(tx, plan, day, "delivery failed — "+string(fault)+"-fault refund"); err != nil {
				return err
			}
			// Belt-and-braces: when escrow is OFF, RefundDay returns before touching the
			// hold, leaving it `disputed` on a now-refunded day — which an unrelated
			// order-issue clear could still flip to release_eligible. Drive disputed →
			// withheld so a refunded day is never payable. Idempotent (no-op once RefundDay
			// has already moved it).
			if err := withholdDisputedDayHold(tx, day.ID); err != nil {
				return err
			}
			if err := EnqueueEvent(tx, SubjectMealPlanDayRefunded, "meal_plan.day_refunded", plan.CustomerID, map[string]any{
				"meal_plan_id": plan.ID.String(), "day_id": day.ID.String(), "reason": "delivery failed",
			}); err != nil {
				return err
			}
			return completePlanIfAllDaysTerminal(tx, plan.ID)
		})
	default: // FaultAmbiguous or anything unrecognized
		return ErrAmbiguousFault
	}
}

// claimFailedDay is the concurrency/idempotency gate: a guarded UPDATE that flips the day
// out of `failed` to the terminal `to` status, returning ok=false (no error) when the day
// is no longer `failed` (already resolved by a concurrent call). When stampDelivered is
// set the delivered_at timestamp is set to the resolution time (a paid-terminal day needs
// a delivered_at for the payout-queue age ordering).
func claimFailedDay(tx *gorm.DB, dayID uuid.UUID, to models.MealPlanDayStatus, stampDelivered bool) (bool, error) {
	updates := map[string]any{"status": to}
	if stampDelivered {
		updates["delivered_at"] = time.Now()
	}
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND status = ?", dayID, models.MealPlanDayFailed).
		Updates(updates)
	if res.Error != nil {
		return false, fmt.Errorf("meal-plan day resolve: claim day %s: %w", dayID, res.Error)
	}
	return res.RowsAffected == 1, nil
}

// releaseDisputedDayHoldForCustomerFault drives a customer-fault day's hold
// disputed → release_eligible (the explicit admin authorization to pay the chef), emitting
// the release-eligible event for the #388 admin payout queue. Guarded to disputed only, so
// a settled hold is never disturbed; a no-op emits nothing.
func releaseDisputedDayHoldForCustomerFault(tx *gorm.DB, dayID uuid.UUID) error {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status = ?", dayID, models.PayoutHoldDisputed).
		Update("payout_hold_status", models.PayoutHoldReleaseEligible)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: release customer-fault day %s: %w", dayID, res.Error)
	}
	if res.RowsAffected == 0 {
		// This is the SOLE writer expected to fire here (slice A always freezes the hold
		// to disputed atomically with the failed status). A zero-match means the day was
		// marked paid-terminal but its hold is NOT in the pay queue — surface it rather
		// than silently 200 the admin, so a broken freeze invariant is visible.
		log.Printf("meal-plan day resolve: customer-fault day %s had no disputed hold to release — hold NOT queued for payout", dayID)
		return nil
	}
	return emitHoldEvent(tx, models.PayoutHoldReleaseEligible, aggTypeMealPlanDay, dayID)
}

// withholdDisputedDayHold drives a still-disputed hold → withheld (a terminal non-payable
// state). Used on the refund path as a flag-off backstop; a no-op once RefundDay has
// already driven the hold. No event, no money (mirrors WithholdHold's semantics).
func withholdDisputedDayHold(tx *gorm.DB, dayID uuid.UUID) error {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status = ?", dayID, models.PayoutHoldDisputed).
		Update("payout_hold_status", models.PayoutHoldWithheld)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: withhold refunded day %s: %w", dayID, res.Error)
	}
	if res.RowsAffected > 0 {
		// Records the "why" for the flags-off path (RefundDay no-op'd, so no crossguard
		// audit row was written) — mirrors withholdOrReverseHoldForRefund's audit.
		LogSystemAudit(nil, "payout.hold.day_refund_withheld", string(aggTypeMealPlanDay), dayID.String(),
			models.PayoutHoldDisputed, models.PayoutHoldWithheld)
	}
	return nil
}

// completePlanIfAllDaysTerminal re-checks the plan (with the just-committed day status,
// read within the same tx) and marks it completed once every day is terminal. Guarded on
// the confirmed/active source status so a concurrent transition wins (mirrors
// completeFinishedPlans). A `failed` day is non-terminal, so a plan with any other
// unresolved failed day stays open.
func completePlanIfAllDaysTerminal(tx *gorm.DB, planID uuid.UUID) error {
	var statuses []models.MealPlanDayStatus
	if err := tx.Model(&models.MealPlanDay{}).
		Where("meal_plan_id = ?", planID).Pluck("status", &statuses).Error; err != nil {
		return fmt.Errorf("meal-plan day resolve: reload day statuses for plan %s: %w", planID, err)
	}
	days := make([]models.MealPlanDay, len(statuses))
	for i, s := range statuses {
		days[i].Status = s
	}
	if !allDaysTerminal(&models.MealPlan{Days: days}) {
		return nil
	}
	res := tx.Model(&models.MealPlan{}).
		Where("id = ? AND status IN ?", planID,
			[]models.MealPlanStatus{models.MealPlanConfirmed, models.MealPlanActive}).
		Update("status", models.MealPlanCompleted)
	if res.Error != nil {
		return fmt.Errorf("meal-plan day resolve: complete plan %s: %w", planID, res.Error)
	}
	return nil
}
