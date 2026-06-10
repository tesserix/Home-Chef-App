package services

// availability_cron.go — auto-resume for timed kitchen pauses.
//
// Chefs pause receiving for {15,30,60} min (paused_until set, accepting_orders
// flipped false). This cron wakes every minute and reopens any kitchen whose
// pause window has elapsed, then nudges the chef. Runs reliably because
// homechef-api holds min-scale:1 (a warm pod is always present); fire-on-start
// also catches up anything missed during a deploy.

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const availabilityResumeInterval = 1 * time.Minute

// StartAvailabilityResumeCron launches the auto-resume loop. Returns
// immediately; lives for the life of ctx.
func StartAvailabilityResumeCron(ctx context.Context) {
	go func() {
		runAvailabilityResumeScan(ctx)

		ticker := time.NewTicker(availabilityResumeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("availability-resume: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runAvailabilityResumeScan(ctx)
			}
		}
	}()
	log.Println("availability-resume: cron started (interval=1m)")
}

func runAvailabilityResumeScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("availability-resume: panic recovered: %v", r)
		}
	}()

	now := time.Now().UTC()

	// Grab the chefs about to be reopened (for the nudge) before clearing.
	var due []models.ChefProfile
	if err := database.DB.
		Where("paused_until IS NOT NULL AND paused_until <= ?", now).
		Find(&due).Error; err != nil {
		log.Printf("availability-resume: query failed: %v", err)
		return
	}
	if len(due) == 0 {
		return
	}

	ids := make([]uuid.UUID, 0, len(due))
	for _, ch := range due {
		ids = append(ids, ch.ID)
	}
	if err := database.DB.Model(&models.ChefProfile{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"accepting_orders": true,
			"paused_until":     nil,
		}).Error; err != nil {
		log.Printf("availability-resume: reopen failed: %v", err)
		return
	}

	for _, ch := range due {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = SendPushNotification(ch.UserID, "You're back online",
			"Your kitchen is accepting orders again.",
			map[string]string{"type": "availability_resumed", "deeplink": "homechef-vendor:///"})
	}
	log.Printf("availability-resume: reopened %d kitchen(s)", len(due))
}
