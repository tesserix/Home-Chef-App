package services

// meal_plan_day_skip.go — #422 policy change. A customer's "skip a day" is no longer a
// self-service auto-credit; it raises an ADMIN-reviewed request. The customer handler
// freezes the confirmed day to `skip_req` and disputes its payout hold (handlers/meal_plan.go
// SkipMealPlanDay). This file executes the admin's decision:
//
//   - approve → the day becomes terminally `skipped`; the customer is refunded the day's
//     FOOD price MINUS the platform commission (perDaySkipRefund) — they forfeit GST +
//     delivery + commission — and the chef's held transfer is FULLY reversed (0 for a day
//     never cooked). The plan completes if every day is now terminal.
//   - reject → the day returns to `confirmed`; its frozen hold is restored to none and the
//     customer is told the skip was declined. No money moves.
//
// Models the approve path on ResolveMealPlanDayFailure's platform/chef-fault branch (claim
// → refund + hold claw-back → terminalize → notify → complete). Every transition is guarded
// on the day currently being `skip_req`, so a duplicate/racing resolution loses the claim
// and returns ErrIssueAlreadyHandled.

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// ErrNotSkipRequest is returned when a skip resolution targets a day that isn't in the
// `skip_req` state (never requested, or already resolved by a concurrent call).
var ErrNotSkipRequest = errors.New("day is not in a skip-requested state")

// ResolveMealPlanDaySkip executes the admin's decision on a `skip_req` day. plan MUST
// carry its snapshotted totals (Subtotal/Tax/Total) + the target day for the refund basis.
// approve=true credits perDaySkipRefund and terminalizes the day `skipped`; approve=false
// returns the day to `confirmed`. Idempotent via the guarded claim on `skip_req`.
func ResolveMealPlanDaySkip(db *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, approve bool, resolvedBy uuid.UUID) error {
	if day.Status != models.MealPlanDaySkipRequested {
		return ErrNotSkipRequest
	}
	if approve {
		return db.Transaction(func(tx *gorm.DB) error { return approveSkip(tx, plan, day, resolvedBy) })
	}
	return db.Transaction(func(tx *gorm.DB) error { return rejectSkip(tx, plan, day) })
}

// approveSkip terminalizes a skip_req day `skipped`, refunds food-minus-commission, and
// reverses the chef's hold. Mirrors ResolveMealPlanDayFailure's refund branch.
func approveSkip(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, resolvedBy uuid.UUID) error {
	ok, err := claimSkipDay(tx, day.ID, models.MealPlanDaySkipped)
	if err != nil {
		return err
	}
	if !ok {
		return ErrIssueAlreadyHandled
	}
	// Refund basis: the day's frozen commission rate (what the held transfer was sized at),
	// falling back to the live/default rate for legacy days that predate the frozen column.
	rate := day.CommissionRate
	if rate <= 0 || rate >= 1 {
		rate = GetCommissionRate(tx)
	}
	amount := perDaySkipRefund(plan, day, rate)
	// Refund food − platform fee + FULLY reverse the chef's held transfer (flag-gated; a
	// no-op when escrow is OFF). refundDayAmount stamps refund_txn_id and drives the hold
	// out of the releasable set (disputed → withheld/reversed) via reverseRefundedDayHold.
	if err := refundDayAmount(tx, plan, day, amount, "customer skip approved — food refund less platform fee"); err != nil {
		return err
	}
	// Flags-OFF backstop: when escrow is off refundDayAmount returns before touching the
	// hold, leaving it `disputed` on a now-skipped day. Drive disputed → withheld so a
	// skipped day is never payable. Idempotent (no-op once refundDayAmount has moved it).
	if err := withholdDisputedDayHold(tx, day.ID); err != nil {
		return err
	}
	// Customer: the day was refunded (partial) to their wallet.
	if err := EnqueueEvent(tx, SubjectMealPlanDayRefunded, "meal_plan.day_skip_refunded", plan.CustomerID, map[string]any{
		"meal_plan_id": plan.ID.String(), "day_id": day.ID.String(), "reason": "skip approved", "resolved_by": resolvedBy.String(),
	}); err != nil {
		return err
	}
	// Chef: the day was skipped, so they don't cook it (#422). Best-effort — a missing chef
	// row/user must not fail the money resolution (the notification handler drops a nil user).
	var chef models.ChefProfile
	if err := tx.Select("id", "user_id").First(&chef, "id = ?", plan.ChefID).Error; err == nil && chef.UserID != uuid.Nil {
		if err := EnqueueEvent(tx, SubjectMealPlanDaySkippedChef, "meal_plan.day_skipped_chef", chef.UserID, map[string]any{
			"meal_plan_id": plan.ID.String(), "day_id": day.ID.String(),
			"date": day.Date.Format("2006-01-02"),
		}); err != nil {
			return err
		}
	}
	return completePlanIfAllDaysTerminal(tx, plan.ID)
}

// rejectSkip returns a skip_req day to `confirmed`, restores its frozen hold, and tells
// the customer the skip was declined. No money moves.
func rejectSkip(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay) error {
	ok, err := claimSkipDay(tx, day.ID, models.MealPlanDayConfirmed)
	if err != nil {
		return err
	}
	if !ok {
		return ErrIssueAlreadyHandled
	}
	// The skip froze a confirmed, un-delivered day's hold from `none` → `disputed`; restore
	// it to `none` (a confirmed un-delivered day is not yet payable). Guarded to disputed so
	// a settled hold is never disturbed.
	if err := restoreDisputedDayHoldToNone(tx, day.ID); err != nil {
		return err
	}
	return EnqueueEvent(tx, SubjectMealPlanDaySkipDeclined, "meal_plan.day_skip_declined", plan.CustomerID, map[string]any{
		"meal_plan_id": plan.ID.String(), "day_id": day.ID.String(),
	})
}

// claimSkipDay is the concurrency/idempotency gate for a skip resolution: a guarded UPDATE
// that flips the day out of `skip_req` to `to`, returning ok=false (no error) when the day
// is no longer `skip_req` (already resolved by a concurrent call).
func claimSkipDay(tx *gorm.DB, dayID uuid.UUID, to models.MealPlanDayStatus) (bool, error) {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND status = ?", dayID, models.MealPlanDaySkipRequested).
		Update("status", to)
	if res.Error != nil {
		return false, fmt.Errorf("meal-plan day skip: claim day %s: %w", dayID, res.Error)
	}
	return res.RowsAffected == 1, nil
}

// restoreDisputedDayHoldToNone reverses the skip freeze: disputed → none, for a day
// returning to `confirmed`. Guarded to disputed so a hold that moved on for any other
// reason is untouched. No event/no money (mirrors withholdDisputedDayHold's shape).
func restoreDisputedDayHoldToNone(tx *gorm.DB, dayID uuid.UUID) error {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status = ?", dayID, models.PayoutHoldDisputed).
		Update("payout_hold_status", models.PayoutHoldNone)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: restore skip-declined day %s: %w", dayID, res.Error)
	}
	return nil
}
