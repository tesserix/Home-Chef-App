package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// admin_loyalty.go — #40. Admin surface over the loyalty program config stored as
// flat PlatformSettings `loyalty.*` keys, plus a read-only analytics overview.
// The customer endpoints + points engine read the same keys via GetLoyaltyConfig,
// so changes here take effect with no deploy — mirrors admin_referral.go.

// GetLoyaltyConfig returns the effective loyalty program config.
func (h *AdminHandler) GetLoyaltyConfig(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetLoyaltyConfig(database.DB))
}

// UpdateLoyaltyConfig upserts the provided fields (partial PUT supported).
func (h *AdminHandler) UpdateLoyaltyConfig(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Enabled         *bool    `json:"enabled"`
		PointsPerRupee  *float64 `json:"pointsPerRupee"`
		RedeemRate      *float64 `json:"redeemRate"`
		MinRedeem       *float64 `json:"minRedeem"`
		StreakThreshold *int     `json:"streakThreshold"`
		StreakBonus     *float64 `json:"streakBonus"`
		StreakGraceDays *int     `json:"streakGraceDays"`
		TierSilverAt    *float64 `json:"tierSilverAt"`
		TierGoldAt      *float64 `json:"tierGoldAt"`

		MaxRedeemPct     *float64 `json:"maxRedeemPct"`
		MonthlyRedeemCap *float64 `json:"monthlyRedeemCap"`
		ExpiryDays       *float64 `json:"expiryDays"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	num := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	if req.Enabled != nil {
		setPlatformSetting("loyalty.enabled", strconv.FormatBool(*req.Enabled), userID)
	}
	if req.PointsPerRupee != nil {
		setPlatformSetting("loyalty.points_per_rupee", num(*req.PointsPerRupee), userID)
	}
	if req.RedeemRate != nil {
		setPlatformSetting("loyalty.redeem_rate", num(*req.RedeemRate), userID)
	}
	if req.MinRedeem != nil {
		setPlatformSetting("loyalty.min_redeem", num(*req.MinRedeem), userID)
	}
	if req.StreakThreshold != nil {
		setPlatformSetting("loyalty.streak_threshold", strconv.Itoa(*req.StreakThreshold), userID)
	}
	if req.StreakBonus != nil {
		setPlatformSetting("loyalty.streak_bonus", num(*req.StreakBonus), userID)
	}
	if req.StreakGraceDays != nil {
		setPlatformSetting("loyalty.streak_grace_days", strconv.Itoa(*req.StreakGraceDays), userID)
	}
	if req.TierSilverAt != nil {
		setPlatformSetting("loyalty.tier_silver_at", num(*req.TierSilverAt), userID)
	}
	if req.TierGoldAt != nil {
		setPlatformSetting("loyalty.tier_gold_at", num(*req.TierGoldAt), userID)
	}
	if req.MaxRedeemPct != nil {
		setPlatformSetting("loyalty.max_redeem_pct", num(*req.MaxRedeemPct), userID)
	}
	if req.MonthlyRedeemCap != nil {
		setPlatformSetting("loyalty.monthly_redeem_cap", num(*req.MonthlyRedeemCap), userID)
	}
	if req.ExpiryDays != nil {
		setPlatformSetting("loyalty.expiry_days", num(*req.ExpiryDays), userID)
	}

	c.JSON(http.StatusOK, services.GetLoyaltyConfig(database.DB))
}

// GetLoyaltyAnalytics returns the read-only program overview.
func (h *AdminHandler) GetLoyaltyAnalytics(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetLoyaltyAnalytics(database.DB))
}

type grantLoyaltyRequest struct {
	UserID string  `json:"userId" binding:"required"`
	Points float64 `json:"points" binding:"required"`
	Reason string  `json:"reason"`
}

// GrantLoyaltyPoints manually credits or debits a customer's points balance
// (points<0 removes), stamped with the acting admin.
// POST /admin/loyalty/grant  { userId, points, reason }
func (h *AdminHandler) GrantLoyaltyPoints(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	var req grantLoyaltyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userId"})
		return
	}
	txn, err := services.AdminAdjustLoyalty(database.DB, uid, req.Points, req.Reason, adminID)
	if err != nil {
		if errors.Is(err, services.ErrInsufficientLoyaltyPoints) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User has fewer points than the deduction"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to adjust points"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"pointsBalance": txn.PointsAfter})
}
