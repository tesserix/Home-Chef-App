package services

// winback_cron.go — the daily lapse scan that drives the customer-lapse win-back
// trigger (#42/#275). Reconciles open offers, then issues a win-back to freshly
// lapsed customers. Mirrors the fssai-reminder cron: in-process ticker fallback +
// Temporal-schedule activity, panic-safe, Redis SETNX idempotency.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const winbackScanInterval = 24 * time.Hour

// winbackScanLimit drip-feeds offers: at most N new offers per daily scan so a
// first run can't blast the entire lapsed backlog at once.
const winbackScanLimit = 200

// StartWinbackCron runs the daily scan as an in-process ticker (the Temporal
// schedule path calls runWinbackScan directly).
func StartWinbackCron(ctx context.Context) {
	go func() {
		runWinbackScan(ctx)
		ticker := time.NewTicker(winbackScanInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("winback-scan: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runWinbackScan(ctx)
			}
		}
	}()
	log.Println("winback-scan: cron started (interval=24h)")
}

// runWinbackScan reconciles open offers, then offers a win-back to freshly-lapsed
// customers. Never takes down the pod (panic-safe). Idempotent across instances via
// a per-user Redis SETNX guard layered on the DB cooldown dedup in OfferWinback.
func runWinbackScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("winback-scan: panic recovered: %v", r)
		}
	}()

	cfg := GetWinbackConfig(database.DB)

	// Always reconcile so reactivation/expiry analytics stay fresh even when new
	// offers are paused.
	react, exp := ReconcileWinbackOffers(database.DB)
	if react > 0 || exp > 0 {
		log.Printf("winback-scan: reconciled %d reactivated, %d expired", react, exp)
	}
	if !cfg.Enabled {
		return
	}

	lapsed := FindLapsedCustomers(database.DB, cfg.LapseThresholdDays, cfg.CooldownDays, winbackScanLimit)
	offered := 0
	for _, uid := range lapsed {
		if !claimWinbackScan(ctx, uid, cfg.CooldownDays) {
			continue
		}
		o, err := OfferWinback(database.DB, uid, models.WinbackAudienceCustomer, models.WinbackTriggerLapsed, nil)
		if err != nil {
			log.Printf("winback-scan: offer failed for user=%s: %v", uid, err)
			continue
		}
		if o != nil {
			offered++
		}
	}
	if offered > 0 {
		log.Printf("winback-scan: offered win-back to %d lapsed customers", offered)
	}
}

// claimWinbackScan stops concurrent scans (multi-instance) double-offering the same
// user — a per-user Redis SETNX with a cooldown TTL. No Redis → allow (OfferWinback's
// DB cooldown dedup still applies).
func claimWinbackScan(ctx context.Context, userID uuid.UUID, cooldownDays int) bool {
	r := GetRedisClient()
	if !r.IsConnected() {
		return true
	}
	ttl := time.Duration(cooldownDays) * 24 * time.Hour
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	dedupCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	acquired, err := r.SetNX(dedupCtx, fmt.Sprintf("winback_offer:%s", userID), "1", ttl)
	if err != nil {
		return true
	}
	return acquired
}
