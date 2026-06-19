package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// admin_subscription_pricing.go — #44. A friendly, structured admin surface over
// the chef subscription pricing that's stored as flat PlatformSettings keys
// (`subscription.{CC}.chef.*`). The platform owner edits standard + premium
// prices and the premium commission here; GetPlanSettings reads the same keys, so
// GetAvailablePlans (and therefore the vendor premium page) reflect changes with
// no deploy.

type tierPricing struct {
	Monthly   float64 `json:"monthly"`
	Quarterly float64 `json:"quarterly"`
	Yearly    float64 `json:"yearly"`
}

type subscriptionPricingResponse struct {
	Country              string      `json:"country"`
	Currency             string      `json:"currency"`
	TrialDays            int         `json:"trialDays"`
	MinEarningsThreshold float64     `json:"minEarningsThreshold"`
	Standard             tierPricing `json:"standard"`
	Premium              tierPricing `json:"premium"`
	PremiumCommissionRate float64    `json:"premiumCommissionRate"`
}

func pricingCountry(c *gin.Context) string {
	cc := strings.ToUpper(strings.TrimSpace(c.Query("country")))
	if cc == "" {
		cc = "IN"
	}
	return cc
}

// GetSubscriptionPricing returns the effective chef subscription pricing for a
// country (query ?country=IN), resolved through the same GetPlanSettings the
// vendor plan endpoint uses — so the admin always sees what chefs would see.
func (h *AdminHandler) GetSubscriptionPricing(c *gin.Context) {
	cc := pricingCountry(c)
	cfg, err := services.GetPlanSettings(cc, models.SubscriberChef)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load pricing"})
		return
	}
	c.JSON(http.StatusOK, subscriptionPricingResponse{
		Country:              cc,
		Currency:             cfg.Currency,
		TrialDays:            cfg.TrialDays,
		MinEarningsThreshold: cfg.MinEarningsThreshold,
		Standard:             tierPricing{Monthly: cfg.MonthlyPrice, Quarterly: cfg.QuarterlyPrice, Yearly: cfg.YearlyPrice},
		Premium:              tierPricing{Monthly: cfg.PremiumMonthlyPrice, Quarterly: cfg.PremiumQuarterlyPrice, Yearly: cfg.PremiumYearlyPrice},
		PremiumCommissionRate: cfg.PremiumCommissionRate,
	})
}

// UpdateSubscriptionPricing upserts the provided pricing fields for a country.
// Only non-nil fields are written, so a partial PUT (e.g. just the premium
// monthly price) leaves everything else untouched.
func (h *AdminHandler) UpdateSubscriptionPricing(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Country  string `json:"country"`
		Currency *string `json:"currency"`
		Standard *struct {
			Monthly   *float64 `json:"monthly"`
			Quarterly *float64 `json:"quarterly"`
			Yearly    *float64 `json:"yearly"`
		} `json:"standard"`
		Premium *struct {
			Monthly   *float64 `json:"monthly"`
			Quarterly *float64 `json:"quarterly"`
			Yearly    *float64 `json:"yearly"`
		} `json:"premium"`
		PremiumCommissionRate *float64 `json:"premiumCommissionRate"`
		TrialDays             *int     `json:"trialDays"`
		MinEarningsThreshold  *float64 `json:"minEarningsThreshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	cc := strings.ToUpper(strings.TrimSpace(req.Country))
	if cc == "" {
		cc = "IN"
	}
	prefix := "subscription." + cc + ".chef."

	set := func(suffix, value string) {
		setPlatformSetting(prefix+suffix, value, userID)
	}
	money := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

	if req.Currency != nil {
		set("currency", strings.ToUpper(*req.Currency))
	}
	if req.Standard != nil {
		if req.Standard.Monthly != nil {
			set("monthly_price", money(*req.Standard.Monthly))
		}
		if req.Standard.Quarterly != nil {
			set("quarterly_price", money(*req.Standard.Quarterly))
		}
		if req.Standard.Yearly != nil {
			set("yearly_price", money(*req.Standard.Yearly))
		}
	}
	if req.Premium != nil {
		if req.Premium.Monthly != nil {
			set("premium_monthly_price", money(*req.Premium.Monthly))
		}
		if req.Premium.Quarterly != nil {
			set("premium_quarterly_price", money(*req.Premium.Quarterly))
		}
		if req.Premium.Yearly != nil {
			set("premium_yearly_price", money(*req.Premium.Yearly))
		}
	}
	if req.PremiumCommissionRate != nil {
		set("premium_commission_rate", money(*req.PremiumCommissionRate))
	}
	if req.TrialDays != nil {
		set("trial_days", strconv.Itoa(*req.TrialDays))
	}
	if req.MinEarningsThreshold != nil {
		set("min_earnings_threshold", money(*req.MinEarningsThreshold))
	}

	// Return the fresh effective pricing so the admin UI can confirm.
	cfg, _ := services.GetPlanSettings(cc, models.SubscriberChef)
	c.JSON(http.StatusOK, subscriptionPricingResponse{
		Country:              cc,
		Currency:             cfg.Currency,
		TrialDays:            cfg.TrialDays,
		MinEarningsThreshold: cfg.MinEarningsThreshold,
		Standard:             tierPricing{Monthly: cfg.MonthlyPrice, Quarterly: cfg.QuarterlyPrice, Yearly: cfg.YearlyPrice},
		Premium:              tierPricing{Monthly: cfg.PremiumMonthlyPrice, Quarterly: cfg.PremiumQuarterlyPrice, Yearly: cfg.PremiumYearlyPrice},
		PremiumCommissionRate: cfg.PremiumCommissionRate,
	})
}

// setPlatformSetting upserts a single PlatformSettings key (same semantics as the
// generic admin UpdateSettings, reused so the write path stays consistent).
func setPlatformSetting(key, value string, updatedBy uuid.UUID) {
	result := database.DB.Model(&models.PlatformSettings{}).
		Where("key = ?", key).
		Updates(map[string]interface{}{"value": value, "updated_by": updatedBy})
	if result.RowsAffected == 0 {
		database.DB.Create(&models.PlatformSettings{Key: key, Value: value, UpdatedBy: &updatedBy})
	}
}
