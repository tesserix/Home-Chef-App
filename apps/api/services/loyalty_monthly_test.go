package services

// loyalty_monthly_test.go — the rolling window that makes MonthlyRedeemCap bind.
//
// The cap is measured over the POINTS ledger, not over wallet credits. Two reasons:
// a checkout redemption writes no wallet row at all (so a wallet-based counter
// misses it entirely, letting a customer draw the full cap through each route),
// and refunding a loyalty slice back as wallet credit would otherwise inflate the
// counter as if the customer had redeemed a second time.
//
// The cross-route test that proves both routes share one pool lives in
// loyalty_order_redeem_test.go, next to the checkout redemption it exercises.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestMonthlyRedeemedPaise_CountsBothRoutesInsideWindow(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	now := time.Now()

	insert := func(src models.LoyaltyTxnSource, typ models.LoyaltyTxnType, pts float64, at time.Time) {
		require.NoError(t, db.Create(&models.LoyaltyTransaction{
			ID: uuid.New(), UserID: uid, Type: typ, Source: src,
			Points: pts, IdempotencyKey: uuid.New().String(), CreatedAt: at,
		}).Error)
	}
	insert(models.LoyaltySourceRedeem, models.LoyaltyDebit, 1000, now.Add(-2*24*time.Hour))           // ₹50
	insert(models.LoyaltySourceOrderRedemption, models.LoyaltyDebit, 2000, now.Add(-10*24*time.Hour)) // ₹100
	insert(models.LoyaltySourceOrderRedemption, models.LoyaltyDebit, 4000, now.Add(-40*24*time.Hour)) // outside the window
	insert(models.LoyaltySourceOrder, models.LoyaltyCredit, 5000, now)                                // an EARN, not a redemption

	got, err := MonthlyRedeemedPaise(db, uid)
	require.NoError(t, err)
	require.Equal(t, 15000, got, "50.00 + 100.00 — window and source filtered")
}

func TestMonthlyRedeemedPaise_NoHistoryIsZero(t *testing.T) {
	db := setupLoyaltyDB(t)
	got, err := MonthlyRedeemedPaise(db, uuid.New())
	require.NoError(t, err)
	require.Equal(t, 0, got)
}

// A refund credited back to the wallet must NOT read as a fresh redemption. A
// wallet-based counter would double-count a redeem-then-refund cycle.
func TestMonthlyRedeemedPaise_WalletRefundDoesNotInflateTheCount(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	require.NoError(t, db.Create(&models.LoyaltyTransaction{
		ID: uuid.New(), UserID: uid, Type: models.LoyaltyDebit,
		Source: models.LoyaltySourceOrderRedemption, Points: 2000,
		IdempotencyKey: uuid.New().String(), CreatedAt: time.Now(),
	}).Error)

	// The order is refunded and the loyalty slice returns as wallet credit.
	_, err := CreditWallet(db, uid, 100, models.WalletSourceLoyalty, nil,
		"refund — loyalty portion", "refund-loyalty:x", nil)
	require.NoError(t, err)

	got, err := MonthlyRedeemedPaise(db, uid)
	require.NoError(t, err)
	require.Equal(t, 10000, got, "still ₹100 — the refund credit is not a redemption")
}

// RedeemLoyalty must reject a conversion that would breach the cap, measured on
// the shared ledger counter rather than on wallet credits.
func TestRedeemLoyalty_RejectsBreachOfMonthlyCap(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 20000, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.LoyaltyTransaction{
		ID: uuid.New(), UserID: uid, Type: models.LoyaltyDebit,
		Source: models.LoyaltySourceOrderRedemption, Points: 5800, // ₹290 of the ₹300 cap
		IdempotencyKey: uuid.New().String(), CreatedAt: time.Now(),
	}).Error)

	_, _, err = RedeemLoyalty(db, uid, 1000) // ₹50 — would breach
	require.ErrorIs(t, err, ErrLoyaltyMonthlyCap)
}

func TestRedeemLoyalty_AllowsRedemptionInsideRemainingCap(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 20000, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)

	_, wt, err := RedeemLoyalty(db, uid, 1000) // ₹50, well inside the ₹300 cap
	require.NoError(t, err)
	require.Equal(t, 50.0, wt.Amount)
}
