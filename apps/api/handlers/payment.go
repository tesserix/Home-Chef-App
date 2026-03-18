package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type PaymentHandler struct{}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{}
}

// CreateOrderPayment creates a Razorpay order with Route transfers for an order.
//
// Payment flow:
//   Customer pays total → Razorpay splits automatically:
//     - Chef gets: Subtotal + ChefTip (food cost + chef tip)
//     - Driver gets: DeliveryFee + DriverTip (delivery fee + driver tip)
//     - Fe3dr gets: ₹0 from orders (revenue comes only from subscriptions)
//
// POST /payments/order/:orderId/create
func (h *PaymentHandler) CreateOrderPayment(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Chef").Preload("Delivery.DeliveryPartner").
		Where("id = ? AND customer_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.PaymentStatus == models.PaymentCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order already paid"})
		return
	}

	// Build Route transfer splits
	totalPaise := services.ToPaise(order.Total)
	transfers := []services.TransferSpec{}

	// Chef transfer: subtotal + tax + chef tip (food amount goes directly to chef)
	chefAmount := order.Subtotal + order.Tax + order.ChefTip
	if order.Chef.RazorpayAccountID != "" && chefAmount > 0 {
		transfers = append(transfers, services.TransferSpec{
			Account:  order.Chef.RazorpayAccountID,
			Amount:   services.ToPaise(chefAmount),
			Currency: "INR",
			Notes: map[string]string{
				"purpose":      "food_payment",
				"order_number": order.OrderNumber,
			},
			OnHold: true, // Hold until order is delivered/confirmed
		})
	}

	// Driver transfer: delivery fee + driver tip
	if order.Delivery != nil && order.Delivery.DeliveryPartner.RazorpayAccountID != "" {
		driverAmount := order.DeliveryFee + order.DriverTip
		if driverAmount > 0 {
			transfers = append(transfers, services.TransferSpec{
				Account:  order.Delivery.DeliveryPartner.RazorpayAccountID,
				Amount:   services.ToPaise(driverAmount),
				Currency: "INR",
				Notes: map[string]string{
					"purpose":      "delivery_payment",
					"order_number": order.OrderNumber,
				},
				OnHold: true, // Hold until delivery is confirmed
			})
		}
	}

	// Create Razorpay order
	rzOrder, err := rz.CreateOrder(&services.OrderRequest{
		Amount:    totalPaise,
		Currency:  "INR",
		Receipt:   order.OrderNumber,
		Transfers: transfers,
		Notes: map[string]string{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"customer_id":  userID.String(),
		},
	})
	if err != nil {
		log.Printf("Failed to create Razorpay order: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}

	// Save Razorpay order ID
	database.DB.Model(&order).Update("razorpay_order_id", rzOrder.ID)

	c.JSON(http.StatusOK, gin.H{
		"razorpayOrderId": rzOrder.ID,
		"razorpayKeyId":   services.GetRazorpay().GetKeyID(),
		"amount":          totalPaise,
		"currency":        "INR",
		"orderNumber":     order.OrderNumber,
		"prefill": gin.H{
			"name":  order.Customer.FirstName + " " + order.Customer.LastName,
			"email": order.Customer.Email,
			"phone": order.Customer.Phone,
		},
	})
}

// VerifyPayment verifies a payment after Razorpay checkout on the client.
// Called by the frontend after successful Razorpay checkout.
// POST /payments/order/:orderId/verify
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		RazorpayPaymentID  string `json:"razorpayPaymentId" binding:"required"`
		RazorpayOrderID    string `json:"razorpayOrderId" binding:"required"`
		RazorpaySignature  string `json:"razorpaySignature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND razorpay_order_id = ?", orderID, req.RazorpayOrderID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Verify payment with Razorpay
	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	payment, err := rz.FetchPayment(req.RazorpayPaymentID)
	if err != nil {
		log.Printf("Failed to fetch payment %s: %v", req.RazorpayPaymentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}

	if payment.Status != "captured" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Payment not captured, status: %s", payment.Status)})
		return
	}

	// Update order payment status
	database.DB.Model(&order).Updates(map[string]interface{}{
		"payment_status":     models.PaymentCompleted,
		"payment_method":     payment.Method,
		"razorpay_payment_id": req.RazorpayPaymentID,
	})

	// Publish event
	if err := services.PublishEvent("orders.paid", "order.paid", order.CustomerID, map[string]interface{}{
		"order_id":     order.ID.String(),
		"order_number": order.OrderNumber,
		"amount":       services.FromPaise(payment.Amount),
		"method":       payment.Method,
	}); err != nil {
		log.Printf("Failed to publish order paid event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment verified", "status": "completed"})
}

// InitiateRefund processes a refund for an order.
// Refunds go back to the customer via Razorpay. Route transfers are auto-reversed.
//
// Refund policies:
//   - Chef initiates refund via their dashboard (chef is responsible for refund decisions)
//   - Admin can force refund in dispute cases
//   - Before pickup: full refund
//   - After pickup/preparing: chef decides partial or full refund
//   - After delivery: no automatic refund (handled as dispute)
//
// POST /payments/order/:orderId/refund
func (h *PaymentHandler) InitiateRefund(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		Amount float64 `json:"amount"` // 0 = full refund
		Reason string  `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Chef").Where("id = ?", orderID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Determine who is initiating the refund
	initiatedBy := "system"
	// Check if the requester is the chef
	if order.Chef.UserID == userID {
		initiatedBy = "chef"
	} else {
		// Check if admin
		var user models.User
		if err := database.DB.First(&user, "id = ?", userID).Error; err == nil {
			if user.Role == models.RoleAdmin {
				initiatedBy = "admin"
			}
		}
	}

	if initiatedBy == "system" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the chef or admin can initiate refunds"})
		return
	}

	if order.PaymentStatus != models.PaymentCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only refund completed payments"})
		return
	}

	if order.RazorpayPaymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No payment found for this order"})
		return
	}

	// Validate refund amount
	refundAmount := req.Amount
	if refundAmount == 0 {
		refundAmount = order.Total // Full refund
	}
	if refundAmount > order.Total {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refund amount cannot exceed order total"})
		return
	}

	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	// Create Razorpay refund — this automatically reverses Route transfers
	rzRefund, err := rz.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
		Amount: services.ToPaise(refundAmount),
		Speed:  "normal",
		Notes: map[string]string{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"reason":       req.Reason,
			"initiated_by": initiatedBy,
		},
		Receipt: fmt.Sprintf("refund-%s", order.OrderNumber),
	})
	if err != nil {
		log.Printf("Failed to create refund for order %s: %v", order.OrderNumber, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
		return
	}

	now := time.Now()
	database.DB.Model(&order).Updates(map[string]interface{}{
		"payment_status":    models.PaymentRefunded,
		"status":            models.OrderStatusRefunded,
		"refund_id":         rzRefund.ID,
		"refund_amount":     refundAmount,
		"refund_reason":     req.Reason,
		"refund_initiated_by": initiatedBy,
		"refunded_at":       &now,
	})

	// Publish event
	if err := services.PublishEvent("orders.refunded", "order.refunded", userID, map[string]interface{}{
		"order_id":      order.ID.String(),
		"order_number":  order.OrderNumber,
		"refund_amount": refundAmount,
		"reason":        req.Reason,
		"initiated_by":  initiatedBy,
		"refund_id":     rzRefund.ID,
	}); err != nil {
		log.Printf("Failed to publish order refunded event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Refund initiated",
		"refundId":     rzRefund.ID,
		"refundAmount": refundAmount,
		"status":       rzRefund.Status,
	})
}

// RazorpayWebhook handles Razorpay webhook events.
// POST /webhooks/razorpay (no auth — verified via HMAC signature)
func (h *PaymentHandler) RazorpayWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	// Verify webhook signature
	signature := c.GetHeader("X-Razorpay-Signature")
	if !services.VerifyWebhookSignature(body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var event struct {
		Event   string          `json:"event"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	log.Printf("Razorpay webhook received: %s", event.Event)

	switch event.Event {
	case "payment.captured":
		h.handlePaymentCaptured(event.Payload)
	case "payment.failed":
		h.handlePaymentFailed(event.Payload)
	case "refund.processed":
		h.handleRefundProcessed(event.Payload)
	case "transfer.processed":
		log.Printf("Transfer processed event received")
	case "subscription.charged":
		h.handleSubscriptionCharged(event.Payload)
	case "subscription.halted":
		h.handleSubscriptionHalted(event.Payload)
	default:
		log.Printf("Unhandled Razorpay webhook event: %s", event.Event)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *PaymentHandler) handlePaymentCaptured(payload json.RawMessage) {
	var data struct {
		Payment struct {
			Entity services.PaymentResponse `json:"entity"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse payment.captured payload: %v", err)
		return
	}

	payment := data.Payment.Entity
	log.Printf("Payment captured: %s (order: %s, amount: %d)", payment.ID, payment.OrderID, payment.Amount)

	// Update order
	database.DB.Model(&models.Order{}).
		Where("razorpay_order_id = ?", payment.OrderID).
		Updates(map[string]interface{}{
			"payment_status":      models.PaymentCompleted,
			"payment_method":      payment.Method,
			"razorpay_payment_id": payment.ID,
		})
}

func (h *PaymentHandler) handlePaymentFailed(payload json.RawMessage) {
	var data struct {
		Payment struct {
			Entity services.PaymentResponse `json:"entity"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse payment.failed payload: %v", err)
		return
	}

	payment := data.Payment.Entity
	log.Printf("Payment failed: %s (order: %s)", payment.ID, payment.OrderID)

	database.DB.Model(&models.Order{}).
		Where("razorpay_order_id = ?", payment.OrderID).
		Update("payment_status", models.PaymentFailed)
}

func (h *PaymentHandler) handleRefundProcessed(payload json.RawMessage) {
	var data struct {
		Refund struct {
			Entity services.RefundResponse `json:"entity"`
		} `json:"refund"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse refund.processed payload: %v", err)
		return
	}

	refund := data.Refund.Entity
	log.Printf("Refund processed: %s (payment: %s, amount: %d)", refund.ID, refund.PaymentID, refund.Amount)

	// Update refund status
	now := time.Now()
	database.DB.Model(&models.Order{}).
		Where("razorpay_payment_id = ?", refund.PaymentID).
		Updates(map[string]interface{}{
			"refund_id":   refund.ID,
			"refunded_at": &now,
		})
}

func (h *PaymentHandler) handleSubscriptionCharged(payload json.RawMessage) {
	log.Printf("Subscription charged webhook received")
	// Update subscription invoice status
	var data struct {
		Subscription struct {
			Entity struct {
				ID string `json:"id"`
			} `json:"entity"`
		} `json:"subscription"`
		Payment struct {
			Entity struct {
				ID string `json:"id"`
			} `json:"entity"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse subscription.charged: %v", err)
		return
	}

	// Mark latest pending invoice as paid
	now := time.Now()
	database.DB.Model(&models.SubscriptionInvoice{}).
		Where("gateway_payment_id = '' AND status = ?", models.InvoicePending).
		Joins("JOIN subscriptions ON subscriptions.id = subscription_invoices.subscription_id AND subscriptions.gateway_sub_id = ?", data.Subscription.Entity.ID).
		Updates(map[string]interface{}{
			"status":             models.InvoicePaid,
			"gateway_payment_id": data.Payment.Entity.ID,
			"paid_at":            &now,
		})
}

func (h *PaymentHandler) handleSubscriptionHalted(payload json.RawMessage) {
	log.Printf("Subscription halted webhook received")
	var data struct {
		Subscription struct {
			Entity struct {
				ID string `json:"id"`
			} `json:"entity"`
		} `json:"subscription"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse subscription.halted: %v", err)
		return
	}

	// Mark subscription as suspended
	database.DB.Model(&models.Subscription{}).
		Where("gateway_sub_id = ?", data.Subscription.Entity.ID).
		Update("status", models.SubStatusSuspended)
}

