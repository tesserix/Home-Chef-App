package services

// meal_subscription_cron.go — the hourly scan that places each active meal
// subscription's order(s) for today once the chef's cutoff (IST) has passed (#282).
// Idempotent: GenerateMealSubscriptionDay is keyed per (sub, date, slot), and a
// per-(sub, date) Redis SETNX dedups across instances. Panic-safe like the other crons.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const mealSubOrderScanInterval = 1 * time.Hour

// StartMealSubscriptionOrderCron runs the order-generation scan as an in-process
// ticker (the Temporal-schedule path calls runMealSubscriptionDailyOrders directly).
func StartMealSubscriptionOrderCron(ctx context.Context) {
	go func() {
		runMealSubscriptionDailyOrders(ctx)
		ticker := time.NewTicker(mealSubOrderScanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("meal-sub-orders: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runMealSubscriptionDailyOrders(ctx)
			}
		}
	}()
	log.Println("meal-sub-orders: cron started (interval=1h)")
}

func runMealSubscriptionDailyOrders(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("meal-sub-orders: panic recovered: %v", r)
		}
	}()

	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.UTC
	}
	nowIST := time.Now().In(ist)
	today := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), 0, 0, 0, 0, ist)

	var subs []models.MealSubscription
	database.DB.Where("status = ?", models.MealSubStatusActive).Find(&subs)

	placedTotal := 0
	for i := range subs {
		sub := subs[i]
		if !IsMealDeliveryDay(&sub, today) {
			continue
		}
		cfg := GetChefSubscriptionConfig(database.DB, sub.ChefID)
		if cfg == nil || !cutoffPassed(nowIST, cfg.CutoffTime, ist) {
			continue
		}
		addr, ok := defaultAddress(sub.CustomerID)
		if !ok {
			continue // no delivery address on file — skip (the customer is nudged elsewhere)
		}
		if !claimMealSubGen(ctx, sub.ID, today) {
			continue
		}
		placed, gErr := GenerateMealSubscriptionDay(database.DB, &sub, today, addr)
		if gErr != nil {
			log.Printf("meal-sub-orders: generate failed for sub=%s: %v", sub.ID, gErr)
			continue
		}
		placedTotal += placed
	}
	if placedTotal > 0 {
		log.Printf("meal-sub-orders: placed %d subscription order(s)", placedTotal)
	}
}

// cutoffPassed reports whether the current IST time-of-day is at/after the chef's
// HH:MM cutoff. A malformed cutoff is treated as already passed (fail-open: deliver).
func cutoffPassed(nowIST time.Time, cutoff string, ist *time.Location) bool {
	t, err := time.ParseInLocation("15:04", cutoff, ist)
	if err != nil {
		return true
	}
	cutoffToday := time.Date(nowIST.Year(), nowIST.Month(), nowIST.Day(), t.Hour(), t.Minute(), 0, 0, ist)
	return !nowIST.Before(cutoffToday)
}

// claimMealSubGen dedups generation across instances — one (sub, date) per day.
func claimMealSubGen(ctx context.Context, subID uuid.UUID, date time.Time) bool {
	r := GetRedisClient()
	if !r.IsConnected() {
		return true // DB idempotency (per sub/date/slot) still guards duplicates
	}
	key := fmt.Sprintf("mealsub_gen:%s:%s", subID, date.Format("2006-01-02"))
	dedupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	acquired, err := r.SetNX(dedupCtx, key, "1", 25*time.Hour)
	if err != nil {
		return true
	}
	return acquired
}
