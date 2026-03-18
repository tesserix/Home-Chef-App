package handlers

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type DeliveryHandler struct{}

func NewDeliveryHandler() *DeliveryHandler {
	return &DeliveryHandler{}
}

// GetProfile returns the delivery partner profile for the authenticated user
func (h *DeliveryHandler) GetProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Preload("User").
		Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	c.JSON(http.StatusOK, partner.ToDetailResponse())
}

// UpdateProfile updates the delivery partner profile
func (h *DeliveryHandler) UpdateProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	var req struct {
		VehicleType   *string `json:"vehicleType"`
		VehicleNumber *string `json:"vehicleNumber"`
		LicenseNumber *string `json:"licenseNumber"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.VehicleType != nil {
		partner.VehicleType = *req.VehicleType
	}
	if req.VehicleNumber != nil {
		partner.VehicleNumber = *req.VehicleNumber
	}
	if req.LicenseNumber != nil {
		partner.LicenseNumber = *req.LicenseNumber
	}

	if err := database.DB.Save(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, partner.ToDetailResponse())
}

// GetStats returns dashboard statistics for the delivery partner
func (h *DeliveryHandler) GetStats(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	// Today's stats
	today := time.Now().Truncate(24 * time.Hour)
	var todayDeliveries int64
	var todayEarnings float64

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ? AND delivered_at >= ?",
			partner.ID, models.DeliveryDelivered, today).
		Count(&todayDeliveries)

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ? AND delivered_at >= ?",
			partner.ID, models.DeliveryDelivered, today).
		Select("COALESCE(SUM(total_payout), 0)").
		Scan(&todayEarnings)

	// This week stats
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))
	var weekDeliveries int64
	var weekEarnings float64

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ? AND delivered_at >= ?",
			partner.ID, models.DeliveryDelivered, weekStart).
		Count(&weekDeliveries)

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ? AND delivered_at >= ?",
			partner.ID, models.DeliveryDelivered, weekStart).
		Select("COALESCE(SUM(total_payout), 0)").
		Scan(&weekEarnings)

	// This month stats
	monthStart := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	var monthDeliveries int64
	var monthEarnings float64

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ? AND delivered_at >= ?",
			partner.ID, models.DeliveryDelivered, monthStart).
		Count(&monthDeliveries)

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ? AND delivered_at >= ?",
			partner.ID, models.DeliveryDelivered, monthStart).
		Select("COALESCE(SUM(total_payout), 0)").
		Scan(&monthEarnings)

	// Active delivery count
	var activeDeliveries int64
	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status IN ?",
			partner.ID, []models.DeliveryStatus{
				models.DeliveryAssigned,
				models.DeliveryPickedUp,
				models.DeliveryInTransit,
			}).
		Count(&activeDeliveries)

	// Available deliveries count (orders ready but unassigned)
	var availableCount int64
	database.DB.Model(&models.Order{}).
		Where("status = ? AND delivery_id IS NULL", models.OrderStatusReady).
		Count(&availableCount)

	c.JSON(http.StatusOK, gin.H{
		"partner": gin.H{
			"id":       partner.ID,
			"isOnline": partner.IsOnline,
			"rating":   partner.Rating,
		},
		"today": gin.H{
			"deliveries": todayDeliveries,
			"earnings":   todayEarnings,
		},
		"week": gin.H{
			"deliveries": weekDeliveries,
			"earnings":   weekEarnings,
		},
		"month": gin.H{
			"deliveries": monthDeliveries,
			"earnings":   monthEarnings,
		},
		"active":          activeDeliveries,
		"availableOrders": availableCount,
		"totalDeliveries": partner.TotalDeliveries,
		"totalReviews":    partner.TotalReviews,
	})
}

// ToggleOnline toggles the delivery partner's online/offline status
func (h *DeliveryHandler) ToggleOnline(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	var req struct {
		IsOnline bool `json:"isOnline"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	partner.IsOnline = req.IsOnline
	if err := database.DB.Save(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"isOnline": partner.IsOnline,
	})
}

// UpdateLocation updates the delivery partner's current location
func (h *DeliveryHandler) UpdateLocation(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	var req struct {
		Latitude  float64 `json:"latitude" binding:"required"`
		Longitude float64 `json:"longitude" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	partner.CurrentLatitude = req.Latitude
	partner.CurrentLongitude = req.Longitude

	if err := database.DB.Save(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update location"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetCurrentDelivery returns the current active delivery
func (h *DeliveryHandler) GetCurrentDelivery(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	var delivery models.Delivery
	err := database.DB.Preload("Order").Preload("Order.Items").Preload("Order.Chef").
		Where("delivery_partner_id = ? AND status IN ?",
			partner.ID, []models.DeliveryStatus{
				models.DeliveryAssigned,
				models.DeliveryPickedUp,
				models.DeliveryInTransit,
			}).
		Order("assigned_at DESC").
		First(&delivery).Error

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"delivery": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"delivery": deliveryDetailResponse(&delivery),
	})
}

// GetAvailableDeliveries returns orders ready for pickup that haven't been assigned
func (h *DeliveryHandler) GetAvailableDeliveries(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	if !partner.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account not verified yet"})
		return
	}

	if !partner.IsOnline {
		c.JSON(http.StatusOK, gin.H{
			"data":    []interface{}{},
			"message": "Go online to see available deliveries",
		})
		return
	}

	// Find orders that are ready for pickup and don't have a delivery assigned
	var orders []models.Order
	query := database.DB.Preload("Items").Preload("Chef").
		Where("status = ? AND delivery_id IS NULL", models.OrderStatusReady)

	// If partner has location, sort by distance
	if partner.CurrentLatitude != 0 && partner.CurrentLongitude != 0 {
		// Simple distance ordering using Haversine approximation
		query = query.Order("created_at ASC")
	} else {
		query = query.Order("created_at ASC")
	}

	if err := query.Limit(20).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch available deliveries"})
		return
	}

	type AvailableDelivery struct {
		OrderID         uuid.UUID   `json:"orderId"`
		OrderNumber     string      `json:"orderNumber"`
		ChefName        string      `json:"chefName"`
		ItemCount       int         `json:"itemCount"`
		PickupAddress   string      `json:"pickupAddress"`
		DropoffAddress  string      `json:"dropoffAddress"`
		Distance        float64     `json:"distance"`
		EstimatedPayout float64     `json:"estimatedPayout"`
		CreatedAt       time.Time   `json:"createdAt"`
	}

	available := make([]AvailableDelivery, 0, len(orders))
	for _, order := range orders {
		distance := 0.0
		if partner.CurrentLatitude != 0 && order.Chef.Latitude != 0 {
			distance = haversine(
				partner.CurrentLatitude, partner.CurrentLongitude,
				order.Chef.Latitude, order.Chef.Longitude,
			)
		}

		available = append(available, AvailableDelivery{
			OrderID:     order.ID,
			OrderNumber: order.OrderNumber,
			ChefName:    order.Chef.BusinessName,
			ItemCount:   len(order.Items),
			PickupAddress: order.Chef.AddressLine1 + ", " + order.Chef.City,
			DropoffAddress: order.DeliveryAddressLine1 + ", " + order.DeliveryAddressCity,
			Distance:        distance,
			EstimatedPayout: order.DeliveryFee + order.Tip, // 100% — subscription model, no platform cut
			CreatedAt:       order.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": available})
}

// AcceptDelivery accepts a delivery assignment for an order
func (h *DeliveryHandler) AcceptDelivery(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	if !partner.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account not verified"})
		return
	}

	if !partner.IsOnline {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You must be online to accept deliveries"})
		return
	}

	// Check for existing active delivery
	var activeCount int64
	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status IN ?",
			partner.ID, []models.DeliveryStatus{
				models.DeliveryAssigned,
				models.DeliveryPickedUp,
				models.DeliveryInTransit,
			}).
		Count(&activeCount)

	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "You already have an active delivery"})
		return
	}

	// Start transaction
	tx := database.DB.Begin()

	// Get the order and lock it
	var order models.Order
	if err := tx.Preload("Chef").
		Where("id = ? AND status = ? AND delivery_id IS NULL", orderUUID, models.OrderStatusReady).
		First(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not available for delivery"})
		return
	}

	// Calculate distance and estimated duration
	distance := haversine(
		order.Chef.Latitude, order.Chef.Longitude,
		order.DeliveryLatitude, order.DeliveryLongitude,
	)
	estimatedDuration := int(math.Ceil(distance / 0.5)) // ~30km/h average speed, in minutes
	if estimatedDuration < 10 {
		estimatedDuration = 10
	}

	// Create delivery
	delivery := models.Delivery{
		OrderID:            order.ID,
		DeliveryPartnerID:  partner.ID,
		Status:             models.DeliveryAssigned,
		PickupAddressLine1: order.Chef.AddressLine1,
		PickupAddressCity:  order.Chef.City,
		PickupLatitude:     order.Chef.Latitude,
		PickupLongitude:    order.Chef.Longitude,
		DropoffAddressLine1: order.DeliveryAddressLine1,
		DropoffAddressCity:  order.DeliveryAddressCity,
		DropoffLatitude:     order.DeliveryLatitude,
		DropoffLongitude:    order.DeliveryLongitude,
		Distance:            distance,
		EstimatedDuration:   estimatedDuration,
		DeliveryFee:         order.DeliveryFee,
		Tip:                 order.Tip,
		TotalPayout:         order.DeliveryFee + order.Tip, // 100% to driver — subscription model
	}

	if err := tx.Create(&delivery).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delivery"})
		return
	}

	// Update order with delivery reference
	order.DeliveryID = &delivery.ID
	order.Status = models.OrderStatusPickedUp
	now := time.Now()
	order.PickedUpAt = &now
	order.EstimatedDeliveryTime = estimatedDuration

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	tx.Commit()

	// Publish delivery assigned event
	go func() {
		if err := services.PublishEvent(services.SubjectDeliveryAssigned, "delivery.assigned", partner.UserID, map[string]interface{}{
			"delivery_id":  delivery.ID.String(),
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"partner_id":   partner.ID.String(),
		}); err != nil {
			log.Printf("Failed to publish delivery assigned event: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"delivery": delivery.ToResponse(),
		"message":  "Delivery accepted successfully",
	})
}

// UpdateDeliveryStatus updates the status of a delivery
func (h *DeliveryHandler) UpdateDeliveryStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	deliveryID := c.Param("id")

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	var delivery models.Delivery
	if err := database.DB.Preload("Order").
		Where("id = ? AND delivery_partner_id = ?", deliveryID, partner.ID).
		First(&delivery).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery not found"})
		return
	}

	var req struct {
		Status       models.DeliveryStatus `json:"status" binding:"required"`
		CancelReason string                `json:"cancelReason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status transitions
	validTransitions := map[models.DeliveryStatus][]models.DeliveryStatus{
		models.DeliveryPending:   {models.DeliveryAssigned, models.DeliveryCancelled},
		models.DeliveryAssigned:  {models.DeliveryAtPickup, models.DeliveryPickedUp, models.DeliveryCancelled},
		models.DeliveryAtPickup:  {models.DeliveryPickedUp, models.DeliveryCancelled},
		models.DeliveryPickedUp:  {models.DeliveryInTransit, models.DeliveryCancelled},
		models.DeliveryInTransit: {models.DeliveryAtDropoff, models.DeliveryDelivered, models.DeliveryCancelled},
		models.DeliveryAtDropoff: {models.DeliveryDelivered, models.DeliveryFailed, models.DeliveryCancelled},
		models.DeliveryFailed:    {models.DeliveryReturned},
	}

	allowed, exists := validTransitions[delivery.Status]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery is in a terminal state"})
		return
	}

	valid := false
	for _, s := range allowed {
		if s == req.Status {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status transition"})
		return
	}

	now := time.Now()
	delivery.Status = req.Status

	switch req.Status {
	case models.DeliveryPickedUp:
		delivery.PickedUpAt = &now
		// Update order status
		database.DB.Model(&delivery.Order).Updates(map[string]interface{}{
			"status":     models.OrderStatusDelivering,
			"picked_up_at": now,
		})
	case models.DeliveryInTransit:
		// Order stays in delivering status
	case models.DeliveryDelivered:
		delivery.DeliveredAt = &now
		delivery.ActualDuration = int(now.Sub(delivery.AssignedAt).Minutes())
		// Update order status
		database.DB.Model(&delivery.Order).Updates(map[string]interface{}{
			"status":       models.OrderStatusDelivered,
			"delivered_at": now,
		})
		// Update partner stats
		database.DB.Model(&partner).Updates(map[string]interface{}{
			"total_deliveries": partner.TotalDeliveries + 1,
		})

		// Publish delivery completed event
		go func() {
			services.PublishEvent(services.SubjectDeliveryPickedUp, "delivery.delivered", partner.UserID, map[string]interface{}{
				"delivery_id":  delivery.ID.String(),
				"order_id":     delivery.OrderID.String(),
				"partner_id":   partner.ID.String(),
				"total_payout": delivery.TotalPayout,
			})
		}()

		// Record earnings for subscription billing (driver)
		go func() {
			var sub models.Subscription
			if err := database.DB.Where("user_id = ? AND subscriber_type = ? AND status IN ?",
				partner.UserID, models.SubscriberDriver,
				[]models.SubscriptionStatus{models.SubStatusTrial, models.SubStatusActive}).
				First(&sub).Error; err == nil {
				services.RecordEarning(partner.UserID, sub.ID, models.EarningDeliveryFee, delivery.DeliveryFee, "INR", nil, &delivery.ID)
				if delivery.Tip > 0 {
					services.RecordEarning(partner.UserID, sub.ID, models.EarningTip, delivery.Tip, "INR", nil, &delivery.ID)
				}
				services.CheckEarningsThreshold(sub.ID)
			}
		}()

		// Record chef earnings for subscription billing
		go func() {
			var order models.Order
			if err := database.DB.Preload("Chef").First(&order, delivery.OrderID).Error; err == nil {
				var chefSub models.Subscription
				if err := database.DB.Where("user_id = ? AND subscriber_type = ? AND status IN ?",
					order.Chef.UserID, models.SubscriberChef,
					[]models.SubscriptionStatus{models.SubStatusTrial, models.SubStatusActive}).
					First(&chefSub).Error; err == nil {
					services.RecordEarning(order.Chef.UserID, chefSub.ID, models.EarningOrderRevenue, order.Subtotal, "INR", &order.ID, nil)
					services.CheckEarningsThreshold(chefSub.ID)
				}
			}
		}()

		// Generate order invoice for the customer
		go func() {
			var fullOrder models.Order
			if err := database.DB.Preload("Items").Preload("Chef").Preload("Customer").
				First(&fullOrder, delivery.OrderID).Error; err == nil {
				if _, err := services.GenerateOrderInvoice(&fullOrder); err != nil {
					log.Printf("Failed to generate order invoice for order %s: %v", delivery.OrderID, err)
				}
			}
		}()

	case models.DeliveryCancelled:
		delivery.CancelledAt = &now
		delivery.CancelReason = req.CancelReason
		// Reset order - remove delivery assignment so another driver can pick up
		database.DB.Model(&delivery.Order).Updates(map[string]interface{}{
			"status":      models.OrderStatusReady,
			"delivery_id": nil,
		})
	}

	if err := database.DB.Save(&delivery).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update delivery status"})
		return
	}

	c.JSON(http.StatusOK, delivery.ToResponse())
}

// GetDeliveryHistory returns past deliveries for the partner
func (h *DeliveryHandler) GetDeliveryHistory(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	status := c.Query("status")

	query := database.DB.Where("delivery_partner_id = ?", partner.ID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.Delivery{}).Count(&total)

	var deliveries []models.Delivery
	if err := query.Preload("Order").Preload("Order.Items").Preload("Order.Chef").
		Order("assigned_at DESC").
		Offset(offset).Limit(limit).
		Find(&deliveries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch delivery history"})
		return
	}

	responses := make([]gin.H, len(deliveries))
	for i, d := range deliveries {
		responses[i] = deliveryDetailResponse(&d)
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

// GetEarnings returns earnings breakdown for the delivery partner
func (h *DeliveryHandler) GetEarnings(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	period := c.DefaultQuery("period", "week") // day, week, month, all
	var since time.Time
	now := time.Now()

	switch period {
	case "day":
		since = now.Truncate(24 * time.Hour)
	case "week":
		since = now.AddDate(0, 0, -7)
	case "month":
		since = now.AddDate(0, -1, 0)
	case "all":
		since = time.Time{}
	default:
		since = now.AddDate(0, 0, -7)
	}

	query := database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ?", partner.ID, models.DeliveryDelivered)

	if !since.IsZero() {
		query = query.Where("delivered_at >= ?", since)
	}

	var totalDeliveries int64
	var totalEarnings float64
	var totalTips float64
	var totalDeliveryFees float64

	query.Count(&totalDeliveries)

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ?", partner.ID, models.DeliveryDelivered).
		Where("delivered_at >= ? OR ?", since, since.IsZero()).
		Select("COALESCE(SUM(total_payout), 0)").Scan(&totalEarnings)

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ?", partner.ID, models.DeliveryDelivered).
		Where("delivered_at >= ? OR ?", since, since.IsZero()).
		Select("COALESCE(SUM(tip), 0)").Scan(&totalTips)

	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status = ?", partner.ID, models.DeliveryDelivered).
		Where("delivered_at >= ? OR ?", since, since.IsZero()).
		Select("COALESCE(SUM(delivery_fee), 0)").Scan(&totalDeliveryFees)

	// Daily breakdown for the period
	type DailyEarning struct {
		Date       string  `json:"date"`
		Deliveries int     `json:"deliveries"`
		Earnings   float64 `json:"earnings"`
	}

	var dailyEarnings []DailyEarning
	database.DB.Model(&models.Delivery{}).
		Select("DATE(delivered_at) as date, COUNT(*) as deliveries, COALESCE(SUM(total_payout), 0) as earnings").
		Where("delivery_partner_id = ? AND status = ?", partner.ID, models.DeliveryDelivered).
		Where("delivered_at >= ? OR ?", since, since.IsZero()).
		Group("DATE(delivered_at)").
		Order("date DESC").
		Limit(30).
		Scan(&dailyEarnings)

	c.JSON(http.StatusOK, gin.H{
		"period":          period,
		"totalDeliveries": totalDeliveries,
		"totalEarnings":   totalEarnings,
		"totalTips":       totalTips,
		"deliveryFees":    totalDeliveryFees,
		"avgPerDelivery":  safeDiv(totalEarnings, float64(totalDeliveries)),
		"daily":           dailyEarnings,
	})
}

// Onboarding registers a new delivery partner profile
func (h *DeliveryHandler) Onboarding(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Check if already has a partner profile
	var existingPartner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&existingPartner).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Delivery partner profile already exists"})
		return
	}

	var req struct {
		VehicleType   string `json:"vehicleType" binding:"required"`
		VehicleNumber string `json:"vehicleNumber" binding:"required"`
		LicenseNumber string `json:"licenseNumber" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	partner := models.DeliveryPartner{
		UserID:        userID,
		VehicleType:   req.VehicleType,
		VehicleNumber: req.VehicleNumber,
		LicenseNumber: req.LicenseNumber,
		IsVerified:    false,
		IsActive:      true,
		IsOnline:      false,
	}

	if err := database.DB.Create(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delivery partner profile"})
		return
	}

	// Update user role to delivery
	database.DB.Model(&models.User{}).Where("id = ?", userID).Update("role", models.RoleDelivery)

	c.JSON(http.StatusCreated, partner.ToDetailResponse())
}

// GetOnboardingStatus returns the delivery partner onboarding status
func (h *DeliveryHandler) GetOnboardingStatus(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"completed":  false,
			"status":     "not_started",
			"isVerified": false,
		})
		return
	}

	status := "pending_verification"
	if partner.IsVerified {
		status = "approved"
	}

	c.JSON(http.StatusOK, gin.H{
		"completed":  partner.IsVerified,
		"status":     status,
		"isVerified": partner.IsVerified,
		"profile": gin.H{
			"vehicleType":   partner.VehicleType,
			"vehicleNumber": partner.VehicleNumber,
			"licenseNumber": partner.LicenseNumber,
		},
	})
}

// --- Admin endpoints for delivery management ---

// AdminGetDeliveryPartners returns all delivery partners (admin only)
func (h *DeliveryHandler) AdminGetDeliveryPartners(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	status := c.Query("status")
	search := c.Query("search")

	query := database.DB.Preload("User")

	if status == "verified" {
		query = query.Where("is_verified = ?", true)
	} else if status == "pending" {
		query = query.Where("is_verified = ?", false)
	} else if status == "online" {
		query = query.Where("is_online = ? AND is_verified = ?", true, true)
	}

	if search != "" {
		query = query.Joins("JOIN users ON users.id = delivery_partners.user_id").
			Where("users.first_name ILIKE ? OR users.last_name ILIKE ? OR users.email ILIKE ? OR delivery_partners.vehicle_number ILIKE ?",
				"%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.DeliveryPartner{}).Count(&total)

	var partners []models.DeliveryPartner
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&partners).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch delivery partners"})
		return
	}

	responses := make([]gin.H, len(partners))
	for i, p := range partners {
		responses[i] = gin.H{
			"id":              p.ID,
			"userId":          p.UserID,
			"name":            p.User.FirstName + " " + p.User.LastName,
			"email":           maskEmail(p.User.Email),
			"phone":           maskPhone(p.User.Phone),
			"vehicleType":     p.VehicleType,
			"vehicleNumber":   maskID(p.VehicleNumber),
			"isVerified":      p.IsVerified,
			"isOnline":        p.IsOnline,
			"isActive":        p.IsActive,
			"rating":          p.Rating,
			"totalDeliveries": p.TotalDeliveries,
			"createdAt":       p.CreatedAt,
		}
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

// AdminVerifyPartner verifies a delivery partner
func (h *DeliveryHandler) AdminVerifyPartner(c *gin.Context) {
	partnerID := c.Param("id")

	var partner models.DeliveryPartner
	if err := database.DB.Where("id = ?", partnerID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner not found"})
		return
	}

	adminUserID, _ := middleware.GetUserID(c)

	now := time.Now()
	partner.IsVerified = true
	partner.VerifiedAt = &now
	partner.VerificationStatus = models.VerificationApproved
	partner.VerifiedByID = &adminUserID

	if err := database.DB.Save(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify partner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Delivery partner verified", "id": partner.ID})
}

// AdminSuspendPartner suspends a delivery partner
func (h *DeliveryHandler) AdminSuspendPartner(c *gin.Context) {
	partnerID := c.Param("id")

	var partner models.DeliveryPartner
	if err := database.DB.Where("id = ?", partnerID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner not found"})
		return
	}

	partner.IsActive = false
	partner.IsOnline = false

	if err := database.DB.Save(&partner).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to suspend partner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Delivery partner suspended", "id": partner.ID})
}

// AdminGetDeliveryStats returns platform-wide delivery statistics
func (h *DeliveryHandler) AdminGetDeliveryStats(c *gin.Context) {
	var totalPartners int64
	var verifiedPartners int64
	var onlinePartners int64
	var totalDeliveries int64
	var activeDeliveries int64

	database.DB.Model(&models.DeliveryPartner{}).Count(&totalPartners)
	database.DB.Model(&models.DeliveryPartner{}).Where("is_verified = ?", true).Count(&verifiedPartners)
	database.DB.Model(&models.DeliveryPartner{}).Where("is_online = ? AND is_verified = ?", true, true).Count(&onlinePartners)
	database.DB.Model(&models.Delivery{}).Where("status = ?", models.DeliveryDelivered).Count(&totalDeliveries)
	database.DB.Model(&models.Delivery{}).
		Where("status IN ?", []models.DeliveryStatus{
			models.DeliveryAssigned, models.DeliveryPickedUp, models.DeliveryInTransit,
		}).Count(&activeDeliveries)

	// Today's stats
	today := time.Now().Truncate(24 * time.Hour)
	var todayDeliveries int64
	var todayEarnings float64
	database.DB.Model(&models.Delivery{}).
		Where("status = ? AND delivered_at >= ?", models.DeliveryDelivered, today).
		Count(&todayDeliveries)
	database.DB.Model(&models.Delivery{}).
		Where("status = ? AND delivered_at >= ?", models.DeliveryDelivered, today).
		Select("COALESCE(SUM(total_payout), 0)").Scan(&todayEarnings)

	c.JSON(http.StatusOK, gin.H{
		"totalPartners":    totalPartners,
		"verifiedPartners": verifiedPartners,
		"onlinePartners":   onlinePartners,
		"totalDeliveries":  totalDeliveries,
		"activeDeliveries": activeDeliveries,
		"todayDeliveries":  todayDeliveries,
		"todayEarnings":    todayEarnings,
	})
}

// AdminListDeliveries returns paginated delivery records for admin dashboard
func (h *DeliveryHandler) AdminListDeliveries(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	statusFilter := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Delivery{}).
		Preload("Order").
		Preload("Order.Chef").
		Preload("DeliveryPartner.User")

	if statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var total int64
	query.Count(&total)

	var deliveries []models.Delivery
	query.Order("assigned_at DESC").Offset(offset).Limit(limit).Find(&deliveries)

	responses := make([]gin.H, len(deliveries))
	for i, d := range deliveries {
		resp := gin.H{
			"id":        d.ID,
			"orderId":   d.OrderID,
			"status":    d.Status,
			"distance":  d.Distance,
			"pickup": gin.H{
				"address": d.PickupAddressLine1,
				"city":    d.PickupAddressCity,
			},
			"dropoff": gin.H{
				"address": d.DropoffAddressLine1,
				"city":    d.DropoffAddressCity,
			},
			"deliveryFee":  d.DeliveryFee,
			"tip":          d.Tip,
			"totalPayout":  d.TotalPayout,
			"assignedAt":   d.AssignedAt,
			"pickedUpAt":   d.PickedUpAt,
			"deliveredAt":  d.DeliveredAt,
			"cancelledAt":  d.CancelledAt,
			"cancelReason": d.CancelReason,
		}
		if d.Order.ID != uuid.Nil {
			resp["orderNumber"] = d.Order.OrderNumber
			resp["orderTotal"] = d.Order.Total
			resp["orderStatus"] = d.Order.Status
			if d.Order.Chef.ID != uuid.Nil {
				resp["chefName"] = d.Order.Chef.BusinessName
			}
		}
		if d.DeliveryPartner.ID != uuid.Nil {
			resp["driverName"] = d.DeliveryPartner.User.FirstName + " " + d.DeliveryPartner.User.LastName
			resp["driverPhone"] = maskPhone(d.DeliveryPartner.User.Phone)
			resp["vehicleType"] = d.DeliveryPartner.VehicleType
		}
		responses[i] = resp
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

// --- Helper functions ---

func deliveryDetailResponse(d *models.Delivery) gin.H {
	resp := gin.H{
		"id":                d.ID,
		"orderId":           d.OrderID,
		"deliveryPartnerId": d.DeliveryPartnerID,
		"status":            d.Status,
		"pickup": gin.H{
			"address":   d.PickupAddressLine1,
			"city":      d.PickupAddressCity,
			"latitude":  d.PickupLatitude,
			"longitude": d.PickupLongitude,
		},
		"dropoff": gin.H{
			"address":   d.DropoffAddressLine1,
			"city":      d.DropoffAddressCity,
			"latitude":  d.DropoffLatitude,
			"longitude": d.DropoffLongitude,
		},
		"distance":          d.Distance,
		"estimatedDuration": d.EstimatedDuration,
		"actualDuration":    d.ActualDuration,
		"deliveryFee":       d.DeliveryFee,
		"tip":               d.Tip,
		"totalPayout":       d.TotalPayout,
		"assignedAt":        d.AssignedAt,
		"pickedUpAt":        d.PickedUpAt,
		"deliveredAt":       d.DeliveredAt,
	}

	if d.Order.ID != uuid.Nil {
		resp["order"] = gin.H{
			"id":          d.Order.ID,
			"orderNumber": d.Order.OrderNumber,
			"status":      d.Order.Status,
			"total":       d.Order.Total,
			"itemCount":   len(d.Order.Items),
			"items":       d.Order.Items,
			"specialInstructions":  d.Order.SpecialInstructions,
			"deliveryInstructions": d.Order.DeliveryInstructions,
		}
		if d.Order.Chef.ID != uuid.Nil {
			resp["chef"] = gin.H{
				"name": d.Order.Chef.BusinessName,
			}
		}
	}

	return resp
}

// haversine calculates the distance between two lat/long coordinates in km
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return math.Round(a/b*100) / 100
}

// --- Fleet Management endpoints (for fleet managers via delivery portal) ---

// FleetOverview returns online/offline counts, active deliveries, and unassigned queue depth
func (h *DeliveryHandler) FleetOverview(c *gin.Context) {
	var totalPartners int64
	var onlinePartners int64
	var offlinePartners int64
	var activeDeliveries int64
	var unassignedOrders int64
	var verifiedPartners int64
	var pendingVerification int64

	database.DB.Model(&models.DeliveryPartner{}).Where("is_active = ?", true).Count(&totalPartners)
	database.DB.Model(&models.DeliveryPartner{}).Where("is_online = ? AND is_active = ?", true, true).Count(&onlinePartners)
	database.DB.Model(&models.DeliveryPartner{}).Where("is_online = ? AND is_active = ?", false, true).Count(&offlinePartners)
	database.DB.Model(&models.DeliveryPartner{}).Where("is_verified = ? AND is_active = ?", true, true).Count(&verifiedPartners)
	database.DB.Model(&models.DeliveryPartner{}).Where("verification_status = ?", models.VerificationPending).Count(&pendingVerification)

	database.DB.Model(&models.Delivery{}).
		Where("status IN ?", []models.DeliveryStatus{
			models.DeliveryAssigned, models.DeliveryAtPickup, models.DeliveryPickedUp,
			models.DeliveryInTransit, models.DeliveryAtDropoff,
		}).Count(&activeDeliveries)

	database.DB.Model(&models.Order{}).
		Where("status = ? AND delivery_id IS NULL", models.OrderStatusReady).
		Count(&unassignedOrders)

	// Today's completed
	today := time.Now().Truncate(24 * time.Hour)
	var todayCompleted int64
	var todayEarnings float64
	database.DB.Model(&models.Delivery{}).
		Where("status = ? AND delivered_at >= ?", models.DeliveryDelivered, today).
		Count(&todayCompleted)
	database.DB.Model(&models.Delivery{}).
		Where("status = ? AND delivered_at >= ?", models.DeliveryDelivered, today).
		Select("COALESCE(SUM(total_payout), 0)").Scan(&todayEarnings)

	c.JSON(http.StatusOK, gin.H{
		"totalPartners":       totalPartners,
		"onlinePartners":      onlinePartners,
		"offlinePartners":     offlinePartners,
		"verifiedPartners":    verifiedPartners,
		"pendingVerification": pendingVerification,
		"activeDeliveries":    activeDeliveries,
		"unassignedOrders":    unassignedOrders,
		"todayCompleted":      todayCompleted,
		"todayEarnings":       todayEarnings,
	})
}

// GetPartnerDetail returns full detail for a delivery partner including documents and metrics
func (h *DeliveryHandler) GetPartnerDetail(c *gin.Context) {
	partnerID := c.Param("id")

	var partner models.DeliveryPartner
	if err := database.DB.Preload("User").Preload("Documents").
		Where("id = ?", partnerID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner not found"})
		return
	}

	// Get active delivery count
	var activeCount int64
	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status IN ?", partner.ID,
			[]models.DeliveryStatus{models.DeliveryAssigned, models.DeliveryAtPickup, models.DeliveryPickedUp, models.DeliveryInTransit, models.DeliveryAtDropoff}).
		Count(&activeCount)

	resp := partner.ToDetailResponse()
	c.JSON(http.StatusOK, gin.H{
		"partner":          resp,
		"activeDeliveries": activeCount,
	})
}

// ManualAssignDelivery allows a fleet manager to manually assign an order to a partner
func (h *DeliveryHandler) ManualAssignDelivery(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	partnerID := c.Param("id")

	var req struct {
		OrderID string `json:"orderId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderUUID, err := uuid.Parse(req.OrderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var partner models.DeliveryPartner
	if err := database.DB.Where("id = ?", partnerID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner not found"})
		return
	}

	if !partner.IsVerified || !partner.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Partner is not verified or active"})
		return
	}

	// Check concurrent delivery limit
	var activeCount int64
	database.DB.Model(&models.Delivery{}).
		Where("delivery_partner_id = ? AND status IN ?", partner.ID,
			[]models.DeliveryStatus{models.DeliveryAssigned, models.DeliveryAtPickup, models.DeliveryPickedUp, models.DeliveryInTransit, models.DeliveryAtDropoff}).
		Count(&activeCount)

	if int(activeCount) >= partner.MaxConcurrent {
		c.JSON(http.StatusConflict, gin.H{"error": "Partner has reached maximum concurrent deliveries"})
		return
	}

	tx := database.DB.Begin()

	var order models.Order
	if err := tx.Preload("Chef").
		Where("id = ? AND status = ? AND delivery_id IS NULL", orderUUID, models.OrderStatusReady).
		First(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not available for delivery"})
		return
	}

	distance := haversine(
		order.Chef.Latitude, order.Chef.Longitude,
		order.DeliveryLatitude, order.DeliveryLongitude,
	)
	estimatedDuration := int(math.Ceil(distance / 0.5))
	if estimatedDuration < 10 {
		estimatedDuration = 10
	}

	delivery := models.Delivery{
		OrderID:            order.ID,
		DeliveryPartnerID:  partner.ID,
		Status:             models.DeliveryAssigned,
		AssignmentType:     models.AssignmentManual,
		AssignedByID:       &userID,
		PickupAddressLine1: order.Chef.AddressLine1,
		PickupAddressCity:  order.Chef.City,
		PickupLatitude:     order.Chef.Latitude,
		PickupLongitude:    order.Chef.Longitude,
		DropoffAddressLine1: order.DeliveryAddressLine1,
		DropoffAddressCity:  order.DeliveryAddressCity,
		DropoffLatitude:     order.DeliveryLatitude,
		DropoffLongitude:    order.DeliveryLongitude,
		Distance:            distance,
		EstimatedDuration:   estimatedDuration,
		DeliveryFee:         order.DeliveryFee,
		Tip:                 order.Tip,
		TotalPayout:         order.DeliveryFee + order.Tip, // 100% to driver — subscription model
	}

	if err := tx.Create(&delivery).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create delivery"})
		return
	}

	order.DeliveryID = &delivery.ID
	order.Status = models.OrderStatusPickedUp
	now := time.Now()
	order.PickedUpAt = &now
	order.EstimatedDeliveryTime = estimatedDuration

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	tx.Commit()

	go func() {
		if err := services.PublishEvent(services.SubjectDeliveryAssigned, "delivery.manual_assigned", partner.UserID, map[string]interface{}{
			"delivery_id":  delivery.ID.String(),
			"order_id":     order.ID.String(),
			"partner_id":   partner.ID.String(),
			"assigned_by":  userID.String(),
		}); err != nil {
			log.Printf("Failed to publish manual assign event: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"delivery": delivery.ToResponse(),
		"message":  "Delivery manually assigned",
	})
}

// --- Partner document endpoints ---

// UploadPartnerDocument handles document upload for delivery partner verification
func (h *DeliveryHandler) UploadPartnerDocument(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	docType := models.PartnerDocType(c.PostForm("type"))
	validTypes := map[models.PartnerDocType]bool{
		models.PartnerDocDrivingLicense:      true,
		models.PartnerDocVehicleRC:           true,
		models.PartnerDocInsurance:           true,
		models.PartnerDocAadhaar:             true,
		models.PartnerDocPanCard:             true,
		models.PartnerDocPhoto:               true,
		models.PartnerDocPoliceVerification:  true,
		models.PartnerDocVehicleFront:        true,
		models.PartnerDocVehicleBack:         true,
		models.PartnerDocVehicleLeft:         true,
		models.PartnerDocVehicleRight:        true,
		models.PartnerDocVehicleTop:          true,
		models.PartnerDocVehicleNumberPlate:  true,
	}
	if !validTypes[docType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document type"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}
	defer file.Close()

	isPhoto := docType == models.PartnerDocPhoto || models.IsVehiclePhoto(docType)

	// File size limits: 5MB for profile photo, 10MB for documents
	maxSize := int64(10 * 1024 * 1024) // 10MB for documents
	if isPhoto {
		maxSize = 5 * 1024 * 1024 // 5MB for profile photo
	}
	if header.Size > maxSize {
		if isPhoto {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Profile photo too large (max 5MB)"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 10MB)"})
		}
		return
	}

	contentType := header.Header.Get("Content-Type")

	// Profile photo: JPEG/PNG only. Documents: JPEG/PNG/WebP/PDF
	if isPhoto {
		photoTypes := map[string]bool{"image/jpeg": true, "image/png": true}
		if !photoTypes[contentType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Profile photo must be JPEG or PNG"})
			return
		}
	} else {
		docContentTypes := map[string]bool{
			"image/jpeg":      true,
			"image/png":       true,
			"image/webp":      true,
			"application/pdf": true,
		}
		if !docContentTypes[contentType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Allowed: JPEG, PNG, WebP, PDF"})
			return
		}
	}

	// Upload to storage — photos go to public bucket, verification docs to private
	folder := fmt.Sprintf("delivery-partners/%s/%s", partner.ID, docType)
	var uploadedPath string
	var bucket string

	if docType == models.PartnerDocPhoto {
		bucket = config.AppConfig.GCSPublicBucket
		uploadedPath, err = services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, contentType)
	} else {
		bucket = config.AppConfig.GCSPrivateBucket
		uploadedPath, err = services.UploadPrivateFile(c.Request.Context(), folder, header.Filename, file, contentType)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
		return
	}

	// Upsert: replace existing doc of same type
	var existingDoc models.DeliveryPartnerDocument
	if err := database.DB.Where("partner_id = ? AND type = ?", partner.ID, docType).First(&existingDoc).Error; err == nil {
		existingDoc.FileName = header.Filename
		existingDoc.FilePath = uploadedPath
		existingDoc.Bucket = bucket
		existingDoc.ContentType = contentType
		existingDoc.FileSize = header.Size
		existingDoc.Status = models.DocStatusPending
		existingDoc.RejectionReason = ""
		database.DB.Save(&existingDoc)
		c.JSON(http.StatusOK, existingDoc.ToResponse())
		return
	}

	doc := models.DeliveryPartnerDocument{
		PartnerID:   partner.ID,
		Type:        docType,
		FileName:    header.Filename,
		FilePath:    uploadedPath,
		Bucket:      bucket,
		ContentType: contentType,
		FileSize:    header.Size,
		Status:      models.DocStatusPending,
	}

	if err := database.DB.Create(&doc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document"})
		return
	}

	c.JSON(http.StatusCreated, doc.ToResponse())
}

// GetPartnerDocuments returns the authenticated partner's documents
func (h *DeliveryHandler) GetPartnerDocuments(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var partner models.DeliveryPartner
	if err := database.DB.Where("user_id = ?", userID).First(&partner).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Delivery partner profile not found"})
		return
	}

	var docs []models.DeliveryPartnerDocument
	database.DB.Where("partner_id = ?", partner.ID).Order("created_at DESC").Find(&docs)

	responses := make([]models.PartnerDocumentResponse, len(docs))
	for i, d := range docs {
		responses[i] = d.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"data": responses})
}

// --- Zone CRUD endpoints ---

// ListZones returns all delivery zones
func (h *DeliveryHandler) ListZones(c *gin.Context) {
	city := c.Query("city")
	country := c.Query("country")
	tier := c.Query("tier")
	activeOnly := c.DefaultQuery("active", "true")

	query := database.DB.Model(&models.DeliveryZone{})
	if city != "" {
		query = query.Where("city ILIKE ?", "%"+city+"%")
	}
	if country != "" {
		query = query.Where("country = ?", country)
	}
	if tier != "" {
		query = query.Where("tier = ?", tier)
	}
	if activeOnly == "true" {
		query = query.Where("is_active = ?", true)
	}

	var zones []models.DeliveryZone
	if err := query.Order("country, state, city, name").Find(&zones).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch zones"})
		return
	}

	responses := make([]models.DeliveryZoneResponse, len(zones))
	for i, z := range zones {
		responses[i] = z.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"data": responses})
}

// CreateZone creates a new delivery zone
func (h *DeliveryHandler) CreateZone(c *gin.Context) {
	var req struct {
		Name              string  `json:"name" binding:"required"`
		City              string  `json:"city" binding:"required"`
		State             string  `json:"state"`
		Country           string  `json:"country"`
		Tier              string  `json:"tier"`
		MinLatitude       float64 `json:"minLatitude"`
		MaxLatitude       float64 `json:"maxLatitude"`
		MinLongitude      float64 `json:"minLongitude"`
		MaxLongitude      float64 `json:"maxLongitude"`
		Boundary          string  `json:"boundary"`
		Currency          string  `json:"currency"`
		BaseFare          float64 `json:"baseFare"`
		PerKmRate         float64 `json:"perKmRate"`
		MinimumFare       float64 `json:"minimumFare"`
		TipEnabled        *bool   `json:"tipEnabled"`
		DefaultTipPercent float64 `json:"defaultTipPercent"`
		MaxTipAmount      float64 `json:"maxTipAmount"`
		DriverPayoutPercent float64 `json:"driverPayoutPercent"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	country := req.Country
	if country == "" { country = "IN" }
	currency := req.Currency
	if currency == "" { currency = "INR" }
	tier := req.Tier
	if tier == "" { tier = "standard" }
	tipEnabled := true
	if req.TipEnabled != nil { tipEnabled = *req.TipEnabled }
	defaultTip := req.DefaultTipPercent
	if defaultTip == 0 { defaultTip = 10 }
	driverPayout := req.DriverPayoutPercent
	if driverPayout == 0 { driverPayout = 80 }

	zone := models.DeliveryZone{
		Name:                req.Name,
		City:                req.City,
		State:               req.State,
		Country:             country,
		Tier:                tier,
		MinLatitude:         req.MinLatitude,
		MaxLatitude:         req.MaxLatitude,
		MinLongitude:        req.MinLongitude,
		MaxLongitude:        req.MaxLongitude,
		Boundary:            req.Boundary,
		Currency:            currency,
		BaseFare:            req.BaseFare,
		PerKmRate:           req.PerKmRate,
		MinimumFare:         req.MinimumFare,
		SurgeMultiplier:     1.0,
		TipEnabled:          tipEnabled,
		DefaultTipPercent:   defaultTip,
		MaxTipAmount:        req.MaxTipAmount,
		DriverPayoutPercent: driverPayout,
		IsActive:            true,
	}

	if err := database.DB.Create(&zone).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create zone"})
		return
	}

	c.JSON(http.StatusCreated, zone.ToResponse())
}

// UpdateZone updates a delivery zone
func (h *DeliveryHandler) UpdateZone(c *gin.Context) {
	zoneID := c.Param("id")

	var zone models.DeliveryZone
	if err := database.DB.Where("id = ?", zoneID).First(&zone).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Zone not found"})
		return
	}

	var req struct {
		Name              *string  `json:"name"`
		City              *string  `json:"city"`
		State             *string  `json:"state"`
		Country           *string  `json:"country"`
		Tier              *string  `json:"tier"`
		MinLatitude       *float64 `json:"minLatitude"`
		MaxLatitude       *float64 `json:"maxLatitude"`
		MinLongitude      *float64 `json:"minLongitude"`
		MaxLongitude      *float64 `json:"maxLongitude"`
		Boundary          *string  `json:"boundary"`
		Currency          *string  `json:"currency"`
		BaseFare          *float64 `json:"baseFare"`
		PerKmRate         *float64 `json:"perKmRate"`
		MinimumFare       *float64 `json:"minimumFare"`
		SurgeMultiplier   *float64 `json:"surgeMultiplier"`
		TipEnabled        *bool    `json:"tipEnabled"`
		DefaultTipPercent *float64 `json:"defaultTipPercent"`
		MaxTipAmount      *float64 `json:"maxTipAmount"`
		DriverPayoutPercent *float64 `json:"driverPayoutPercent"`
		IsActive          *bool    `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil { zone.Name = *req.Name }
	if req.City != nil { zone.City = *req.City }
	if req.State != nil { zone.State = *req.State }
	if req.Country != nil { zone.Country = *req.Country }
	if req.Tier != nil { zone.Tier = *req.Tier }
	if req.MinLatitude != nil { zone.MinLatitude = *req.MinLatitude }
	if req.MaxLatitude != nil { zone.MaxLatitude = *req.MaxLatitude }
	if req.MinLongitude != nil { zone.MinLongitude = *req.MinLongitude }
	if req.MaxLongitude != nil { zone.MaxLongitude = *req.MaxLongitude }
	if req.Boundary != nil { zone.Boundary = *req.Boundary }
	if req.Currency != nil { zone.Currency = *req.Currency }
	if req.BaseFare != nil { zone.BaseFare = *req.BaseFare }
	if req.PerKmRate != nil { zone.PerKmRate = *req.PerKmRate }
	if req.MinimumFare != nil { zone.MinimumFare = *req.MinimumFare }
	if req.SurgeMultiplier != nil { zone.SurgeMultiplier = *req.SurgeMultiplier }
	if req.TipEnabled != nil { zone.TipEnabled = *req.TipEnabled }
	if req.DefaultTipPercent != nil { zone.DefaultTipPercent = *req.DefaultTipPercent }
	if req.MaxTipAmount != nil { zone.MaxTipAmount = *req.MaxTipAmount }
	if req.DriverPayoutPercent != nil { zone.DriverPayoutPercent = *req.DriverPayoutPercent }
	if req.IsActive != nil { zone.IsActive = *req.IsActive }

	if err := database.DB.Save(&zone).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update zone"})
		return
	}

	c.JSON(http.StatusOK, zone.ToResponse())
}

// DeleteZone soft-deletes a delivery zone
func (h *DeliveryHandler) DeleteZone(c *gin.Context) {
	zoneID := c.Param("id")

	var zone models.DeliveryZone
	if err := database.DB.Where("id = ?", zoneID).First(&zone).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Zone not found"})
		return
	}

	if err := database.DB.Delete(&zone).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete zone"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Zone deleted"})
}
