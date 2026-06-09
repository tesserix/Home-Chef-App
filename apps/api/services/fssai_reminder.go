package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// reminderWindowsDays is the set of days-before-expiry at which we ping
// the chef. Mirrors the Wave 2 spec — 30/15/7 day countdown, balancing
// "you have time" with "your doc actually expires tomorrow."
var reminderWindowsDays = []int{30, 15, 7}

// fssaiReminderInterval controls how often the cron wakes. 24h matches
// "daily" — actual wall-clock time-of-day depends on when the pod
// started, which is acceptable: doc expiry is a multi-day window so
// firing at any time of day is fine.
const fssaiReminderInterval = 24 * time.Hour

// StartFSSAIReminderCron launches a background goroutine that scans
// every fssaiReminderInterval for documents expiring at the configured
// windows + pushes the chef. Returns immediately; the cron lives for
// the life of the context. Pass main.go's root context so SIGTERM
// shuts it down cleanly.
//
// Idempotency: each (document_id, window_days, day) tuple is gated
// through Redis SETNX so two pods running the cron concurrently won't
// double-push. The key TTL is 26h — comfortably longer than the
// interval so we never race the rollover window.
func StartFSSAIReminderCron(ctx context.Context) {
	go func() {
		// Fire once immediately so a fresh deploy doesn't have to wait
		// 24h for the first scan. Subsequent fires are interval-spaced.
		runFSSAIReminderScan(ctx)

		ticker := time.NewTicker(fssaiReminderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("fssai-reminder: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runFSSAIReminderScan(ctx)
			}
		}
	}()
	log.Println("fssai-reminder: cron started (interval=24h, windows=30/15/7 days)")
}

func runFSSAIReminderScan(ctx context.Context) {
	defer func() {
		// Cron must NEVER take down the pod — recover from any panic
		// (nil pointer in a model, transient DB error, etc.) and log.
		// Better to skip a day's reminders than crash a serving pod.
		if r := recover(); r != nil {
			log.Printf("fssai-reminder: panic recovered: %v", r)
		}
	}()

	today := time.Now().UTC().Truncate(24 * time.Hour)
	totalSent := 0

	for _, daysOut := range reminderWindowsDays {
		// expiry_date == today + daysOut, matched on the date portion.
		// Postgres comparisons against a midnight-UTC bound work
		// because expiry_date is stored as a timestamp; we widen by 24h
		// to catch any tz drift in legacy rows.
		windowStart := today.AddDate(0, 0, daysOut)
		windowEnd := windowStart.AddDate(0, 0, 1)

		var docs []models.ChefDocument
		err := database.DB.Where(
			"type = ? AND expiry_date >= ? AND expiry_date < ?",
			models.DocFSSAILicense, windowStart, windowEnd,
		).Find(&docs).Error
		if err != nil {
			log.Printf("fssai-reminder: query failed for window=%dd: %v", daysOut, err)
			continue
		}

		for _, doc := range docs {
			if !shouldSendReminder(ctx, doc.ID, daysOut, today) {
				continue
			}
			if err := sendFSSAIReminderPush(doc.ChefID, doc.ExpiryDate, daysOut); err != nil {
				log.Printf("fssai-reminder: push failed for chef=%s daysOut=%d: %v",
					doc.ChefID, daysOut, err)
				continue
			}
			totalSent++
		}
	}

	if totalSent > 0 {
		log.Printf("fssai-reminder: scan complete, sent=%d", totalSent)
	}
}

// shouldSendReminder gates each (doc, window, day) tuple through Redis
// SETNX so two pods running the cron simultaneously don't both push.
// First caller wins; subsequent callers (or this pod on restart later
// today) see the existing key and skip. 26h TTL means a missed day
// stays missed rather than firing twice during clock skew.
//
// If Redis is unavailable we fall OPEN — better to risk a duplicate
// reminder than to miss it entirely. Matches Wave 1 Redis fail-mode.
func shouldSendReminder(ctx context.Context, docID uuid.UUID, daysOut int, today time.Time) bool {
	r := GetRedisClient()
	if !r.IsConnected() {
		return true
	}
	key := fmt.Sprintf("fssai_reminder:%s:%d:%s", docID, daysOut, today.Format("2006-01-02"))
	dedupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	acquired, err := r.SetNX(dedupCtx, key, "1", 26*time.Hour)
	if err != nil {
		return true
	}
	return acquired
}

func sendFSSAIReminderPush(chefID uuid.UUID, expiryDate *time.Time, daysOut int) error {
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		return fmt.Errorf("chef lookup: %w", err)
	}

	title := "FSSAI license expiring soon"
	var body string
	if daysOut <= 7 {
		body = fmt.Sprintf("Your FSSAI license expires in %d days. Renew now to keep accepting orders.", daysOut)
	} else {
		body = fmt.Sprintf("Your FSSAI license expires in %d days. Tap to renew.", daysOut)
	}

	data := map[string]string{
		"type":       "fssai_expiring",
		"daysOut":    fmt.Sprintf("%d", daysOut),
		"deeplink":   "homechef-vendor:///documents/renew",
	}
	if expiryDate != nil {
		data["expiryDate"] = expiryDate.Format("2006-01-02")
	}

	return SendPushNotification(chef.UserID, title, body, data)
}
