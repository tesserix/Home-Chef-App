package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chef_rewards.go — vendor-facing rewards surface: the chef's referral code +
// per-referral milestone progress, the loyalty points balance, and the
// points → cashback conversion. Cash lands via ChefBonus on the next weekly
// settlement (services/chef_referral.go, chef_loyalty.go).

type ChefRewardsHandler struct{}

func NewChefRewardsHandler() *ChefRewardsHandler { return &ChefRewardsHandler{} }

// chefFromContext loads the caller's chef profile (routes are RequireChef-gated,
// so a missing profile is a real error, not a role problem).
func chefFromContext(c *gin.Context) (*models.ChefProfile, bool) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return nil, false
	}
	return &chef, true
}

// GetChefRewards returns everything the vendor Rewards screen needs: referral
// code + link + progress, points balance + conversion terms, and totals.
func (h *ChefRewardsHandler) GetChefRewards(c *gin.Context) {
	chef, ok := chefFromContext(c)
	if !ok {
		return
	}

	code, err := services.GetOrCreateReferralCode(database.DB, chef.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load your referral code"})
		return
	}

	refCfg := services.GetChefReferralConfig(database.DB)
	loyCfg := services.GetChefLoyaltyConfig(database.DB)
	progress := services.GetChefReferralProgress(database.DB, chef.ID)

	var totalEarned float64
	for _, p := range progress {
		if p.Status == string(models.ChefReferralRewarded) {
			totalEarned += p.Reward
		}
	}

	var acct models.ChefLoyaltyAccount
	database.DB.Where("chef_id = ?", chef.ID).First(&acct)

	// Pending settlement credits, so the screen can show "on your next payout".
	var pendingBonus float64
	database.DB.Model(&models.ChefBonus{}).
		Where("chef_id = ? AND status = ?", chef.ID, models.ChefBonusPending).
		Select("COALESCE(SUM(amount), 0)").Scan(&pendingBonus)

	c.JSON(http.StatusOK, gin.H{
		"referral": gin.H{
			"enabled":         refCfg.Enabled,
			"code":            code,
			"link":            services.ChefReferralLink(code),
			"referrerAmount":  refCfg.ReferrerAmount,
			"refereeAmount":   refCfg.RefereeAmount,
			"milestoneOrders": refCfg.MilestoneOrders,
			"currency":        "INR",
			"totalEarned":     totalEarned,
			"referrals":       progress,
		},
		"loyalty": gin.H{
			"enabled":          loyCfg.Enabled,
			"points":           acct.Points,
			"lifetimePoints":   acct.LifetimePoints,
			"earnRate":         loyCfg.EarnRate,
			"redeemRate":       loyCfg.RedeemRate,
			"minConvertPoints": loyCfg.MinConvertPoints,
			"canConvert":       loyCfg.Enabled && acct.Points >= loyCfg.MinConvertPoints,
			"convertValue":     models.RoundAmount(acct.Points * loyCfg.RedeemRate),
		},
		"pendingPayoutCredit": pendingBonus,
	})
}

// GetChefRewardsHistory lists the chef's loyalty point movements, newest first.
func (h *ChefRewardsHandler) GetChefRewardsHistory(c *gin.Context) {
	chef, ok := chefFromContext(c)
	if !ok {
		return
	}
	var txns []models.ChefLoyaltyTxn
	database.DB.Where("chef_id = ?", chef.ID).Order("created_at DESC").Limit(100).Find(&txns)
	c.JSON(http.StatusOK, gin.H{"data": txns, "count": len(txns)})
}

// ConvertChefRewards converts the chef's points balance into a cashback credit
// on their next settlement, once the threshold is met.
func (h *ChefRewardsHandler) ConvertChefRewards(c *gin.Context) {
	chef, ok := chefFromContext(c)
	if !ok {
		return
	}
	bonus, points, err := services.ConvertChefLoyalty(database.DB, chef.ID, chef.UserID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrChefLoyaltyDisabled):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrChefLoyaltyBelowMinimum):
			cfg := services.GetChefLoyaltyConfig(database.DB)
			c.JSON(http.StatusConflict, gin.H{
				"error": fmt.Sprintf("You need at least %.0f points to convert.", cfg.MinConvertPoints),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not convert your points"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"convertedPoints": points,
		"cashback":        bonus.Amount,
		"currency":        bonus.Currency,
		"status":          string(bonus.Status),
		"message":         "Cashback will be added to your next weekly payout.",
	})
}
