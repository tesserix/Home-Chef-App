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
// delivery; one Cashfree charge Easy-Splits 100% to the chef's vendor account (no
// platform commission, no tax). Charge → client verify (and/or webhook) → mark
// paid + notify. Checkout-time tips (Order.ChefTip/DriverTip) are a separate,
// pre-existing concept and are untouched here.

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
// delivered + owned by the customer, resolves the chef's payout account, and
// creates a Cashfree order that splits the tip 100% to them.
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

	// A tip is a new charge, so it is minted on Cashfree whatever gateway the
	// order it thanks was stamped with (#1086). The old dispatch sent every
	// non-Cashfree order to the retired gateway's split product, which demanded a
	// linked account NO chef on the platform has — the whole tip surface
	// answered 409 "This chef can't receive tips right now".
	h.createCashfreeTip(c, &order, &tip, req, customerID)
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
	var tip models.Tip
	if err := database.DB.Where("id = ? AND customer_id = ?", tipID, customerID).First(&tip).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tip not found"})
		return
	}
	if tip.Status == models.TipPaid {
		c.JSON(http.StatusOK, gin.H{"tip": tip, "alreadyPaid": true})
		return
	}

	// A tip has no client-supplied payment id or signature to check — the
	// authority is a server-side fetch from the gateway that holds the charge.
	if !h.verifyCashfreeTip(c, &tip) {
		return
	}
	h.settleTip(c, &tip)
}

// GetChefTips — GET /chef/tips. Tips the authed chef has received (paid only).
//
// The rider leg counts when the chef carried the order themselves (#1080):
// that money reaches their account, so a list that only reads chef_amount shows
// a chef income they cannot account for. On a third-party delivery the rider is
// someone else and rider_user_id will not match, so nothing leaks.
func (h *TipHandler) GetChefTips(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var tips []models.Tip
	database.DB.
		Where("status = ?", models.TipPaid).
		Where(database.DB.Where("chef_user_id = ? AND chef_amount > 0", userID).
			Or("rider_user_id = ? AND rider_amount > 0", userID)).
		Preload("Order").Order("created_at DESC").Limit(100).Find(&tips)
	c.JSON(http.StatusOK, gin.H{"data": tips})
}

// markTipPaidTx flips a tip to paid (status-guarded → idempotent) and stages the
// beneficiary tip-received events. Shared by the verify endpoint and the webhook.
func markTipPaidTx(tx *gorm.DB, tip *models.Tip, paymentID string) error {
	res := tx.Model(&models.Tip{}).
		Where("id = ? AND status <> ?", tip.ID, models.TipPaid).
		Updates(map[string]any{"status": models.TipPaid, "gateway_payment_id": paymentID})
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

// markTipPaidByGatewayOrder is the webhook path: confirm a tip charge by its
// gateway order id (idempotent). Called from handlePaymentCaptured.
func markTipPaidByGatewayOrder(gatewayOrderID, paymentID string) {
	var tip models.Tip
	if err := database.DB.Where("gateway_order_id = ?", gatewayOrderID).First(&tip).Error; err != nil {
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
// The tip is charged as its own gateway order whose Easy Split
// order_splits allocate 100% of the chef's share to the chef's vendor account,
// so the money settles to the chef directly and never sits on the platform's
// balance. Same promise the screen makes — "100% goes straight to your chef,
// with no platform cut" — expressed in the gateway the platform actually uses.
//
// The chef's share is the WHOLE chef tip: a tip carries no commission and no tax
// (INV-6), so unlike BuildOrderSplit there is no fee to subtract.

// cashfreeTipPlan is where each leg of a tip settles. Both legs name the same
// vendor on a chef-delivered order, because the chef carried it.
type cashfreeTipPlan struct {
	VendorID    string
	ChefAmount  float64
	RiderAmount float64
	ChefUserID  *uuid.UUID
	RiderUserID *uuid.UUID
}

func (p cashfreeTipPlan) Total() float64 { return p.ChefAmount + p.RiderAmount }

// planCashfreeTip resolves a tip's beneficiaries, or the HTTP status and message
// explaining why it cannot be paid. A zero status means the plan is payable.
//
// The rider leg is gated on the ORDER's fulfilment type, not on a global flag
// (#1080): today every delivery is chef_delivery and the chef is the driver, so
// the rider's tip has a real destination. The day a fleet exists, an order
// carried by a DeliveryPartner still has no Cashfree route and must keep its
// refusal rather than quietly paying the rider's tip to the kitchen.
func planCashfreeTip(order *models.Order, req createTipRequest) (cashfreeTipPlan, int, string) {
	if req.RiderAmount > 0 && order.FulfillmentType != models.FulfillmentChefDelivery {
		return cashfreeTipPlan{}, http.StatusConflict, "Rider tips aren't available on this payment method yet"
	}
	if req.ChefAmount <= 0 && req.RiderAmount <= 0 {
		return cashfreeTipPlan{}, http.StatusBadRequest, "Nothing to tip"
	}

	chef := &order.Chef
	if chef.VendorID() == "" || !strings.EqualFold(chef.VendorStatus(), services.CashfreeVendorActive) {
		// A real, explainable state — the chef's payout registration is not live —
		// rather than the blanket message the retired gateway leg used to give everyone.
		return cashfreeTipPlan{}, http.StatusConflict,
			"This chef's payout account isn't active yet, so tips can't reach them"
	}

	plan := cashfreeTipPlan{VendorID: chef.VendorID()}
	userID := chef.UserID
	if req.ChefAmount > 0 {
		plan.ChefAmount = req.ChefAmount
		plan.ChefUserID = &userID
	}
	if req.RiderAmount > 0 {
		plan.RiderAmount = req.RiderAmount
		plan.RiderUserID = &userID
	}
	return plan, 0, ""
}

// createCashfreeTip charges a tip through Cashfree and hands the client a
// checkout session. Mirrors respondCashfreeSession's payload so the app opens
// the tip sheet with exactly the branch it already uses for an order.
//
// Both legs ride ONE gateway order with a single split line: they settle to the
// same vendor, so a second charge would double the gateway fee to reach the same
// account, and Cashfree takes one split per vendor. The legs stay independently
// traceable on our side — the tip row keeps chef_amount and rider_amount apart,
// which is what the two tip-received notifications are raised from.
func (h *TipHandler) createCashfreeTip(c *gin.Context, order *models.Order, tip *models.Tip, req createTipRequest, customerID uuid.UUID) {
	cf := services.GetCashfreeFor(order.Mode)
	if cf == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	plan, status, msg := planCashfreeTip(order, req)
	if status != 0 {
		c.JSON(status, gin.H{"error": msg})
		return
	}

	tip.ChefAmount = plan.ChefAmount
	tip.ChefUserID = plan.ChefUserID
	tip.RiderAmount = plan.RiderAmount
	tip.RiderUserID = plan.RiderUserID
	tip.Amount = plan.Total()

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
			VendorID:    plan.VendorID,
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
	tip.GatewayOrderID = cfOrder.OrderID
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
		"cashfreeEnv":              env,
		// mode is the same value under the name the mobile client reads.
		"mode":         env,
		"amount":       tipPaise,
		"amountRupees": tip.Amount,
		"currency":     "INR",
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
	payment, err := cf.SuccessfulPayment(tip.GatewayOrderID)
	if err != nil {
		log.Printf("tip: fetch cashfree payments for %s failed: %v", tip.GatewayOrderID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Could not verify the tip payment — please try again in a moment"})
		return false
	}
	if payment == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment not completed"})
		return false
	}
	if payment.OrderID != tip.GatewayOrderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID mismatch"})
		return false
	}
	if payment.AmountPaise.Paise() < services.ToPaise(tip.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment amount does not match the tip amount"})
		return false
	}
	tip.GatewayPaymentID = payment.CFPaymentID.String()
	return true
}

// settleTip marks a verified tip paid and notifies the beneficiaries. Shared by
// both gateway legs so "what happens once the money is confirmed" has one
// implementation, whichever gateway confirmed it.
func (h *TipHandler) settleTip(c *gin.Context, tip *models.Tip) {
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return markTipPaidTx(tx, tip, tip.GatewayPaymentID)
	}); err != nil {
		log.Printf("tip: mark paid %s failed: %v", tip.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record tip"})
		return
	}
	tip.Status = models.TipPaid
	c.JSON(http.StatusOK, gin.H{"tip": tip, "paymentVerified": true})
}
