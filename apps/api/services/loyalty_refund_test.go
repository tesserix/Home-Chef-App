package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReverseOrderLoyalty_DebitsUnredeemedEarn(t *testing.T) {
	db := setupLoyaltyDB(t)
	u, oid := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, subtotal, total) VALUES (?,?,?,?)`,
		oid.String(), u.String(), 500.0, 620.0).Error)
	_, err := AwardOrderLoyalty(db, u, oid) // +50 pts, lot tagged with order_id
	require.NoError(t, err)
	acct, err := LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 50.0, acct.Balance)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return ReverseOrderLoyalty(tx, oid) }))
	acct, err = LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 0.0, acct.Balance)

	// Idempotent — a second reversal is a no-op.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return ReverseOrderLoyalty(tx, oid) }))
	acct, err = LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 0.0, acct.Balance)
}
