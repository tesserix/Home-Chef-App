package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAwardOrderLoyalty_EarnsOnSubtotal(t *testing.T) {
	db := setupLoyaltyDB(t)
	u, oid := uuid.New(), uuid.New()
	// subtotal 500, total 620 (fees/GST) — must earn on 500 → floor(500 * 0.1) = 50 pts.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, subtotal, total) VALUES (?,?,?,?)`,
		oid.String(), u.String(), 500.0, 620.0).Error)
	awarded, err := AwardOrderLoyalty(db, u, oid)
	require.NoError(t, err)
	require.Equal(t, 50.0, awarded)
}
