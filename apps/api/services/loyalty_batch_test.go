package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func setupBatchDB(t *testing.T) *gorm.DB {
	return setupLoyaltyDB(t)
}

func batchRemaining(t *testing.T, db *gorm.DB, u uuid.UUID) float64 {
	var sum float64
	db.Raw(`SELECT COALESCE(SUM(points_remaining),0) FROM loyalty_earn_batches WHERE user_id = ?`, u.String()).Scan(&sum)
	return sum
}

func TestCreateEarnBatch_Idempotent(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	require.NoError(t, createEarnBatch(db, u, 100, models.LoyaltySourceOrder, nil, 365, "k1"))
	require.NoError(t, createEarnBatch(db, u, 100, models.LoyaltySourceOrder, nil, 365, "k1")) // dup no-op
	require.Equal(t, 100.0, batchRemaining(t, db, u))
}

func TestConsumeBatchesFIFO_SoonestExpiryFirst(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	// Two lots: older-expiring 30 pts, later-expiring 100 pts.
	old := models.LoyaltyEarnBatch{ID: uuid.New(), UserID: u, Source: models.LoyaltySourceOrder, Points: 30, PointsRemaining: 30,
		EarnedAt: time.Now().Add(-48 * time.Hour), ExpiresAt: time.Now().Add(24 * time.Hour), IdempotencyKey: "b-old"}
	newer := models.LoyaltyEarnBatch{ID: uuid.New(), UserID: u, Source: models.LoyaltySourceOrder, Points: 100, PointsRemaining: 100,
		EarnedAt: time.Now(), ExpiresAt: time.Now().Add(240 * time.Hour), IdempotencyKey: "b-new"}
	require.NoError(t, db.Create(&old).Error)
	require.NoError(t, db.Create(&newer).Error)

	require.NoError(t, consumeBatchesFIFO(db, u, 50)) // drains the 30-lot, then 20 from the 100-lot
	require.Equal(t, 80.0, batchRemaining(t, db, u))

	var oldRow models.LoyaltyEarnBatch
	db.First(&oldRow, "idempotency_key = ?", "b-old")
	require.Equal(t, 0.0, oldRow.PointsRemaining)
}

func TestConsumeBatchesFIFO_Insufficient(t *testing.T) {
	db := setupBatchDB(t)
	u := uuid.New()
	require.NoError(t, createEarnBatch(db, u, 10, models.LoyaltySourceOrder, nil, 365, "k"))
	require.ErrorIs(t, consumeBatchesFIFO(db, u, 50), ErrInsufficientLoyaltyPoints)
}
