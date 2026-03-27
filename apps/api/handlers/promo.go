package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
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
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	discount, promo, err := validateAndCalculateDiscount(req.Code, userID, req.OrderTotal)
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

// ---------- Shared validation logic ----------

// validateAndCalculateDiscount validates a promo code and returns the calculated discount.
// Returns (discount, promoCode, errorMessage).
func validateAndCalculateDiscount(code string, userID uuid.UUID, orderTotal float64) (float64, *models.PromoCode, interface{}) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, nil, "Promo code is required"
	}

	var promo models.PromoCode
	if err := database.DB.Where("code = ? AND is_active = ?", code, true).First(&promo).Error; err != nil {
		return 0, nil, "Invalid promo code"
	}

	now := time.Now()

	// Check validity period
	if now.Before(promo.ValidFrom) {
		return 0, nil, "Promo code is not yet active"
	}
	if promo.ValidUntil != nil && now.After(*promo.ValidUntil) {
		return 0, nil, "Promo code has expired"
	}

	// Check global usage limit
	if promo.UsageLimit > 0 && promo.UsageCount >= promo.UsageLimit {
		return 0, nil, "Promo code usage limit reached"
	}

	// Check per-user limit
	if promo.PerUserLimit > 0 {
		var userUsageCount int64
		database.DB.Model(&models.PromoCodeUsage{}).
			Where("promo_code_id = ? AND user_id = ?", promo.ID, userID).
			Count(&userUsageCount)
		if int(userUsageCount) >= promo.PerUserLimit {
			return 0, nil, "You have already used this promo code the maximum number of times"
		}
	}

	// Check minimum order amount
	if promo.MinOrderAmount > 0 && orderTotal < promo.MinOrderAmount {
		return 0, nil, map[string]interface{}{
			"message":        "Minimum order amount not met",
			"minOrderAmount": promo.MinOrderAmount,
		}
	}

	// Check applicableTo
	if promo.ApplicableTo != "all" {
		var orderCount int64
		database.DB.Model(&models.Order{}).Where("customer_id = ?", userID).Count(&orderCount)
		if promo.ApplicableTo == "new_users" && orderCount > 0 {
			return 0, nil, "This promo code is only for new users"
		}
		if promo.ApplicableTo == "returning_users" && orderCount == 0 {
			return 0, nil, "This promo code is only for returning users"
		}
	}

	// Calculate discount
	var discount float64
	if promo.DiscountType == "percentage" {
		discount = orderTotal * promo.DiscountValue / 100
		if promo.MaxDiscount > 0 && discount > promo.MaxDiscount {
			discount = promo.MaxDiscount
		}
	} else {
		discount = promo.DiscountValue
	}

	// Discount should not exceed order total
	if discount > orderTotal {
		discount = orderTotal
	}

	return discount, &promo, nil
}
