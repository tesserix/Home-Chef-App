package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// createEarnBatch records a dated lot for a point CREDIT. Idempotent on idempotencyKey
// (a redelivered event or retried grant writes no second lot).
func createEarnBatch(tx *gorm.DB, userID uuid.UUID, points float64, source models.LoyaltyTxnSource, orderID *uuid.UUID, expiryDays float64, idempotencyKey string) error {
	if points <= 0 {
		return nil
	}
	var existing models.LoyaltyEarnBatch
	err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
	if err == nil {
		return nil // already recorded
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	now := time.Now()
	if expiryDays <= 0 {
		expiryDays = 365
	}
	return tx.Create(&models.LoyaltyEarnBatch{
		ID: uuid.New(), UserID: userID, Source: source, Points: points, PointsRemaining: points,
		EarnedAt: now, ExpiresAt: now.Add(time.Duration(expiryDays) * 24 * time.Hour),
		OrderID: orderID, IdempotencyKey: idempotencyKey,
	}).Error
}

// consumeBatchesFIFO decrements points_remaining from the soonest-expiring non-empty lots
// until `points` is consumed. Returns ErrInsufficientLoyaltyPoints if the lots can't cover it
// (the caller's balance check should already prevent this — this is defense in depth).
// Must run inside the same transaction that holds the loyalty account's row lock (see
// applyLoyaltyTxnInTx), so two concurrent redemptions for the same user can't both read
// and then clobber the same batch snapshot.
func consumeBatchesFIFO(tx *gorm.DB, userID uuid.UUID, points float64) error {
	if points <= 0 {
		return nil
	}
	var batches []models.LoyaltyEarnBatch
	query := tx.Where("user_id = ? AND points_remaining > 0", userID).Order("expires_at ASC")
	if tx.Dialector.Name() == "postgres" {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.Find(&batches).Error; err != nil {
		return err
	}
	remaining := points
	for i := range batches {
		if remaining <= 0 {
			break
		}
		take := batches[i].PointsRemaining
		if take > remaining {
			take = remaining
		}
		if err := tx.Model(&batches[i]).Update("points_remaining", batches[i].PointsRemaining-take).Error; err != nil {
			return err
		}
		remaining -= take
	}
	if remaining > 1e-6 {
		return ErrInsufficientLoyaltyPoints
	}
	return nil
}
