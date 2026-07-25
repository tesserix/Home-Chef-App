package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestRedeemLoyalty_MonthlyCap(t *testing.T) {
	db := setupLoyaltyDB(t)
	u := uuid.New()
	cfg := GetLoyaltyConfig(db) // redeem_rate 0.05, monthly cap 300 → cap = 6000 pts of redemption/mo
	// Seed a big balance.
	_, _, err := applyLoyaltyTxnInTx(db, u, 20000, models.LoyaltyCredit, models.LoyaltySourceOrder, nil, "seed", "seed", nil, cfg)
	require.NoError(t, err)

	// Redeem 6000 pts = ₹300 (exactly the cap) — allowed.
	_, _, err = RedeemLoyalty(db, u, 6000)
	require.NoError(t, err)
	// Another 500 pts = ₹25 would exceed ₹300/mo — rejected.
	_, _, err = RedeemLoyalty(db, u, 500)
	require.ErrorIs(t, err, ErrLoyaltyMonthlyCap)
}
