package services

// referral_loyalty_test.go — the referral reward is paid in LOYALTY POINTS.
//
// It used to credit the wallet. Points are the better instrument for an
// acquisition reward: they expire, they are capped per order and per month at
// checkout, and they only convert into value when the customer comes back and
// spends — which is the behaviour a referral is meant to buy. Wallet credit does
// none of that; it is an unbounded, permanent liability the moment it is issued.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// pointsOf reads a user's live loyalty balance.
func pointsOf(t *testing.T, db *gorm.DB, userID uuid.UUID) float64 {
	t.Helper()
	acct, err := LoyaltyBalance(db, userID)
	require.NoError(t, err)
	return acct.Balance
}

// Both sides are paid, in points, on the referee's first paid order.
func TestMaybeGrantReward_PaysBothSidesInPoints(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)

	require.Equal(t, 1500.0, pointsOf(t, db, referrer), "referrer gets 1500 pts (₹75.00)")
	require.Equal(t, 1100.0, pointsOf(t, db, referee), "referee gets 1100 pts (₹55.00)")
}

// The points must land as dated EARN LOTS, not just an account balance — that is
// what makes them expire and what the FIFO spend path consumes.
func TestMaybeGrantReward_PointsLandAsExpiringLots(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	MaybeGrantReward(db, seedPaidOrder(t, db, referee))

	var lots []models.LoyaltyEarnBatch
	require.NoError(t, db.Where("user_id = ?", referrer).Find(&lots).Error)
	require.Len(t, lots, 1)
	require.Equal(t, 1500.0, lots[0].PointsRemaining)
	require.True(t, lots[0].ExpiresAt.After(lots[0].EarnedAt), "the lot must expire")
}

// A redelivered webhook must not pay twice.
func TestMaybeGrantReward_PointsAreIdempotent(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	orderID := seedPaidOrder(t, db, referee)

	MaybeGrantReward(db, orderID)
	MaybeGrantReward(db, orderID)

	require.Equal(t, 1500.0, pointsOf(t, db, referrer), "paid once, not twice")
	require.Equal(t, 1100.0, pointsOf(t, db, referee))
}

// The reward is recorded against the referral in RUPEES, so the monthly spend
// cap and the admin reporting keep working in the unit they are expressed in.
func TestMaybeGrantReward_RecordsRupeeValueOnTheReferral(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	refID := seedPendingReferral(t, db, referrer, referee, "")
	MaybeGrantReward(db, seedPaidOrder(t, db, referee))

	var ref models.Referral
	require.NoError(t, db.First(&ref, "id = ?", refID).Error)
	require.Equal(t, string(models.ReferralStateRewarded), string(ref.Status))
	require.Equal(t, 75.0, ref.ReferrerReward, "1500 pts x 0.05")
	require.Equal(t, 55.0, ref.RefereeReward, "1100 pts x 0.05")
}

// Points are an acquisition reward, so nothing is paid until the referee has
// actually bought something.
func TestMaybeGrantReward_PaysNothingWithoutAPaidOrder(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")

	// An order that never completed payment.
	unpaid := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, payment_status) VALUES (?, ?, 'pending')`,
		unpaid.String(), referee.String()).Error)
	MaybeGrantReward(db, unpaid)

	require.Equal(t, 0.0, pointsOf(t, db, referrer))
	require.Equal(t, 0.0, pointsOf(t, db, referee))
}

// The wallet is no longer touched by a referral — the reward moved rails
// entirely, rather than paying on both.
func TestMaybeGrantReward_DoesNotCreditTheWallet(t *testing.T) {
	db := setupRewardDB(t)
	referrer, referee := uuid.New(), uuid.New()
	seedPendingReferral(t, db, referrer, referee, "")
	MaybeGrantReward(db, seedPaidOrder(t, db, referee))

	var n int64
	require.NoError(t, db.Model(&models.WalletTxn{}).
		Where("user_id IN ?", []string{referrer.String(), referee.String()}).Count(&n).Error)
	require.Equal(t, int64(0), n, "referral rewards are points, not store credit")
}
