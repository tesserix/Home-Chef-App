package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
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
		ReferrerPoints  *float64 `json:"referrerPoints"`
		RefereePoints   *float64 `json:"refereePoints"`
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
	// Rewards are POINTS now, not rupees — see services.ReferralConfig.
	if req.ReferrerPoints != nil {
		setPlatformSetting("referral.referrer_points", money(*req.ReferrerPoints), userID)
	}
	if req.RefereePoints != nil {
		setPlatformSetting("referral.referee_points", money(*req.RefereePoints), userID)
	}
	if req.MonthlySpendCap != nil {
		setPlatformSetting("referral.monthly_spend_cap", money(*req.MonthlySpendCap), userID)
	}

	c.JSON(http.StatusOK, services.GetReferralConfig(database.DB))
}

// GetChefReferralProgramConfig returns the chef-refers-chef program config plus
// the chef loyalty (points → cashback) config, one payload for the admin card.
func (h *AdminHandler) GetChefReferralProgramConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"referral": services.GetChefReferralConfig(database.DB),
		"loyalty":  services.GetChefLoyaltyConfig(database.DB),
	})
}

// UpdateChefReferralProgramConfig upserts the provided fields (partial PUT).
// Rewards here are RUPEES (settlement cash), unlike the customer program's points.
func (h *AdminHandler) UpdateChefReferralProgramConfig(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Referral *struct {
			Enabled         *bool    `json:"enabled"`
			ReferrerAmount  *float64 `json:"referrerAmount"`
			RefereeAmount   *float64 `json:"refereeAmount"`
			MilestoneOrders *int     `json:"milestoneOrders"`
			MonthlySpendCap *float64 `json:"monthlySpendCap"`
		} `json:"referral"`
		Loyalty *struct {
			Enabled          *bool    `json:"enabled"`
			EarnRate         *float64 `json:"earnRate"`
			RedeemRate       *float64 `json:"redeemRate"`
			MinConvertPoints *float64 `json:"minConvertPoints"`
		} `json:"loyalty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	money := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	if r := req.Referral; r != nil {
		if r.Enabled != nil {
			setPlatformSetting("chef_referral.enabled", strconv.FormatBool(*r.Enabled), userID)
		}
		if r.ReferrerAmount != nil {
			setPlatformSetting("chef_referral.referrer_amount", money(*r.ReferrerAmount), userID)
		}
		if r.RefereeAmount != nil {
			setPlatformSetting("chef_referral.referee_amount", money(*r.RefereeAmount), userID)
		}
		if r.MilestoneOrders != nil && *r.MilestoneOrders > 0 {
			setPlatformSetting("chef_referral.milestone_orders", strconv.Itoa(*r.MilestoneOrders), userID)
		}
		if r.MonthlySpendCap != nil {
			setPlatformSetting("chef_referral.monthly_spend_cap", money(*r.MonthlySpendCap), userID)
		}
	}
	if l := req.Loyalty; l != nil {
		if l.Enabled != nil {
			setPlatformSetting("chef_loyalty.enabled", strconv.FormatBool(*l.Enabled), userID)
		}
		if l.EarnRate != nil {
			setPlatformSetting("chef_loyalty.earn_rate", money(*l.EarnRate), userID)
		}
		if l.RedeemRate != nil {
			setPlatformSetting("chef_loyalty.redeem_rate", money(*l.RedeemRate), userID)
		}
		if l.MinConvertPoints != nil && *l.MinConvertPoints > 0 {
			setPlatformSetting("chef_loyalty.min_convert_points", money(*l.MinConvertPoints), userID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"referral": services.GetChefReferralConfig(database.DB),
		"loyalty":  services.GetChefLoyaltyConfig(database.DB),
	})
}

// ListChefReferrals is the admin oversight list of kitchen referrals with the
// referrer/referee kitchen names, newest first.
func (h *AdminHandler) ListChefReferrals(c *gin.Context) {
	var refs []models.ChefReferral
	q := database.DB.Order("created_at DESC").Limit(200)
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	q.Find(&refs)

	// Batch-load the kitchen names for both sides.
	idSet := map[uuid.UUID]struct{}{}
	for _, r := range refs {
		idSet[r.ReferrerChefID] = struct{}{}
		idSet[r.RefereeChefID] = struct{}{}
	}
	ids := make([]uuid.UUID, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	names := map[uuid.UUID]string{}
	if len(ids) > 0 {
		type row struct {
			ID           uuid.UUID
			BusinessName string
		}
		var rows []row
		database.DB.Model(&models.ChefProfile{}).Select("id, business_name").Where("id IN ?", ids).Scan(&rows)
		for _, r := range rows {
			names[r.ID] = r.BusinessName
		}
	}

	type item struct {
		models.ChefReferral
		ReferrerKitchen string `json:"referrerKitchen"`
		RefereeKitchen  string `json:"refereeKitchen"`
	}
	out := make([]item, len(refs))
	for i, r := range refs {
		out[i] = item{ChefReferral: r, ReferrerKitchen: names[r.ReferrerChefID], RefereeKitchen: names[r.RefereeChefID]}
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "count": len(out)})
}
