package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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

	tip := models.Tip{ModePartition: models.ModePartition{Mode: order.Mode, TestSessionID: order.TestSessionID}, OrderID: order.ID, CustomerID: customerID, Currency: "INR", Status: models.TipPending}

	// A tip rides the SAME gateway as the order it thanks — a test-mode order's
	// tip must not become a real charge, and a Cashfree order's tip cannot be
	// routed by Razorpay.
	//
	// This dispatch is the fix for the tip surface being unreachable platform-wide:
	// the flow was written against Razorpay Route and never moved when payouts did,
	// so it demanded a chef.razorpay_account_id that NO chef on the platform has.
	// Every attempt answered 409 "This chef can't receive tips right now".
	if models.NormalizeProvider(order.PaymentProvider) == models.PaymentProviderCashfree {
		h.createCashfreeTip(c, &order, &tip, req, customerID)
		return
	}

	rz := services.GetRazorpayFor(order.Mode)
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

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
	RazorpaySignature string `json:"razorpaySignature"`
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

	// A Cashfree tip has no client-supplied payment id or signature to check —
	// the authority is a server-side fetch, exactly as the order path's Cashfree
	// leg works. Dispatch before touching any Razorpay-shaped field.
	if strings.HasPrefix(tip.RazorpayOrderID, "tip-") {
		if !h.verifyCashfreeTip(c, &tip) {
			return
		}
		h.settleTip(c, &tip)
		return
	}

	rz := services.GetRazorpayFor(tip.Mode)
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
	// SECURITY (#395·4): bind the captured amount + Checkout signature to THIS tip,
	// mirroring the main-order VerifyPayment. Without these a mismatched/under-amount
	// captured payment on the tip's razorpay order (payment.Amount comes from
	// Razorpay, unforgeable) could settle the tip in full, and a captured payment
	// from another order could be reused. Signature enforced when present (the app
	// always sends it); the amount check is the hard gate and never trusts the client.
	if payment.Amount < services.ToPaise(tip.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment amount does not match the tip amount"})
		return
	}
	if req.RazorpaySignature != "" &&
		!services.VerifyPaymentSignature(tip.RazorpayOrderID, req.RazorpayPaymentID, req.RazorpaySignature) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment signature verification failed"})
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

// --- Cashfree tip leg ---
//
// The Razorpay leg above splits a tip with Route linked accounts. Cashfree's
// equivalent is Easy Split: the tip is charged as its own gateway order whose
// order_splits allocate 100% of the chef's share to the chef's vendor account,
// so the money settles to the chef directly and never sits on the platform's
// balance. Same promise the screen makes — "100% goes straight to your chef,
// with no platform cut" — expressed in the gateway the platform actually uses.
//
// The chef's share is the WHOLE chef tip: a tip carries no commission and no tax
// (INV-6), so unlike BuildOrderSplit there is no fee to subtract.

// createCashfreeTip charges a tip through Cashfree and hands the client a
// checkout session. Mirrors respondCashfreeSession's payload so the app opens
// the tip sheet with exactly the branch it already uses for an order.
func (h *TipHandler) createCashfreeTip(c *gin.Context, order *models.Order, tip *models.Tip, req createTipRequest, customerID uuid.UUID) {
	cf := services.GetCashfreeFor(order.Mode)
	if cf == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	// Rider tips have no Cashfree route: DeliveryPartner carries only a Razorpay
	// linked account. Say that plainly rather than reusing the chef's message,
	// which is what made the original defect read as a chef-account problem.
	if req.RiderAmount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Rider tips aren't available on this payment method yet"})
		return
	}
	if req.ChefAmount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nothing to tip"})
		return
	}

	chef := &order.Chef
	if chef.CashfreeVendorID == "" || !strings.EqualFold(chef.CashfreeVendorStatus, services.CashfreeVendorActive) {
		// A real, explainable state — the chef's payout registration is not live —
		// rather than the blanket message the Razorpay leg used to give everyone.
		c.JSON(http.StatusConflict, gin.H{
			"error": "This chef's payout account isn't active yet, so tips can't reach them",
		})
		return
	}

	tip.ChefAmount = req.ChefAmount
	chefUserID := chef.UserID
	tip.ChefUserID = &chefUserID
	tip.Amount = req.ChefAmount

	tipPaise := services.ToPaise(tip.Amount)
	// Deterministic order id keyed on the tip, so a retry lands on the SAME
	// Cashfree order and reuses its session instead of minting a parallel charge.
	tip.ID = uuid.New()
	cfOrderID := "tip-" + strings.ReplaceAll(tip.ID.String(), "-", "")

	cfOrder, err := cf.CreateOrder(&services.CashfreeOrderRequest{
		OrderID:     cfOrderID,
		AmountPaise: services.CashfreeAmountFromPaise(tipPaise),
		Currency:    "INR",
		Customer: services.CashfreeCustomerDetails{
			CustomerID:    strings.ReplaceAll(customerID.String(), "-", ""),
			CustomerPhone: order.Customer.Phone,
			CustomerName:  strings.TrimSpace(order.Customer.FirstName + " " + order.Customer.LastName),
			CustomerEmail: order.Customer.Email,
		},
		// 100% to the chef — the platform keeps nothing.
		Splits: []services.CashfreeVendorSplit{{
			VendorID:    chef.CashfreeVendorID,
			AmountPaise: services.CashfreeAmountFromPaise(tipPaise),
		}},
		Tags: map[string]string{
			"purpose":  "tip",
			"order_id": order.ID.String(),
			"tip_id":   tip.ID.String(),
		},
		IdempotencyKey: "tip:" + tip.ID.String(),
	})
	if err != nil {
		log.Printf("tip: create cashfree order for %s failed: %v", order.OrderNumber, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not start the tip payment"})
		return
	}

	// Same column the order path stores its Cashfree id in — see payment_cashfree.go.
	tip.RazorpayOrderID = cfOrder.OrderID
	if err := database.DB.Create(tip).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record tip"})
		return
	}

	env := "PRODUCTION"
	if cf.IsSandbox() {
		env = "SANDBOX"
	}
	c.JSON(http.StatusCreated, gin.H{
		"tipId":                    tip.ID,
		"provider":                 models.PaymentProviderCashfree,
		"cashfreeOrderId":          cfOrder.OrderID,
		"cashfreePaymentSessionId": cfOrder.PaymentSessionID,
		"cashfreeAppId":            cf.GetAppID(),
		"mode":                     env,
		"amount":                   tipPaise,
		"amountRupees":             tip.Amount,
		"currency":                 "INR",
	})
}

// verifyCashfreeTip confirms a Cashfree tip captured. The gate mirrors
// SettleCashfreeOrder: the gateway is asked directly, and the captured amount is
// bound to THIS tip so an under-amount or foreign payment cannot settle it.
func (h *TipHandler) verifyCashfreeTip(c *gin.Context, tip *models.Tip) bool {
	cf := services.GetCashfreeFor(tip.Mode)
	if cf == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return false
	}
	payment, err := cf.SuccessfulPayment(tip.RazorpayOrderID)
	if err != nil {
		log.Printf("tip: fetch cashfree payments for %s failed: %v", tip.RazorpayOrderID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not verify the tip payment — please try again in a moment"})
		return false
	}
	if payment == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment not completed"})
		return false
	}
	if payment.OrderID != tip.RazorpayOrderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID mismatch"})
		return false
	}
	if payment.AmountPaise.Paise() < services.ToPaise(tip.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment amount does not match the tip amount"})
		return false
	}
	tip.RazorpayPaymentID = payment.CFPaymentID.String()
	return true
}

// settleTip marks a verified tip paid and notifies the beneficiaries. Shared by
// both gateway legs so "what happens once the money is confirmed" has one
// implementation, whichever gateway confirmed it.
func (h *TipHandler) settleTip(c *gin.Context, tip *models.Tip) {
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return markTipPaidTx(tx, tip, tip.RazorpayPaymentID)
	}); err != nil {
		log.Printf("tip: mark paid %s failed: %v", tip.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record tip"})
		return
	}
	tip.Status = models.TipPaid
	c.JSON(http.StatusOK, gin.H{"tip": tip, "paymentVerified": true})
}
