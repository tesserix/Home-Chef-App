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
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"gorm.io/gorm"
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
//
//	Customer pays total → Razorpay splits automatically:
//	  - Chef gets: Subtotal + ChefTip (food cost + chef tip)
//	  - Driver gets: DeliveryFee + DriverTip (delivery fee + driver tip)
//	  - Fe3dr gets: ₹0 from orders (revenue comes only from subscriptions)
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

	// Optional wallet store-credit to apply at checkout (#141). Body is optional,
	// so a malformed/empty body just means "no wallet". Gated by a feature flag;
	// wallet is INR-only (Razorpay Route), so it never applies to the Stripe path.
	var body struct {
		WalletAmount float64 `json:"walletAmount"`
	}
	_ = c.ShouldBindJSON(&body)
	requestedWalletPaise := 0
	if config.AppConfig.WalletCheckoutEnabled && body.WalletAmount > 0 {
		requestedWalletPaise = services.ToPaise(body.WalletAmount)
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
		h.createRazorpayPayment(c, &order, userID, requestedWalletPaise)
	}
}

// createRazorpayPayment is the original INR + Razorpay Route flow. Unchanged
// from the pre-multi-gateway implementation except that the order's
// payment_provider column is now stamped so VerifyPayment / InitiateRefund
// know which code path to run later.
func (h *PaymentHandler) createRazorpayPayment(c *gin.Context, order *models.Order, userID uuid.UUID, requestedWalletPaise int) {
	rz := services.GetRazorpay()
	if rz == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
		return
	}

	totalPaise := services.ToPaise(order.Total)

	// FSSAI hard lockout (#32/#93): audit + withhold the chef payout when their
	// food-safety licence has lapsed. orderSettlements() clears the chef account so
	// no transfer is built; here we record the freeze for the regulatory trail.
	if services.IsChefFSSAIExpired(&order.Chef) {
		chefAmount := chefGrossPayout(order)
		middleware.RecordFSSAILockout("payout_withheld")
		log.Printf("fssai-lockout: withholding chef payout order=%s chef=%s amount=%.2f",
			order.OrderNumber, order.Chef.ID, chefAmount)
		services.LogSystemAudit(c, "chef.payout.fssai_withheld", "chef", order.Chef.ID.String(), nil, map[string]any{
			"orderNumber":    order.OrderNumber,
			"withheldAmount": chefAmount,
			"reason":         "fssai_licence_expired",
		})
	}

	settlements := orderSettlements(order)

	// Clamp the requested wallet credit against the live balance, then plan the
	// split: payment-funded transfers (bounded by the capture) + platform-funded
	// top-ups for whatever the capture can't cover.
	balancePaise := 0
	if requestedWalletPaise > 0 {
		if w, err := services.WalletBalance(database.DB, userID); err == nil && w != nil {
			balancePaise = services.ToPaise(w.Balance)
		}
	}
	plan := services.PlanWalletFunding(totalPaise, balancePaise, requestedWalletPaise, settlements)
	walletApplied := services.FromPaise(plan.WalletAppliedPaise)

	// Full-wallet order: the credit covers the entire total, so there is no gateway
	// payment. Settle the chef/driver from the platform balance, debit the wallet,
	// and mark the order paid in one shot.
	if plan.FullWallet {
		h.settleFullWalletOrder(c, order, plan, walletApplied)
		return
	}

	rzOrder, err := rz.CreateOrder(&services.OrderRequest{
		Amount:    plan.CapturePaise,
		Currency:  "INR",
		Receipt:   order.OrderNumber,
		Transfers: plan.PaymentTransfers,
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
		"wallet_applied":    walletApplied,
	})

	c.JSON(http.StatusOK, gin.H{
		"provider":        "razorpay",
		"razorpayOrderId": rzOrder.ID,
		"razorpayKeyId":   rz.GetKeyID(),
		"amount":          plan.CapturePaise,
		"walletApplied":   walletApplied,
		"currency":        "INR",
		"orderNumber":     order.OrderNumber,
		"prefill": gin.H{
			"name":  order.Customer.FirstName + " " + order.Customer.LastName,
			"email": order.Customer.Email,
			"phone": order.Customer.Phone,
		},
	})
}

// orderSettlements derives the chef + driver payouts for an order, chef first so
// its (larger) food payout stays a single payment-linked transfer when the capture
// allows (#141). The chef account is cleared when its FSSAI licence has lapsed, so
// that slice is withheld and never transferred. Requires Chef and
// Delivery.DeliveryPartner preloaded. Deterministic, so create and verify produce
// the same split for a given order.
// chefGrossPayout is the chef's pre-Route payout for an order: food revenue +
// tax + chef tip, less any chef-funded promo discount the chef bears (#39).
// Platform-funded promos leave ChefFundedDiscount at 0, so the chef stays whole.
// Single source of truth shared by the Route split, the Stripe transfer, and the
// FSSAI-withhold audit so the three can't drift.
func chefGrossPayout(order *models.Order) float64 {
	amount := order.Subtotal + order.Tax + order.ChefTip - order.ChefFundedDiscount
	if amount < 0 {
		amount = 0
	}
	return amount
}

func orderSettlements(order *models.Order) []services.Settlement {
	chefAccount := order.Chef.RazorpayAccountID
	if services.IsChefFSSAIExpired(&order.Chef) {
		chefAccount = ""
	}
	driverAccount := ""
	if order.Delivery != nil {
		driverAccount = order.Delivery.DeliveryPartner.RazorpayAccountID
	}
	return []services.Settlement{
		{Account: chefAccount, Amount: services.ToPaise(chefGrossPayout(order)), Hold: true,
			Notes: map[string]string{"purpose": "food_payment", "order_number": order.OrderNumber}},
		{Account: driverAccount, Amount: services.ToPaise(order.DeliveryFee + order.DriverTip), Hold: true,
			Notes: map[string]string{"purpose": "delivery_payment", "order_number": order.OrderNumber}},
	}
}

// debitOrderWallet debits the customer's store credit for the wallet applied to an
// order, idempotent on the order so a retry never double-debits (#141).
func debitOrderWallet(order *models.Order) error {
	if order.WalletApplied <= 0 {
		return nil
	}
	_, err := services.DebitWallet(database.DB, order.CustomerID, order.WalletApplied,
		models.WalletSourceOrderPayment, &order.ID, "checkout", "wallet-debit:"+order.ID.String(), nil)
	return err
}

// settleWalletTopUps funds the chef/driver portion that the gateway capture could
// not cover, via direct transfers from the platform balance (#141). Failures are
// logged but not fatal — the money is already captured and the reconciliation job
// retries; failing here would wrongly tell the client the order is unpaid.
func settleWalletTopUps(order *models.Order, topUps []services.TransferSpec) {
	rz := services.GetRazorpay()
	if rz == nil {
		return
	}
	for _, t := range topUps {
		if _, err := rz.CreateTransfer(&services.DirectTransferRequest{
			Account: t.Account, Amount: t.Amount, Currency: t.Currency, OnHold: t.OnHold, Notes: t.Notes,
		}); err != nil {
			log.Printf("wallet-topup: direct transfer failed order=%s account=%s amount=%d: %v",
				order.OrderNumber, t.Account, t.Amount, err)
		}
	}
}

// settleFullWalletOrder handles an order fully covered by store credit: there is no
// gateway payment, so the chef/driver are paid entirely from the platform balance,
// the wallet is debited, and the order is marked paid (#141).
func (h *PaymentHandler) settleFullWalletOrder(c *gin.Context, order *models.Order, plan services.FundingPlan, walletApplied float64) {
	order.WalletApplied = walletApplied
	if err := debitOrderWallet(order); err != nil {
		log.Printf("full-wallet: debit failed order=%s: %v", order.OrderNumber, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not apply wallet credit"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(order).Updates(map[string]interface{}{
			"payment_status":   models.PaymentCompleted,
			"payment_method":   "wallet",
			"payment_provider": "wallet",
			"wallet_applied":   walletApplied,
		}).Error; err != nil {
			return err
		}
		return services.EnqueueEvent(tx, "orders.paid", "order.paid", order.CustomerID, map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       order.Total,
			"method":       "wallet",
			"provider":     "wallet",
		})
	}); err != nil {
		log.Printf("full-wallet: mark-paid failed order=%s: %v", order.OrderNumber, err)
	} else {
		// Referral reward (#38) on the referee's first paid order — idempotent.
		services.MaybeGrantReward(database.DB, order.ID)
	}

	// Pay the chef/driver from the platform balance (the whole split is a top-up).
	settleWalletTopUps(order, plan.DirectTopUps)

	c.JSON(http.StatusOK, gin.H{
		"provider":      "wallet",
		"paid":          true,
		"amount":        0,
		"walletApplied": walletApplied,
		"orderNumber":   order.OrderNumber,
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

	// Chef receives: subtotal + tax + chefTip (less any chef-funded promo). Platform
	// keeps the rest (deliveryFee + driverTip) as application_fee; driver gets paid
	// out of platform balance via a follow-up Transfer on delivery confirmation.
	chefAmount := chefGrossPayout(order)
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
	// Preload Chef + Delivery so wallet-funded orders can recompute the chef/driver
	// split and settle their platform-balance top-ups after capture (#141).
	if err := database.DB.Preload("Chef").Preload("Delivery.DeliveryPartner").
		Where("id = ? AND customer_id = ?", orderID, userID).First(&order).Error; err != nil {
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

	// Mark the order paid and stage the order.paid event atomically (transactional
	// outbox). Payment already captured at the gateway, so a DB hiccup must not
	// 500 the client — it's logged + sent to Sentry and the reconciliation cron
	// catches any drift.
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(order).Updates(map[string]interface{}{
			"payment_status":      models.PaymentCompleted,
			"payment_method":      payment.Method,
			"razorpay_payment_id": paymentID,
		}).Error; err != nil {
			return err
		}
		return services.EnqueueEvent(tx, "orders.paid", "order.paid", order.CustomerID, map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       services.FromPaise(payment.Amount),
			"method":       payment.Method,
			"provider":     "razorpay",
		})
	}); err != nil {
		log.Printf("Failed to persist payment completion + event for order %s: %v", order.ID, err)
		services.CaptureBackgroundError(err)
	} else {
		// Referral reward (#38) on the referee's first paid order — idempotent,
		// so a later webhook for the same order won't double-pay.
		services.MaybeGrantReward(database.DB, order.ID)
	}

	// Wallet-at-checkout settlement (#141): now that the gateway capture is
	// confirmed, debit the applied store credit (idempotent) and top up the
	// chef/driver portion the capture couldn't cover, from the platform balance.
	// Recomputes the same split planned at create time (deterministic per order).
	if order.WalletApplied > 0 {
		if err := debitOrderWallet(order); err != nil {
			log.Printf("wallet-debit failed order=%s: %v", order.OrderNumber, err)
			services.CaptureBackgroundError(err)
		}
		appliedPaise := services.ToPaise(order.WalletApplied)
		plan := services.PlanWalletFunding(services.ToPaise(order.Total), appliedPaise, appliedPaise, orderSettlements(order))
		settleWalletTopUps(order, plan.DirectTopUps)
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

	eventCurrency := strings.ToLower(order.Currency)
	if eventCurrency == "" {
		eventCurrency = "inr"
	}

	// Mark paid + stage the order.paid event atomically (transactional outbox).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(order).Updates(map[string]interface{}{
			"payment_status": models.PaymentCompleted,
			"payment_method": "card",
		}).Error; err != nil {
			return err
		}
		return services.EnqueueEvent(tx, "orders.paid", "order.paid", order.CustomerID, map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       services.FromMinor(pi.Amount, eventCurrency),
			"method":       "card",
			"provider":     "stripe",
			"currency":     order.Currency,
		})
	}); err != nil {
		log.Printf("Failed to persist payment completion + event for order %s: %v", order.ID, err)
		services.CaptureBackgroundError(err)
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
		Amount   float64 `json:"amount"` // 0 = full refund
		Reason   string  `json:"reason" binding:"required"`
		ToWallet bool    `json:"toWallet"` // credit store credit instead of reversing the gateway charge (#33)
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

	// Release reserved daily capacity (#48) only on a FULL refund of an order that
	// hasn't been delivered yet — those dishes won't be made. Partial and
	// post-delivery refunds keep the capacity consumed.
	releaseCapOnRefund := refundAmount >= order.Total &&
		order.Status != models.OrderStatusDelivered &&
		order.Status != models.OrderStatusCancelled &&
		order.Status != models.OrderStatusRefunded
	refundCapDay := services.CapacityDay(order.CreatedAt)
	releaseRefundCapacity := func(tx *gorm.DB) error {
		if !releaseCapOnRefund {
			return nil
		}
		var items []models.OrderItem
		if err := tx.Where("order_id = ?", order.ID).Find(&items).Error; err != nil {
			return err
		}
		for _, it := range items {
			if err := services.ReleaseCapacity(tx, it.MenuItemID, it.Quantity, refundCapDay); err != nil {
				return err
			}
		}
		// Release the scheduled delivery-slot booking too (#51), keyed to the
		// order's scheduled delivery day.
		if order.DeliverySlot != "" && order.ScheduledFor != nil {
			if err := services.ReleaseSlot(tx, order.ChefID, order.DeliverySlot, 1, services.CapacityDay(*order.ScheduledFor)); err != nil {
				return err
			}
		}
		return nil
	}

	// Refund-to-wallet (#33): credit the customer's store credit instead of
	// reversing the gateway charge. Faster for the customer (no gateway round
	// trip) and the platform keeps the cash. The idempotency key ties the credit
	// to the order so a retried refund can't double-credit. Doesn't touch the
	// chef/driver splits — the original payment already settled.
	if req.ToWallet {
		txn, werr := services.CreditWallet(database.DB, order.CustomerID, refundAmount,
			models.WalletSourceRefund, &order.ID,
			fmt.Sprintf("Refund for order %s: %s", order.OrderNumber, req.Reason),
			"refund:"+order.ID.String(), nil)
		if werr != nil {
			log.Printf("refund-to-wallet failed for order %s: %v", order.OrderNumber, werr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to credit wallet"})
			return
		}
		now := time.Now()
		// Persist the refund and stage the order.refunded event atomically.
		if err := database.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&order).Updates(map[string]interface{}{
				"payment_status":      models.PaymentRefunded,
				"status":              models.OrderStatusRefunded,
				"refund_id":           "wallet:" + txn.ID.String(),
				"refund_amount":       refundAmount,
				"refund_reason":       req.Reason,
				"refund_initiated_by": initiatedBy,
				"refunded_at":         &now,
			}).Error; err != nil {
				return err
			}
			if err := releaseRefundCapacity(tx); err != nil {
				return err
			}
			return services.EnqueueEvent(tx, "orders.refunded", "order.refunded", userID, map[string]interface{}{
				"order_id": order.ID.String(), "order_number": order.OrderNumber,
				"refund_amount": refundAmount, "reason": req.Reason, "initiated_by": initiatedBy,
				"refund_id": "wallet:" + txn.ID.String(), "provider": "wallet",
			})
		}); err != nil {
			log.Printf("Failed to persist wallet refund + event for order %s: %v", order.ID, err)
			services.CaptureBackgroundError(err)
		}
		services.LogSystemAudit(c, "order.refund.to_wallet", "order", order.ID.String(), nil, map[string]any{
			"amount": refundAmount, "walletTxnId": txn.ID.String(), "reason": req.Reason, "initiatedBy": initiatedBy,
		})
		c.JSON(http.StatusOK, gin.H{
			"message":             "Refund credited to wallet",
			"refundAmount":        refundAmount,
			"provider":            "wallet",
			"walletTransactionId": txn.ID,
		})
		return
	}

	provider := strings.ToLower(order.PaymentProvider)
	if provider == "" {
		provider = "razorpay"
	}

	// Wallet-at-checkout refunds (#141): a wallet-funded order only captured
	// (Total − WalletApplied) at the gateway, so the gateway can't refund more than
	// that. Re-credit the wallet-covered slice as store credit and cap the gateway
	// refund to the captured amount. NOTE: direct-transfer top-ups
	// (settleWalletTopUps) are NOT auto-reversed by Razorpay Route the way
	// payment-linked transfers are — reversing them needs a transfer-reversal call,
	// tracked for the sandbox-verification follow-up before this flag goes live.
	if order.WalletApplied > 0 && provider != "wallet" {
		capture := order.Total - order.WalletApplied
		if refundAmount > capture {
			walletPortion := refundAmount - capture
			if _, werr := services.CreditWallet(database.DB, order.CustomerID, walletPortion,
				models.WalletSourceRefund, &order.ID,
				fmt.Sprintf("Wallet-portion refund for order %s: %s", order.OrderNumber, req.Reason),
				"refund-wallet:"+order.ID.String(), nil); werr != nil {
				log.Printf("wallet-portion re-credit failed order=%s: %v", order.OrderNumber, werr)
				services.CaptureBackgroundError(werr)
			}
			refundAmount = capture
		}
	}

	var refundID, refundStatus string

	switch provider {
	case "wallet":
		// Full-wallet order (no gateway payment): the entire refund returns as
		// store credit.
		txn, werr := services.CreditWallet(database.DB, order.CustomerID, refundAmount,
			models.WalletSourceRefund, &order.ID,
			fmt.Sprintf("Refund for order %s: %s", order.OrderNumber, req.Reason),
			"refund:"+order.ID.String(), nil)
		if werr != nil {
			log.Printf("wallet-order refund failed order=%s: %v", order.OrderNumber, werr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to credit wallet"})
			return
		}
		refundID = "wallet:" + txn.ID.String()
		refundStatus = "processed"
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
	// Persist the refund and stage the order.refunded event atomically.
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"payment_status":      models.PaymentRefunded,
			"status":              models.OrderStatusRefunded,
			"refund_id":           refundID,
			"refund_amount":       refundAmount,
			"refund_reason":       req.Reason,
			"refund_initiated_by": initiatedBy,
			"refunded_at":         &now,
		}).Error; err != nil {
			return err
		}
		if err := releaseRefundCapacity(tx); err != nil {
			return err
		}
		return services.EnqueueEvent(tx, "orders.refunded", "order.refunded", userID, map[string]interface{}{
			"order_id":      order.ID.String(),
			"order_number":  order.OrderNumber,
			"refund_amount": refundAmount,
			"reason":        req.Reason,
			"initiated_by":  initiatedBy,
			"refund_id":     refundID,
			"provider":      provider,
		})
	}); err != nil {
		log.Printf("Failed to persist refund + event for order %s: %v", order.ID, err)
		services.CaptureBackgroundError(err)
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
	} else {
		// The order just became paid — try the referral reward (#38). Idempotent
		// + best-effort: a referral failure must never affect the captured payment.
		var ord models.Order
		if err := database.DB.Select("id").Where("razorpay_order_id = ?", payment.OrderID).First(&ord).Error; err == nil {
			services.MaybeGrantReward(database.DB, ord.ID)
		}
	}
	// A post-delivery tip is a separate Razorpay order (#45); confirm it here too
	// (idempotent). Harmless no-op when payment.OrderID isn't a tip charge.
	markTipPaidByRazorpayOrder(payment.OrderID, payment.ID)
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

	// Customer meal subscription (#281): activate the cycle + generate the paid
	// invoice. If it matched a meal sub, we're done; else fall through to the
	// chef/driver SaaS subscription path below.
	if inv, mErr := services.ActivateMealSubscriptionOnCharge(database.DB, data.Subscription.Entity.ID, data.Payment.Entity.ID); mErr != nil {
		log.Printf("meal-subscription charge activation failed: %v", mErr)
	} else if inv != nil {
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

	// Customer meal subscription (#281): a failed recurring charge halts order
	// generation (past_due). If it matched a meal sub, we're done.
	if services.HaltMealSubscriptionOnFailure(database.DB, data.Subscription.Entity.ID) {
		return
	}

	// Mark subscription as suspended, then win-back (#42): offer a discounted
	// re-subscription to the suspended (payment-failed) chef/driver. Best-effort.
	var sub models.Subscription
	if err := database.DB.Where("gateway_sub_id = ?", data.Subscription.Entity.ID).First(&sub).Error; err != nil {
		return
	}
	database.DB.Model(&models.Subscription{}).Where("id = ?", sub.ID).Update("status", models.SubStatusSuspended)
	if _, werr := services.OfferWinback(database.DB, sub.UserID, string(sub.SubscriberType), models.WinbackTriggerSubSuspended, &sub.ID); werr != nil {
		log.Printf("winback: offer on suspend failed for user=%s: %v", sub.UserID, werr)
	}
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
		ID   string `json:"id"`
		Type string `json:"type"`
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
