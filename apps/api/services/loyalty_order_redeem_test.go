package services

// loyalty_order_redeem_test.go — spending points directly on an order.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestRedeemLoyaltyToOrder_DebitsFIFOAndIsIdempotent(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid, oid := uuid.New(), uuid.New()
	_, err := EarnLoyalty(db, uid, 2000, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)

	require.NoError(t, RedeemLoyaltyToOrder(db, uid, oid, 500))
	acct, err := LoyaltyBalance(db, uid)
	require.NoError(t, err)
	require.Equal(t, float64(1500), acct.Balance)

	// A retried settle must not debit a second time.
	require.NoError(t, RedeemLoyaltyToOrder(db, uid, oid, 500))
	acct, err = LoyaltyBalance(db, uid)
	require.NoError(t, err)
	require.Equal(t, float64(1500), acct.Balance, "idempotent per order")
}

// The lots must actually be drawn down, not just the account balance — otherwise
// the expiry sweep would later find points that have already been spent.
func TestRedeemLoyaltyToOrder_DrawsDownTheEarnLots(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 2000, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)

	require.NoError(t, RedeemLoyaltyToOrder(db, uid, uuid.New(), 800))

	var remaining float64
	require.NoError(t, db.Model(&models.LoyaltyEarnBatch{}).
		Where("user_id = ?", uid).Select("COALESCE(SUM(points_remaining),0)").
		Scan(&remaining).Error)
	require.Equal(t, float64(1200), remaining)
}

func TestRedeemLoyaltyToOrder_RejectsOverdraw(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 100, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)
	require.Error(t, RedeemLoyaltyToOrder(db, uid, uuid.New(), 500))
}

func TestRedeemLoyaltyToOrder_ZeroIsNoOp(t *testing.T) {
	db := setupLoyaltyDB(t)
	require.NoError(t, RedeemLoyaltyToOrder(db, uuid.New(), uuid.New(), 0))
}

// THE HOLE the shared counter closes: redeem-to-wallet and checkout redemption
// must draw on ONE monthly pool. Counting only wallet credits let a customer take
// the full cap through each route and spend double the configured limit.
func TestRedeemLoyaltyToOrder_ConsumesTheSameMonthlyCapAsRedeemToWallet(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 20000, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)

	// Spend ₹290 of the ₹300 monthly cap at CHECKOUT (5800 pts x ₹0.05).
	require.NoError(t, RedeemLoyaltyToOrder(db, uid, uuid.New(), 5800))

	// Redeem-to-wallet may now only take the remaining ₹10, so a 1000-point
	// (₹50) conversion must be refused.
	_, _, err = RedeemLoyalty(db, uid, 1000)
	require.ErrorIs(t, err, ErrLoyaltyMonthlyCap,
		"checkout redemptions must consume the same monthly pool")
}
