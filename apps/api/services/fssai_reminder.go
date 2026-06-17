package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

	// Post-expiry: nudge chefs whose licence has actually lapsed (now locked),
	// once per day until they renew. Reuses IsChefFSSAIExpired so a verified
	// renewal correctly lifts the nudge.
	totalSent += runFSSAIExpiredNudge(ctx, today)

	if totalSent > 0 {
		log.Printf("fssai-reminder: scan complete, sent=%d", totalSent)
	}
}

// runFSSAIExpiredNudge pings chefs whose FSSAI licence has lapsed (and who are
// therefore locked out of new orders + payouts) with a daily "orders paused —
// renew" push, until they renew. Returns the number of pushes sent.
//
// Candidates are chefs with a verified FSSAI doc already past expiry; the
// per-chef IsChefFSSAIExpired check then narrows to the genuinely-locked (a
// later verified renewal lifts the lock), so the cron and the order/payout
// enforcement never disagree.
func runFSSAIExpiredNudge(ctx context.Context, today time.Time) int {
	cutoff := time.Now().AddDate(0, 0, -1)

	var chefIDs []uuid.UUID
	if err := database.DB.Model(&models.ChefDocument{}).
		Distinct("chef_id").
		Where("type = ? AND status = ? AND expiry_date IS NOT NULL AND expiry_date < ?",
			models.DocFSSAILicense, models.DocStatusVerified, cutoff).
		Pluck("chef_id", &chefIDs).Error; err != nil {
		log.Printf("fssai-reminder: expired-nudge query failed: %v", err)
		return 0
	}

	sent := 0
	for _, chefID := range chefIDs {
		var chef models.ChefProfile
		if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
			continue
		}
		if !IsChefFSSAIExpired(&chef) {
			continue // renewed / not actually locked
		}
		if !shouldSendExpiredNudge(ctx, chefID, today) {
			continue
		}
		if err := sendFSSAIExpiredPush(chef.UserID); err != nil {
			log.Printf("fssai-reminder: expired push failed for chef=%s: %v", chefID, err)
			continue
		}
		sent++
	}
	if sent > 0 {
		log.Printf("fssai-reminder: expired-nudge sent=%d", sent)
	}
	return sent
}

// shouldSendExpiredNudge gates one expired-nudge per chef per day through Redis
// SETNX (fails open if Redis is down — better a duplicate nudge than none).
func shouldSendExpiredNudge(ctx context.Context, chefID uuid.UUID, today time.Time) bool {
	r := GetRedisClient()
	if !r.IsConnected() {
		return true
	}
	key := fmt.Sprintf("fssai_expired_nudge:%s:%s", chefID, today.Format("2006-01-02"))
	dedupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	acquired, err := r.SetNX(dedupCtx, key, "1", 26*time.Hour)
	if err != nil {
		return true
	}
	return acquired
}

func sendFSSAIExpiredPush(userID uuid.UUID) error {
	title := "FSSAI licence expired — orders paused"
	body := "Your food-safety (FSSAI) licence has expired, so new orders are paused. Renew now to go back online."
	data := map[string]string{
		"type":     "fssai_expired",
		"deeplink": "homechef-vendor:///documents/renew",
	}
	return SendPushNotification(userID, title, body, data)
}

// SendFSSAIBackOnlinePush tells a chef their FSSAI renewal was verified and the
// kitchen is live again (#92). Exported so the document-verification handler can
// fire it the moment a renewal lifts the lockout (event-driven, no cron lag).
func SendFSSAIBackOnlinePush(userID uuid.UUID) error {
	title := "You're back online"
	body := "Your FSSAI licence renewal is verified — your kitchen is live and accepting orders again."
	data := map[string]string{"type": "fssai_back_online"}
	return SendPushNotification(userID, title, body, data)
}

// SendFSSAIConfirmLicencePush asks a chef to confirm the expiry date of an FSSAI
// licence we already have on file but with no recorded expiry (a legacy upload).
// Backs the one-time expiry backfill (#93): until the date is captured the chef
// can't be expiry-locked, so this closes a compliance blind-spot. Exported so the
// admin-triggered backfill endpoint can fire it.
func SendFSSAIConfirmLicencePush(userID uuid.UUID) error {
	title := "Confirm your FSSAI licence"
	body := "Please re-enter your FSSAI licence expiry date so we can keep your kitchen compliant and online."
	data := map[string]string{
		"type":     "fssai_confirm_expiry",
		"deeplink": "homechef-vendor:///documents/renew",
	}
	return SendPushNotification(userID, title, body, data)
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
		"type":     "fssai_expiring",
		"daysOut":  fmt.Sprintf("%d", daysOut),
		"deeplink": "homechef-vendor:///documents/renew",
	}
	if expiryDate != nil {
		data["expiryDate"] = expiryDate.Format("2006-01-02")
	}

	return SendPushNotification(chef.UserID, title, body, data)
}

// fssaiLockedGauge tracks how many chefs are currently FSSAI-locked, refreshed
// by StartFSSAILockedGaugeUpdater. Defined here (services) rather than in
// middleware/metrics.go to avoid an import cycle — middleware imports services.
var fssaiLockedGauge = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "homechef_fssai_locked_chefs",
	Help: "Number of chefs currently locked out by an expired FSSAI licence (#94)",
})

const fssaiGaugeInterval = 10 * time.Minute

// StartFSSAILockedGaugeUpdater keeps homechef_fssai_locked_chefs current by
// recounting locked chefs every fssaiGaugeInterval (and once on startup). Cheap:
// it only re-checks chefs that already have an expired verified FSSAI doc, then
// narrows with IsChefFSSAIExpired so a verified renewal doesn't inflate the
// count. Pass main.go's root context so SIGTERM stops it cleanly.
func StartFSSAILockedGaugeUpdater(ctx context.Context) {
	go func() {
		updateFSSAILockedGauge()
		ticker := time.NewTicker(fssaiGaugeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("fssai-gauge: shutting down on ctx cancel")
				return
			case <-ticker.C:
				updateFSSAILockedGauge()
			}
		}
	}()
	log.Println("fssai-gauge: locked-chefs updater started (interval=10m)")
}

func updateFSSAILockedGauge() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("fssai-gauge: panic recovered: %v", r)
		}
	}()

	cutoff := time.Now().AddDate(0, 0, -1)
	var chefIDs []uuid.UUID
	if err := database.DB.Model(&models.ChefDocument{}).
		Distinct("chef_id").
		Where("type = ? AND status = ? AND expiry_date IS NOT NULL AND expiry_date < ?",
			models.DocFSSAILicense, models.DocStatusVerified, cutoff).
		Pluck("chef_id", &chefIDs).Error; err != nil {
		log.Printf("fssai-gauge: query failed: %v", err)
		return
	}

	locked := 0
	for _, chefID := range chefIDs {
		var chef models.ChefProfile
		if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
			continue
		}
		if IsChefFSSAIExpired(&chef) {
			locked++
		}
	}
	fssaiLockedGauge.Set(float64(locked))
}
