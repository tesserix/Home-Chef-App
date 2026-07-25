package services

// refund_funding_split_test.go — dividing a refund across the rails that funded
// the order.

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The owner's worked example: wallet 400 + loyalty 100 + card 500, refund 50%.
func TestSplitRefundByFunding_ProRata(t *testing.T) {
	o := &models.Order{Total: 1000, WalletApplied: 400, LoyaltyApplied: 100}
	s := SplitRefundByFunding(o, ToPaise(500))
	require.Equal(t, ToPaise(200), s.WalletPaise)
	require.Equal(t, ToPaise(50), s.LoyaltyPaise)
	require.Equal(t, ToPaise(250), s.CardPaise)
}

// The parts must reconstitute the refund exactly — no paise created or destroyed
// by three independent roundings.
func TestSplitRefundByFunding_CardAbsorbsRounding(t *testing.T) {
	o := &models.Order{Total: 300.03, WalletApplied: 100.01, LoyaltyApplied: 100.01}
	r := ToPaise(100.01)
	s := SplitRefundByFunding(o, r)
	require.Equal(t, r, s.WalletPaise+s.LoyaltyPaise+s.CardPaise)
}

// Two successive partials must never return more than a rail ever funded.
func TestSplitRefundByFunding_RepeatedPartialsCannotOverReturn(t *testing.T) {
	o := &models.Order{Total: 1000, WalletApplied: 400, LoyaltyApplied: 100,
		WalletRefunded: 360, LoyaltyRefunded: 90}
	s := SplitRefundByFunding(o, ToPaise(500))
	require.Equal(t, ToPaise(40), s.WalletPaise, "only 40.00 of wallet remains")
	require.Equal(t, ToPaise(10), s.LoyaltyPaise)
	require.Equal(t, ToPaise(450), s.CardPaise, "the rest goes back to the card")
}

func TestSplitRefundByFunding_NoCreditOrderIsAllCard(t *testing.T) {
	o := &models.Order{Total: 1000}
	s := SplitRefundByFunding(o, ToPaise(1000))
	require.Equal(t, 0, s.WalletPaise)
	require.Equal(t, 0, s.LoyaltyPaise)
	require.Equal(t, ToPaise(1000), s.CardPaise)
}

// A fully credit-funded order returns nothing to the card.
func TestSplitRefundByFunding_FullyCreditFundedReturnsNoCash(t *testing.T) {
	o := &models.Order{Total: 1000, WalletApplied: 600, LoyaltyApplied: 400}
	s := SplitRefundByFunding(o, ToPaise(1000))
	require.Equal(t, ToPaise(600), s.WalletPaise)
	require.Equal(t, ToPaise(400), s.LoyaltyPaise)
	require.Equal(t, 0, s.CardPaise)
}

func TestSplitRefundByFunding_ZeroRefundIsEmpty(t *testing.T) {
	o := &models.Order{Total: 1000, WalletApplied: 400}
	require.Equal(t, FundingSplit{}, SplitRefundByFunding(o, 0))
}

// Whatever the inputs, the three parts sum to exactly the refund and no rail is
// ever returned more than it funded.
func TestSplitRefundByFunding_InvariantsHoldUnderFuzz(t *testing.T) {
	r := rand.New(rand.NewSource(725))
	for i := 0; i < 20000; i++ {
		total := float64(r.Intn(500000)) / 100
		wallet := float64(r.Intn(int(total*100)+1)) / 100
		loyalty := float64(r.Intn(int((total-wallet)*100)+1)) / 100
		wRef := float64(r.Intn(int(wallet*100)+1)) / 100
		lRef := float64(r.Intn(int(loyalty*100)+1)) / 100
		o := &models.Order{Total: total, WalletApplied: wallet, LoyaltyApplied: loyalty,
			WalletRefunded: wRef, LoyaltyRefunded: lRef}

		refund := r.Intn(ToPaise(total) + 1)
		s := SplitRefundByFunding(o, refund)

		require.Equal(t, refund, s.WalletPaise+s.LoyaltyPaise+s.CardPaise,
			"the parts must sum to exactly the refund")
		require.LessOrEqual(t, s.WalletPaise, ToPaise(wallet)-ToPaise(wRef))
		require.LessOrEqual(t, s.LoyaltyPaise, ToPaise(loyalty)-ToPaise(lRef))
		require.GreaterOrEqual(t, s.WalletPaise, 0)
		require.GreaterOrEqual(t, s.LoyaltyPaise, 0)
		require.GreaterOrEqual(t, s.CardPaise, 0)
	}
}
