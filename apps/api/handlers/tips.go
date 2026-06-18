package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// tips.go — post-delivery tips for chefs / riders (#45). The customer tips after
// delivery; one Razorpay charge Route-splits 100% to the chef and/or rider linked
// accounts (no platform commission, no tax). Charge → client verify (and/or
// webhook) → mark paid + notify. Checkout-time tips (Order.ChefTip/DriverTip) are
// a separate, pre-existing concept and are untouched here.

const (
	minTipAmount = 1.0    // ₹1 minimum per charge
	maxTipAmount = 5000.0 // sanity cap per charge
)

// TipHandler owns the customer tip-charge endpoints + chef "tips received".
type TipHandler struct{}

func NewTipHandler() *TipHandler { return &TipHandler{} }

// validateTipAmounts checks a (chef, rider) tip split. Extracted for unit testing.
func validateTipAmounts(chef, rider float64) error {
	if chef < 0 || rider < 0 {
		return fmt.Errorf("tip amounts cannot be negative")
	}
	total := chef + rider
	if total < minTipAmount {
		return fmt.Errorf("a tip must be at least ₹%.0f", minTipAmount)
	}
	if total > maxTipAmount {
		return fmt.Errorf("a tip cannot exceed ₹%.0f", maxTipAmount)
	}
	return nil
}

type createTipRequest struct {
	ChefAmount  float64 `json:"chefAmount"`
	RiderAmount float64 `json:"riderAmount"`
}

// CreateOrderTip — POST /payments/order/:orderId/tip. Validates the order is
// delivered + owned by the customer, resolves the chef/rider linked accounts,
// and creates a Razorpay order that Route-splits the tip 100% to them.
func (h *TipHandler) CreateOrderTip(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req createTipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateTipAmounts(req.ChefAmount, req.RiderAmount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Chef").Preload("Delivery.DeliveryPartner").
		Where("id = ? AND customer_id = ?", orderID, customerID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.Status != models.OrderStatusDelivered {
		c.JSON(http.StatusConflict, gin.H{"error": "You can only tip after the order is delivered"})
		return
	}

	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	tip := models.Tip{OrderID: order.ID, CustomerID: customerID, Currency: "INR", Status: models.TipPending}
	var transfers []services.TransferSpec

	if req.ChefAmount > 0 {
		acct := order.Chef.RazorpayAccountID
		if acct == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "This chef can't receive tips right now"})
			return
		}
		transfers = append(transfers, services.TransferSpec{
			Account: acct, Amount: services.ToPaise(req.ChefAmount), Currency: "INR", OnHold: false,
			Notes: map[string]string{"purpose": "tip", "beneficiary": "chef", "order_id": order.ID.String()},
		})
		tip.ChefAmount = req.ChefAmount
		chefUserID := order.Chef.UserID
		tip.ChefUserID = &chefUserID
	}

	if req.RiderAmount > 0 {
		if order.Delivery == nil || order.Delivery.DeliveryPartnerID == nil || order.Delivery.DeliveryPartner.RazorpayAccountID == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "This delivery has no rider to tip"})
			return
		}
		transfers = append(transfers, services.TransferSpec{
			Account: order.Delivery.DeliveryPartner.RazorpayAccountID, Amount: services.ToPaise(req.RiderAmount),
			Currency: "INR", OnHold: false,
			Notes: map[string]string{"purpose": "tip", "beneficiary": "rider", "order_id": order.ID.String()},
		})
		tip.RiderAmount = req.RiderAmount
		riderUserID := order.Delivery.DeliveryPartner.UserID
		tip.RiderUserID = &riderUserID
	}

	tip.Amount = tip.ChefAmount + tip.RiderAmount
	if len(transfers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nothing to tip"})
		return
	}

	rzOrder, err := rz.CreateOrder(&services.OrderRequest{
		Amount:    services.ToPaise(tip.Amount),
		Currency:  "INR",
		Receipt:   "TIP-" + order.OrderNumber,
		Notes:     map[string]string{"purpose": "tip", "order_id": order.ID.String(), "customer_id": customerID.String()},
		Transfers: transfers,
	})
	if err != nil {
		log.Printf("tip: create razorpay order for %s failed: %v", order.OrderNumber, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not start the tip payment"})
		return
	}
	tip.RazorpayOrderID = rzOrder.ID

	if err := database.DB.Create(&tip).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record tip"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"tipId":           tip.ID,
		"razorpayOrderId": rzOrder.ID,
		"razorpayKeyId":   rz.GetKeyID(),
		"amount":          rzOrder.Amount, // paise — fed straight to the checkout sheet
		"amountRupees":    tip.Amount,
		"currency":        "INR",
	})
}

type verifyTipRequest struct {
	RazorpayPaymentID string `json:"razorpayPaymentId"`
	RazorpayOrderID   string `json:"razorpayOrderId"`
}

// VerifyTip — POST /payments/tip/:tipId/verify. Confirms the tip charge captured
// and marks it paid + notifies the beneficiaries (idempotent; the webhook is a
// second, equivalent path).
func (h *TipHandler) VerifyTip(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	tipID, err := uuid.Parse(c.Param("tipId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tip ID"})
		return
	}
	var req verifyTipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tip models.Tip
	if err := database.DB.Where("id = ? AND customer_id = ?", tipID, customerID).First(&tip).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tip not found"})
		return
	}
	if tip.Status == models.TipPaid {
		c.JSON(http.StatusOK, gin.H{"tip": tip, "alreadyPaid": true})
		return
	}

	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}
	if req.RazorpayPaymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "razorpayPaymentId is required"})
		return
	}
	payment, err := rz.FetchPayment(req.RazorpayPaymentID)
	if err != nil {
		log.Printf("tip: fetch payment %s failed: %v", req.RazorpayPaymentID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify payment"})
		return
	}
	if payment.Status != "captured" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Payment not captured, status: %s", payment.Status)})
		return
	}
	if payment.OrderID != tip.RazorpayOrderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID mismatch"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return markTipPaidTx(tx, &tip, req.RazorpayPaymentID)
	}); err != nil {
		log.Printf("tip: mark paid %s failed: %v", tip.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record tip"})
		return
	}
	tip.Status = models.TipPaid
	tip.RazorpayPaymentID = req.RazorpayPaymentID
	c.JSON(http.StatusOK, gin.H{"tip": tip, "paymentVerified": true})
}

// GetChefTips — GET /chef/tips. Tips the authed chef has received (paid only).
func (h *TipHandler) GetChefTips(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var tips []models.Tip
	database.DB.Where("chef_user_id = ? AND status = ? AND chef_amount > 0", userID, models.TipPaid).
		Preload("Order").Order("created_at DESC").Limit(100).Find(&tips)
	c.JSON(http.StatusOK, gin.H{"data": tips})
}

// markTipPaidTx flips a tip to paid (status-guarded → idempotent) and stages the
// beneficiary tip-received events. Shared by the verify endpoint and the webhook.
func markTipPaidTx(tx *gorm.DB, tip *models.Tip, paymentID string) error {
	res := tx.Model(&models.Tip{}).
		Where("id = ? AND status <> ?", tip.ID, models.TipPaid).
		Updates(map[string]any{"status": models.TipPaid, "razorpay_payment_id": paymentID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil // already paid by the other path
	}
	if tip.ChefUserID != nil && tip.ChefAmount > 0 {
		if err := services.EnqueueEvent(tx, services.SubjectChefTipReceived, "chef.tip_received", *tip.ChefUserID, map[string]any{
			"tip_id": tip.ID.String(), "order_id": tip.OrderID.String(), "amount": tip.ChefAmount,
		}); err != nil {
			return err
		}
	}
	if tip.RiderUserID != nil && tip.RiderAmount > 0 {
		if err := services.EnqueueEvent(tx, services.SubjectDriverTipReceived, "driver.tip_received", *tip.RiderUserID, map[string]any{
			"tip_id": tip.ID.String(), "order_id": tip.OrderID.String(), "amount": tip.RiderAmount,
		}); err != nil {
			return err
		}
	}
	return nil
}

// markTipPaidByRazorpayOrder is the webhook path: confirm a tip charge by its
// Razorpay order id (idempotent). Called from handlePaymentCaptured.
func markTipPaidByRazorpayOrder(rzOrderID, paymentID string) {
	var tip models.Tip
	if err := database.DB.Where("razorpay_order_id = ?", rzOrderID).First(&tip).Error; err != nil {
		return // not a tip charge
	}
	if tip.Status == models.TipPaid {
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return markTipPaidTx(tx, &tip, paymentID)
	}); err != nil {
		log.Printf("tip: webhook mark paid %s failed: %v", tip.ID, err)
	}
}
