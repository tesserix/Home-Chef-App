package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type PromotionHandler struct{}

func NewPromotionHandler() *PromotionHandler {
	return &PromotionHandler{}
}

// GetFeaturedAdPricing returns the featured ad price for the chef's country.
// GET /chef/promotion/pricing
func (h *PromotionHandler) GetFeaturedAdPricing(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Resolve country from chef's city/zone
	countryCode := resolveChefCountry(userID)
	pricing := models.DefaultPromotionPricing(countryCode)

	// Check for override in PlatformSettings
	var setting models.PlatformSettings
	key := "promotion." + countryCode + ".chef.featured_monthly_price"
	if err := database.DB.Where("key = ?", key).First(&setting).Error; err == nil {
		if val, err := strconv.ParseFloat(setting.Value, 64); err == nil {
			pricing.MonthlyPrice = val
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"monthlyPrice": pricing.MonthlyPrice,
		"currency":     pricing.Currency,
		"countryCode":  countryCode,
		"duration":     30,
		"description":  "Your kitchen will appear at the top of search results and on the featured section for 30 days.",
	})
}

// PurchaseFeaturedAd creates a Razorpay order for the featured ad and activates on payment.
// POST /chef/promotion/purchase
func (h *PromotionHandler) PurchaseFeaturedAd(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Check if already featured
	if chef.IsFeatured && chef.FeaturedUntil != nil && chef.FeaturedUntil.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        "Your kitchen is already featured",
			"featuredUntil": chef.FeaturedUntil,
		})
		return
	}

	countryCode := resolveChefCountry(userID)
	pricing := models.DefaultPromotionPricing(countryCode)

	// Check for override
	var setting models.PlatformSettings
	key := "promotion." + countryCode + ".chef.featured_monthly_price"
	if err := database.DB.Where("key = ?", key).First(&setting).Error; err == nil {
		if val, err := strconv.ParseFloat(setting.Value, 64); err == nil {
			pricing.MonthlyPrice = val
		}
	}

	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 1, 0) // 30 days

	// Create promotion record
	promo := models.ChefPromotion{
		ChefID:    chef.ID,
		Status:    models.PromotionPending,
		Amount:    pricing.MonthlyPrice,
		Currency:  pricing.Currency,
		Duration:  30,
		StartsAt:  now,
		ExpiresAt: expiresAt,
	}

	if err := database.DB.Create(&promo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create promotion"})
		return
	}

	// Create Razorpay order (no Route transfers — this goes to Fe3dr's account)
	rzOrder, err := rz.CreateOrder(&services.OrderRequest{
		Amount:   services.ToPaise(pricing.MonthlyPrice),
		Currency: pricing.Currency,
		Receipt:  "promo-" + promo.ID.String()[:8],
		Notes: map[string]string{
			"type":         "featured_ad",
			"promotion_id": promo.ID.String(),
			"chef_id":      chef.ID.String(),
		},
	})
	if err != nil {
		log.Printf("Failed to create Razorpay order for promotion: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}

	// Save Razorpay order ID
	database.DB.Model(&promo).Update("razorpay_order_id", rzOrder.ID)

	c.JSON(http.StatusOK, gin.H{
		"promotionId":     promo.ID,
		"razorpayOrderId": rzOrder.ID,
		"razorpayKeyId":   rz.GetKeyID(),
		"amount":          services.ToPaise(pricing.MonthlyPrice),
		"currency":        pricing.Currency,
	})
}

// ConfirmFeaturedAd confirms payment and activates the featured ad.
// POST /chef/promotion/confirm
func (h *PromotionHandler) ConfirmFeaturedAd(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		PromotionID       string `json:"promotionId" binding:"required"`
		RazorpayPaymentID string `json:"razorpayPaymentId" binding:"required"`
		RazorpayOrderID   string `json:"razorpayOrderId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	promoID, err := uuid.Parse(req.PromotionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid promotion ID"})
		return
	}

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var promo models.ChefPromotion
	if err := database.DB.Where("id = ? AND chef_id = ? AND status = ?", promoID, chef.ID, models.PromotionPending).
		First(&promo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Promotion not found or already activated"})
		return
	}

	// Verify payment
	rz := services.GetRazorpay()
	if rz != nil {
		payment, err := rz.FetchPayment(req.RazorpayPaymentID)
		if err != nil || payment.Status != "captured" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment not captured"})
			return
		}
	}

	now := time.Now()
	expiresAt := now.AddDate(0, 1, 0)

	// Activate promotion
	database.DB.Model(&promo).Updates(map[string]interface{}{
		"status":              models.PromotionActive,
		"razorpay_payment_id": req.RazorpayPaymentID,
		"starts_at":           now,
		"expires_at":          expiresAt,
	})

	// Set chef as featured
	database.DB.Model(&chef).Updates(map[string]interface{}{
		"is_featured":    true,
		"featured_until": expiresAt,
	})

	// Publish event
	if err := services.PublishEvent("promotion.activated", "promotion.activated", userID, map[string]interface{}{
		"promotion_id": promo.ID.String(),
		"chef_id":      chef.ID.String(),
		"amount":       promo.Amount,
		"currency":     promo.Currency,
		"expires_at":   expiresAt.Format(time.RFC3339),
	}); err != nil {
		log.Printf("Failed to publish promotion event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Featured ad activated!",
		"featuredUntil": expiresAt,
	})
}

// GetMyPromotions returns the chef's promotion history.
// GET /chef/promotion/history
func (h *PromotionHandler) GetMyPromotions(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
		return
	}

	var promos []models.ChefPromotion
	database.DB.Where("chef_id = ?", chef.ID).Order("created_at DESC").Find(&promos)

	c.JSON(http.StatusOK, gin.H{
		"data":       promos,
		"isFeatured": chef.IsFeatured && chef.FeaturedUntil != nil && chef.FeaturedUntil.After(time.Now()),
		"featuredUntil": chef.FeaturedUntil,
	})
}

// AdminGetPromotionStats returns promotion revenue stats.
// GET /admin/promotions/stats
func (h *PromotionHandler) AdminGetPromotionStats(c *gin.Context) {
	var totalRevenue float64
	var activeCount int64
	var totalCount int64

	database.DB.Model(&models.ChefPromotion{}).
		Where("status = ?", models.PromotionActive).
		Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	database.DB.Model(&models.ChefPromotion{}).
		Where("status = ? AND expires_at > ?", models.PromotionActive, time.Now()).
		Count(&activeCount)

	database.DB.Model(&models.ChefPromotion{}).
		Where("status IN ?", []models.PromotionStatus{models.PromotionActive, models.PromotionExpired}).
		Count(&totalCount)

	c.JSON(http.StatusOK, gin.H{
		"totalRevenue":    totalRevenue,
		"activePromotions": activeCount,
		"totalPurchases":  totalCount,
	})
}

// AdminListPromotions returns paginated promotion records.
// GET /admin/promotions
func (h *PromotionHandler) AdminListPromotions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 { page = 1 }
	if limit > 100 { limit = 100 }
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.ChefPromotion{}).Count(&total)

	var promos []models.ChefPromotion
	database.DB.Preload("Chef.User").Order("created_at DESC").Offset(offset).Limit(limit).Find(&promos)

	c.JSON(http.StatusOK, gin.H{
		"data": promos,
		"pagination": gin.H{
			"page": page, "limit": limit, "total": total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// resolveChefCountry resolves the country code for a chef from their delivery zone or city.
func resolveChefCountry(userID uuid.UUID) string {
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		return "IN"
	}
	// Try to match chef's city to a delivery zone for country code
	var zone models.DeliveryZone
	if err := database.DB.Where("city ILIKE ?", "%"+chef.City+"%").First(&zone).Error; err == nil {
		return zone.Country
	}
	return "IN" // Default to India
}
