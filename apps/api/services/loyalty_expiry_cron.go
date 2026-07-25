package services

// loyalty_expiry_cron.go — daily sweep that expires past-due loyalty point
// lots (#40 points core, phase 1). Each due, non-empty lot is debited via the
// normal points ledger path (source=expiry) so the ledger, cached balance,
// and dated lots all move together in one transaction — the same invariant
// applyLoyaltyTxnInTx already guarantees for every other mutation.
//
// Lots are processed soonest-expiry-first and the sweep does NOT force a
// lot's points_remaining to zero after debiting it. applyLoyaltyTxnInTx's
// debit path already calls consumeBatchesFIFO, which drains the
// soonest-expiring non-empty lot for that user; because we work the due list
// in the same soonest-first order, the lot consumeBatchesFIFO drains is
// always exactly the lot we're expiring. Force-zeroing on top of that would
// double-remove points whenever a user has two or more lots due in the same
// sweep, desyncing the cached account balance from Σ(points_remaining).

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
// (source=expiry, idempotent per lot on "loyalty:expiry:<batch_id>"). Lots are
// processed soonest-expiry-first so the debit's internal FIFO consumption drains
// exactly the lot being expired, keeping account balance == Σ(points_remaining).
// Returns how many lots were actually expired.
func ExpireLoyaltyBatches(db *gorm.DB, now time.Time) (int, error) {
	var due []models.LoyaltyEarnBatch
	if err := db.Where("points_remaining > 0 AND expires_at <= ?", now).Find(&due).Error; err != nil {
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
			_, created, err := applyLoyaltyTxnInTx(tx, cur.UserID, cur.PointsRemaining,
				models.LoyaltyDebit, models.LoyaltyTxnSource("expiry"), nil, "Points expired",
				"loyalty:expiry:"+cur.ID.String(), nil, cfg)
			if err != nil {
				return err
			}
			didExpire = created
			// TEMP: brief's buggy force-zero, to prove the regression test catches it.
			return tx.Model(&models.LoyaltyEarnBatch{}).Where("id = ?", cur.ID).Update("points_remaining", 0).Error
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
