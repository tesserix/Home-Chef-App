package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
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

	// Wallet + loyalty credit to apply at checkout. The client sends INTENT — which
	// rails, and optionally how much of each — and the server recomputes the whole
	// allocation from live balances and the real order. Its answer is the only one
	// that counts.
	//
	// This is deliberate: the client used to compute a payable from its own cached
	// balance and post a rupee amount, so any drift between the two views showed the
	// customer one figure and charged another. An older build posting a bare
	// {"walletAmount": N} is read as an explicit wallet request.
	rawBody, _ := io.ReadAll(c.Request.Body)
	var creditReq services.CreditRequest
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &creditReq)
	}
	if !creditReq.UseWallet && creditReq.WalletAmount != nil && *creditReq.WalletAmount > 0 {
		creditReq.UseWallet = true
	}

	// A body-less call is a RETRY of an order whose credit was already chosen — the
	// "Pay now" button on an unpaid order, which has no checkout screen to ask
	// again. Reuse what is stamped on the order rather than reading the silence as
	// "no credit", which would quietly charge the customer the full total and drop
	// the credit they had already applied.
	if len(bytes.TrimSpace(rawBody)) <= 2 { // "" or "{}"
		if order.WalletApplied > 0 {
			amt := order.WalletApplied
			creditReq.UseWallet, creditReq.WalletAmount = true, &amt
		}
		if order.LoyaltyPointsSpent > 0 {
			pts := order.LoyaltyPointsSpent
			creditReq.UseLoyalty, creditReq.LoyaltyPoints = true, &pts
		}
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

	// Resolve the gateway from the chef's configured provider. The branch taken is
	// also what stamps order.payment_provider, so the gateway that takes the money
	// and the gateway a later refund goes to cannot disagree. Anything selection
	// cannot charge is refused rather than sent down a gateway's branch by
	// default — that fallthrough is how a Cashfree order used to reach the
	// Razorpay code and fail on an empty payment id.
	switch provider := services.SelectCheckoutGateway(order.Chef.PaymentProvider, order.Mode); provider {
	case models.PaymentProviderStripe:
		h.createStripePayment(c, &order, userID)
	case models.PaymentProviderCashfree:
		h.createCashfreePayment(c, &order, userID, creditReq)
	default:
		log.Printf("checkout: no charge path for provider %q on order %s", provider, order.OrderNumber)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payments aren't available for this kitchen right now"})
	}
}

// settleFullWalletOrder handles an order fully covered by credit: there is no
// gateway payment, so the chef/driver are paid entirely from the platform balance,
// the wallet and points are debited, and the order is marked paid (#141).
func (h *PaymentHandler) settleFullWalletOrder(c *gin.Context, order *models.Order, plan services.FundingPlan, walletApplied, loyaltyApplied, pointsSpent float64) {
	order.WalletApplied = walletApplied
	order.LoyaltyApplied = loyaltyApplied
	order.LoyaltyPointsSpent = pointsSpent
	if err := services.DebitOrderWallet(order); err != nil {
		log.Printf("full-wallet: debit failed order=%s: %v", order.OrderNumber, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not apply wallet credit"})
		return
	}
	if pointsSpent > 0 {
		if err := services.RedeemLoyaltyToOrder(database.DB, order.CustomerID, order.ID, pointsSpent); err != nil {
			log.Printf("full-wallet: loyalty debit failed order=%s: %v", order.OrderNumber, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Could not apply loyalty points"})
			return
		}
	}

	// #555: guarded completion — emit order.paid + chef push ONLY on the single
	// pending→completed transition (a retried full-wallet settle no longer double-emits).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := services.CompleteOrderPaymentTx(tx, order,
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
		// Store credit covered the whole total — still a payment, and the only
		// confirmation the customer gets (#notify-money).
		services.NotifyPaymentSucceeded(database.DB, order.ID)
	}

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
	chefAmount := services.ChefNetPayoutFor(order)
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
		Description:         fmt.Sprintf("Fe3dr order %s", order.OrderNumber),
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
		// Cashfree. Only the order id — there is no client-side payment id or
		// signature to send, and none would be trusted: the Cashfree leg reads
		// the captured payment from the gateway itself.
		CashfreeOrderID string `json:"cashfreeOrderId"`
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

	switch provider := models.NormalizeProvider(order.PaymentProvider); provider {
	case models.PaymentProviderStripe:
		h.verifyStripePayment(c, &order, req.StripePaymentIntentID)
	case models.PaymentProviderCashfree:
		h.verifyCashfreePayment(c, &order, req.CashfreeOrderID)
	default:
		// Legacy Razorpay orders and anything unrecognised. No order can be
		// captured on Razorpay since #1101, so there is nothing here to verify
		// against it (#1086).
		c.JSON(http.StatusBadRequest, gin.H{"error": "This order was paid through a gateway that is no longer in service — contact support"})
	}
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
	// outbox). #555: the guarded services.CompleteOrderPaymentTx emits order.paid ONLY on the
	// single pending→completed transition — a re-verify or a verify/webhook race no
	// longer double-emits (the old path here updated + emitted unconditionally).
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		_, err := services.CompleteOrderPaymentTx(tx, order,
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
// refundWalletIdempotencyKey makes a refund-to-wallet credit idempotent per refund
// INSTANCE, not per order. It folds in how much was already refunded before this one
// (paise): a legitimate second partial goodwill refund — which #549 made repeatable by
// leaving the order non-terminal — gets a fresh key and actually credits, while a retry
// of the SAME refund (prior-refunded total unchanged) collides on the key and no-ops.
// Pre-#549 the per-order key was safe only because the first refund flipped the order
// terminal and blocked all repeats; now that repeats are legitimate, a per-order key
// would silently short the customer on the second refund while still clawing back the chef.
func refundWalletIdempotencyKey(prefix string, orderID uuid.UUID, priorRefunded float64) string {
	return fmt.Sprintf("%s:%s:%d", prefix, orderID, services.ToPaise(priorRefunded))
}

// claimRefundForProcessing atomically claims an order for refund processing by
// flipping completed→refunded in ONE conditional UPDATE. Returns won=true only for
// the caller that wins the flip; a concurrent duplicate (double-click / client retry)
// sees payment_status already flipped and gets won=false → the handler answers 409.
// This is the single serialization point shared by the to-wallet and gateway refund
// branches (#567 — the wallet branch previously had no such mutex, so two concurrent
// submits could lose-update refund_amount and double-claw the chef). The refund path
// reverts to completed after a PARTIAL refund so sequential partial goodwill refunds
// can re-claim (#549); a FULL refund keeps it refunded.
func claimRefundForProcessing(db *gorm.DB, orderID uuid.UUID) (bool, error) {
	res := db.Model(&models.Order{}).
		Where("id = ? AND payment_status = ?", orderID, models.PaymentCompleted).
		Update("payment_status", models.PaymentRefunded)
	return res.RowsAffected == 1, res.Error
}

// crossGuardRefundHold drives the payout hold to the state a refund implies. A
// FULL refund withholds/reverses the whole hold; a PARTIAL one leaves it
// releasable — the chef eats the refunded amount and keeps the remainder (#549),
// which the statement path already reflects, so there is nothing to move here.
func crossGuardRefundHold(orderID uuid.UUID, refundAmount float64, reason string, fullRefund, persistOK bool) error {
	if fullRefund {
		return services.WithholdOrReverseOrderHoldForRefund(database.DB, orderID, reason)
	}
	return nil
}

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

	// Best-effort pre-lock UX validation (friendly 400s for the common non-racing case).
	// The AUTHORITATIVE cap + claim + reserve happens under a row lock in
	// services.ReserveRefund below. Partial refunds via the chef-cancel path increment
	// order.RefundAmount but leave the order "completed", so RemainingRefundable subtracts
	// them — otherwise a chef could partial-refund there and then refund the whole total again.
	if req.Amount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refund amount cannot be negative"})
		return
	}
	// #560/#527: RemainingRefundable, NOT Total − RefundAmount — after a per-line cancel
	// the naive difference double-subtracts the refunded lines (Total was already reduced)
	// and would wrongly cap this refund at ~0, stranding the remaining live items' money.
	if pre := services.RemainingRefundable(&order); pre <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order has already been fully refunded"})
		return
	} else if req.Amount > pre {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refund amount cannot exceed the remaining refundable amount"})
		return
	}

	// #600: dedup a client retry-after-success. Once a PARTIAL refund fully succeeds RefundAmount
	// advances, so an app re-submitting the identical "refund ₹X" (its HTTP response was dropped)
	// computes a new gateway key + re-reserves → a second real refund. Claim the submission BEFORE
	// the reserve — keyed by the caller's Idempotency-Key header when present, else the
	// order+amount+reason within a short window. A duplicate returns idempotently without
	// refunding. Released on any pre-commit failure (via releaseReservation) so a legit retry
	// re-attempts; KEPT once the money moves so a retry dedups.
	clientKey := services.NormalizeRefundClientKey(c.GetHeader("Idempotency-Key"))
	proceed, dedupKey, dErr := services.ClaimRefundRequest(database.DB, order.ID, clientKey, int64(roundPaise(req.Amount)), req.Reason)
	if dErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
		return
	}
	if !proceed {
		var cur models.Order
		_ = database.DB.Where("id = ?", order.ID).First(&cur).Error
		c.JSON(http.StatusOK, gin.H{
			"message": "Duplicate refund request ignored", "duplicate": true,
			"refundAmount": cur.RefundAmount,
		})
		return
	}

	// #611: reserve the refund UNDER A ROW LOCK — the same discipline the partial path
	// (RefundIssueToWallet) uses. ReserveRefund reads remaining + claims payment_status
	// (completed→refunded, the concurrency mutex) + increments refund_amount atomically in
	// ONE transaction, so a concurrent partial committing between the read and the claim can
	// no longer over-refund live customer money or clobber the increment. This replaces the
	// old unlocked RemainingRefundable read + separate claimRefundForProcessing + the stale
	// `order.RefundAmount + refundAmount` read-modify-write persists below.
	//   - reserved:      the amount reserved = min(req.Amount-or-full, fresh remaining); what we refund.
	//   - priorRefunded: refund_amount observed UNDER THE LOCK — the prior-cumulative basis for the
	//                    idempotency keys (replaces the stale in-memory order.RefundAmount).
	//   - fullRefund:    true iff the reservation exhausted the remaining → the caller stamps the
	//                    terminal marker (a PARTIAL must leave status/payment_status/refunded_at
	//                    untouched so the payout hold stays releasable for the remainder, #549).
	//   - won:           false ⇒ a sibling refund path claimed it first → 409.
	reserved, priorRefunded, fullRefund, won, rErr := services.ReserveRefund(database.DB, order.ID, req.Amount)
	if rErr != nil {
		services.ReleaseRefundRequestClaim(database.DB, dedupKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
		return
	}
	if !won {
		services.ReleaseRefundRequestClaim(database.DB, dedupKey)
		c.JSON(http.StatusConflict, gin.H{"error": "A refund for this order is already in progress or has completed"})
		return
	}
	refundAmount := reserved
	// On any downstream gateway/wallet failure, release the reservation (revert payment_status
	// → completed AND decrement refund_amount) so a retry can re-reserve. ALWAYS releases the
	// FULL reserved amount: the wallet-at-checkout split below lowers the gateway `refundAmount`,
	// but the reservation covers the whole customer refund, so releasing only the lowered gateway
	// portion would leave refund_amount inflated by the wallet slice (#609 wallet-split gotcha).
	// #600: also drop the refund-request claim so a legit retry re-attempts (money didn't move);
	// a persisted / gateway-succeeded refund never calls this, so its claim persists → dedups.
	releaseReservation := func() {
		services.ReleaseRefundReservation(database.DB, order.ID, reserved)
		services.ReleaseRefundRequestClaim(database.DB, dedupKey)
	}

	// Release reserved daily capacity (#48) only on a FULL refund of an order that
	// hasn't been delivered yet — those dishes won't be made. Partial and
	// post-delivery refunds keep the capacity consumed. Uses the same fullRefund
	// (RemainingRefundable-based) notion as the terminal-marker decision, so a
	// partial refund after a per-line cancel can't wrongly free the remaining
	// dishes' capacity (the old `>= order.Total` double-counted per-line cancels).
	releaseCapOnRefund := fullRefund &&
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
	// to THIS refund instance (order + prior-refunded) so a retry can't double-credit
	// yet a legitimate second partial refund still credits (#549). Doesn't touch the
	// chef/driver splits — the original payment already settled.
	if req.ToWallet {
		// On-demand guard (refund-v2 policy): a plain à-la-carte "menu" order NEVER refunds to
		// the wallet — only meal-plan and group orders are wallet-eligible. Reject and require the
		// original payment method. Gated so behaviour is unchanged until the v2 flow is enabled.
		if services.MealPlanRefundFlowV2Active() && !services.WalletRefundEligible(database.DB, order.ID) {
			releaseReservation()
			c.JSON(http.StatusBadRequest, gin.H{"error": "This order can only be refunded to the original payment method, not the wallet"})
			return
		}
		txn, werr := services.CreditWallet(database.DB, order.CustomerID, refundAmount,
			models.WalletSourceRefund, &order.ID,
			fmt.Sprintf("Refund for order %s: %s", order.OrderNumber, req.Reason),
			refundWalletIdempotencyKey("refund", order.ID, priorRefunded), nil)
		if werr != nil {
			log.Printf("refund-to-wallet failed for order %s: %v", order.OrderNumber, werr)
			releaseReservation()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to credit wallet"})
			return
		}
		// Persist the refund and stage the order.refunded event atomically. Target
		// the row by id (not the preloaded &order struct) so we don't spuriously
		// upsert the belongs-to Chef association on a refund. #611: refund_amount was
		// already incremented atomically by ReserveRefund — never re-write it here (that
		// was the stale read-modify-write that clobbered a concurrent partial's increment).
		refundUpdates := map[string]interface{}{
			"refund_id":           "wallet:" + txn.ID.String(),
			"refund_reason":       req.Reason,
			"refund_initiated_by": initiatedBy,
		}
		if fullRefund {
			now := time.Now()
			refundUpdates["payment_status"] = models.PaymentRefunded
			refundUpdates["status"] = models.OrderStatusRefunded
			refundUpdates["refunded_at"] = &now
		} else {
			// PARTIAL (#549/#567): the claim above flipped completed→refunded as the
			// concurrency mutex; revert it so the payout hold stays releasable and
			// sequential partial goodwill refunds can proceed. Leave status/refunded_at unset.
			refundUpdates["payment_status"] = models.PaymentCompleted
		}
		persistErr := database.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(refundUpdates).Error; err != nil {
				return err
			}
			if err := releaseRefundCapacity(tx); err != nil {
				return err
			}
			if err := services.ReverseOrderLoyalty(tx, order.ID); err != nil {
				return err
			}
			return services.EnqueueEvent(tx, "orders.refunded", "order.refunded", userID, map[string]interface{}{
				"order_id": order.ID.String(), "order_number": order.OrderNumber,
				"refund_amount": refundAmount, "reason": req.Reason, "initiated_by": initiatedBy,
				"refund_id": "wallet:" + txn.ID.String(), "provider": "wallet",
			})
		})
		if persistErr != nil {
			log.Printf("Failed to persist wallet refund + event for order %s: %v", order.ID, persistErr)
			services.CaptureBackgroundError(persistErr)
			// #602: do NOT release the reservation on a persist failure. services.ReserveRefund
			// already committed `refund_amount += reserved` in its own tx BEFORE the wallet
			// credit, so on a persist failure refund_amount is CORRECT (the customer got the
			// credit). Decrementing it back (the old releaseReservation) would ERASE a refund
			// that actually happened → the next distinct refund over-refunds and collides the
			// amount-based idempotency key (razorpay rejects → stuck; wallet silently
			// under-credits). The money-safe state is to leave the order STUCK at refunded with
			// the ledger correct; reconcileStuckRefunds finalizes it (payment_status=refunded
			// AND refunded_at IS NULL is the stuck-mid-refund signal). Same for FULL — its
			// unconditional cross-guard below keeps the payout blocked meanwhile.
		}
		// Cross-guard the payout hold (#457/#549/#568): a FULL refund drives the whole
		// hold to withheld/reversed (unconditional); a PARTIAL refund claws back only
		// the refunded portion from the chef's transfer and leaves the hold releasable,
		// gated on the refund having persisted so a retry can't double-claw. #611: claw the
		// full RESERVED amount (what the customer was refunded), not a wallet-split-lowered
		// value — here they are equal (the to-wallet branch never splits), but keeping
		// `reserved` matches the gateway branch + RefundOrder. Best-effort — never change the response.
		if hErr := crossGuardRefundHold(order.ID, reserved, req.Reason, fullRefund, persistErr == nil); hErr != nil {
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

	provider := models.NormalizeProvider(order.PaymentProvider)

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
				refundWalletIdempotencyKey("refund-wallet", order.ID, priorRefunded), nil); werr != nil {
				log.Printf("wallet-portion re-credit failed order=%s: %v", order.OrderNumber, werr)
				services.CaptureBackgroundError(werr)
			}
			refundAmount = capture
		}
	}

	// The atomic reserve (completed→refunded + refund_amount increment) + releaseReservation
	// were taken above, before the to-wallet / gateway split (#567/#611), so both paths
	// serialize concurrent double-submits under a row lock. The gateway CreateRefund below
	// also carries the #574 idempotency key, but this reserve is what stops two concurrent
	// requests both reserving + issuing a real gateway refund.
	var refundID, refundStatus string

	switch provider {
	case models.PaymentProviderWallet:
		// Full-wallet order (no gateway payment): the entire refund returns as
		// store credit.
		txn, werr := services.CreditWallet(database.DB, order.CustomerID, refundAmount,
			models.WalletSourceRefund, &order.ID,
			fmt.Sprintf("Refund for order %s: %s", order.OrderNumber, req.Reason),
			refundWalletIdempotencyKey("refund", order.ID, priorRefunded), nil)
		if werr != nil {
			log.Printf("wallet-order refund failed order=%s: %v", order.OrderNumber, werr)
			releaseReservation()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to credit wallet"})
			return
		}
		refundID = "wallet:" + txn.ID.String()
		refundStatus = "processed"
	case models.PaymentProviderCashfree:
		if order.RazorpayOrderID == "" {
			// Cashfree refunds are issued against the ORDER, not the payment — so
			// the gateway order id is what's required here, unlike the Razorpay
			// branch below which needs the payment id.
			releaseReservation()
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Cashfree payment found for this order"})
			return
		}
		cf := services.GetCashfreeFor(order.Mode)
		if cf == nil {
			releaseReservation()
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Payment gateway not configured"})
			return
		}
		r, err := cf.CreateRefund(order.RazorpayOrderID, &services.CashfreeRefundRequest{
			AmountPaise: services.CashfreeAmountFromPaise(services.ToPaise(refundAmount)),
			Note:        fmt.Sprintf("refund-%s: %s", order.OrderNumber, req.Reason),
			// Same prior-refunded basis as every other branch (#611): the atomic
			// reserve serializes concurrent submits, and a retry re-derives this key
			// so Cashfree dedups it — here natively, since the key becomes the
			// refund_id itself rather than a header. #574.
			IdempotencyKey: services.RefundPartialIdempotencyKey(order.ID, services.ToPaise(priorRefunded)),
			// Stated explicitly so the vendor's share is reversed the same way
			// whether or not Cashfree's proportional debiting is on. See
			// BuildRefundSplits.
			Splits: services.BuildRefundSplits(&order, services.ToPaise(priorRefunded), services.ToPaise(refundAmount)),
		})
		if err != nil {
			log.Printf("Failed to create Cashfree refund for order %s: %v", order.OrderNumber, err)
			releaseReservation()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
			return
		}
		refundID = r.RefundID
		refundStatus = services.PlatformRefundStatus(r.RefundStatus)
	case models.PaymentProviderStripe:
		if order.StripePaymentIntentID == "" {
			releaseReservation()
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Stripe payment found for this order"})
			return
		}
		st := services.GetStripe()
		if st == nil {
			releaseReservation()
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
			releaseReservation()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
			return
		}
		refundID = r.ID
		refundStatus = r.Status
	default: // razorpay
		if order.RazorpayPaymentID == "" {
			releaseReservation()
			c.JSON(http.StatusBadRequest, gin.H{"error": "No Razorpay payment found for this order"})
			return
		}
		rz := services.GetRazorpayFor(order.Mode)
		if rz == nil {
			releaseReservation()
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
			// Same prior-refunded basis as the wallet branch above (priorRefunded from the
			// reserve, #611); the atomic reserve (#567/#611) serializes concurrent submits,
			// and a retry re-derives the same key so Razorpay dedups it. #574.
			IdempotencyKey: services.RefundPartialIdempotencyKey(order.ID, services.ToPaise(priorRefunded)),
		})
		if err != nil {
			log.Printf("Failed to create Razorpay refund for order %s: %v", order.OrderNumber, err)
			releaseReservation()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process refund"})
			return
		}
		refundID = r.ID
		refundStatus = r.Status
	}

	// Persist the refund and stage the order.refunded event atomically. Target the
	// row by id (not the preloaded &order struct) so a refund doesn't spuriously
	// upsert the belongs-to Chef association. #611: refund_amount was already incremented
	// atomically by ReserveRefund — never re-write it here (the stale read-modify-write
	// that clobbered a concurrent partial's increment).
	refundUpdates := map[string]interface{}{
		"refund_id":           refundID,
		"refund_reason":       req.Reason,
		"refund_initiated_by": initiatedBy,
	}
	if fullRefund {
		now := time.Now()
		refundUpdates["payment_status"] = models.PaymentRefunded
		refundUpdates["status"] = models.OrderStatusRefunded
		refundUpdates["refunded_at"] = &now
	} else {
		// PARTIAL (#549): the reserve above flipped completed→refunded only to serialize
		// the gateway round-trip. Revert it to completed and leave status/refunded_at
		// untouched so the chef's hold stays releasable for the remainder.
		refundUpdates["payment_status"] = models.PaymentCompleted
	}
	persistErr := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(refundUpdates).Error; err != nil {
			return err
		}
		if err := releaseRefundCapacity(tx); err != nil {
			return err
		}
		if err := services.ReverseOrderLoyalty(tx, order.ID); err != nil {
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
	})
	if persistErr != nil {
		log.Printf("Failed to persist refund + event for order %s: %v", order.ID, persistErr)
		services.CaptureBackgroundError(persistErr)
	}
	// Cross-guard the payout hold (#457/#549/#568): FULL → withhold/reverse the whole
	// hold (unconditional); PARTIAL → claw back only the refunded portion, leave the
	// hold releasable, gated on the refund having persisted so a retry can't double-claw.
	// #611: claw the full RESERVED amount = what the customer was refunded in total, NOT the
	// `refundAmount` the wallet-at-checkout split lowered to the gateway `capture` portion —
	// the wallet-re-credited portion is also money returned to the customer that must come out
	// of the chef's held transfer, so clawing only `capture` under-claws the chef (platform
	// eats the wallet portion). Best-effort — never change the HTTP response.
	if hErr := crossGuardRefundHold(order.ID, reserved, req.Reason, fullRefund, persistErr == nil); hErr != nil {
		log.Printf("payout cross-guard failed for refunded order %s: %v", order.ID, hErr)
		services.CaptureBackgroundError(hErr)
	}
	// #885: the payment gateway's transaction-fee loss on a CHEF-INITIATED Cashfree refund,
	// recovered from the chef through the same best-effort mechanism the chef_order_cancel.go
	// paths use. Admin-initiated refunds never levy (decision 1 — ambiguous fault, and a wrong
	// levy takes real money from a chef); non-Cashfree providers never levy (decision-locked
	// Cashfree-only scope). refundAmount here is READ AFTER the wallet-at-checkout capping
	// logic above has already possibly lowered it — that final, capped value is exactly the
	// amount sent to Cashfree, which is the correct basis (not the raw requested amount).
	if provider == models.PaymentProviderCashfree && initiatedBy == "chef" {
		if fp, fErr := services.LevyGatewayFeePenalty(database.DB, order.ChefID, userID, order.ID,
			order.OrderNumber, models.PaymentProviderCashfree, refundAmount,
			services.RefundPartialIdempotencyKey(order.ID, services.ToPaise(priorRefunded))); fErr != nil {
			log.Printf("initiate refund: gateway-fee levy failed for order %s: %v", order.OrderNumber, fErr)
			services.CaptureBackgroundError(fErr)
		} else if fp != nil {
			services.LogAudit(c, "chef.penalty.levy", "chef_penalty", fp.ID.String(), nil,
				gin.H{"orderId": order.ID.String(), "amount": fp.Amount, "kind": "gateway_fee"})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Refund initiated",
		"refundId":     refundID,
		"refundAmount": refundAmount,
		"status":       refundStatus,
		"provider":     provider,
	})
}

// confirmMealPlanAdvanceFromWebhook is the payment.captured fallback for a meal-plan
// ADVANCE order — its gateway order id lives on meal_plans, not orders, so the regular
// order UPDATE above never matches it. It confirms the plan + holds the chef payouts
// durably server-side, so a lost client verify-payment (e.g. the RN Razorpay SDK
// returning dismiss on the success auto-redirect) can't strand a captured advance.
// Returns confirmed=true only on the transition it performed; (false, nil) when there
// is no pending meal-plan advance for this order (the common case: a regular order
// that was already completed). Errors are transient so the webhook layer redelivers.
func (h *PaymentHandler) confirmMealPlanAdvanceFromWebhook(orderID, paymentID string) (bool, error) {
	if !services.MealPlanEscrowActive() {
		return false, nil
	}
	var plan models.MealPlan
	err := database.DB.Preload("Days").
		Where("razorpay_order_id = ? AND status = ?", orderID, models.MealPlanAwaitingCustomer).
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil // not a pending meal-plan advance — nothing to do
	}
	if err != nil {
		return false, err
	}
	// ConfirmMealPlanAdvance manages its own confirm tx + holds payouts outside it.
	return services.ConfirmMealPlanAdvance(database.DB, &plan, paymentID, "")
}

// confirmFssaiRequestFromWebhook is the payment-captured fallback for an FSSAI
// filing request — its gateway order id lives on fssai_requests, so the order
// UPDATE never matches it.
//
// Without this the request is confirmed ONLY when the chef returns from the
// hosted checkout. A chef who pays and closes the browser leaves us holding
// their money against a request that was never submitted, never emailed to
// onboarding, and invisible in the admin queue (which hides unpaid drafts).
//
// Returns confirmed=true only for the transition it performed. Errors are
// transient so the webhook layer redelivers.
func (h *PaymentHandler) confirmFssaiRequestFromWebhook(orderID, paymentID, mode string) (bool, error) {
	var req models.FssaiRequest
	err := database.DB.Preload("Documents").
		Where("gateway_order = ? AND status = ? AND mode = ?",
			orderID, models.FssaiAwaitingPayment, mode).
		First(&req).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil // not a pending FSSAI request — nothing to do
	}
	if err != nil {
		return false, err
	}
	// Checkout refuses to mint an order until the documents are attached, so
	// this should be unreachable. If it ever fires we are holding money for a
	// request we cannot file, which is a person's problem to fix, not a retry's.
	if req.NeedsDocuments() {
		log.Printf("FSSAI request %s: PAID (payment %s) but documents are missing — needs manual follow-up",
			req.ID, paymentID)
		return false, nil
	}
	// Idempotent: the status guard lives in MarkFssaiPaid's WHERE clause, so a
	// webhook racing the chef's own confirm settles on one submission and one
	// onboarding email.
	if err := services.MarkFssaiPaid(database.DB, &req, paymentID); err != nil {
		return false, err
	}
	return true, nil
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
		Where("stripe_payment_intent_id = ? AND payment_status NOT IN ?", pi.ID, services.CompletionBlockedStatuses).
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
	services.NotifyPaymentSucceeded(database.DB, ord.ID)
	if err := services.NotifyChefNewOrderTx(database.DB, &ord); err != nil {
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
	res := database.DB.Model(&models.Order{}).
		Where("stripe_payment_intent_id = ? AND payment_status NOT IN ?", pi.ID, []models.PaymentStatus{
			models.PaymentCompleted, models.PaymentFailed, models.PaymentRefunded,
		}).
		Update("payment_status", models.PaymentFailed)
	if res.Error != nil {
		log.Printf("Failed to apply stripe payment_intent.payment_failed for %s: %v", pi.ID, res.Error)
		return
	}
	// Single transition only, so a redelivery can't notify twice (#notify-money).
	if res.RowsAffected > 0 {
		var ord models.Order
		if err := database.DB.Select("id").
			Where("stripe_payment_intent_id = ?", pi.ID).First(&ord).Error; err == nil {
			services.NotifyPaymentFailed(database.DB, ord.ID, "card declined")
		}
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
