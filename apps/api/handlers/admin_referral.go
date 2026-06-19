package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// admin_referral.go — #38. A friendly, structured admin surface over the referral
// program config stored as flat PlatformSettings `referral.*` keys. The customer
// endpoints + reward engine read the same keys via GetReferralConfig, so changes
// here take effect with no deploy.

// GetReferralConfig returns the effective referral program config.
func (h *AdminHandler) GetReferralConfig(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetReferralConfig(database.DB))
}

// UpdateReferralConfig upserts the provided fields (partial PUT supported).
func (h *AdminHandler) UpdateReferralConfig(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Enabled         *bool    `json:"enabled"`
		ReferrerReward  *float64 `json:"referrerReward"`
		RefereeReward   *float64 `json:"refereeReward"`
		MonthlySpendCap *float64 `json:"monthlySpendCap"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	money := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	if req.Enabled != nil {
		setPlatformSetting("referral.enabled", strconv.FormatBool(*req.Enabled), userID)
	}
	if req.ReferrerReward != nil {
		setPlatformSetting("referral.referrer_reward", money(*req.ReferrerReward), userID)
	}
	if req.RefereeReward != nil {
		setPlatformSetting("referral.referee_reward", money(*req.RefereeReward), userID)
	}
	if req.MonthlySpendCap != nil {
		setPlatformSetting("referral.monthly_spend_cap", money(*req.MonthlySpendCap), userID)
	}

	c.JSON(http.StatusOK, services.GetReferralConfig(database.DB))
}
