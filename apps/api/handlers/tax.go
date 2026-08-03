package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// TaxHandler owns per-country tax CRUD for admins + a public lookup the
// checkout UI uses to preview the tax line before the order is placed.
type TaxHandler struct{}

func NewTaxHandler() *TaxHandler {
	return &TaxHandler{}
}

// GetPublicTaxRate returns the active tax rule for a country (and optional
// region) without requiring auth. Used by the checkout page to render the
// tax line estimate before the order exists — the real tax is recomputed
// server-side at order creation, so a drifted preview never overcharges.
//
// GET /tax-rates/lookup?country=IN&region=KA
func (h *TaxHandler) GetPublicTaxRate(c *gin.Context) {
	country := strings.ToUpper(c.Query("country"))
	region := strings.ToUpper(c.Query("region"))
	if country == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country is required"})
		return
	}
	rule := services.ResolveTaxRate(country, region)
	rates := rule.ComponentRates(services.DeliveryByPlatform(models.FulfillmentDelivery, services.ThirdPartyDeliveryEnabled()))
	c.JSON(http.StatusOK, gin.H{
		"country":   country,
		"region":    region,
		"taxName":   rule.TaxName,
		"rate":      rule.Rate,
		"inclusive": rule.Inclusive,
		// The resolved per-supply rates — what each part of an order is actually
		// taxed at, with any component left unset taking `rate`.
		"rates": rates,
	})
}

// AdminListTaxRates returns every configured rule, grouped by country.
// GET /admin/tax-rates
func (h *TaxHandler) AdminListTaxRates(c *gin.Context) {
	var rates []models.TaxRate
	if err := database.DB.Order("country_code, region").Find(&rates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tax rates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rates})
}

// AdminUpsertTaxRate creates or updates a tax rule for a country/region
// pair. Region is optional — empty region means "country-wide default"
// and is used as the fallback when no matching region row exists.
//
// POST /admin/tax-rates
func (h *TaxHandler) AdminUpsertTaxRate(c *gin.Context) {
	var req struct {
		CountryCode string  `json:"countryCode" binding:"required"`
		Region      string  `json:"region"`
		TaxName     string  `json:"taxName" binding:"required"`
		Rate        float64 `json:"rate"`
		Inclusive   bool    `json:"inclusive"`
		Notes       string  `json:"notes"`
		IsActive    *bool   `json:"isActive"`
		// Per-supply overrides. Omit one and it is left as-is; send 0 and that
		// supply falls back to Rate. Setting ServicePercent is how the platform fee
		// moves off the restaurant rate, and DeliveryPlatformPercent how a
		// platform-arranged rider does — both without a deploy.
		FoodPercent             *float64 `json:"foodPercent"`
		ServicePercent          *float64 `json:"servicePercent"`
		ServiceInclusive        *bool    `json:"serviceInclusive"`
		DeliverySelfPercent     *float64 `json:"deliverySelfPercent"`
		DeliveryPlatformPercent *float64 `json:"deliveryPlatformPercent"`
		SubscriptionPercent     *float64 `json:"subscriptionPercent"`
		RegistrationIDLabel     *string  `json:"registrationIdLabel"`
		CompanyTaxID            *string  `json:"companyTaxId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Rate < 0 || req.Rate > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rate must be between 0 and 100"})
		return
	}
	for label, v := range map[string]*float64{
		"foodPercent": req.FoodPercent, "servicePercent": req.ServicePercent,
		"deliverySelfPercent": req.DeliverySelfPercent, "deliveryPlatformPercent": req.DeliveryPlatformPercent,
		"subscriptionPercent": req.SubscriptionPercent,
	} {
		if v != nil && (*v < 0 || *v > 100) {
			c.JSON(http.StatusBadRequest, gin.H{"error": label + " must be between 0 and 100"})
			return
		}
	}
	country := strings.ToUpper(req.CountryCode)
	region := strings.ToUpper(req.Region)

	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}

	var row models.TaxRate
	err := database.DB.
		Where("country_code = ? AND region = ?", country, region).
		First(&row).Error

	// Only the per-supply fields actually sent are touched, so an admin editing the
	// headline rate never silently resets an override someone else chose.
	applyOverrides := func(row *models.TaxRate) {
		if req.FoodPercent != nil {
			row.FoodPercent = *req.FoodPercent
		}
		if req.ServicePercent != nil {
			row.ServicePercent = *req.ServicePercent
		}
		if req.ServiceInclusive != nil {
			row.ServiceInclusive = *req.ServiceInclusive
		}
		if req.DeliverySelfPercent != nil {
			row.DeliverySelfPercent = *req.DeliverySelfPercent
		}
		if req.DeliveryPlatformPercent != nil {
			row.DeliveryPlatformPercent = *req.DeliveryPlatformPercent
		}
		if req.SubscriptionPercent != nil {
			row.SubscriptionPercent = *req.SubscriptionPercent
		}
		if req.RegistrationIDLabel != nil {
			row.RegistrationIDLabel = *req.RegistrationIDLabel
		}
		if req.CompanyTaxID != nil {
			row.CompanyTaxID = *req.CompanyTaxID
		}
	}

	if err != nil {
		row = models.TaxRate{
			CountryCode: country,
			Region:      region,
			TaxName:     req.TaxName,
			Rate:        req.Rate,
			Inclusive:   req.Inclusive,
			Notes:       req.Notes,
			IsActive:    active,
		}
		applyOverrides(&row)
		if err := database.DB.Create(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tax rate"})
			return
		}
	} else {
		row.TaxName = req.TaxName
		row.Rate = req.Rate
		row.Inclusive = req.Inclusive
		row.Notes = req.Notes
		row.IsActive = active
		applyOverrides(&row)
		if err := database.DB.Save(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tax rate"})
			return
		}
	}

	services.InvalidateTaxCache()
	c.JSON(http.StatusOK, row)
}

// AdminDeleteTaxRate removes a rule entirely. Use IsActive=false instead
// if you want to keep history; this endpoint is for rows that were seeded
// incorrectly or for countries you no longer serve.
//
// DELETE /admin/tax-rates/:id
func (h *TaxHandler) AdminDeleteTaxRate(c *gin.Context) {
	id := c.Param("id")
	rateID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tax rate ID"})
		return
	}
	if err := database.DB.Delete(&models.TaxRate{}, "id = ?", rateID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tax rate"})
		return
	}
	services.InvalidateTaxCache()
	c.JSON(http.StatusOK, gin.H{"deleted": rateID})
}
