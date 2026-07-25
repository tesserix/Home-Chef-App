package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestGetLoyaltyConfig_NewDefaults(t *testing.T) {
	db := setupLoyaltyDB(t)
	cfg := GetLoyaltyConfig(db)
	require.Equal(t, 0.1, cfg.PointsPerRupee)
	require.Equal(t, 0.05, cfg.RedeemRate)
	require.Equal(t, 500.0, cfg.MinRedeem)
	require.Equal(t, 0.10, cfg.MaxRedeemPct)
	require.Equal(t, 300.0, cfg.MonthlyRedeemCap)
	require.Equal(t, 365.0, cfg.ExpiryDays)
}

func TestGetLoyaltyConfig_ReadsNewKeys(t *testing.T) {
	db := setupLoyaltyDB(t)
	for k, v := range map[string]string{
		"loyalty.redeem_rate":        "0.1",
		"loyalty.max_redeem_pct":     "0.2",
		"loyalty.monthly_redeem_cap": "500",
		"loyalty.expiry_days":        "180",
	} {
		require.NoError(t, db.Create(&models.PlatformSettings{Key: k, Value: v}).Error)
	}
	cfg := GetLoyaltyConfig(db)
	require.Equal(t, 0.1, cfg.RedeemRate)
	require.Equal(t, 0.2, cfg.MaxRedeemPct)
	require.Equal(t, 500.0, cfg.MonthlyRedeemCap)
	require.Equal(t, 180.0, cfg.ExpiryDays)
}
