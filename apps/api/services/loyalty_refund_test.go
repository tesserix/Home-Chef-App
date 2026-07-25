package services

import (
	"testing"
	"time"

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

// TestReverseOrderLoyalty_MultiLotNoCrossContamination pins the fix for the cross-order lot
// leak: order A earns 50 (expiring SOONER) and order C earns 40 (expiring later), giving a
// balance of 90. Refunding C first, then A, must claw back exactly 90 total — 40 from C's own
// lot and 50 from A's own lot — never draining A's still-un-refunded lot when C is reversed.
// On the old FIFO-consuming code this fails (only 50 gets debited, leaking 40 points).
func TestReverseOrderLoyalty_MultiLotNoCrossContamination(t *testing.T) {
	db := setupLoyaltyDB(t)
	u := uuid.New()
	oidA, oidC := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, subtotal, total) VALUES (?,?,?,?)`,
		oidA.String(), u.String(), 500.0, 620.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, subtotal, total) VALUES (?,?,?,?)`,
		oidC.String(), u.String(), 400.0, 496.0).Error)

	_, err := AwardOrderLoyalty(db, u, oidA) // +50 pts, order A's lot
	require.NoError(t, err)
	_, err = AwardOrderLoyalty(db, u, oidC) // +40 pts, order C's lot
	require.NoError(t, err)

	// Backdate A's lot to expire SOONER than C's, so a naive FIFO reversal (soonest-expiring
	// first) would drain A's lot instead of C's when C is refunded.
	now := time.Now()
	require.NoError(t, db.Exec(`UPDATE loyalty_earn_batches SET expires_at = ? WHERE order_id = ?`,
		now.Add(240*time.Hour), oidA.String()).Error)
	require.NoError(t, db.Exec(`UPDATE loyalty_earn_batches SET expires_at = ? WHERE order_id = ?`,
		now.Add(720*time.Hour), oidC.String()).Error)

	acct, err := LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 90.0, acct.Balance)

	// Refund order timing is unrelated to expiry order: reverse C first, then A.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return ReverseOrderLoyalty(tx, oidC) }))
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return ReverseOrderLoyalty(tx, oidA) }))

	acct, err = LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 0.0, acct.Balance)
	require.Equal(t, 0.0, batchRemaining(t, db, u))
}

// TestReverseOrderLoyalty_OnlyUnredeemedPortion proves a partial redeem then a refund only
// claws back the leftover unredeemed points — the already-redeemed portion went to the wallet
// and is handled by the wallet refund, not the loyalty reversal.
func TestReverseOrderLoyalty_OnlyUnredeemedPortion(t *testing.T) {
	db := setupLoyaltyDB(t)
	setLoyaltySetting(t, db, "loyalty.min_redeem", "100")
	u, oid := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, subtotal, total) VALUES (?,?,?,?)`,
		oid.String(), u.String(), 5000.0, 6000.0).Error)

	_, err := AwardOrderLoyalty(db, u, oid) // +500 pts, order's single lot
	require.NoError(t, err)

	_, _, err = RedeemLoyalty(db, u, 300) // drains 300 from the order's lot -> 200 remaining
	require.NoError(t, err)

	acct, err := LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 200.0, acct.Balance)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return ReverseOrderLoyalty(tx, oid) }))

	acct, err = LoyaltyBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, 0.0, acct.Balance)
	require.Equal(t, 0.0, batchRemaining(t, db, u))
}
