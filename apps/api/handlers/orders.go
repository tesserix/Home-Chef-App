package handlers

import (
	"fmt"
	"log"
	"math/rand"
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

type OrderHandler struct{}

func NewOrderHandler() *OrderHandler {
	return &OrderHandler{}
}

// CreateOrderRequest represents the order creation payload
type CreateOrderRequest struct {
	ChefID               uuid.UUID            `json:"chefId" binding:"required"`
	Items                []CreateOrderItem    `json:"items" binding:"required,min=1"`
	DeliveryAddressID    *uuid.UUID           `json:"deliveryAddressId"`
	DeliveryAddress      *CreateAddressRequest `json:"deliveryAddress"`
	DeliveryInstructions string               `json:"deliveryInstructions"`
	SpecialInstructions  string               `json:"specialInstructions"`
	Tip                  float64              `json:"tip"`
	PromoCode            string               `json:"promoCode"`
	PaymentMethodID      *uuid.UUID           `json:"paymentMethodId"`
	ScheduledFor         *time.Time           `json:"scheduledFor"`
}

type CreateOrderItem struct {
	MenuItemID uuid.UUID `json:"menuItemId" binding:"required"`
	Quantity   int       `json:"quantity" binding:"required,min=1"`
	Notes      string    `json:"notes"`
}

type CreateAddressRequest struct {
	Line1      string  `json:"line1" binding:"required"`
	Line2      string  `json:"line2"`
	City       string  `json:"city" binding:"required"`
	State      string  `json:"state" binding:"required"`
	PostalCode string  `json:"postalCode" binding:"required"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

// CreateOrder creates a new order
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify chef exists and is active
	var chef models.ChefProfile
	if err := database.DB.Where("id = ? AND is_active = ? AND is_verified = ?", req.ChefID, true, true).
		First(&chef).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chef not found or not available"})
		return
	}

	if !chef.AcceptingOrders {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chef is not accepting orders"})
		return
	}

	// Get menu items and calculate totals
	var subtotal float64
	orderItems := make([]models.OrderItem, len(req.Items))

	for i, item := range req.Items {
		var menuItem models.MenuItem
		if err := database.DB.Where("id = ? AND chef_id = ? AND is_available = ?",
			item.MenuItemID, req.ChefID, true).First(&menuItem).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Menu item %s not found or unavailable", item.MenuItemID)})
			return
		}

		itemSubtotal := menuItem.Price * float64(item.Quantity)
		subtotal += itemSubtotal

		orderItems[i] = models.OrderItem{
			MenuItemID: item.MenuItemID,
			Name:       menuItem.Name,
			Price:      menuItem.Price,
			Quantity:   item.Quantity,
			Subtotal:   itemSubtotal,
			Notes:      item.Notes,
		}
	}

	// Check minimum order
	if subtotal < chef.MinimumOrder {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":        fmt.Sprintf("Minimum order is $%.2f", chef.MinimumOrder),
			"minimumOrder": chef.MinimumOrder,
		})
		return
	}

	// Calculate fees
	deliveryFee := 2.99 // Base fee, could be calculated based on distance
	serviceFee := subtotal * 0.10 // 10% service fee
	tax := subtotal * 0.08 // 8% tax
	tip := req.Tip
	discount := 0.0

	// TODO: Apply promo code discount

	total := subtotal + deliveryFee + serviceFee + tax + tip - discount

	// Get or create delivery address
	var deliveryAddr CreateAddressRequest
	if req.DeliveryAddressID != nil {
		var addr models.Address
		if err := database.DB.Where("id = ? AND user_id = ?", *req.DeliveryAddressID, userID).
			First(&addr).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery address not found"})
			return
		}
		deliveryAddr = CreateAddressRequest{
			Line1:      addr.Line1,
			Line2:      addr.Line2,
			City:       addr.City,
			State:      addr.State,
			PostalCode: addr.PostalCode,
			Latitude:   addr.Latitude,
			Longitude:  addr.Longitude,
		}
	} else if req.DeliveryAddress != nil {
		deliveryAddr = *req.DeliveryAddress
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery address required"})
		return
	}

	// Generate order number
	orderNumber := generateOrderNumber()

	// Create order
	order := models.Order{
		OrderNumber:               orderNumber,
		CustomerID:                userID,
		ChefID:                    chef.ID,
		Status:                    models.OrderStatusPending,
		PaymentStatus:             models.PaymentPending,
		Subtotal:                  subtotal,
		DeliveryFee:               deliveryFee,
		ServiceFee:                serviceFee,
		Tax:                       tax,
		Tip:                       tip,
		Discount:                  discount,
		Total:                     total,
		PromoCode:                 req.PromoCode,
		DeliveryAddressLine1:      deliveryAddr.Line1,
		DeliveryAddressLine2:      deliveryAddr.Line2,
		DeliveryAddressCity:       deliveryAddr.City,
		DeliveryAddressState:      deliveryAddr.State,
		DeliveryAddressPostalCode: deliveryAddr.PostalCode,
		DeliveryLatitude:          deliveryAddr.Latitude,
		DeliveryLongitude:         deliveryAddr.Longitude,
		DeliveryInstructions:      req.DeliveryInstructions,
		SpecialInstructions:       req.SpecialInstructions,
		ScheduledFor:              req.ScheduledFor,
		EstimatedPrepTime:         30, // Default, could be calculated
	}

	// Start transaction
	tx := database.DB.Begin()

	if err := tx.Create(&order).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// Add order items
	for i := range orderItems {
		orderItems[i].OrderID = order.ID
	}

	if err := tx.Create(&orderItems).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order items"})
		return
	}

	// Clear user's cart for this chef
	tx.Where("user_id = ? AND chef_id = ?", userID, chef.ID).Delete(&models.Cart{})

	tx.Commit()

	// Load the created order with items
	database.DB.Preload("Items").First(&order, order.ID)

	// Publish order created event
	go func() {
		orderEvent := services.OrderEvent{
			OrderID:     order.ID,
			OrderNumber: order.OrderNumber,
			CustomerID:  order.CustomerID,
			ChefID:      order.ChefID,
			Status:      string(order.Status),
			Total:       order.Total,
		}
		if err := services.PublishOrderEvent(services.SubjectOrderCreated, orderEvent); err != nil {
			log.Printf("Failed to publish order created event: %v", err)
		}
		// Also notify the chef
		if err := services.PublishOrderEvent(services.SubjectChefNewOrder, orderEvent); err != nil {
			log.Printf("Failed to publish chef new order event: %v", err)
		}
		// Record metrics
		middleware.RecordOrder(string(order.Status), order.Total)
	}()

	c.JSON(http.StatusCreated, order.ToResponse())
}

// GetOrders returns the user's orders
func (h *OrderHandler) GetOrders(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset := (page - 1) * limit

	query := database.DB.Where("customer_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Model(&models.Order{}).Count(&total)

	var orders []models.Order
	if err := query.Preload("Items").Preload("Chef").
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
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// GetOrder returns a single order
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Items").Preload("Chef").Preload("Delivery").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order.ToResponse())
}

// CancelOrder cancels an order
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Check if order can be cancelled
	if order.Status != models.OrderStatusPending && order.Status != models.OrderStatusAccepted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be cancelled at this stage"})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	now := time.Now()
	order.Status = models.OrderStatusCancelled
	order.CancelledAt = &now
	order.CancelReason = req.Reason

	if err := database.DB.Save(&order).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}

	// Publish order cancelled event
	go func() {
		orderEvent := services.OrderEvent{
			OrderID:     order.ID,
			OrderNumber: order.OrderNumber,
			CustomerID:  order.CustomerID,
			ChefID:      order.ChefID,
			Status:      string(order.Status),
			Total:       order.Total,
		}
		if err := services.PublishOrderEvent(services.SubjectOrderCancelled, orderEvent); err != nil {
			log.Printf("Failed to publish order cancelled event: %v", err)
		}
	}()

	c.JSON(http.StatusOK, order.ToResponse())
}

// TrackOrder returns real-time tracking info for an order
func (h *OrderHandler) TrackOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Delivery").Preload("Chef").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	response := gin.H{
		"orderId":     order.ID,
		"orderNumber": order.OrderNumber,
		"status":      order.Status,
		"chef": gin.H{
			"name": order.Chef.BusinessName,
		},
		"estimatedPrepTime":     order.EstimatedPrepTime,
		"estimatedDeliveryTime": order.EstimatedDeliveryTime,
		"createdAt":             order.CreatedAt,
		"acceptedAt":            order.AcceptedAt,
		"preparedAt":            order.PreparedAt,
		"pickedUpAt":            order.PickedUpAt,
		"deliveredAt":           order.DeliveredAt,
	}

	if order.Delivery != nil {
		response["delivery"] = order.Delivery.ToResponse()
	}

	c.JSON(http.StatusOK, response)
}

// GetOrderInvoice returns the invoice for an order, generating it on the fly if needed
func (h *OrderHandler) GetOrderInvoice(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	// Verify the order belongs to this user
	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderUUID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	invoice, err := services.GetOrderInvoiceByOrderID(orderUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, invoice.ToResponse())
}

// Helper to generate order number
func generateOrderNumber() string {
	timestamp := time.Now().Format("0601021504")
	random := rand.Intn(9999)
	return fmt.Sprintf("HC%s%04d", timestamp, random)
}
