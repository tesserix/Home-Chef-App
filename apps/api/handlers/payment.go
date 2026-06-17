package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// Currency resolution lives in services.CurrencyForCountry; services.ToMinor
// handles the currency-aware major→minor unit conversion (2 decimals for
// most, 0 for JPY/KRW/VND, 3 for KWD/BHD/OMR). Keeping those in one place
// prevents the "÷100 vs ÷1000 vs ÷1" bug from drifting across handlers.

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

	var order models.Order
	if err := database.DB.Preload("Chef").Preload("Customer").Preload("Delivery.DeliveryPartner").
		Where("id = ? AND customer_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if order.PaymentStatus == models.PaymentCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order already paid"})
		return
	}

	// Pick gateway from chef's configured provider. Falls back to razorpay
	// for older chef profiles that don't have the column populated yet.
	provider := strings.ToLower(order.Chef.PaymentProvider)
	if provider == "" {
		provider = "razorpay"
	}

	switch provider {
	case "stripe":
		h.createStripePayment(c, &order, userID)
	default:
		h.createRazorpayPayment(c, &order, userID)
	}
}

// createRazorpayPayment is the original INR + Razorpay Route flow. Unchanged
// from the pre-multi-gateway implementation except that the order's
// payment_provider column is now stamped so VerifyPayment / InitiateRefund
// know which code path to run later.
func (h *PaymentHandler) createRazorpayPayment(c *gin.Context, order *models.Order, userID uuid.UUID) {
	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	// Build Route transfer splits
	totalPaise := services.ToPaise(order.Total)
	transfers := []services.TransferSpec{}

	// Chef transfer: subtotal + tax + chef tip (food amount goes directly to chef)
	chefAmount := order.Subtotal + order.Tax + order.ChefTip
	// FSSAI hard lockout (#32): withhold the chef's payout when their food-safety
	// licence has lapsed. The customer is still charged and the platform/driver
	// are settled; the chef's split stays in the platform account until a verified
	// renewal lifts the lock. Defense-in-depth — CreateOrder already blocks new
	// orders for expired chefs, but any path that reaches payment is covered here.
	chefFSSAIExpired := services.IsChefFSSAIExpired(&order.Chef)
	if chefFSSAIExpired {
		middleware.RecordFSSAILockout("payout_withheld")
		log.Printf("fssai-lockout: withholding chef payout order=%s chef=%s amount=%.2f",
			order.OrderNumber, order.Chef.ID, chefAmount)
	}
	if order.Chef.RazorpayAccountID != "" && chefAmount > 0 && !chefFSSAIExpired {
		transfers = append(transfers, services.TransferSpec{
			Account:  order.Chef.RazorpayAccountID,
			Amount:   services.ToPaise(chefAmount),
			Currency: "INR",
			Notes: map[string]string{
				"purpose":      "food_payment",
				"order_number": order.OrderNumber,
			},
			OnHold: true,
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
				OnHold: true,
			})
		}
	}

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

	database.DB.Model(order).Updates(map[string]interface{}{
		"razorpay_order_id": rzOrder.ID,
		"payment_provider":  "razorpay",
	})

	c.JSON(http.StatusOK, gin.H{
		"provider":        "razorpay",
		"razorpayOrderId": rzOrder.ID,
		"razorpayKeyId":   rz.GetKeyID(),
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

// createStripePayment creates a PaymentIntent against the chef's Connect
// account. Chef receives (subtotal + tax + chefTip) via
// `transfer_data[destination]`; the platform retains (deliveryFee +
// driverTip) as `application_fee_amount` and settles the driver separately.
// Stripe rejects charges whose currency doesn't match the chef's Connect
// country, so we derive currency from PayoutCountry rather than hardcoding.
func (h *PaymentHandler) createStripePayment(c *gin.Context, order *models.Order, userID uuid.UUID) {
	st := services.GetStripe()
	if st == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe gateway not configured"})
		return
	}

	if order.Chef.StripeAccountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chef has not completed Stripe onboarding"})
		return
	}
	if !order.Chef.StripeChargesEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chef's Stripe account is not ready to accept charges — onboarding may be incomplete"})
		return
	}

	// Order.Currency was stamped at creation from the chef's PayoutCountry.
	// Using that frozen value (instead of rederiving from chef) means a
	// mid-flight chef profile edit doesn't change the currency of an
	// already-placed order.
	currency := strings.ToLower(order.Currency)
	if currency == "" {
		currency = services.CurrencyForCountry(order.Chef.PayoutCountry)
	}
	totalMinor := services.ToMinor(order.Total, currency)

	// Chef receives: subtotal + tax + chefTip. Platform keeps the rest
	// (deliveryFee + driverTip) as application_fee; driver gets paid out of
	// platform balance via a follow-up Transfer on delivery confirmation.
	chefAmount := order.Subtotal + order.Tax + order.ChefTip
	chefMinor := services.ToMinor(chefAmount, currency)
	applicationFee := totalMinor - chefMinor
	if applicationFee < 0 {
		applicationFee = 0
	}

	pi, err := st.CreatePaymentIntent(&services.StripePaymentIntentRequest{
		Amount:              totalMinor,
		Currency:            currency,
		ReceiptEmail:        order.Customer.Email,
		DestinationAccount:  order.Chef.StripeAccountID,
		ApplicationFeeCents: applicationFee,
		Description:         fmt.Sprintf("HomeChef order %s", order.OrderNumber),
		Metadata: map[string]string{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"customer_id":  userID.String(),
			"chef_id":      order.ChefID.String(),
		},
	})
	if err != nil {
		log.Printf("Failed to create Stripe PaymentIntent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}

	database.DB.Model(order).Updates(map[string]interface{}{
		"stripe_payment_intent_id": pi.ID,
		"payment_provider":         "stripe",
	})

	c.JSON(http.StatusOK, gin.H{
		"provider":              "stripe",
		"stripePaymentIntentId": pi.ID,
		"clientSecret":          pi.ClientSecret,
		"publishableKey":        st.GetPublishableKey(),
		"amount":                totalMinor,
		"currency":              strings.ToUpper(currency),
		"orderNumber":           order.OrderNumber,
		"prefill": gin.H{
			"name":  order.Customer.FirstName + " " + order.Customer.LastName,
			"email": order.Customer.Email,
			"phone": order.Customer.Phone,
		},
	})
}

// VerifyPayment verifies a payment after checkout on the client. The request
// body differs by provider — Razorpay sends razorpayPaymentId/OrderId/Signature,
// Stripe sends stripePaymentIntentId — so we look at the already-stamped
// order.payment_provider first to pick the code path.
// POST /payments/order/:orderId/verify
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req struct {
		// Razorpay
		RazorpayPaymentID string `json:"razorpayPaymentId"`
		RazorpayOrderID   string `json:"razorpayOrderId"`
		RazorpaySignature string `json:"razorpaySignature"`
		// Stripe
		StripePaymentIntentID string `json:"stripePaymentIntentId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Scope by customer_id so a user cannot verify another user's order
	// (IDOR). Returning 404 (not 403) avoids leaking existence of other
	// orders.
	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderID, userID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	provider := strings.ToLower(order.PaymentProvider)
	if provider == "" {
		provider = "razorpay"
	}

	switch provider {
	case "stripe":
		h.verifyStripePayment(c, &order, req.StripePaymentIntentID)
	default:
		h.verifyRazorpayPayment(c, &order, req.RazorpayPaymentID, req.RazorpayOrderID)
	}
}

func (h *PaymentHandler) verifyRazorpayPayment(c *gin.Context, order *models.Order, paymentID, rzOrderID string) {
	if paymentID == "" || rzOrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "razorpayPaymentId and razorpayOrderId are required"})
		return
	}
	if order.RazorpayOrderID != rzOrderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID mismatch"})
		return
	}

	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	payment, err := rz.FetchPayment(paymentID)
	if err != nil {
		log.Printf("Failed to fetch Razorpay payment %s: %v", paymentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}

	if payment.Status != "captured" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Payment not captured, status: %s", payment.Status)})
		return
	}

	database.DB.Model(order).Updates(map[string]interface{}{
		"payment_status":      models.PaymentCompleted,
		"payment_method":      payment.Method,
		"razorpay_payment_id": paymentID,
	})

	if err := services.PublishEvent("orders.paid", "order.paid", order.CustomerID, map[string]interface{}{
		"order_id":     order.ID.String(),
		"order_number": order.OrderNumber,
		"amount":       services.FromPaise(payment.Amount),
		"method":       payment.Method,
		"provider":     "razorpay",
	}); err != nil {
		log.Printf("Failed to publish order paid event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment verified", "status": "completed"})
}

func (h *PaymentHandler) verifyStripePayment(c *gin.Context, order *models.Order, piID string) {
	if piID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stripePaymentIntentId is required"})
		return
	}
	if order.StripePaymentIntentID != piID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PaymentIntent ID mismatch"})
		return
	}

	st := services.GetStripe()
	if st == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe gateway not configured"})
		return
	}

	pi, err := st.FetchPaymentIntent(piID)
	if err != nil {
		log.Printf("Failed to fetch Stripe PaymentIntent %s: %v", piID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}

	if pi.Status != "succeeded" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Payment not succeeded, status: %s", pi.Status)})
		return
	}

	database.DB.Model(order).Updates(map[string]interface{}{
		"payment_status": models.PaymentCompleted,
		"payment_method": "card",
	})

	eventCurrency := strings.ToLower(order.Currency)
	if eventCurrency == "" {
		eventCurrency = "inr"
	}
	if err := services.PublishEvent("orders.paid", "order.paid", order.CustomerID, map[string]interface{}{
		"order_id":     order.ID.String(),
		"order_number": order.OrderNumber,
		"amount":       services.FromMinor(pi.Amount, eventCurrency),
		"method":       "card",
		"provider":     "stripe",
		"currency":     order.Currency,
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

	// Validate refund amount
	refundAmount := req.Amount
	if refundAmount == 0 {
		refundAmount = order.Total // Full refund
	}
	if refundAmount > order.Total {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refund amount cannot exceed order total"})
		return
	}

	provider := strings.ToLower(order.PaymentProvider)
	if provider == "" {
		provider = "razorpay"
	}

	var refundID, refundStatus string

	switch provider {
	case "stripe":
		if order.StripePaymentIntentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Stripe payment found for this order"})
			return
		}
		st := services.GetStripe()
		if st == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Stripe gateway not configured"})
			return
		}
		refundCurrency := strings.ToLower(order.Currency)
		if refundCurrency == "" {
			refundCurrency = services.CurrencyForCountry(order.Chef.PayoutCountry)
		}
		r, err := st.CreateRefund(&services.StripeRefundRequest{
			PaymentIntent:        order.StripePaymentIntentID,
			Amount:               services.ToMinor(refundAmount, refundCurrency),
			Reason:               "requested_by_customer",
			ReverseTransfer:      true, // pull money back from the chef's Connect account
			RefundApplicationFee: true, // also refund our platform cut
			Metadata: map[string]string{
				"order_id":     order.ID.String(),
				"order_number": order.OrderNumber,
				"reason":       req.Reason,
				"initiated_by": initiatedBy,
			},
		})
		if err != nil {
			log.Printf("Failed to create Stripe refund for order %s: %v", order.OrderNumber, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
			return
		}
		refundID = r.ID
		refundStatus = r.Status
	default: // razorpay
		if order.RazorpayPaymentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Razorpay payment found for this order"})
			return
		}
		rz := services.GetRazorpay()
		if rz == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
			return
		}
		r, err := rz.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
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
			log.Printf("Failed to create Razorpay refund for order %s: %v", order.OrderNumber, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
			return
		}
		refundID = r.ID
		refundStatus = r.Status
	}

	now := time.Now()
	database.DB.Model(&order).Updates(map[string]interface{}{
		"payment_status":      models.PaymentRefunded,
		"status":              models.OrderStatusRefunded,
		"refund_id":           refundID,
		"refund_amount":       refundAmount,
		"refund_reason":       req.Reason,
		"refund_initiated_by": initiatedBy,
		"refunded_at":         &now,
	})

	if err := services.PublishEvent("orders.refunded", "order.refunded", userID, map[string]interface{}{
		"order_id":      order.ID.String(),
		"order_number":  order.OrderNumber,
		"refund_amount": refundAmount,
		"reason":        req.Reason,
		"initiated_by":  initiatedBy,
		"refund_id":     refundID,
		"provider":      provider,
	}); err != nil {
		log.Printf("Failed to publish order refunded event: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Refund initiated",
		"refundId":     refundID,
		"refundAmount": refundAmount,
		"status":       refundStatus,
		"provider":     provider,
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

	// Idempotent update: only flip orders that are not already marked
	// completed. Razorpay retries webhooks; without this guard repeated
	// deliveries would re-emit downstream effects.
	res := database.DB.Model(&models.Order{}).
		Where("razorpay_order_id = ? AND payment_status <> ?", payment.OrderID, models.PaymentCompleted).
		Updates(map[string]interface{}{
			"payment_status":      models.PaymentCompleted,
			"payment_method":      payment.Method,
			"razorpay_payment_id": payment.ID,
		})
	if res.Error != nil {
		log.Printf("Failed to apply payment.captured for order %s: %v", payment.OrderID, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		log.Printf("payment.captured already processed for order %s (payment %s) — skipping", payment.OrderID, payment.ID)
	}
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

	// Idempotent: only transition from a non-terminal state. Avoids
	// overwriting a completed payment if events arrive out-of-order.
	database.DB.Model(&models.Order{}).
		Where("razorpay_order_id = ? AND payment_status NOT IN ?", payment.OrderID, []models.PaymentStatus{
			models.PaymentCompleted,
			models.PaymentFailed,
			models.PaymentRefunded,
		}).
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

	// Idempotent: skip if we already stamped this refund_id on the order
	// (Razorpay retries refund.processed; without this we'd re-write the
	// refunded_at timestamp on every redelivery).
	now := time.Now()
	database.DB.Model(&models.Order{}).
		Where("razorpay_payment_id = ? AND (refund_id IS NULL OR refund_id <> ?)", refund.PaymentID, refund.ID).
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

// StripeWebhook handles Stripe webhook events.
// POST /webhooks/stripe (no auth — verified via Stripe-Signature HMAC).
//
// Only the event types relevant to our order lifecycle are handled;
// everything else is acknowledged with 200 so Stripe stops retrying.
func (h *PaymentHandler) StripeWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
		return
	}

	signature := c.GetHeader("Stripe-Signature")
	if !services.VerifyStripeWebhookSignature(body, signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}

	var event struct {
		ID   string          `json:"id"`
		Type string          `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	log.Printf("Stripe webhook received: %s (%s)", event.Type, event.ID)

	switch event.Type {
	case "payment_intent.succeeded":
		h.handleStripePaymentSucceeded(event.Data.Object)
	case "payment_intent.payment_failed":
		h.handleStripePaymentFailed(event.Data.Object)
	case "charge.refunded":
		// Top-level object is a Charge with a nested refunds.data[]. Pick
		// the most recent refund to stamp on the order.
		h.handleStripeChargeRefunded(event.Data.Object)
	case "refund.updated":
		h.handleStripeRefund(event.Data.Object)
	case "account.updated":
		h.handleStripeAccountUpdated(event.Data.Object)
	default:
		log.Printf("Unhandled Stripe webhook event: %s", event.Type)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *PaymentHandler) handleStripePaymentSucceeded(obj json.RawMessage) {
	var pi services.StripePaymentIntent
	if err := json.Unmarshal(obj, &pi); err != nil {
		log.Printf("Failed to parse stripe payment_intent: %v", err)
		return
	}
	log.Printf("Stripe payment succeeded: %s (amount: %d %s)", pi.ID, pi.Amount, pi.Currency)

	database.DB.Model(&models.Order{}).
		Where("stripe_payment_intent_id = ?", pi.ID).
		Updates(map[string]interface{}{
			"payment_status": models.PaymentCompleted,
			"payment_method": "card",
		})
}

func (h *PaymentHandler) handleStripePaymentFailed(obj json.RawMessage) {
	var pi services.StripePaymentIntent
	if err := json.Unmarshal(obj, &pi); err != nil {
		log.Printf("Failed to parse stripe payment_intent: %v", err)
		return
	}
	log.Printf("Stripe payment failed: %s", pi.ID)

	database.DB.Model(&models.Order{}).
		Where("stripe_payment_intent_id = ?", pi.ID).
		Update("payment_status", models.PaymentFailed)
}

// handleStripeAccountUpdated syncs the cached capability flags on the chef
// profile whenever Stripe signals a Connect account change. Keeps the
// CreateOrderPayment guard (StripeChargesEnabled) accurate without needing
// a Stripe round-trip on every order.
func (h *PaymentHandler) handleStripeAccountUpdated(obj json.RawMessage) {
	var acct struct {
		ID      string `json:"id"`
		Charges bool   `json:"charges_enabled"`
		Payouts bool   `json:"payouts_enabled"`
		Country string `json:"country"`
	}
	if err := json.Unmarshal(obj, &acct); err != nil {
		log.Printf("Failed to parse stripe account.updated: %v", err)
		return
	}
	if acct.ID == "" {
		return
	}
	log.Printf("Stripe account.updated: %s (charges=%v, payouts=%v)", acct.ID, acct.Charges, acct.Payouts)

	database.DB.Model(&models.ChefProfile{}).
		Where("stripe_account_id = ?", acct.ID).
		Updates(map[string]interface{}{
			"stripe_charges_enabled": acct.Charges,
			"stripe_payouts_enabled": acct.Payouts,
		})
}

// handleStripeRefund handles refund.updated where the top-level object IS a
// Refund with payment_intent at the root.
func (h *PaymentHandler) handleStripeRefund(obj json.RawMessage) {
	var r struct {
		ID            string `json:"id"`
		PaymentIntent string `json:"payment_intent"`
		Amount        int    `json:"amount"`
		Status        string `json:"status"`
	}
	if err := json.Unmarshal(obj, &r); err != nil {
		log.Printf("Failed to parse stripe refund: %v", err)
		return
	}
	if r.PaymentIntent == "" {
		return
	}
	log.Printf("Stripe refund.updated: %s (payment_intent: %s, status: %s)", r.ID, r.PaymentIntent, r.Status)

	now := time.Now()
	database.DB.Model(&models.Order{}).
		Where("stripe_payment_intent_id = ?", r.PaymentIntent).
		Updates(map[string]interface{}{
			"refund_id":   r.ID,
			"refunded_at": &now,
		})
}

// handleStripeChargeRefunded handles charge.refunded where the top-level
// object is a Charge; refunds live at charge.refunds.data[]. Picks the last
// refund entry so we record the most recent ID/state.
func (h *PaymentHandler) handleStripeChargeRefunded(obj json.RawMessage) {
	var ch struct {
		ID            string `json:"id"`
		PaymentIntent string `json:"payment_intent"`
		Refunded      bool   `json:"refunded"`
		Refunds       struct {
			Data []struct {
				ID     string `json:"id"`
				Amount int    `json:"amount"`
				Status string `json:"status"`
			} `json:"data"`
		} `json:"refunds"`
	}
	if err := json.Unmarshal(obj, &ch); err != nil {
		log.Printf("Failed to parse stripe charge.refunded: %v", err)
		return
	}
	if ch.PaymentIntent == "" || len(ch.Refunds.Data) == 0 {
		return
	}
	latest := ch.Refunds.Data[len(ch.Refunds.Data)-1]
	log.Printf("Stripe charge.refunded: charge=%s pi=%s refund=%s status=%s",
		ch.ID, ch.PaymentIntent, latest.ID, latest.Status)

	now := time.Now()
	database.DB.Model(&models.Order{}).
		Where("stripe_payment_intent_id = ?", ch.PaymentIntent).
		Updates(map[string]interface{}{
			"refund_id":   latest.ID,
			"refunded_at": &now,
		})
}

