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

	// Freeze the platform commission rate on the order ONCE at checkout (#390),
	// for BOTH gateway paths. After this, order.CommissionRate is the single source
	// for the chef/driver split — a later admin retune of the runtime rate can no
	// longer make the settlement statement disagree with the transfer already sent.
	// Skip if already frozen (idempotent on a retry of an unpaid order).
	if order.CommissionRate <= 0 {
		rate := services.GetCommissionRate(database.DB)
		database.DB.Model(&order).Update("commission_rate", rate)
		order.CommissionRate = rate // same request uses the frozen rate
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
		chefAmount := chefNetPayout(order)
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

// chefNetPayout is the chef's actual Route/Transfer payout for an order: NET of
// platform commission and TDS (#390). It is the SINGLE SOURCE OF TRUTH shared by
// the Route split, the Stripe transfer, and the FSSAI-withhold audit, so the three
// can never drift — and it equals ComputeOrderEarnings(order).NetPayout, the exact
// figure on the settlement statement.
//
//	gross = Subtotal + Tax + ChefTip (less any chef-funded promo the chef bears);
//	net   = gross − commission − TDS.
//
// The commission rate is read from the FROZEN order.CommissionRate (stamped at
// checkout), so a mid-flight admin rate change cannot make the statement disagree
// with a transfer already sent. A legacy 0 falls back to DefaultCommissionRate
// inside ComputeOrderEarnings. Delivery is the DRIVER's money and is excluded.
// Requires order.Chef preloaded (for the intra/inter-state GST state).
func chefNetPayout(order *models.Order) float64 {
	return services.ComputeOrderEarnings(services.EarningsInput{
		ItemRevenue:        order.Subtotal,
		Tax:                order.Tax,
		ChefTip:            order.ChefTip,
		DeliveryFee:        order.DeliveryFee,
		ChefFundedDiscount: order.ChefFundedDiscount,
		DeliveryState:      order.DeliveryAddressState,
		CommissionRate:     order.CommissionRate,
	}, order.Chef.State).NetPayout
}

// orderSettlements derives the chef + driver payouts for an order, chef first so
// its (larger) food payout stays a single payment-linked transfer when the capture
// allows (#141). The chef slice is NET (chefNetPayout, reading the frozen
// order.CommissionRate); the chef account is cleared when its FSSAI licence has
// lapsed, so that slice is withheld and never transferred. Requires Chef and
// Delivery.DeliveryPartner preloaded. Deterministic — because the rate is frozen
// on the row, create and verify produce the identical split for a given order.
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
		{Account: chefAccount, Amount: services.ToPaise(chefNetPayout(order)), Hold: true,
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
	settleWalletTopUpsWith(order.ID, order.OrderNumber, topUps, func(t services.TransferSpec) error {
		_, err := rz.CreateTransfer(&services.DirectTransferRequest{
			Account: t.Account, Amount: t.Amount, Currency: t.Currency, OnHold: t.OnHold, Notes: t.Notes,
		})
		return err
	})
}

// settleWalletTopUpsWith issues each platform-funded top-up transfer AT MOST ONCE per
// (order, account), idempotently (#554). A retried VerifyPayment used to re-issue the
// same real money transfer because CreateTransfer had no dedup. Now each (order,
// account) is claimed in the processed_events ledger before the transfer; a repeat
// settlement finds the claim and skips. On a transfer failure the claim is released so
// the NEXT settlement re-attempts it — so a gateway blip retries without ever
// double-paying. doTransfer is the gateway seam (real in prod, a fake in tests). A
// crash between claim and a successful transfer strands that one top-up (recoverable
// by the settlement reconcile — #398/#3), which is the safe side of the trade-off:
// never a double transfer.
func settleWalletTopUpsWith(orderID uuid.UUID, orderNumber string, topUps []services.TransferSpec, doTransfer func(services.TransferSpec) error) {
	for _, t := range topUps {
		firstTime, err := services.ClaimWalletTopUp(database.DB, orderID, t.Account)
		if err != nil {
			log.Printf("wallet-topup: claim failed order=%s account=%s: %v", orderNumber, t.Account, err)
			continue
		}
		if !firstTime {
			continue // already transferred for this (order, account) — no double
		}
		if err := doTransfer(t); err != nil {
			services.ReleaseWalletTopUp(database.DB, orderID, t.Account) // let a retry re-attempt
			log.Printf("wallet-topup: direct transfer failed order=%s account=%s amount=%d: %v",
				orderNumber, t.Account, t.Amount, err)
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

	// #555: guarded completion — emit order.paid + chef push ONLY on the single
	// pending→completed transition (a retried full-wallet settle no longer double-emits).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := completeOrderPaymentTx(tx, order,
			map[string]interface{}{
				"payment_method":   "wallet",
				"payment_provider": "wallet",
				"wallet_applied":   walletApplied,
			},
			map[string]interface{}{
				"order_id":     order.ID.String(),
				"order_number": order.OrderNumber,
				"amount":       order.Total,
				"method":       "wallet",
				"provider":     "wallet",
			})
		return err
	}); err != nil {
		log.Printf("full-wallet: mark-paid failed order=%s: %v", order.OrderNumber, err)
	} else {
		// Referral reward (#38) on the referee's first paid order — idempotent.
		services.MaybeGrantReward(database.DB, order.ID)
		// Start the durable order saga (#122) — gated, idempotent, no-op when off.
		services.StartOrderSaga(order.ID)
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
// account. Chef receives their NET payout (gross − commission − TDS, where gross
// is subtotal + tax + chefTip) via `transfer_data[destination]`; the platform
// retains the rest (commission + TDS + deliveryFee + driverTip) as
// `application_fee_amount` and settles the driver separately. Stripe rejects
// charges whose currency doesn't match the chef's Connect country, so we derive
// currency from PayoutCountry rather than hardcoding.
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

	// Chef receives NET: gross (subtotal + tax + chefTip, less any chef-funded
	// promo) minus commission and TDS (#390). Platform keeps the rest (commission +
	// TDS + deliveryFee + driverTip) as application_fee; the driver is paid out of
	// the platform balance via a follow-up Transfer on delivery confirmation.
	chefAmount := chefNetPayout(order)
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

// notifyChefNewOrderTx stages the actionable "new order" push to the chef
// within a payment-completion transaction. Orders are created pre-payment, so
// the chef is only notified once money is captured (previously this fired in
// CreateOrder, pushing unpaid/abandoned orders to the kitchen — see
// handlers/orders.go). Callers MUST guard the call on the pending→completed
// transition so a webhook arriving after the client verify (or vice-versa)
// doesn't double-notify; the OrderEvent mirrors the one CreateOrder used.
func notifyChefNewOrderTx(tx *gorm.DB, order *models.Order) error {
	return services.EnqueueOrderEvent(tx, services.SubjectChefNewOrder, services.OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	})
}

// completionBlockedStatuses are the payment states a success/capture/verify must NOT
// re-complete from: already `completed` (idempotent — no duplicate side effects) and
// `refunded` (a refund is terminal; re-stamping it `completed` would silently re-enable
// the chef payout on money already returned to the customer — #563). `failed` is
// deliberately NOT here, so a retry-after-decline can still complete the order.
var completionBlockedStatuses = []models.PaymentStatus{models.PaymentCompleted, models.PaymentRefunded}

// completeOrderPaymentTx flips an order pending→completed exactly ONCE and, only on
// that single transition, notifies the chef and emits order.paid — the provider-generic
// core shared by the Razorpay / Stripe / wallet verify+settle paths (#395 item 2, #555).
// Every one of them used to read `wasUnpaid` from the in-memory order and then update
// unconditionally + emit order.paid unconditionally, so a webhook or re-verify that
// completed the order underneath left a duplicate chef "new order" push AND a duplicate
// order.paid event. The guarded UPDATE (WHERE payment_status <> 'completed') makes
// exactly one racing path perform the transition; the chef notify + event fire only on
// RowsAffected>0. `updates` are the provider-specific columns to stamp alongside
// payment_status=completed; `event` is the order.paid payload. Returns whether THIS call
// performed the transition.
func completeOrderPaymentTx(tx *gorm.DB, order *models.Order, updates, event map[string]interface{}) (bool, error) {
	cols := map[string]interface{}{"payment_status": models.PaymentCompleted}
	for k, v := range updates {
		cols[k] = v
	}
	res := tx.Model(&models.Order{}).
		Where("id = ? AND payment_status NOT IN ?", order.ID, completionBlockedStatuses).
		Updates(cols)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil // a concurrent webhook/verify already completed it — no re-fire
	}
	// Keep the in-memory order consistent for any post-tx idempotent steps.
	order.PaymentStatus = models.PaymentCompleted
	if err := notifyChefNewOrderTx(tx, order); err != nil {
		return false, err
	}
	if err := services.EnqueueEvent(tx, "orders.paid", "order.paid", order.CustomerID, event); err != nil {
		return false, err
	}
	return true, nil
}

// completeRazorpayOrderTx is the Razorpay-specific wrapper over completeOrderPaymentTx
// (#395). The caller's post-tx steps (referral reward, saga, wallet DEBIT) are
// idempotently keyed; the wallet TOP-UP transfer is NOT (see #395 follow-up) — but this
// helper does not invoke it, and on a raced (RowsAffected==0) verify the top-up
// recomputes the same deterministic split, so this does not add a double-transfer path.
func completeRazorpayOrderTx(tx *gorm.DB, order *models.Order, method, paymentID string, amountPaise int) (bool, error) {
	return completeOrderPaymentTx(tx, order,
		map[string]interface{}{"payment_method": method, "razorpay_payment_id": paymentID},
		map[string]interface{}{
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"amount":       services.FromPaise(amountPaise),
			"method":       method,
			"provider":     "razorpay",
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
		h.verifyRazorpayPayment(c, &order, req.RazorpayPaymentID, req.RazorpayOrderID, req.RazorpaySignature)
	}
}

func (h *PaymentHandler) verifyRazorpayPayment(c *gin.Context, order *models.Order, paymentID, rzOrderID, signature string) {
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

	// SECURITY: bind the fetched payment to THIS order and its amount. Without
	// these checks any captured payment on the merchant account (e.g. a ₹1
	// payment reused across orders) would settle this order. payment.OrderID and
	// payment.Amount come from Razorpay (via FetchPayment), so they can't be
	// forged by the client.
	if payment.OrderID != order.RazorpayOrderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment does not belong to this order"})
		return
	}
	expectedPaise := services.ToPaise(order.Total) - services.ToPaise(order.WalletApplied)
	if expectedPaise < 0 {
		expectedPaise = 0
	}
	if payment.Amount < expectedPaise {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment amount does not match the order total"})
		return
	}
	// Verify the Checkout signature (order_id|payment_id) the client received
	// from Razorpay. Enforced when present (the customer app always sends it);
	// tolerated-if-absent since the binding + amount checks above are the hard
	// gate and don't rely on the client.
	if signature != "" && !services.VerifyPaymentSignature(rzOrderID, paymentID, signature) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment signature verification failed"})
		return
	}

	// Mark the order paid and stage the chef push + order.paid event atomically
	// (transactional outbox). Payment already captured at the gateway, so a DB hiccup
	// must not 500 the client — it's logged + sent to Sentry and the reconciliation
	// cron catches any drift.
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := completeRazorpayOrderTx(tx, order, payment.Method, paymentID, payment.Amount)
		return err
	}); err != nil {
		log.Printf("Failed to persist payment completion + event for order %s: %v", order.ID, err)
		services.CaptureBackgroundError(err)
	} else {
		// Referral reward (#38) on the referee's first paid order — idempotent,
		// so a later webhook for the same order won't double-pay.
		services.MaybeGrantReward(database.DB, order.ID)
		// Start the durable order saga (#122) — gated, idempotent, no-op when off.
		services.StartOrderSaga(order.ID)
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
		// orderSettlements recomputes the IDENTICAL split here: the order was
		// reloaded with its persisted commission_rate, and chefNetPayout reads that
		// frozen rate — so create and verify are deterministic, never re-resolving
		// the live rate (#390).
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

	// Mark paid + stage the chef push + order.paid event atomically (transactional
	// outbox). #555: the guarded completeOrderPaymentTx emits order.paid ONLY on the
	// single pending→completed transition — a re-verify or a verify/webhook race no
	// longer double-emits (the old path here updated + emitted unconditionally).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := completeOrderPaymentTx(tx, order,
			map[string]interface{}{"payment_method": "card"},
			map[string]interface{}{
				"order_id":     order.ID.String(),
				"order_number": order.OrderNumber,
				"amount":       services.FromMinor(pi.Amount, eventCurrency),
				"method":       "card",
				"provider":     "stripe",
				"currency":     order.Currency,
			})
		return err
	}); err != nil {
		log.Printf("Failed to persist payment completion + event for order %s: %v", order.ID, err)
		services.CaptureBackgroundError(err)
	} else {
		// Start the durable order saga (#122) — gated, idempotent, no-op when off.
		services.StartOrderSaga(order.ID)
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

	// #394: refuse orders that are refund-managed by a typed escrow flow. A
	// meal-plan-day or group order spawns a regular Order reachable here, but its
	// refund keyspace (mealplan-refund:<dayID> / grouporder-refund:<id>) is disjoint
	// from this endpoint's refund:<orderID> and bypasses Order.RefundAmount — a
	// generic refund would credit the customer a second time AND leave the chef's
	// held direct transfer unreversed. Route the caller to the correct flow.
	switch kind, kErr := services.TypedRefundOrderKind(database.DB, orderID); {
	case kErr != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check order type"})
		return
	case kind == services.TypedRefundMealPlanDay:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "This order is part of a meal plan; refund it through the meal-plan refund flow, not the generic order refund"})
		return
	case kind == services.TypedRefundGroupOrder:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "This is a group order; refund participants through the group-order cancellation flow, not the generic order refund"})
		return
	}

	if order.PaymentStatus != models.PaymentCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Can only refund completed payments"})
		return
	}

	// Validate refund amount against what is STILL refundable. Partial refunds
	// via the chef-cancel path increment order.RefundAmount but leave the order
	// "completed", so we must subtract them — otherwise a chef could partial-refund
	// via that path and then refund the whole order total again here (and the
	// to-wallet branch would credit that as platform-funded store credit,
	// bypassing the gateway's cumulative-refund ceiling).
	if req.Amount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refund amount cannot be negative"})
		return
	}
	// #560/#527: RemainingRefundable, NOT Total − RefundAmount — after a per-line cancel
	// the naive difference double-subtracts the refunded lines (Total was already reduced)
	// and would wrongly cap this refund at ~0, stranding the remaining live items' money.
	remaining := services.RemainingRefundable(&order)
	if remaining <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order has already been fully refunded"})
		return
	}
	refundAmount := req.Amount
	if refundAmount == 0 {
		refundAmount = remaining // full refund of whatever is still refundable
	}
	if refundAmount > remaining {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refund amount cannot exceed the remaining refundable amount"})
		return
	}

	// Release reserved daily capacity (#48) only on a FULL refund of an order that
	// hasn't been delivered yet — those dishes won't be made. Partial and
	// post-delivery refunds keep the capacity consumed.
	releaseCapOnRefund := (order.RefundAmount+refundAmount) >= order.Total &&
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
				"refund_amount":       order.RefundAmount + refundAmount,
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
		// Cross-guard the payout hold (#457): this primary refund endpoint is
		// reachable on an already-released order, so drive the hold to
		// withheld/reversed. Best-effort — never change the HTTP response.
		if hErr := services.WithholdOrReverseOrderHoldForRefund(database.DB, order.ID, req.Reason); hErr != nil {
			log.Printf("payout cross-guard failed for wallet-refunded order %s: %v", order.ID, hErr)
			services.CaptureBackgroundError(hErr)
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

	// SECURITY: atomically claim the refund before calling the gateway. The
	// gateway CreateRefund calls carry no idempotency key, so two concurrent
	// requests (double-click / client retry) could otherwise both issue a real
	// refund. Flip completed→refunded in one conditional UPDATE; only the winner
	// (RowsAffected==1) proceeds. On any downstream failure we revert to
	// completed so the refund can be retried.
	claim := database.DB.Model(&models.Order{}).
		Where("id = ? AND payment_status = ?", order.ID, models.PaymentCompleted).
		Update("payment_status", models.PaymentRefunded)
	if claim.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
		return
	}
	if claim.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "A refund for this order is already in progress or has completed"})
		return
	}
	revertClaim := func() {
		database.DB.Model(&models.Order{}).Where("id = ?", order.ID).
			Update("payment_status", models.PaymentCompleted)
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
			revertClaim()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to credit wallet"})
			return
		}
		refundID = "wallet:" + txn.ID.String()
		refundStatus = "processed"
	case "stripe":
		if order.StripePaymentIntentID == "" {
			revertClaim()
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Stripe payment found for this order"})
			return
		}
		st := services.GetStripe()
		if st == nil {
			revertClaim()
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
			revertClaim()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
			return
		}
		refundID = r.ID
		refundStatus = r.Status
	default: // razorpay
		if order.RazorpayPaymentID == "" {
			revertClaim()
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Razorpay payment found for this order"})
			return
		}
		rz := services.GetRazorpay()
		if rz == nil {
			revertClaim()
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
			revertClaim()
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
			"refund_amount":       order.RefundAmount + refundAmount,
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
	// Cross-guard the payout hold (#457): the gateway-persist branch sets
	// status=refunded, so drive the hold to withheld/reversed. Best-effort — never
	// change the HTTP response.
	if hErr := services.WithholdOrReverseOrderHoldForRefund(database.DB, order.ID, req.Reason); hErr != nil {
		log.Printf("payout cross-guard failed for refunded order %s: %v", order.ID, hErr)
		services.CaptureBackgroundError(hErr)
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

	// Event-level replay dedup (#462). The signature proves authenticity, not
	// freshness — a provider retry or a captured+replayed event would re-fire the
	// handlers' side effects (the chef new-order push, referral grant) that aren't
	// covered by the per-effect conditional-UPDATE guards. Claim (webhook:razorpay,
	// event-id) AFTER the signature check so a forged request can't poison the
	// ledger. Header id when Razorpay sends one, else a body hash.
	const rzConsumer = "webhook:razorpay"
	eventID := services.WebhookEventID(c.GetHeader("X-Razorpay-Event-Id"), body)
	firstTime, err := services.ClaimWebhookEvent(database.DB, rzConsumer, eventID, event.Event)
	if err != nil {
		log.Printf("razorpay webhook: claim failed for %s: %v", eventID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dedup unavailable"})
		return
	}
	if !firstTime {
		log.Printf("razorpay webhook: duplicate event %s (%s) — skipping", eventID, event.Event)
		c.JSON(http.StatusOK, gin.H{"status": "duplicate"})
		return
	}

	var derr error
	switch event.Event {
	case "payment.captured":
		derr = h.handlePaymentCaptured(event.Payload)
	case "payment.failed":
		derr = h.handlePaymentFailed(event.Payload)
	case "refund.processed":
		derr = h.handleRefundProcessed(event.Payload)
	case "transfer.processed":
		log.Printf("Transfer processed event received")
	case "subscription.charged":
		derr = h.handleSubscriptionCharged(event.Payload)
	case "subscription.halted":
		derr = h.handleSubscriptionHalted(event.Payload)
	default:
		log.Printf("Unhandled Razorpay webhook event: %s", event.Event)
	}
	if derr != nil {
		// A transient processing failure releases the claim so a later (re)delivery
		// of this event id can re-run — otherwise the dedup would strand the event.
		// We still ACK 200 (Razorpay isn't retried on 5xx here today either).
		services.ReleaseWebhookEvent(database.DB, rzConsumer, eventID)
		log.Printf("razorpay webhook: handler error for %s (%s): %v", eventID, event.Event, derr)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handlePaymentCaptured returns a non-nil error only for a TRANSIENT failure (a DB
// error) so the webhook layer releases the dedup claim and a redelivery re-runs.
// A parse failure is permanent (nil → keep the claim; a retry won't parse either).
func (h *PaymentHandler) handlePaymentCaptured(payload json.RawMessage) error {
	var data struct {
		Payment struct {
			Entity services.PaymentResponse `json:"entity"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse payment.captured payload: %v", err)
		return nil
	}

	payment := data.Payment.Entity
	log.Printf("Payment captured: %s (order: %s, amount: %d)", payment.ID, payment.OrderID, payment.Amount)

	// Idempotent update: only flip orders that aren't already completed OR refunded.
	// Razorpay retries webhooks; without the `completed` guard repeated deliveries would
	// re-emit downstream effects, and without the `refunded` guard a late/duplicate
	// capture could re-stamp a refunded order back to completed (#563).
	res := database.DB.Model(&models.Order{}).
		Where("razorpay_order_id = ? AND payment_status NOT IN ?", payment.OrderID, completionBlockedStatuses).
		Updates(map[string]interface{}{
			"payment_status":      models.PaymentCompleted,
			"payment_method":      payment.Method,
			"razorpay_payment_id": payment.ID,
		})
	if res.Error != nil {
		log.Printf("Failed to apply payment.captured for order %s: %v", payment.OrderID, res.Error)
		return res.Error
	}
	if res.RowsAffected == 0 {
		log.Printf("payment.captured already processed for order %s (payment %s) — skipping", payment.OrderID, payment.ID)
	} else {
		// The order just became paid — try the referral reward (#38). Idempotent
		// + best-effort: a referral failure must never affect the captured payment.
		var ord models.Order
		if err := database.DB.Where("razorpay_order_id = ?", payment.OrderID).First(&ord).Error; err == nil {
			services.MaybeGrantReward(database.DB, ord.ID)
			// Start the durable order saga (#122) — gated, idempotent, no-op when off.
			services.StartOrderSaga(ord.ID)
			// Notify the chef now that the order is paid. The conditional update
			// above (RowsAffected > 0) guarantees this is the single
			// pending→completed transition, so the client verify path won't also
			// fire it. Best-effort — a notify failure must not affect the payment.
			if err := notifyChefNewOrderTx(database.DB, &ord); err != nil {
				log.Printf("Failed to enqueue chef new-order push for order %s: %v", ord.ID, err)
				services.CaptureBackgroundError(err)
			}
		}
	}
	// A post-delivery tip is a separate Razorpay order (#45); confirm it here too
	// (idempotent). Harmless no-op when payment.OrderID isn't a tip charge.
	markTipPaidByRazorpayOrder(payment.OrderID, payment.ID)
	return nil
}

func (h *PaymentHandler) handlePaymentFailed(payload json.RawMessage) error {
	var data struct {
		Payment struct {
			Entity services.PaymentResponse `json:"entity"`
		} `json:"payment"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse payment.failed payload: %v", err)
		return nil
	}

	payment := data.Payment.Entity
	log.Printf("Payment failed: %s (order: %s)", payment.ID, payment.OrderID)

	// Idempotent: only transition from a non-terminal state. Avoids
	// overwriting a completed payment if events arrive out-of-order.
	return database.DB.Model(&models.Order{}).
		Where("razorpay_order_id = ? AND payment_status NOT IN ?", payment.OrderID, []models.PaymentStatus{
			models.PaymentCompleted,
			models.PaymentFailed,
			models.PaymentRefunded,
		}).
		Update("payment_status", models.PaymentFailed).Error
}

func (h *PaymentHandler) handleRefundProcessed(payload json.RawMessage) error {
	var data struct {
		Refund struct {
			Entity services.RefundResponse `json:"entity"`
		} `json:"refund"`
	}
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("Failed to parse refund.processed payload: %v", err)
		return nil
	}

	refund := data.Refund.Entity
	log.Printf("Refund processed: %s (payment: %s, amount: %d)", refund.ID, refund.PaymentID, refund.Amount)

	// Idempotent: skip if we already stamped this refund_id on the order
	// (Razorpay retries refund.processed; without this we'd re-write the
	// refunded_at timestamp on every redelivery).
	now := time.Now()
	return database.DB.Model(&models.Order{}).
		Where("razorpay_payment_id = ? AND (refund_id IS NULL OR refund_id <> ?)", refund.PaymentID, refund.ID).
		Updates(map[string]interface{}{
			"refund_id":   refund.ID,
			"refunded_at": &now,
		}).Error
}

// handleSubscriptionCharged is best-effort (subscription billing, not escrow) —
// it returns nil so the dedup claim is kept regardless; failures are logged.
func (h *PaymentHandler) handleSubscriptionCharged(payload json.RawMessage) error {
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
		return nil
	}

	// Customer meal subscription (#281): activate the cycle + generate the paid
	// invoice. If it matched a meal sub, we're done; else fall through to the
	// chef/driver SaaS subscription path below.
	if inv, mErr := services.ActivateMealSubscriptionOnCharge(database.DB, data.Subscription.Entity.ID, data.Payment.Entity.ID); mErr != nil {
		log.Printf("meal-subscription charge activation failed: %v", mErr)
	} else if inv != nil {
		return nil
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
	return nil
}

// handleSubscriptionHalted is best-effort (subscription billing) — returns nil.
func (h *PaymentHandler) handleSubscriptionHalted(payload json.RawMessage) error {
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
		return nil
	}

	// Customer meal subscription (#281): a failed recurring charge halts order
	// generation (past_due). If it matched a meal sub, we're done.
	if services.HaltMealSubscriptionOnFailure(database.DB, data.Subscription.Entity.ID) {
		return nil
	}

	// Mark subscription as suspended, then win-back (#42): offer a discounted
	// re-subscription to the suspended (payment-failed) chef/driver. Best-effort.
	var sub models.Subscription
	if err := database.DB.Where("gateway_sub_id = ?", data.Subscription.Entity.ID).First(&sub).Error; err != nil {
		return nil
	}
	database.DB.Model(&models.Subscription{}).Where("id = ?", sub.ID).Update("status", models.SubStatusSuspended)
	if _, werr := services.OfferWinback(database.DB, sub.UserID, string(sub.SubscriberType), models.WinbackTriggerSubSuspended, &sub.ID); werr != nil {
		log.Printf("winback: offer on suspend failed for user=%s: %v", sub.UserID, werr)
	}
	return nil
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

	// Guarded completion mirroring handlePaymentCaptured (#563 — this was an
	// UNCONDITIONAL update that would re-stamp a refunded/failed order back to completed
	// on a webhook replay). Only flip an order that isn't already completed/refunded; the
	// chef notify + reward + saga fire on the single transition (RowsAffected > 0), so the
	// client verify path (verifyStripePayment) and this webhook can't both re-fire them.
	res := database.DB.Model(&models.Order{}).
		Where("stripe_payment_intent_id = ? AND payment_status NOT IN ?", pi.ID, completionBlockedStatuses).
		Updates(map[string]interface{}{
			"payment_status": models.PaymentCompleted,
			"payment_method": "card",
		})
	if res.Error != nil {
		log.Printf("Failed to apply stripe payment_intent.succeeded for %s: %v", pi.ID, res.Error)
		return
	}
	if res.RowsAffected == 0 {
		return // already completed/refunded — dup delivery or raced with verify
	}
	var ord models.Order
	if err := database.DB.Where("stripe_payment_intent_id = ?", pi.ID).First(&ord).Error; err != nil {
		// Order was just marked completed but we can't re-read it — the chef notify +
		// saga won't fire. Surface it (reconciliation is the backstop) rather than drop silently.
		log.Printf("stripe succeeded: completed order for intent %s but re-read failed: %v", pi.ID, err)
		services.CaptureBackgroundError(err)
		return
	}
	services.MaybeGrantReward(database.DB, ord.ID)
	services.StartOrderSaga(ord.ID)
	if err := notifyChefNewOrderTx(database.DB, &ord); err != nil {
		log.Printf("Failed to enqueue chef new-order push for order %s: %v", ord.ID, err)
		services.CaptureBackgroundError(err)
	}
}

func (h *PaymentHandler) handleStripePaymentFailed(obj json.RawMessage) {
	var pi services.StripePaymentIntent
	if err := json.Unmarshal(obj, &pi); err != nil {
		log.Printf("Failed to parse stripe payment_intent: %v", err)
		return
	}
	log.Printf("Stripe payment failed: %s", pi.ID)

	// Guarded: only transition from a non-terminal state (#563 — was UNCONDITIONAL and
	// could overwrite a completed/refunded order on an out-of-order/duplicate delivery).
	if err := database.DB.Model(&models.Order{}).
		Where("stripe_payment_intent_id = ? AND payment_status NOT IN ?", pi.ID, []models.PaymentStatus{
			models.PaymentCompleted, models.PaymentFailed, models.PaymentRefunded,
		}).
		Update("payment_status", models.PaymentFailed).Error; err != nil {
		log.Printf("Failed to apply stripe payment_intent.payment_failed for %s: %v", pi.ID, err)
	}
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
