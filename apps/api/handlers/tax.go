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
	c.JSON(http.StatusOK, gin.H{
		"country":   country,
		"region":    region,
		"taxName":   rule.TaxName,
		"rate":      rule.Rate,
		"inclusive": rule.Inclusive,
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Rate < 0 || req.Rate > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rate must be between 0 and 100"})
		return
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
