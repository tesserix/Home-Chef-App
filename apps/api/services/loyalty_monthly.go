package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// loyalty_monthly.go — the rolling redemption window that makes
// LoyaltyConfig.MonthlyRedeemCap bind.

// loyaltyMonthlyWindow is the rolling period the cap is measured over.
const loyaltyMonthlyWindow = 30 * 24 * time.Hour

// MonthlyRedeemedPaise sums the rupee value of the points a customer has redeemed
// in the last 30 days across BOTH routes — checkout redemptions and
// redeem-to-wallet conversions.
//
// One cap, one pool. Counting the two routes separately would let a customer take
// the full cap through each and draw double the configured limit.
//
// It reads the POINTS ledger rather than wallet credits deliberately. A checkout
// redemption writes no wallet row at all, so a wallet-based counter would miss it
// entirely; and refunding a loyalty slice back as wallet credit would inflate a
// wallet-based counter as though the customer had redeemed a second time.
func MonthlyRedeemedPaise(db *gorm.DB, userID uuid.UUID) (int, error) {
	cfg := GetLoyaltyConfig(db)
	var points float64
	err := db.Model(&models.LoyaltyTransaction{}).
		Where("user_id = ? AND type = ? AND source IN ? AND created_at >= ?",
			userID, models.LoyaltyDebit,
			[]models.LoyaltyTxnSource{models.LoyaltySourceRedeem, models.LoyaltySourceOrderRedemption},
			time.Now().Add(-loyaltyMonthlyWindow)).
		Select("COALESCE(SUM(points), 0)").Scan(&points).Error
	if err != nil {
		return 0, err
	}
	return pointsToPaise(points, cfg.RedeemRate), nil
}
