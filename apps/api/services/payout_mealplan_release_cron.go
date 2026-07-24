package services

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// payout_mealplan_release_cron.go — the per-day tiffin payout auto-release sweep.
//
// This is the FINAL stage of the meal-plan-day payout pipeline and runs AFTER the
// auto-confirm sweep (services/payout_auto_confirm_cron.go). The flow for one
// tiffin day is:
//
//	delivered → awaiting_customer_confirmation   (hold parked, no money moves)
//	  ── customer confirms, OR the confirm window lapses (auto-confirm sweep) ──▶
//	release_eligible                             (customer_confirmed_at stamped)
//	  ── this sweep: ~24h maturation AFTER that confirmation ──▶
//	released                                     (money seam + payout_settled_at)
//
// So a day is released roughly TWO days after delivery: ~24h for the customer
// confirm / auto-confirm window, then a further ~24h maturation here as a last
// window in which a late dispute or refund can still claw the hold back before
// the chef is paid. The maturation is measured from customer_confirmed_at (when
// the day became release_eligible), NOT from delivery.
//
// The transition + flag-gated money seam live in ReleaseHold
// (services/payout_release.go); this sweep only selects matured, release_eligible
// day holds and calls it. ReleaseHold is idempotent and re-checks the
// refund/dispute cross-guards atomically with the flip, so repeated or concurrent
// runs are safe and a row that stopped being eligible is skipped, not forced. The
// money seam self-no-ops while meal-plan escrow is OFF, so the DB advance commits
// regardless — no gateway is required.
//
// Gated by the payout.mealplan_auto_release_enabled platform setting, which is
// opt-OUT: unset/empty means ON. Only an explicit false/0/off turns it off. This
// differs from the order-release sweep's opt-IN kill switch — day escrow releases
// per-day money the platform already holds and ReleaseHold's own cross-guards are
// the terminal safety net, so the default is to keep matured days flowing.

// mealPlanReleaseInterval is how often the legacy in-process ticker fires.
const mealPlanReleaseInterval = 15 * time.Minute

// defaultMealPlanMaturation is the hold applied AFTER a day becomes
// release_eligible, used when payout.mealplan_maturation_hours is unset,
// unparseable, or non-positive. Never zero — that would pay the chef the instant
// the customer confirms, removing the window a late refund/dispute needs.
const defaultMealPlanMaturation = 24 * time.Hour

// mealPlanAutoReleaseEnabled reports whether the per-day auto-release sweep may
// run. Opt-OUT: unset/empty → true; only an explicit "false"/"0"/"off"
// (case-insensitive) disables it. Deliberately NOT settingBool, which defaults
// false — here the safe/default state is ON.
func mealPlanAutoReleaseEnabled(db *gorm.DB) bool {
	v := strings.ToLower(strings.TrimSpace(settingValue(db, "payout.mealplan_auto_release_enabled")))
	switch v {
	case "":
		return true // unset → default ON (opt-out)
	case "true", "1", "on", "yes", "enabled":
		return true
	case "false", "0", "off", "no", "disabled":
		return false
	default:
		// A non-empty, non-canonical value is almost certainly an ops typo. For a
		// money mover, fail toward HOLDING money: treat it as OFF and log, rather
		// than silently keeping payouts flowing when an admin tried to pause them.
		log.Printf("mealplan-release-sweep: payout.mealplan_auto_release_enabled=%q not recognized — treating as OFF (paused). Use true/1/on to enable.", v)
		return false
	}
}

// mealPlanReleaseMaturation reads the post-confirmation hold (whole hours) from
// payout.mealplan_maturation_hours, falling back to defaultMealPlanMaturation on
// anything unset, unparseable, or non-positive.
func mealPlanReleaseMaturation(db *gorm.DB) time.Duration {
	hours, err := strconv.Atoi(strings.TrimSpace(settingValue(db, "payout.mealplan_maturation_hours")))
	if err != nil || hours <= 0 {
		return defaultMealPlanMaturation
	}
	return time.Duration(hours) * time.Hour
}

// sweepMaturedMealPlanDays releases every release_eligible meal-plan day whose
// customer confirmation has matured past the configured window. It is the pure,
// unit-testable core: the caller supplies now (a deterministic clock in tests),
// and it returns the number of candidate rows scanned this pass.
//
// A row is a candidate when it is release_eligible, not yet settled, and was
// confirmed (customer_confirmed_at) at least the maturation window ago. Each is
// driven through ReleaseHold; an ErrHoldNotEligible (a refund/dispute landed, or
// a concurrent pass won the race) is a quiet skip, and any other error is logged
// and the batch continues so one bad row cannot abort the sweep.
func sweepMaturedMealPlanDays(db *gorm.DB, now time.Time) int {
	if !mealPlanAutoReleaseEnabled(db) {
		return 0
	}
	cutoff := now.Add(-mealPlanReleaseMaturation(db))

	// Maturation anchor = COALESCE(customer_confirmed_at, delivered_at). A normal day
	// anchors on customer_confirmed_at (stamped at the customer's confirm or the 24h
	// auto-confirm). An admin-resolved customer-fault delivery-failed day reaches
	// release_eligible via releaseDisputedDayHoldForCustomerFault WITHOUT a
	// customer_confirmed_at (the customer never confirmed) but WITH delivered_at
	// stamped at resolution — anchoring those on delivered_at auto-releases the
	// legitimately-owed chef payout ~maturation after resolution, instead of
	// stranding it out of the auto path and relying on a manual admin release.
	var days []models.MealPlanDay
	if err := db.
		Where("payout_hold_status = ? AND payout_settled_at IS NULL AND "+
			"COALESCE(customer_confirmed_at, delivered_at) <= ?",
			models.PayoutHoldReleaseEligible, cutoff).
		Limit(sweepBatchLimit).Find(&days).Error; err != nil {
		log.Printf("mealplan-release-sweep: query matured days failed: %v", err)
		return 0
	}

	for i := range days {
		err := ReleaseHold(db, aggTypeMealPlanDay, days[i].ID)
		if err != nil {
			if !errors.Is(err, ErrHoldNotEligible) {
				// ErrHoldNotEligible = a guard refused (refund/dispute/race) — ordinary
				// skip. Anything else is a real error worth logging. Either way: no notify.
				log.Printf("mealplan-release-sweep: release day %s failed: %v", days[i].ID, err)
			}
			continue
		}
		// Released just now. ReleaseHold's release_eligible→released flip is a single
		// guarded UPDATE (once-only per day), so this notifies the chef exactly once
		// per real release — the reconcile re-drive path goes through settlePayout,
		// not ReleaseHold, so it can't double-notify. Best-effort (logged, swallowed).
		if e := notifyChefDayPayoutReleased(db, &days[i]); e != nil {
			log.Printf("mealplan-release-sweep: notify chef for day %s: %v", days[i].ID, e)
		}
	}
	return len(days)
}

// runMealPlanDayReleaseScan is the panic-guarded entry the ticker (and any
// external scheduler) calls: it fixes now := time.Now() and runs one sweep,
// logging the scanned count only when it did work.
func runMealPlanDayReleaseScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("mealplan-release-sweep: panic recovered: %v", r)
		}
	}()
	now := time.Now()
	if n := sweepMaturedMealPlanDays(database.DB, now); n > 0 {
		log.Printf("mealplan-release-sweep: scanned %d matured meal-plan day(s)", n)
	}
}

// StartMealPlanDayReleaseCron is the legacy in-process fallback ticker (used when
// Temporal is off): fire the sweep every mealPlanReleaseInterval until ctx is
// cancelled.
func StartMealPlanDayReleaseCron(ctx context.Context) {
	go func() {
		t := time.NewTicker(mealPlanReleaseInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runMealPlanDayReleaseScan(ctx)
			}
		}
	}()
}
