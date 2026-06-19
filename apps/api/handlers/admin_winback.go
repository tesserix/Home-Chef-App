package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// admin_winback.go — #42. Admin surface over the win-back program: config stored
// as flat PlatformSettings `winback.*` keys (the cron + lifecycle hooks read the
// same keys via services.GetWinbackConfig, so changes take effect with no deploy)
// + reactivation analytics over the WinbackOffer table.

// GetWinbackConfig returns the effective win-back program config.
func (h *AdminHandler) GetWinbackConfig(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetWinbackConfig(database.DB))
}

// UpdateWinbackConfig upserts the provided fields (partial PUT).
func (h *AdminHandler) UpdateWinbackConfig(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Enabled            *bool    `json:"enabled"`
		DiscountPercent    *float64 `json:"discountPercent"`
		MaxDiscount        *float64 `json:"maxDiscount"`
		ValidityDays       *int     `json:"validityDays"`
		LapseThresholdDays *int     `json:"lapseThresholdDays"`
		CooldownDays       *int     `json:"cooldownDays"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.DiscountPercent != nil && (*req.DiscountPercent <= 0 || *req.DiscountPercent > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "discountPercent must be between 0 and 100"})
		return
	}

	if req.Enabled != nil {
		setPlatformSetting("winback.enabled", strconv.FormatBool(*req.Enabled), userID)
	}
	if req.DiscountPercent != nil {
		setPlatformSetting("winback.discount_percent", strconv.FormatFloat(*req.DiscountPercent, 'f', -1, 64), userID)
	}
	if req.MaxDiscount != nil {
		setPlatformSetting("winback.max_discount", strconv.FormatFloat(*req.MaxDiscount, 'f', -1, 64), userID)
	}
	if req.ValidityDays != nil {
		setPlatformSetting("winback.validity_days", strconv.Itoa(*req.ValidityDays), userID)
	}
	if req.LapseThresholdDays != nil {
		setPlatformSetting("winback.lapse_threshold_days", strconv.Itoa(*req.LapseThresholdDays), userID)
	}
	if req.CooldownDays != nil {
		setPlatformSetting("winback.cooldown_days", strconv.Itoa(*req.CooldownDays), userID)
	}
	c.JSON(http.StatusOK, services.GetWinbackConfig(database.DB))
}

// GetWinbackAnalytics returns reactivation analytics: offer counts by status, the
// reactivation rate (of resolved offers), and a per-trigger breakdown.
func (h *AdminHandler) GetWinbackAnalytics(c *gin.Context) {
	type statusRow struct {
		Status string
		Count  int64
	}
	var rows []statusRow
	database.DB.Model(&models.WinbackOffer{}).Select("status, COUNT(*) AS count").Group("status").Scan(&rows)

	var offered, reactivated, expired, total int64
	for _, r := range rows {
		total += r.Count
		switch r.Status {
		case models.WinbackStatusOffered:
			offered = r.Count
		case models.WinbackStatusReactivated:
			reactivated = r.Count
		case models.WinbackStatusExpired:
			expired = r.Count
		}
	}
	// Rate over RESOLVED offers (reactivated + expired) — pending ones haven't had
	// their chance yet, so they'd unfairly drag the rate down.
	resolved := reactivated + expired
	rate := 0.0
	if resolved > 0 {
		rate = models.RoundAmount(float64(reactivated) / float64(resolved) * 100)
	}

	type triggerRow struct {
		Trigger     string
		Total       int64
		Reactivated int64
	}
	var byTrigger []triggerRow
	database.DB.Model(&models.WinbackOffer{}).
		Select("trigger, COUNT(*) AS total, SUM(CASE WHEN status = 'reactivated' THEN 1 ELSE 0 END) AS reactivated").
		Group("trigger").Scan(&byTrigger)

	c.JSON(http.StatusOK, gin.H{
		"total":            total,
		"offered":          offered,
		"reactivated":      reactivated,
		"expired":          expired,
		"reactivationRate": rate,
		"byTrigger":        byTrigger,
	})
}
