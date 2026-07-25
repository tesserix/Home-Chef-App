package services

// loyalty_expiry_cron.go — daily sweep that expires past-due loyalty point
// lots (#40 points core, phase 1). Each due, non-empty lot is debited via
// applyLoyaltyLedgerTxn (ledger-only, no FIFO) and the sweep zeroes that
// SPECIFIC lot itself — the same targeted approach ReverseOrderLoyalty uses
// for refund clawbacks. Because the sweep never routes through the
// account-wide FIFO consumer, one lot's debit can never drain a different
// lot, so a transient failure mid-sweep can't strand or double-remove points
// from an unrelated lot.

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

const loyaltyExpiryInterval = 24 * time.Hour

// ExpireLoyaltyBatches debits each past-due non-empty lot's remaining points
// (source=expiry, idempotent per lot on "loyalty:expiry:<batch_id>") via
// applyLoyaltyLedgerTxn and zeroes that specific lot directly — no FIFO, so
// one lot's expiry can never touch another lot. Returns how many lots were
// actually expired.
func ExpireLoyaltyBatches(db *gorm.DB, now time.Time) (int, error) {
	var due []models.LoyaltyEarnBatch
	if err := db.Where("points_remaining > 0 AND expires_at <= ?", now).
		Order("expires_at ASC").Find(&due).Error; err != nil {
		return 0, err
	}
	cfg := GetLoyaltyConfig(db)
	expired := 0
	for i := range due {
		b := due[i]
		var didExpire bool
		err := db.Transaction(func(tx *gorm.DB) error {
			var cur models.LoyaltyEarnBatch
			if err := tx.First(&cur, "id = ?", b.ID).Error; err != nil {
				return err
			}
			if cur.PointsRemaining <= 0 {
				return nil // already drained (idempotent re-run)
			}
			_, created, err := applyLoyaltyLedgerTxn(tx, cur.UserID, cur.PointsRemaining,
				models.LoyaltyDebit, models.LoyaltySourceExpiry, nil, "Points expired",
				"loyalty:expiry:"+cur.ID.String(), nil, cfg)
			if err != nil {
				return err
			}
			if !created {
				return nil // already expired in a prior run
			}
			didExpire = true
			// Zero exactly THIS lot (targeted, not FIFO): balance and this lot both drop by
			// cur.PointsRemaining and no other lot is touched, so a transient failure on one
			// lot can never cause another lot's sweep to drain it.
			return tx.Model(&models.LoyaltyEarnBatch{}).Where("id = ?", cur.ID).
				Update("points_remaining", 0).Error
		})
		if err != nil {
			log.Printf("loyalty-expiry: lot %s failed: %v", b.ID, err)
			continue
		}
		if didExpire {
			expired++
		}
	}
	return expired, nil
}

func runLoyaltyExpiryScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("loyalty-expiry: panic recovered: %v", r)
		}
	}()
	if _, err := ExpireLoyaltyBatches(database.DB, time.Now()); err != nil {
		log.Printf("loyalty-expiry: scan failed: %v", err)
	}
}

// StartLoyaltyExpiryCron launches the daily expiry loop. Returns immediately.
func StartLoyaltyExpiryCron(ctx context.Context) {
	go func() {
		runLoyaltyExpiryScan(ctx)
		ticker := time.NewTicker(loyaltyExpiryInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runLoyaltyExpiryScan(ctx)
			}
		}
	}()
	log.Println("loyalty-expiry: cron started (interval=24h)")
}
