package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type DeliveryProviderHandler struct{}

func NewDeliveryProviderHandler() *DeliveryProviderHandler {
	return &DeliveryProviderHandler{}
}

// ListProviders returns a paginated list of delivery providers
func (h *DeliveryProviderHandler) ListProviders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	db := database.DB.Model(&models.DeliveryProvider{})

	// Filters
	if enabled := c.Query("enabled"); enabled != "" {
		db = db.Where("is_enabled = ?", enabled == "true")
	}
	if active := c.Query("active"); active != "" {
		db = db.Where("is_active = ?", active == "true")
	}
	if country := c.Query("country"); country != "" {
		db = db.Where("supported_countries @> ?", fmt.Sprintf(`["%s"]`, country))
	}
	if city := c.Query("city"); city != "" {
		db = db.Where("supported_cities @> ?", fmt.Sprintf(`["%s"]`, city))
	}

	var total int64
	db.Count(&total)

	var providers []models.DeliveryProvider
	if err := db.Order("priority ASC, name ASC").
		Offset(offset).Limit(limit).
		Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch providers"})
		return
	}

	response := make([]models.DeliveryProviderResponse, len(providers))
	for i, p := range providers {
		response[i] = p.ToResponse()
	}

	totalPages := int64(math.Ceil(float64(total) / float64(limit)))
	c.JSON(http.StatusOK, gin.H{
		"data": response,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// GetProvider returns a single provider by ID
func (h *DeliveryProviderHandler) GetProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider models.DeliveryProvider
	if err := database.DB.First(&provider, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Include active delivery count
	var activeDeliveries int64
	database.DB.Model(&models.Delivery{}).
		Where("provider_id = ? AND status NOT IN ?", id, []string{
			string(models.DeliveryDelivered),
			string(models.DeliveryFailed),
			string(models.DeliveryCancelled),
			string(models.DeliveryReturned),
		}).Count(&activeDeliveries)

	c.JSON(http.StatusOK, gin.H{
		"data": provider.ToResponse(),
		"stats": gin.H{
			"activeDeliveries": activeDeliveries,
		},
	})
}

// CreateProvider creates a new delivery provider
func (h *DeliveryProviderHandler) CreateProvider(c *gin.Context) {
	var req struct {
		Name               string  `json:"name" binding:"required"`
		Code               string  `json:"code" binding:"required"`
		Description        string  `json:"description"`
		LogoURL            string  `json:"logoUrl"`
		APIBaseURL         string  `json:"apiBaseUrl"`
		APIKey             string  `json:"apiKey"`
		APISecret          string  `json:"apiSecret"`
		WebhookSecret      string  `json:"webhookSecret"`
		StatusMapping      string  `json:"statusMapping"`
		SupportedCities    string  `json:"supportedCities"`
		SupportedCountries string  `json:"supportedCountries"`
		MaxDistance         *float64 `json:"maxDistance"`
		AvgPickupTime      *int     `json:"avgPickupTime"`
		PricingModel       string  `json:"pricingModel"`
		BaseCost           *float64 `json:"baseCost"`
		PerKmCost          *float64 `json:"perKmCost"`
		Currency           string  `json:"currency"`
		Priority           *int     `json:"priority"`
		MaxConcurrentDeliveries *int `json:"maxConcurrentDeliveries"`
		DailyLimit         *int     `json:"dailyLimit"`
		ContactName        string  `json:"contactName"`
		ContactEmail       string  `json:"contactEmail"`
		ContactPhone       string  `json:"contactPhone"`
		Notes              string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate code: lowercase, alphanumeric + underscore only
	code := strings.ToLower(strings.TrimSpace(req.Code))
	codeRegex := regexp.MustCompile(`^[a-z0-9_]+$`)
	if !codeRegex.MatchString(code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Code must be lowercase alphanumeric with underscores only"})
		return
	}

	// Check code uniqueness
	var existing models.DeliveryProvider
	if err := database.DB.Where("code = ?", code).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A provider with this code already exists"})
		return
	}

	// Validate statusMapping JSON if provided
	if req.StatusMapping != "" {
		var sm map[string]string
		if err := json.Unmarshal([]byte(req.StatusMapping), &sm); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "statusMapping must be a valid JSON object mapping provider statuses to Fe3dr statuses"})
			return
		}
	}

	// Validate supportedCities JSON if provided
	if req.SupportedCities != "" {
		var cities []string
		if err := json.Unmarshal([]byte(req.SupportedCities), &cities); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "supportedCities must be a valid JSON array of strings"})
			return
		}
	}

	// Validate supportedCountries JSON if provided
	if req.SupportedCountries != "" {
		var countries []string
		if err := json.Unmarshal([]byte(req.SupportedCountries), &countries); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "supportedCountries must be a valid JSON array of strings"})
			return
		}
	}

	provider := models.DeliveryProvider{
		Name:               strings.TrimSpace(req.Name),
		Code:               code,
		Description:        req.Description,
		LogoURL:            req.LogoURL,
		APIBaseURL:         req.APIBaseURL,
		APIKey:             req.APIKey,
		APISecret:          req.APISecret,
		WebhookSecret:      req.WebhookSecret,
		StatusMapping:      req.StatusMapping,
		SupportedCities:    req.SupportedCities,
		SupportedCountries: req.SupportedCountries,
		PricingModel:       req.PricingModel,
		Currency:           req.Currency,
		ContactName:        req.ContactName,
		ContactEmail:       req.ContactEmail,
		ContactPhone:       req.ContactPhone,
		Notes:              req.Notes,
		IsEnabled:          false, // must be explicitly enabled after testing
		IsActive:           true,
	}

	if req.MaxDistance != nil {
		provider.MaxDistance = *req.MaxDistance
	}
	if req.AvgPickupTime != nil {
		provider.AvgPickupTime = *req.AvgPickupTime
	}
	if req.BaseCost != nil {
		provider.BaseCost = *req.BaseCost
	}
	if req.PerKmCost != nil {
		provider.PerKmCost = *req.PerKmCost
	}
	if req.Priority != nil {
		provider.Priority = *req.Priority
	}
	if req.MaxConcurrentDeliveries != nil {
		provider.MaxConcurrentDeliveries = *req.MaxConcurrentDeliveries
	}
	if req.DailyLimit != nil {
		provider.DailyLimit = *req.DailyLimit
	}

	// Set defaults for empty JSON fields
	if provider.StatusMapping == "" {
		provider.StatusMapping = "{}"
	}
	if provider.SupportedCities == "" {
		provider.SupportedCities = "[]"
	}
	if provider.SupportedCountries == "" {
		provider.SupportedCountries = `["IN"]`
	}
	if provider.PricingModel == "" {
		provider.PricingModel = "per_delivery"
	}
	if provider.Currency == "" {
		provider.Currency = "INR"
	}

	if err := database.DB.Create(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create provider"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": provider.ToResponse()})
}

// UpdateProvider partially updates a delivery provider
func (h *DeliveryProviderHandler) UpdateProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider models.DeliveryProvider
	if err := database.DB.First(&provider, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	var req struct {
		Name               *string  `json:"name"`
		Description        *string  `json:"description"`
		LogoURL            *string  `json:"logoUrl"`
		APIBaseURL         *string  `json:"apiBaseUrl"`
		APIKey             *string  `json:"apiKey"`
		APISecret          *string  `json:"apiSecret"`
		WebhookSecret      *string  `json:"webhookSecret"`
		StatusMapping      *string  `json:"statusMapping"`
		SupportedCities    *string  `json:"supportedCities"`
		SupportedCountries *string  `json:"supportedCountries"`
		MaxDistance         *float64 `json:"maxDistance"`
		AvgPickupTime      *int     `json:"avgPickupTime"`
		PricingModel       *string  `json:"pricingModel"`
		BaseCost           *float64 `json:"baseCost"`
		PerKmCost          *float64 `json:"perKmCost"`
		Currency           *string  `json:"currency"`
		Priority           *int     `json:"priority"`
		IsEnabled          *bool    `json:"isEnabled"`
		IsActive           *bool    `json:"isActive"`
		MaxConcurrentDeliveries *int `json:"maxConcurrentDeliveries"`
		DailyLimit         *int     `json:"dailyLimit"`
		ContactName        *string  `json:"contactName"`
		ContactEmail       *string  `json:"contactEmail"`
		ContactPhone       *string  `json:"contactPhone"`
		Notes              *string  `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}

	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	}
	if req.APIBaseURL != nil {
		updates["api_base_url"] = *req.APIBaseURL
	}
	if req.APIKey != nil {
		updates["api_key"] = *req.APIKey
	}
	if req.APISecret != nil {
		updates["api_secret"] = *req.APISecret
	}
	if req.WebhookSecret != nil {
		updates["webhook_secret"] = *req.WebhookSecret
	}
	if req.StatusMapping != nil {
		var sm map[string]string
		if err := json.Unmarshal([]byte(*req.StatusMapping), &sm); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "statusMapping must be a valid JSON object"})
			return
		}
		updates["status_mapping"] = *req.StatusMapping
	}
	if req.SupportedCities != nil {
		var cities []string
		if err := json.Unmarshal([]byte(*req.SupportedCities), &cities); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "supportedCities must be a valid JSON array"})
			return
		}
		updates["supported_cities"] = *req.SupportedCities
	}
	if req.SupportedCountries != nil {
		var countries []string
		if err := json.Unmarshal([]byte(*req.SupportedCountries), &countries); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "supportedCountries must be a valid JSON array"})
			return
		}
		updates["supported_countries"] = *req.SupportedCountries
	}
	if req.MaxDistance != nil {
		updates["max_distance"] = *req.MaxDistance
	}
	if req.AvgPickupTime != nil {
		updates["avg_pickup_time"] = *req.AvgPickupTime
	}
	if req.PricingModel != nil {
		updates["pricing_model"] = *req.PricingModel
	}
	if req.BaseCost != nil {
		updates["base_cost"] = *req.BaseCost
	}
	if req.PerKmCost != nil {
		updates["per_km_cost"] = *req.PerKmCost
	}
	if req.Currency != nil {
		updates["currency"] = *req.Currency
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.IsEnabled != nil {
		updates["is_enabled"] = *req.IsEnabled
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.MaxConcurrentDeliveries != nil {
		updates["max_concurrent_deliveries"] = *req.MaxConcurrentDeliveries
	}
	if req.DailyLimit != nil {
		updates["daily_limit"] = *req.DailyLimit
	}
	if req.ContactName != nil {
		updates["contact_name"] = *req.ContactName
	}
	if req.ContactEmail != nil {
		updates["contact_email"] = *req.ContactEmail
	}
	if req.ContactPhone != nil {
		updates["contact_phone"] = *req.ContactPhone
	}
	if req.Notes != nil {
		updates["notes"] = *req.Notes
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	if err := database.DB.Model(&provider).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update provider"})
		return
	}

	// Reload
	database.DB.First(&provider, "id = ?", id)
	c.JSON(http.StatusOK, gin.H{"data": provider.ToResponse()})
}

// DeleteProvider soft-deletes a delivery provider
func (h *DeliveryProviderHandler) DeleteProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider models.DeliveryProvider
	if err := database.DB.First(&provider, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Check for active deliveries
	var activeCount int64
	database.DB.Model(&models.Delivery{}).
		Where("provider_id = ? AND status NOT IN ?", id, []string{
			string(models.DeliveryDelivered),
			string(models.DeliveryFailed),
			string(models.DeliveryCancelled),
			string(models.DeliveryReturned),
		}).Count(&activeCount)

	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":            "Cannot delete provider with active deliveries",
			"activeDeliveries": activeCount,
		})
		return
	}

	if err := database.DB.Delete(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider deleted successfully"})
}

// ToggleProvider toggles the isEnabled status of a provider
func (h *DeliveryProviderHandler) ToggleProvider(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider models.DeliveryProvider
	if err := database.DB.First(&provider, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	provider.IsEnabled = !provider.IsEnabled
	if err := database.DB.Save(&provider).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle provider"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": provider.ToResponse()})
}

// TestConnection tests connectivity to the provider's API
func (h *DeliveryProviderHandler) TestConnection(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider models.DeliveryProvider
	if err := database.DB.First(&provider, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	if provider.APIBaseURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider has no API base URL configured"})
		return
	}

	// Make a HEAD request to check reachability
	client := &http.Client{Timeout: 10 * time.Second}
	start := time.Now()
	resp, err := client.Head(provider.APIBaseURL)
	elapsed := time.Since(start)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success":      false,
			"error":        err.Error(),
			"responseTime": elapsed.Milliseconds(),
		})
		return
	}
	defer resp.Body.Close()

	c.JSON(http.StatusOK, gin.H{
		"success":      resp.StatusCode < 500,
		"statusCode":   resp.StatusCode,
		"responseTime": elapsed.Milliseconds(),
	})
}

// GetProviderStats returns detailed statistics for a provider
func (h *DeliveryProviderHandler) GetProviderStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid provider ID"})
		return
	}

	var provider models.DeliveryProvider
	if err := database.DB.First(&provider, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found"})
		return
	}

	// Overall counts
	var totalDeliveries int64
	var completedDeliveries int64
	var failedDeliveries int64
	var cancelledDeliveries int64
	var activeDeliveries int64

	database.DB.Model(&models.Delivery{}).Where("provider_id = ?", id).Count(&totalDeliveries)
	database.DB.Model(&models.Delivery{}).Where("provider_id = ? AND status = ?", id, models.DeliveryDelivered).Count(&completedDeliveries)
	database.DB.Model(&models.Delivery{}).Where("provider_id = ? AND status = ?", id, models.DeliveryFailed).Count(&failedDeliveries)
	database.DB.Model(&models.Delivery{}).Where("provider_id = ? AND status = ?", id, models.DeliveryCancelled).Count(&cancelledDeliveries)
	database.DB.Model(&models.Delivery{}).Where("provider_id = ? AND status NOT IN ?", id, []string{
		string(models.DeliveryDelivered),
		string(models.DeliveryFailed),
		string(models.DeliveryCancelled),
		string(models.DeliveryReturned),
	}).Count(&activeDeliveries)

	// Average cost
	var avgCost float64
	database.DB.Model(&models.Delivery{}).
		Where("provider_id = ? AND status = ?", id, models.DeliveryDelivered).
		Select("COALESCE(AVG(provider_cost), 0)").Scan(&avgCost)

	// Total cost
	var totalCost float64
	database.DB.Model(&models.Delivery{}).
		Where("provider_id = ?", id).
		Select("COALESCE(SUM(provider_cost), 0)").Scan(&totalCost)

	// Average delivery time (minutes)
	var avgDeliveryTime float64
	database.DB.Model(&models.Delivery{}).
		Where("provider_id = ? AND status = ? AND delivered_at IS NOT NULL AND assigned_at IS NOT NULL", id, models.DeliveryDelivered).
		Select("COALESCE(AVG(EXTRACT(EPOCH FROM (delivered_at - assigned_at)) / 60), 0)").Scan(&avgDeliveryTime)

	// Deliveries per day (last 30 days)
	type DailyStat struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var dailyStats []DailyStat
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	database.DB.Model(&models.Delivery{}).
		Where("provider_id = ? AND assigned_at >= ?", id, thirtyDaysAgo).
		Select("DATE(assigned_at) as date, COUNT(*) as count").
		Group("DATE(assigned_at)").
		Order("date ASC").
		Scan(&dailyStats)

	successRate := float64(0)
	if totalDeliveries > 0 {
		successRate = float64(completedDeliveries) / float64(totalDeliveries) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"provider": provider.ToResponse(),
		"stats": gin.H{
			"totalDeliveries":     totalDeliveries,
			"completedDeliveries": completedDeliveries,
			"failedDeliveries":    failedDeliveries,
			"cancelledDeliveries": cancelledDeliveries,
			"activeDeliveries":    activeDeliveries,
			"successRate":         math.Round(successRate*100) / 100,
			"avgCost":             math.Round(avgCost*100) / 100,
			"totalCost":           math.Round(totalCost*100) / 100,
			"avgDeliveryTime":     math.Round(avgDeliveryTime*100) / 100,
			"dailyStats":          dailyStats,
		},
	})
}

// HandleWebhook processes inbound webhooks from delivery providers
func (h *DeliveryProviderHandler) HandleWebhook(c *gin.Context) {
	providerCode := c.Param("provider")

	var provider models.DeliveryProvider
	if err := database.DB.Where("code = ? AND is_enabled = ? AND is_active = ?", providerCode, true, true).
		First(&provider).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Provider not found or not active"})
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	providerService := services.NewProviderService()
	if err := providerService.HandleProviderWebhook(provider.Code, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process webhook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
