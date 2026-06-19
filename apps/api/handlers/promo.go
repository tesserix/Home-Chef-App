package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type PromoHandler struct{}

func NewPromoHandler() *PromoHandler {
	return &PromoHandler{}
}

// ---------- Customer endpoint ----------

// ValidatePromoCode validates a promo code for a customer.
// POST /promo/validate
func (h *PromoHandler) ValidatePromoCode(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Code       string  `json:"code" binding:"required"`
		OrderTotal float64 `json:"orderTotal" binding:"required"`
		// ChefID scopes the preview to the cart's chef so chef-funded codes are
		// validated accurately. Optional — omitted (e.g. an empty cart) skips the
		// chef check; CreateOrder always enforces it with the real chef.
		ChefID string `json:"chefId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chefID := uuid.Nil
	if req.ChefID != "" {
		if parsed, perr := uuid.Parse(req.ChefID); perr == nil {
			chefID = parsed
		}
	}

	discount, promo, err := validateAndCalculateDiscount(req.Code, userID, chefID, req.OrderTotal)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":         true,
		"code":          promo.Code,
		"discountType":  promo.DiscountType,
		"discountValue": promo.DiscountValue,
		"discount":      discount,
		"description":   promo.Description,
		"fundingSource": promo.FundingSource,
	})
}

// ---------- Admin endpoints ----------

// AdminListPromos lists all promo codes.
// GET /admin/promos
func (h *PromoHandler) AdminListPromos(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.PromoCode{})

	if search != "" {
		query = query.Where("code ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var promos []models.PromoCode
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&promos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch promo codes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": promos,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// AdminCreatePromo creates a new promo code.
// POST /admin/promos
func (h *PromoHandler) AdminCreatePromo(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Code           string     `json:"code" binding:"required"`
		Description    string     `json:"description"`
		DiscountType   string     `json:"discountType" binding:"required"`
		DiscountValue  float64    `json:"discountValue" binding:"required,gt=0"`
		MinOrderAmount float64    `json:"minOrderAmount"`
		MaxDiscount    float64    `json:"maxDiscount"`
		UsageLimit     int        `json:"usageLimit"`
		PerUserLimit   int        `json:"perUserLimit"`
		ValidFrom      time.Time  `json:"validFrom" binding:"required"`
		ValidUntil     *time.Time `json:"validUntil"`
		ApplicableTo   string     `json:"applicableTo"`
		FundingSource  string     `json:"fundingSource"`
		ChefID         string     `json:"chefId"`
		BudgetCap      float64    `json:"budgetCap"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.DiscountType != "percentage" && req.DiscountType != "fixed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "discountType must be 'percentage' or 'fixed'"})
		return
	}

	if req.DiscountType == "percentage" && (req.DiscountValue <= 0 || req.DiscountValue > 100) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Percentage discount must be between 0 and 100"})
		return
	}

	// Funding source (#39): platform (default) or chef. Chef-funded requires a
	// valid chef so the discount can be billed to that kitchen at settlement.
	fundingSource := models.PromoFundingPlatform
	var chefID *uuid.UUID
	if req.FundingSource != "" {
		if req.FundingSource != models.PromoFundingPlatform && req.FundingSource != models.PromoFundingChef {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fundingSource must be 'platform' or 'chef'"})
			return
		}
		fundingSource = req.FundingSource
	}
	if fundingSource == models.PromoFundingChef {
		parsed, perr := uuid.Parse(req.ChefID)
		if perr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "A valid chefId is required for chef-funded promos"})
			return
		}
		var chef models.ChefProfile
		if err := database.DB.Select("id").First(&chef, "id = ?", parsed).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chef not found"})
			return
		}
		chefID = &parsed
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))

	// Check for duplicate code
	var existing models.PromoCode
	if err := database.DB.Where("code = ?", code).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Promo code already exists"})
		return
	}

	applicableTo := "all"
	if req.ApplicableTo != "" {
		validApplicable := map[string]bool{"all": true, "new_users": true, "returning_users": true}
		if !validApplicable[req.ApplicableTo] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid applicableTo value"})
			return
		}
		applicableTo = req.ApplicableTo
	}

	promo := models.PromoCode{
		Code:           code,
		Description:    req.Description,
		DiscountType:   req.DiscountType,
		DiscountValue:  req.DiscountValue,
		MinOrderAmount: req.MinOrderAmount,
		MaxDiscount:    req.MaxDiscount,
		UsageLimit:     req.UsageLimit,
		PerUserLimit:   req.PerUserLimit,
		ValidFrom:      req.ValidFrom,
		ValidUntil:     req.ValidUntil,
		IsActive:       true,
		ApplicableTo:   applicableTo,
		FundingSource:  fundingSource,
		ChefID:         chefID,
		BudgetCap:      req.BudgetCap,
		CreatedByID:    userID,
	}

	if err := database.DB.Create(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create promo code"})
		return
	}

	c.JSON(http.StatusCreated, promo)
}

// AdminUpdatePromo updates an existing promo code.
// PUT /admin/promos/:id
func (h *PromoHandler) AdminUpdatePromo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid promo ID"})
		return
	}

	var promo models.PromoCode
	if err := database.DB.First(&promo, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
		return
	}

	var req struct {
		Description    *string    `json:"description"`
		DiscountType   *string    `json:"discountType"`
		DiscountValue  *float64   `json:"discountValue"`
		MinOrderAmount *float64   `json:"minOrderAmount"`
		MaxDiscount    *float64   `json:"maxDiscount"`
		UsageLimit     *int       `json:"usageLimit"`
		PerUserLimit   *int       `json:"perUserLimit"`
		ValidFrom      *time.Time `json:"validFrom"`
		ValidUntil     *time.Time `json:"validUntil"`
		IsActive       *bool      `json:"isActive"`
		ApplicableTo   *string    `json:"applicableTo"`
		BudgetCap      *float64   `json:"budgetCap"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.DiscountType != nil {
		if *req.DiscountType != "percentage" && *req.DiscountType != "fixed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "discountType must be 'percentage' or 'fixed'"})
			return
		}
		updates["discount_type"] = *req.DiscountType
	}
	if req.DiscountValue != nil {
		updates["discount_value"] = *req.DiscountValue
	}
	if req.MinOrderAmount != nil {
		updates["min_order_amount"] = *req.MinOrderAmount
	}
	if req.MaxDiscount != nil {
		updates["max_discount"] = *req.MaxDiscount
	}
	if req.UsageLimit != nil {
		updates["usage_limit"] = *req.UsageLimit
	}
	if req.PerUserLimit != nil {
		updates["per_user_limit"] = *req.PerUserLimit
	}
	if req.ValidFrom != nil {
		updates["valid_from"] = *req.ValidFrom
	}
	if req.ValidUntil != nil {
		updates["valid_until"] = *req.ValidUntil
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.ApplicableTo != nil {
		updates["applicable_to"] = *req.ApplicableTo
	}
	if req.BudgetCap != nil {
		updates["budget_cap"] = *req.BudgetCap
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&promo).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update promo code"})
			return
		}
	}

	database.DB.First(&promo, "id = ?", id)
	c.JSON(http.StatusOK, promo)
}

// AdminDeletePromo deactivates a promo code.
// DELETE /admin/promos/:id
func (h *PromoHandler) AdminDeletePromo(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid promo ID"})
		return
	}

	result := database.DB.Model(&models.PromoCode{}).Where("id = ?", id).Update("is_active", false)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Promo code deactivated"})
}

// AdminGetPromoUsage returns usage history for a promo code.
// GET /admin/promos/:id/usage
func (h *PromoHandler) AdminGetPromoUsage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid promo ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.PromoCodeUsage{}).Where("promo_code_id = ?", id).Count(&total)

	var usages []models.PromoCodeUsage
	if err := database.DB.Preload("User").
		Where("promo_code_id = ?", id).
		Order("used_at DESC").
		Offset(offset).Limit(limit).
		Find(&usages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch usage history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": usages,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// AdminGetPromoAnalytics returns redemption analytics for a promo code (#39):
// total redemptions, total discount given, unique redeemers, and budget/usage
// utilisation. Powers the admin redemption-analytics view.
// GET /admin/promos/:id/analytics
func (h *PromoHandler) AdminGetPromoAnalytics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid promo ID"})
		return
	}

	var promo models.PromoCode
	if err := database.DB.First(&promo, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promo code not found"})
		return
	}

	var stats struct {
		Redemptions   int64
		TotalDiscount float64
		UniqueUsers   int64
	}
	database.DB.Model(&models.PromoCodeUsage{}).Where("promo_code_id = ?", id).
		Select("COUNT(*) AS redemptions, COALESCE(SUM(discount),0) AS total_discount, COUNT(DISTINCT user_id) AS unique_users").
		Scan(&stats)

	budgetRemaining := 0.0
	budgetUtilisation := 0.0
	if promo.BudgetCap > 0 {
		budgetRemaining = models.RoundAmount(promo.BudgetCap - promo.BudgetSpent)
		if budgetRemaining < 0 {
			budgetRemaining = 0
		}
		budgetUtilisation = models.RoundAmount(promo.BudgetSpent / promo.BudgetCap * 100)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":              promo.Code,
		"fundingSource":     promo.FundingSource,
		"redemptions":       stats.Redemptions,
		"totalDiscount":     models.RoundAmount(stats.TotalDiscount),
		"uniqueUsers":       stats.UniqueUsers,
		"usageLimit":        promo.UsageLimit,
		"usageCount":        promo.UsageCount,
		"budgetCap":         promo.BudgetCap,
		"budgetSpent":       models.RoundAmount(promo.BudgetSpent),
		"budgetRemaining":   budgetRemaining,
		"budgetUtilisation": budgetUtilisation,
	})
}

// ---------- Shared validation logic ----------

// validateAndCalculateDiscount validates a promo code and returns the calculated
// discount. The math + eligibility rules live in services.ComputePromoDiscount /
// CheckPromoEligibility (shared with any other surface, unit-tested); this
// wrapper only resolves the per-customer/order state from the DB. chefID is the
// order's chef (uuid.Nil in preview without a cart chef). Returns
// (discount, promoCode, errorBody) — errorBody is nil, a message string, or a
// {message, minOrderAmount} map for the min-order case.
func validateAndCalculateDiscount(code string, userID, chefID uuid.UUID, orderSubtotal float64) (float64, *models.PromoCode, interface{}) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, nil, "Promo code is required"
	}

	var promo models.PromoCode
	if err := database.DB.Where("code = ? AND is_active = ?", code, true).First(&promo).Error; err != nil {
		return 0, nil, "Invalid promo code"
	}

	// Resolve per-customer state the rules need.
	var userUsageCount int64
	if promo.PerUserLimit > 0 {
		database.DB.Model(&models.PromoCodeUsage{}).
			Where("promo_code_id = ? AND user_id = ?", promo.ID, userID).
			Count(&userUsageCount)
	}
	var userOrderCount int64
	if promo.ApplicableTo != "all" {
		database.DB.Model(&models.Order{}).Where("customer_id = ?", userID).Count(&userOrderCount)
	}

	discount := services.ComputePromoDiscount(&promo, orderSubtotal)
	ctx := services.PromoContext{
		Now:            time.Now(),
		UserOrderCount: userOrderCount,
		UserUsageCount: userUsageCount,
		OrderSubtotal:  orderSubtotal,
		ChefID:         chefID,
	}
	if err := services.CheckPromoEligibility(&promo, ctx, discount); err != nil {
		var minErr services.ErrPromoMinOrder
		if errors.As(err, &minErr) {
			return 0, nil, map[string]interface{}{
				"message":        minErr.Error(),
				"minOrderAmount": minErr.MinOrderAmount,
			}
		}
		return 0, nil, err.Error()
	}

	return discount, &promo, nil
}
