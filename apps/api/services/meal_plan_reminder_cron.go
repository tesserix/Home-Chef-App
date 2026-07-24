package services

// meal_plan_reminder_cron.go — the daily chef "cook reminder" sweep for tiffin
// meal-plan days.
//
// A confirmed meal-plan day (models.MealPlanDayConfirmed) is a meal the chef has
// committed to cook on a given date. This sweep tells each chef, twice a day, how
// many tiffin meals they must cook, split by slot:
//
//   - Night-before window: at 20:00 IST it looks at TOMORROW's confirmed days
//     (window="tomorrow") so the chef can shop/prep the evening before.
//   - Morning-of window:   at 07:00 IST it looks at TODAY's confirmed days
//     (window="today") as a start-of-day cook list.
//
// Outside those two IST hours the scan is a no-op. Only status=confirmed days are
// counted — delivered / prepared / skipped / cancelled / declined / refunded /
// failed days are not "to cook" and are excluded.
//
// Dedup: the sweep runs on a 30-minute ticker, so a window's hour is seen twice.
// reminderAlreadySent gates one reminder per (chef, targetDate, window) through a
// Redis SETNX key (TTL ~20h) so the second tick in the same hour — or a second
// pod — does not re-notify. It fails OPEN (Redis down → fire) so a chef is never
// silently skipped; a rare duplicate is preferable to a missed cook list.
//
// This file only EMITS the notification event (subject
// meal_plans.chef_cook_reminder). The notification HANDLER that renders the in-app
// copy + push is wired separately in notifications.go.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const mealPlanReminderInterval = 30 * time.Minute

// The reminder event is staged under SubjectMealPlanChefReminder (services/nats.go);
// the handler wiring (in-app copy + push) lives in notifications.go.

// chefReminderRow is one aggregated (chef, slot counts) row from the sweep query.
type chefReminderRow struct {
	ChefUserID uuid.UUID
	Lunch      int
	Dinner     int
}

// chefCookReminderNotifier stages the per-chef reminder event. A package var so
// tests can swap it for a recorder instead of hitting the DB outbox. The event's
// userID is the chef's users.id (chef_profiles.user_id) so the handler pushes to
// the chef.
var chefCookReminderNotifier = func(db *gorm.DB, chefUserID uuid.UUID, targetDate string, window string, lunch, dinner int) error {
	return EnqueueEvent(db, SubjectMealPlanChefReminder, "meal_plan.chef_cook_reminder", chefUserID, map[string]any{
		"target_date": targetDate,
		"window":      window,
		"lunch":       lunch,
		"dinner":      dinner,
		"total":       lunch + dinner,
	})
}

// reminderAlreadySent reports whether this (chef, targetDate, window) reminder has
// already fired within the TTL, so a re-tick in the same window does not re-notify.
// A package var so tests can force it on/off. Default: Redis SETNX with a ~20h TTL
// (covers both intra-hour re-ticks and the gap between the two daily windows). It
// fails OPEN — if Redis is nil/unavailable it returns false so the reminder still
// fires (a duplicate is better than a chef never told what to cook).
var reminderAlreadySent = func(chefUserID uuid.UUID, targetDate, window string) bool {
	r := GetRedisClient()
	if !r.IsConnected() {
		return false // fail-open: fire rather than silently suppress
	}
	key := fmt.Sprintf("mpchefreminder:%s:%s:%s", chefUserID, targetDate, window)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	acquired, err := r.SetNX(ctx, key, "1", 20*time.Hour)
	if err != nil {
		return false // fail-open on a Redis error
	}
	// acquired == true means WE set the key first (not yet sent); false means the
	// key already existed (already sent).
	return !acquired
}

// istDayWindow returns the YYYY-MM-DD label and the [start, end) UTC instants of
// the IST calendar day `offsetDays` from now (0 = today IST, 1 = tomorrow IST).
// Reuses scheduleIST (kitchen_schedule_cron.go) rather than loading a new tz. The
// UTC bounds make the date filter correct on Postgres regardless of how the day's
// stored timestamp was offset.
func istDayWindow(now time.Time, offsetDays int) (string, time.Time, time.Time) {
	ist := now.In(scheduleIST)
	start := time.Date(ist.Year(), ist.Month(), ist.Day(), 0, 0, 0, 0, scheduleIST).AddDate(0, 0, offsetDays)
	return start.Format("2006-01-02"), start.UTC(), start.AddDate(0, 0, 1).UTC()
}

// scanChefCookReminders is the testable sweep core (db + now injected). It picks
// the active IST window from now's hour, aggregates confirmed meal-plan days per
// chef for the window's target date, and fires one reminder per not-yet-reminded
// chef. Returns the number of chefs notified. Takes now from the caller — never
// calls time.Now() itself.
func scanChefCookReminders(db *gorm.DB, now time.Time) int {
	var window string
	var offsetDays int
	switch now.In(scheduleIST).Hour() {
	case 20:
		window, offsetDays = "tomorrow", 1
	case 7:
		window, offsetDays = "today", 0
	default:
		return 0 // no window active at this hour
	}

	targetDate, startUTC, endUTC := istDayWindow(now, offsetDays)

	// Aggregate lunch/dinner counts per chef user for confirmed days landing in the
	// target IST day. Join day -> plan (chef_id) -> chef_profile (user_id).
	var rows []chefReminderRow
	if err := db.Table("meal_plan_days AS d").
		Select("cp.user_id AS chef_user_id, "+
			"SUM(CASE WHEN d.slot = ? THEN 1 ELSE 0 END) AS lunch, "+
			"SUM(CASE WHEN d.slot = ? THEN 1 ELSE 0 END) AS dinner",
			models.MealSlotLunch, models.MealSlotDinner).
		Joins("JOIN meal_plans mp ON mp.id = d.meal_plan_id").
		Joins("JOIN chef_profiles cp ON cp.id = mp.chef_id").
		Where("d.status = ? AND d.date >= ? AND d.date < ?",
			models.MealPlanDayConfirmed, startUTC, endUTC).
		Group("cp.user_id").
		Limit(sweepBatchLimit).
		Scan(&rows).Error; err != nil {
		log.Printf("meal-plan-reminder: query failed: %v", err)
		return 0
	}

	notified := 0
	for _, row := range rows {
		if row.Lunch+row.Dinner == 0 {
			continue
		}
		if reminderAlreadySent(row.ChefUserID, targetDate, window) {
			continue
		}
		if err := chefCookReminderNotifier(db, row.ChefUserID, targetDate, window, row.Lunch, row.Dinner); err != nil {
			// Log-and-continue: one chef's failed notify must not abort the batch.
			log.Printf("meal-plan-reminder: notify chef=%s failed: %v", row.ChefUserID, err)
			continue
		}
		notified++
	}
	return notified
}

// runMealPlanChefReminderScan is the panic-recovered wrapper the ticker calls.
func runMealPlanChefReminderScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("meal-plan-reminder: panic recovered: %v", r)
		}
	}()
	now := time.Now()
	if n := scanChefCookReminders(database.DB, now); n > 0 {
		log.Printf("meal-plan-reminder: reminded %d chef(s) of tiffin meals to cook", n)
	}
}

// StartMealPlanReminderCron is the legacy in-process fallback (used when Temporal
// is off): run once, then on a ticker until ctx is cancelled.
func StartMealPlanReminderCron(ctx context.Context) {
	go func() {
		runMealPlanChefReminderScan(ctx)
		ticker := time.NewTicker(mealPlanReminderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("meal-plan-reminder: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runMealPlanChefReminderScan(ctx)
			}
		}
	}()
	log.Printf("meal-plan-reminder: cron started (interval=%s)", mealPlanReminderInterval)
}
