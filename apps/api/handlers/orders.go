package handlers

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	natsclient "github.com/nats-io/nats.go"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// wsUpgrader upgrades HTTP connections to WebSocket.
//
// Browsers always send Origin on WS upgrade; mobile/native clients omit it.
// We accept the request when EITHER:
//   - Origin is empty (native client), OR
//   - Origin matches the API CORS allowlist (browser).
// Anything else is rejected to prevent cross-site WS hijacking.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin:     allowedWSOrigin,
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func allowedWSOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // native / curl / non-browser
	}
	host := r.Host
	// Same-origin (proxied via Cloudflare/Istio) is always fine.
	if strings.EqualFold(origin, "https://"+host) || strings.EqualFold(origin, "http://"+host) {
		return true
	}
	// Trusted production + dev origins. Keep in sync with routes.allowedOrigins().
	for _, allowed := range []string{
		"https://fe3dr.com",
		"https://www.fe3dr.com",
		"https://vendors.fe3dr.com",
		"https://admin.fe3dr.com",
		"https://auth.fe3dr.com",
		"https://delivery.fe3dr.com",
		"http://localhost:5173",
		"http://localhost:3000",
	} {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}

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
	// ISO-3166 alpha-2. Optional on the wire — drives per-country tax
	// lookup, so the frontend should pass it when known. Falls back to
	// the saved address's country or India when absent.
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// CreateOrder creates a new order
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Platform operating-hours gate — admins can set open/close times in
	// Settings → Platform, and IsPlatformOpen evaluates them against the
	// configured timezone. Unconfigured = always open (current behavior).
	if open, msg := services.IsPlatformOpen(); !open {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// Verify chef exists and is active
	var chef models.ChefProfile
	if err := database.DB.Where("id = ? AND is_active = ?", req.ChefID, true).
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

	// Calculate fees from the platform policy (Settings → Platform).
	// Defaults match the prior hardcoded values (10% service, 8% tax, $2.99
	// delivery) so behavior doesn't change until an admin edits the policy.
	// deliveryFee starts at the flat policy fee and is replaced below with a
	// live 3PL quote once the delivery address (and its coords) is resolved.
	policy := services.GetPlatformPolicy()
	deliveryFee := policy.BaseDeliveryFee
	serviceFee := subtotal * (policy.ServiceFeePercent / 100.0)

	// Tax is resolved per customer country (and state/region when known)
	// from the tax_rates table rather than the single global TaxPercent.
	// GST for India, VAT for EU, HST for Ontario, etc. — all picked by
	// the delivery address the customer chose.
	tip := req.Tip
	discount := 0.0

	// Apply promo code discount
	var appliedPromo *models.PromoCode
	if req.PromoCode != "" {
		promoDiscount, promo, promoErr := validateAndCalculateDiscount(req.PromoCode, userID, subtotal)
		if promoErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": promoErr})
			return
		}
		discount = promoDiscount
		appliedPromo = promo
	}

	// Total is computed below, after tax is resolved from the delivery
	// address. Until then we only have the pre-tax pieces.

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
			Country:    addr.Country,
			Latitude:   addr.Latitude,
			Longitude:  addr.Longitude,
		}
	} else if req.DeliveryAddress != nil {
		deliveryAddr = *req.DeliveryAddress
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Delivery address required"})
		return
	}

	// Country defaults to India if the address was created before the
	// column was populated; tax resolver falls back to a zero-rate rule
	// so the order still goes through.
	deliveryCountry := deliveryAddr.Country
	if deliveryCountry == "" {
		deliveryCountry = "IN"
	}

	// Live 3PL delivery quote, computed now that the drop address is known.
	// Falls back to the flat policy fee (already in deliveryFee) when the
	// address has no coordinates yet or no provider can serve the leg, so
	// checkout never blocks on the quote.
	if fee, ok := services.QuoteCheckoutDeliveryFee(chef, deliveryAddr.City, deliveryCountry, deliveryAddr.Latitude, deliveryAddr.Longitude); ok {
		deliveryFee = fee
	}

	taxRule := services.ResolveTaxRate(deliveryCountry, deliveryAddr.State)
	// Tax base: subtotal + deliveryFee + serviceFee, after promo discount.
	// Tip is excluded (direct pass-through to chef/driver).
	taxBase := subtotal + deliveryFee + serviceFee - discount
	if taxBase < 0 {
		taxBase = 0
	}
	var tax float64
	if taxRule.Inclusive {
		// Inclusive rate: the base already contains the tax; back it out
		// for line-item display but don't add it to total.
		tax = taxBase - (taxBase / (1 + taxRule.Rate/100.0))
	} else {
		tax = taxBase * (taxRule.Rate / 100.0)
	}

	// Order total. For inclusive-tax jurisdictions (EU VAT etc.) the tax
	// line is informational — it's part of taxBase already, so we don't
	// add it again or the customer would be charged twice.
	var total float64
	if taxRule.Inclusive {
		total = subtotal + deliveryFee + serviceFee + tip - discount
	} else {
		total = subtotal + deliveryFee + serviceFee + tax + tip - discount
	}

	// If the admin has configured delivery zones, enforce coverage. When no
	// zones exist we skip — feature is opt-in so existing customers aren't
	// suddenly unable to order.
	if services.HasActiveZones() {
		if deliveryAddr.Latitude == 0 && deliveryAddr.Longitude == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Delivery address needs location coordinates. Please select your address from the map.",
			})
			return
		}
		if zone := services.FindZoneForAddress(deliveryAddr.Latitude, deliveryAddr.Longitude); zone == nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "We don't deliver to this address yet. Please pick a supported area.",
			})
			return
		}
	}

	// Generate order number
	orderNumber := generateOrderNumber()

	// Inherit the chef's current gateway so VerifyPayment / refund later
	// read the same provider this order was created against. Falls back to
	// razorpay for older chefs without the column populated.
	orderProvider := chef.PaymentProvider
	if orderProvider == "" {
		orderProvider = "razorpay"
	}

	// Order currency derives from the chef's settlement country so Stripe
	// charges and chef payouts settle in a consistent currency. Defaults
	// to INR for legacy chefs with empty PayoutCountry.
	orderCurrency := strings.ToUpper(services.CurrencyForCountry(chef.PayoutCountry))

	// Create order
	order := models.Order{
		OrderNumber:               orderNumber,
		CustomerID:                userID,
		ChefID:                    chef.ID,
		PaymentProvider:           orderProvider,
		Currency:                  orderCurrency,
		Status:                    models.OrderStatusPending,
		PaymentStatus:             models.PaymentPending,
		Subtotal:                  subtotal,
		DeliveryFee:               deliveryFee,
		ServiceFee:                serviceFee,
		Tax:                       tax,
		TaxRate:                   taxRule.Rate,
		TaxName:                   taxRule.TaxName,
		Tip:                       tip,
		Discount:                  discount,
		Total:                     total,
		PromoCode:                 req.PromoCode,
		DeliveryAddressLine1:      deliveryAddr.Line1,
		DeliveryAddressLine2:      deliveryAddr.Line2,
		DeliveryAddressCity:       deliveryAddr.City,
		DeliveryAddressState:      deliveryAddr.State,
		DeliveryAddressPostalCode: deliveryAddr.PostalCode,
		DeliveryAddressCountry:    deliveryCountry,
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

	// Record promo code usage
	if appliedPromo != nil {
		usage := models.PromoCodeUsage{
			PromoCodeID: appliedPromo.ID,
			UserID:      userID,
			OrderID:     order.ID,
			Discount:    discount,
		}
		if err := tx.Create(&usage).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record promo usage"})
			return
		}
		// Increment usage count
		tx.Model(&models.PromoCode{}).Where("id = ?", appliedPromo.ID).
			Update("usage_count", appliedPromo.UsageCount+1)
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

	// Cancel any booked 3PL delivery (no-op if none exists yet). Off the
	// response path; failure must not fail the order cancellation.
	go func() {
		if err := services.CancelOrderDelivery(order.ID, req.Reason); err != nil {
			log.Printf("Failed to cancel 3PL delivery for order %s: %v", order.ID, err)
			services.CaptureBackgroundError(err)
		}
	}()

	c.JSON(http.StatusOK, order.ToResponse())
}

// TrackOrder returns real-time tracking info for an order
func (h *OrderHandler) TrackOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	var order models.Order
	if err := database.DB.Preload("Delivery").Preload("Delivery.DeliveryPartner").Preload("Chef").
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

// GetOrderInvoicePDF streams the PDF tax invoice for a delivered order.
// Customer-scoped (only the buyer can pull their own invoice).
// GET /api/v1/orders/:id/invoice.pdf
func (h *OrderHandler) GetOrderInvoicePDF(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("id")

	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderUUID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.Status != models.OrderStatusDelivered {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invoice available after delivery"})
		return
	}

	pdfBytes, filename, err := services.GenerateOrderInvoicePDF(orderUUID)
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invoice"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// TrackOrderWS upgrades the HTTP connection to WebSocket and streams real-time driver location
// updates from NATS delivery.location.{deliveryID} to the connected client.
// Security: verifies the authenticated customer owns the order before upgrading (T-04-03).
func (h *OrderHandler) TrackOrderWS(c *gin.Context) {
	orderID := c.Param("id")
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Verify customer owns this order and load the delivery relationship.
	var order models.Order
	if err := database.DB.Preload("Delivery").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order_not_found", "message": "Order not found"})
		return
	}
	if order.Delivery == nil || order.Delivery.ID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no_active_delivery", "message": "No active delivery for this order"})
		return
	}
	deliveryID := order.Delivery.ID.String()

	// Upgrade HTTP → WebSocket.
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed for order %s: %v", orderID, err)
		return // Upgrader writes the error response automatically.
	}
	defer conn.Close()

	// Buffered write channel — only one goroutine calls conn.WriteMessage (gorilla is not concurrent-write-safe).
	// T-04-05: non-blocking send drops messages when full rather than blocking.
	writeCh := make(chan []byte, 32)
	defer close(writeCh)

	// Write pump — serialises all WebSocket writes.
	go func() {
		for msg := range writeCh {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("WS write error for order %s: %v", orderID, err)
				return
			}
		}
	}()

	// Subscribe to core NATS (not JetStream) for live location fan-out.
	subject := fmt.Sprintf("%s.%s", services.SubjectDeliveryLocation, deliveryID)
	sub, err := services.GetNATSClient().Subscribe(subject, func(msg *natsclient.Msg) {
		select {
		case writeCh <- msg.Data:
		default:
			// Channel full — drop this message rather than block the NATS dispatcher.
		}
	})
	if err != nil {
		log.Printf("NATS subscribe failed for delivery %s: %v", deliveryID, err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Internal error"))
		return
	}
	defer sub.Unsubscribe()

	// Read pump — blocks until client disconnects or sends a close frame.
	// T-04-05: SetReadLimit caps inbound frame size at 512 bytes.
	conn.SetReadLimit(512)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// Helper to generate order number
func generateOrderNumber() string {
	timestamp := time.Now().Format("0601021504")
	random := rand.Intn(9999)
	return fmt.Sprintf("HC%s%04d", timestamp, random)
}
