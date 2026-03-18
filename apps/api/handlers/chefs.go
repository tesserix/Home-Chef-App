package handlers

import (
	"context"
	"fmt"
	"log"
	"math"
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

type ChefHandler struct{}

func NewChefHandler() *ChefHandler {
	return &ChefHandler{}
}

// ListChefs returns a paginated list of chefs
func (h *ChefHandler) ListChefs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	cuisine := c.Query("cuisine")
	dietary := c.Query("dietary")
	isOpen := c.Query("isOpen")
	ratingMin := c.Query("rating")
	sortOrder := c.DefaultQuery("order", "desc")

	// Accept both "sortBy" and "sort" (frontend sends "sort")
	sortBy := c.Query("sortBy")
	if sortBy == "" {
		sortBy = c.DefaultQuery("sort", "rating")
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.ChefProfile{}).
		Where("is_verified = ? AND is_active = ?", true, true)

	// Search filter
	if search != "" {
		query = query.Where("business_name ILIKE ? OR description ILIKE ?",
			"%"+search+"%", "%"+search+"%")
	}

	// Cuisine filter
	if cuisine != "" {
		query = query.Where("? = ANY(cuisines)", cuisine)
	}

	// isOpen filter
	if isOpen == "true" {
		query = query.Where("accepting_orders = ?", true)
	}

	// Minimum rating filter
	if ratingMin != "" {
		if r, err := strconv.ParseFloat(ratingMin, 64); err == nil {
			query = query.Where("rating >= ?", r)
		}
	}

	// Dietary filter — find chefs that have at least one menu item with this dietary tag
	if dietary != "" {
		query = query.Where("id IN (SELECT chef_id FROM menu_items WHERE ? = ANY(dietary_tags) AND deleted_at IS NULL)", dietary)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Apply sorting — featured chefs always appear first
	dir := "DESC"
	if sortOrder == "asc" {
		dir = "ASC"
	}

	// Featured chefs ranked first: active featured status (is_featured=true AND featured_until > now)
	featuredOrder := "CASE WHEN is_featured = true AND featured_until > NOW() THEN 0 ELSE 1 END ASC"

	switch sortBy {
	case "rating":
		query = query.Order(featuredOrder + ", rating " + dir)
	case "orders":
		query = query.Order(featuredOrder + ", total_orders " + dir)
	case "newest":
		query = query.Order(featuredOrder + ", created_at " + dir)
	case "price":
		query = query.Order(featuredOrder + ", minimum_order " + dir)
	default:
		query = query.Order(featuredOrder + ", rating " + dir)
	}

	// Get chefs
	var chefs []models.ChefProfile
	if err := query.Offset(offset).Limit(limit).Find(&chefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chefs"})
		return
	}

	// Convert to response
	responses := make([]models.ChefProfileResponse, len(chefs))
	for i, chef := range chefs {
		responses[i] = chef.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
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

// GetChef returns a single chef by ID
func (h *ChefHandler) GetChef(c *gin.Context) {
	id := c.Param("id")
	chefID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}

	var chef models.ChefProfile
	if err := database.DB.Preload("User").First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	var schedules []models.ChefSchedule
	database.DB.Where("chef_id = ?", chef.ID).Find(&schedules)

	c.JSON(http.StatusOK, chef.ToPublicResponse(schedules))
}

// GetChefMenu returns the menu items and categories for a chef
func (h *ChefHandler) GetChefMenu(c *gin.Context) {
	id := c.Param("id")
	chefID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}

	category := c.Query("category")

	query := database.DB.Where("chef_id = ? AND is_available = ? AND is_approved = ?", chefID, true, true).Preload("Images")

	if category != "" {
		query = query.Where("category_id = ?", category)
	}

	var items []models.MenuItem
	if err := query.Order("sort_order, name").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch menu"})
		return
	}

	// Fetch categories for this chef
	var categories []models.MenuCategory
	database.DB.Where("chef_id = ? AND is_active = ?", chefID, true).
		Order("sort_order, name").Find(&categories)

	responses := make([]models.MenuItemResponse, len(items))
	for i, item := range items {
		responses[i] = item.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"categories": categories,
		"items":      responses,
	})
}

// GetChefReviews returns reviews for a chef
func (h *ChefHandler) GetChefReviews(c *gin.Context) {
	id := c.Param("id")
	chefID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	var reviews []models.Review
	var total int64

	database.DB.Model(&models.Review{}).Where("chef_id = ? AND is_approved = ?", chefID, true).Count(&total)

	if err := database.DB.Preload("Customer").
		Where("chef_id = ? AND is_approved = ?", chefID, true).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	responses := make([]models.ReviewResponse, len(reviews))
	for i, review := range reviews {
		responses[i] = review.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
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

// ---- Chef Dashboard Endpoints ----

// GetChefProfile returns the full chef profile for the authenticated chef
func (h *ChefHandler) GetChefProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Load schedules and convert to operatingHours map
	var schedules []models.ChefSchedule
	database.DB.Where("chef_id = ?", chef.ID).Find(&schedules)

	dayNames := map[int]string{
		0: "sunday", 1: "monday", 2: "tuesday", 3: "wednesday",
		4: "thursday", 5: "friday", 6: "saturday",
	}

	operatingHours := make(map[string]interface{})
	for _, s := range schedules {
		name, ok := dayNames[s.DayOfWeek]
		if !ok {
			continue
		}
		if s.IsClosed {
			// Don't include closed days — frontend treats missing = closed
			continue
		}
		operatingHours[name] = map[string]string{
			"open":  s.OpenTime,
			"close": s.CloseTime,
		}
	}

	resp := chef.ToResponse()
	// Merge profile response with operating hours
	result := map[string]interface{}{
		"id":              resp.ID,
		"userId":          resp.UserID,
		"businessName":    resp.BusinessName,
		"description":     resp.Description,
		"profileImage":    resp.ProfileImage,
		"bannerImage":     resp.BannerImage,
		"cuisines":        resp.Cuisines,
		"specialties":     resp.Specialties,
		"prepTime":        resp.PrepTime,
		"minimumOrder":    resp.MinimumOrder,
		"serviceRadius":   resp.ServiceRadius,
		"rating":          resp.Rating,
		"totalReviews":    resp.TotalReviews,
		"totalOrders":     resp.TotalOrders,
		"verified":        resp.IsVerified,
		"acceptingOrders": resp.AcceptingOrders,
		"kitchenPhotos":   resp.KitchenPhotos,
		"city":            resp.City,
		"state":           resp.State,
		"operatingHours":  operatingHours,
	}

	c.JSON(http.StatusOK, result)
}

// GetChefDashboard returns the chef's dashboard data
func (h *ChefHandler) GetChefDashboard(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Get today's stats
	var todayOrders int64
	var todayRevenue float64
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND DATE(created_at) = CURRENT_DATE", chef.ID).
		Count(&todayOrders)
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND DATE(created_at) = CURRENT_DATE AND payment_status = ?", chef.ID, models.PaymentCompleted).
		Select("COALESCE(SUM(total), 0)").Scan(&todayRevenue)

	// Get pending orders
	var pendingOrders int64
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND status = ?", chef.ID, models.OrderStatusPending).
		Count(&pendingOrders)

	// Get this week's stats
	var weekOrders int64
	var weekRevenue float64
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND created_at >= CURRENT_DATE - INTERVAL '7 days'", chef.ID).
		Count(&weekOrders)
	database.DB.Model(&models.Order{}).
		Where("chef_id = ? AND created_at >= CURRENT_DATE - INTERVAL '7 days' AND payment_status = ?", chef.ID, models.PaymentCompleted).
		Select("COALESCE(SUM(total), 0)").Scan(&weekRevenue)

	c.JSON(http.StatusOK, gin.H{
		"todayOrders":    todayOrders,
		"todayRevenue":   todayRevenue,
		"pendingOrders":  pendingOrders,
		"weekOrders":     weekOrders,
		"weekRevenue":    weekRevenue,
		"rating":         chef.Rating,
		"totalReviews":   chef.TotalReviews,
		"totalOrders":    chef.TotalOrders,
		"acceptingOrders": chef.AcceptingOrders,
	})
}

// UpdateChefProfileRequest represents chef profile update
type UpdateChefProfileRequest struct {
	BusinessName    string                       `json:"businessName"`
	Description     string                       `json:"description"`
	ProfileImage    string                       `json:"profileImage"`
	BannerImage     string                       `json:"bannerImage"`
	Cuisines        []string                     `json:"cuisines"`
	Specialties     []string                     `json:"specialties"`
	PrepTime        string                       `json:"prepTime"`
	MinimumOrder    float64                      `json:"minimumOrder"`
	ServiceRadius   float64                      `json:"serviceRadius"`
	AcceptingOrders *bool                        `json:"acceptingOrders"`
	OperatingHours  map[string]*DayHoursUpdate   `json:"operatingHours"`
}

type DayHoursUpdate struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// UpdateChefProfile updates the chef's profile
func (h *ChefHandler) UpdateChefProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req UpdateChefProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	if req.BusinessName != "" {
		chef.BusinessName = req.BusinessName
	}
	if req.Description != "" {
		chef.Description = req.Description
	}
	if req.ProfileImage != "" {
		chef.ProfileImage = req.ProfileImage
	}
	if req.BannerImage != "" {
		chef.BannerImage = req.BannerImage
	}
	if req.Cuisines != nil {
		chef.Cuisines = req.Cuisines
	}
	if req.Specialties != nil {
		chef.Specialties = req.Specialties
	}
	if req.PrepTime != "" {
		chef.PrepTime = req.PrepTime
	}
	if req.MinimumOrder > 0 {
		chef.MinimumOrder = req.MinimumOrder
	}
	if req.ServiceRadius > 0 {
		chef.ServiceRadius = req.ServiceRadius
	}
	if req.AcceptingOrders != nil {
		chef.AcceptingOrders = *req.AcceptingOrders
	}

	if err := database.DB.Save(&chef).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Update operating hours if provided
	if req.OperatingHours != nil {
		dayMap := map[string]int{
			"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3,
			"thursday": 4, "friday": 5, "saturday": 6,
		}

		// Delete existing schedules and recreate
		database.DB.Where("chef_id = ?", chef.ID).Delete(&models.ChefSchedule{})

		for day, dayNum := range dayMap {
			schedule := models.ChefSchedule{
				ChefID:    chef.ID,
				DayOfWeek: dayNum,
				IsClosed:  true,
			}
			if dh, ok := req.OperatingHours[day]; ok && dh != nil {
				schedule.IsClosed = false
				schedule.OpenTime = dh.Open
				schedule.CloseTime = dh.Close
			}
			database.DB.Create(&schedule)
		}
	}

	c.JSON(http.StatusOK, chef.ToResponse())
}

// GetChefOrders returns orders for the chef
func (h *ChefHandler) GetChefOrders(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := database.DB.Where("chef_id = ?", chef.ID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.Order{}).Count(&total)

	var orders []models.Order
	if err := query.Preload("Items").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}

	responses := make([]models.OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = order.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// UpdateOrderStatus updates an order's status
func (h *ChefHandler) UpdateOrderStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderID, chef.ID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order.Status = models.OrderStatus(req.Status)
	if err := database.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	// Publish order status update event
	go func() {
		orderEvent := services.OrderEvent{
			OrderID:     order.ID,
			OrderNumber: order.OrderNumber,
			CustomerID:  order.CustomerID,
			ChefID:      order.ChefID,
			Status:      string(order.Status),
			Total:       order.Total,
		}

		// Determine which subject to publish to based on status
		subject := services.SubjectOrderUpdated
		if order.Status == models.OrderStatusDelivered {
			subject = services.SubjectOrderDelivered
		}

		if err := services.PublishOrderEvent(subject, orderEvent); err != nil {
			log.Printf("Failed to publish order status update event: %v", err)
		}
	}()

	c.JSON(http.StatusOK, order.ToResponse())
}

// GetChefReviewsForDashboard returns all reviews for the authenticated chef
func (h *ChefHandler) GetChefReviewsForDashboard(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset := (page - 1) * limit

	var reviews []models.Review
	var total int64

	database.DB.Model(&models.Review{}).Where("chef_id = ?", chef.ID).Count(&total)

	if err := database.DB.Preload("Customer").
		Where("chef_id = ?", chef.ID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reviews"})
		return
	}

	responses := make([]models.ReviewResponse, len(reviews))
	for i, review := range reviews {
		responses[i] = review.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// ReplyToReview allows a chef to respond to a review
func (h *ChefHandler) ReplyToReview(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	reviewID := c.Param("reviewId")

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var review models.Review
	if err := database.DB.Where("id = ? AND chef_id = ?", reviewID, chef.ID).First(&review).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		return
	}

	var req struct {
		Response string `json:"response" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Response text is required"})
		return
	}

	now := time.Now()
	review.ChefResponse = req.Response
	review.ChefRespondedAt = &now

	if err := database.DB.Save(&review).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save reply"})
		return
	}

	// Reload with customer for response DTO
	database.DB.Preload("Customer").First(&review, review.ID)

	c.JSON(http.StatusOK, review.ToResponse())
}

// GetChefSettings returns the chef's settings (creates defaults if not found)
func (h *ChefHandler) GetChefSettings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var settings models.ChefSettings
	if err := database.DB.Where("chef_id = ?", chef.ID).First(&settings).Error; err != nil {
		// Create default settings
		settings = models.ChefSettings{
			ChefID:              chef.ID,
			AutoAcceptOrders:    false,
			AutoAcceptThreshold: 0,
			PushNewOrder:        true,
			PushOrderUpdate:     true,
			EmailDailySummary:   true,
			EmailWeeklyReport:   true,
			SmsNewOrder:         false,
		}
		database.DB.Create(&settings)
	}

	// Load user to get auth provider
	var user models.User
	database.DB.First(&user, "id = ?", userID)

	// Return in the shape the frontend expects
	c.JSON(http.StatusOK, gin.H{
		"notifications": gin.H{
			"pushNewOrder":     settings.PushNewOrder,
			"pushOrderUpdate":  settings.PushOrderUpdate,
			"emailDailySummary": settings.EmailDailySummary,
			"emailWeeklyReport": settings.EmailWeeklyReport,
			"smsNewOrder":      settings.SmsNewOrder,
		},
		"autoAcceptOrders":    settings.AutoAcceptOrders,
		"autoAcceptThreshold": settings.AutoAcceptThreshold,
		"acceptingOrders":     chef.AcceptingOrders,
		"authProvider":        string(user.AuthProvider),
	})
}

// UpdateChefSettings updates the chef's settings
func (h *ChefHandler) UpdateChefSettings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req struct {
		Notifications struct {
			PushNewOrder     bool `json:"pushNewOrder"`
			PushOrderUpdate  bool `json:"pushOrderUpdate"`
			EmailDailySummary bool `json:"emailDailySummary"`
			EmailWeeklyReport bool `json:"emailWeeklyReport"`
			SmsNewOrder      bool `json:"smsNewOrder"`
		} `json:"notifications"`
		AutoAcceptOrders    bool    `json:"autoAcceptOrders"`
		AutoAcceptThreshold float64 `json:"autoAcceptThreshold"`
		AcceptingOrders     bool    `json:"acceptingOrders"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update acceptingOrders on chef profile
	database.DB.Model(&chef).Update("accepting_orders", req.AcceptingOrders)

	// Upsert chef settings
	var settings models.ChefSettings
	if err := database.DB.Where("chef_id = ?", chef.ID).First(&settings).Error; err != nil {
		settings = models.ChefSettings{ChefID: chef.ID}
	}
	settings.AutoAcceptOrders = req.AutoAcceptOrders
	settings.AutoAcceptThreshold = req.AutoAcceptThreshold
	settings.PushNewOrder = req.Notifications.PushNewOrder
	settings.PushOrderUpdate = req.Notifications.PushOrderUpdate
	settings.EmailDailySummary = req.Notifications.EmailDailySummary
	settings.EmailWeeklyReport = req.Notifications.EmailWeeklyReport
	settings.SmsNewOrder = req.Notifications.SmsNewOrder
	database.DB.Save(&settings)

	c.JSON(http.StatusOK, gin.H{
		"notifications": gin.H{
			"pushNewOrder":     settings.PushNewOrder,
			"pushOrderUpdate":  settings.PushOrderUpdate,
			"emailDailySummary": settings.EmailDailySummary,
			"emailWeeklyReport": settings.EmailWeeklyReport,
			"smsNewOrder":      settings.SmsNewOrder,
		},
		"autoAcceptOrders":    settings.AutoAcceptOrders,
		"autoAcceptThreshold": settings.AutoAcceptThreshold,
		"acceptingOrders":     req.AcceptingOrders,
	})
}

// GetPayoutDetails returns the chef's payout configuration with masked bank details
func (h *ChefHandler) GetPayoutDetails(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// All sensitive payout data lives in GCP Secret Manager — read and mask for display
	vendorID := chef.ID.String()
	ctx := c.Request.Context()

	bankAccountName, _ := services.GetVendorSecret(ctx, vendorID, "bank-account-name")
	bankAccountNumber, _ := services.GetVendorSecret(ctx, vendorID, "bank-account-number")
	bankIFSC, _ := services.GetVendorSecret(ctx, vendorID, "bank-ifsc")
	upiID, _ := services.GetVendorSecret(ctx, vendorID, "upi-id")

	c.JSON(http.StatusOK, gin.H{
		"payoutMethod":      chef.PayoutMethod,
		"bankAccountName":   bankAccountName,
		"bankAccountNumber": maskBankAccount(bankAccountNumber),
		"bankIFSC":          bankIFSC,
		"upiId":             maskEmail(upiID),
		"razorpayConnected": chef.RazorpayAccountID != "",
		"razorpayAccountId": maskID(chef.RazorpayAccountID),
	})
}

// SavePayoutDetails saves the chef's payout information.
// Sensitive fields (account number, UPI ID) are stored in GCP Secret Manager.
// Only masked values are stored in the database for display purposes.
// Also creates a Razorpay Route linked account so payments split directly to the chef.
func (h *ChefHandler) SavePayoutDetails(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		PayoutMethod      string `json:"payoutMethod" binding:"required"`
		BankAccountNumber string `json:"bankAccountNumber"`
		BankIFSC          string `json:"bankIFSC"`
		BankAccountName   string `json:"bankAccountName"`
		UpiID             string `json:"upiId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PayoutMethod != "bank_transfer" && req.PayoutMethod != "upi" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payoutMethod must be 'bank_transfer' or 'upi'"})
		return
	}

	if req.PayoutMethod == "bank_transfer" {
		if req.BankAccountNumber == "" || req.BankIFSC == "" || req.BankAccountName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bankAccountNumber, bankIFSC, and bankAccountName are required for bank_transfer"})
			return
		}
	}

	if req.PayoutMethod == "upi" {
		if req.UpiID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "upiId is required for upi payout method"})
			return
		}
	}

	var chef models.ChefProfile
	if err := database.DB.Preload("User").Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	vendorID := chef.ID.String()

	// DB stores ONLY the payout method (non-sensitive selector).
	// All sensitive fields (account number, IFSC, name, UPI) live exclusively in Secret Manager.
	chef.PayoutMethod = req.PayoutMethod
	chef.BankAccountNumber = "" // No sensitive data in DB
	chef.BankIFSC = ""          // Stored in Secret Manager
	chef.BankAccountName = ""   // Stored in Secret Manager
	chef.UpiID = ""             // Stored in Secret Manager

	if err := database.DB.Save(&chef).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save payout details"})
		return
	}

	log.Printf("Payout details saved for vendor %s (method: %s)", vendorID, req.PayoutMethod)

	// Store sensitive fields in GCP Secret Manager asynchronously
	// (Secret Manager creation can take several seconds on first call)
	go func() {
		ctx := context.Background()
		secretFields := map[string]string{
			"bank-account-number": req.BankAccountNumber,
			"bank-account-name":   req.BankAccountName,
			"bank-ifsc":           req.BankIFSC,
			"upi-id":              req.UpiID,
		}
		for field, value := range secretFields {
			if value != "" {
				if err := services.StoreVendorSecret(ctx, vendorID, field, value); err != nil {
					log.Printf("Warning: failed to store secret %s for vendor %s: %v", field, vendorID, err)
				}
			}
		}
		log.Printf("Secrets stored in Secret Manager for vendor %s", vendorID)
	}()

	// Create Razorpay Route linked account asynchronously if not already created
	razorpayConnected := chef.RazorpayAccountID != ""
	if !razorpayConnected {
		go func() {
			rz := services.GetRazorpay()
			if rz == nil {
				return
			}
			contactName := chef.User.FirstName + " " + chef.User.LastName
			linkedAcct, err := rz.CreateLinkedAccount(&services.LinkedAccountRequest{
				Email:        chef.User.Email,
				Phone:        chef.User.Phone,
				LegalName:    chef.BusinessName,
				BusinessType: "individual",
				ContactName:  contactName,
			})
			if err != nil {
				log.Printf("Failed to create Razorpay linked account for vendor %s", vendorID)
				return
			}
			database.DB.Model(&models.ChefProfile{}).Where("id = ?", chef.ID).Update("razorpay_account_id", linkedAcct.ID)
			log.Printf("Created Razorpay linked account for vendor %s", vendorID)
		}()
	}

	resp := gin.H{
		"message":           "Payout details saved",
		"payoutMethod":      chef.PayoutMethod,
		"bankAccountName":   req.BankAccountName,
		"bankAccountNumber": maskBankAccount(req.BankAccountNumber),
		"bankIFSC":          req.BankIFSC,
		"upiId":             maskEmail(req.UpiID),
		"razorpayConnected": razorpayConnected,
		"razorpayAccountId": maskID(chef.RazorpayAccountID),
	}

	c.JSON(http.StatusOK, resp)
}

// GetChefAnalytics returns analytics data for the authenticated chef
func (h *ChefHandler) GetChefAnalytics(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	// Parse period (7d, 30d, 90d)
	period := c.DefaultQuery("period", "7d")
	var days int
	switch period {
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		days = 7
	}

	since := time.Now().AddDate(0, 0, -days)

	// ── Order & Revenue Trends (grouped by date) ──
	type dailyStat struct {
		Date     string  `json:"date"`
		Orders   int     `json:"orders"`
		Revenue  float64 `json:"revenue"`
	}
	var dailyStats []dailyStat
	database.DB.Raw(`
		SELECT DATE(created_at) as date,
		       COUNT(*) as orders,
		       COALESCE(SUM(total), 0) as revenue
		FROM orders
		WHERE chef_id = ? AND created_at >= ? AND deleted_at IS NULL
		GROUP BY DATE(created_at)
		ORDER BY date
	`, chef.ID, since).Scan(&dailyStats)

	// Build label→value maps for the full date range
	dateMap := make(map[string]dailyStat)
	for _, ds := range dailyStats {
		dateMap[ds.Date] = ds
	}

	var orderLabels []string
	var orderData []int
	var revenueLabels []string
	var revenueData []float64

	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i)
		dateStr := d.Format("2006-01-02")
		var label string
		if days <= 7 {
			label = d.Format("Mon")
		} else {
			label = d.Format("Jan 2")
		}

		ds := dateMap[dateStr]
		orderLabels = append(orderLabels, label)
		orderData = append(orderData, ds.Orders)
		revenueLabels = append(revenueLabels, label)
		revenueData = append(revenueData, math.Round(ds.Revenue*100)/100)
	}

	// ── Popular Items (top 5 by order count) ──
	type popularItem struct {
		Name   string `json:"name"`
		Orders int    `json:"orders"`
	}
	var topItems []popularItem
	database.DB.Raw(`
		SELECT oi.name, SUM(oi.quantity) as orders
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.chef_id = ? AND o.created_at >= ? AND o.deleted_at IS NULL
		GROUP BY oi.name
		ORDER BY orders DESC
		LIMIT 5
	`, chef.ID, since).Scan(&topItems)

	// Calculate percentages
	var totalItemOrders int
	for _, it := range topItems {
		totalItemOrders += it.Orders
	}
	popularItemsResp := make([]gin.H, len(topItems))
	for i, it := range topItems {
		pct := 0
		if totalItemOrders > 0 {
			pct = int(math.Round(float64(it.Orders) / float64(totalItemOrders) * 100))
		}
		popularItemsResp[i] = gin.H{
			"name":       it.Name,
			"orders":     it.Orders,
			"percentage": pct,
		}
	}

	// ── Peak Hours ──
	type hourStat struct {
		Hour   int `json:"hour"`
		Orders int `json:"orders"`
	}
	var hourStats []hourStat
	database.DB.Raw(`
		SELECT EXTRACT(HOUR FROM created_at)::int as hour, COUNT(*) as orders
		FROM orders
		WHERE chef_id = ? AND created_at >= ? AND deleted_at IS NULL
		GROUP BY hour
		ORDER BY hour
	`, chef.ID, since).Scan(&hourStats)

	hourMap := make(map[int]int)
	for _, hs := range hourStats {
		hourMap[hs.Hour] = hs.Orders
	}
	peakHours := make([]gin.H, 0)
	for h := 8; h <= 22; h++ {
		label := fmt.Sprintf("%d %s", func() int {
			if h == 12 { return 12 }
			if h > 12 { return h - 12 }
			return h
		}(), func() string {
			if h >= 12 { return "PM" }
			return "AM"
		}())
		peakHours = append(peakHours, gin.H{
			"hour":   label,
			"orders": hourMap[h],
		})
	}

	// ── Revenue by Category ──
	type categoryStat struct {
		Category string  `json:"category"`
		Revenue  float64 `json:"revenue"`
	}
	var catStats []categoryStat
	database.DB.Raw(`
		SELECT COALESCE(mc.name, 'Uncategorized') as category,
		       COALESCE(SUM(oi.subtotal), 0) as revenue
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		LEFT JOIN menu_items mi ON mi.id = oi.menu_item_id
		LEFT JOIN menu_categories mc ON mc.id = mi.category_id
		WHERE o.chef_id = ? AND o.created_at >= ? AND o.deleted_at IS NULL
		GROUP BY mc.name
		ORDER BY revenue DESC
	`, chef.ID, since).Scan(&catStats)

	var totalCatRevenue float64
	for _, cs := range catStats {
		totalCatRevenue += cs.Revenue
	}
	revByCat := make([]gin.H, len(catStats))
	for i, cs := range catStats {
		pct := 0
		if totalCatRevenue > 0 {
			pct = int(math.Round(cs.Revenue / totalCatRevenue * 100))
		}
		revByCat[i] = gin.H{
			"category":   cs.Category,
			"revenue":    math.Round(cs.Revenue*100) / 100,
			"percentage": pct,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"orderTrends":       gin.H{"labels": orderLabels, "data": orderData},
		"revenueTrends":     gin.H{"labels": revenueLabels, "data": revenueData},
		"popularItems":      popularItemsResp,
		"peakHours":         peakHours,
		"revenueByCategory": revByCat,
	})
}
