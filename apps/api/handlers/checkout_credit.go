package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// checkout_credit.go — the read-only credit quote the checkout screen renders.

// QuoteOrderCredit returns the authoritative wallet + loyalty breakdown for an
// order without moving any money.
//
// The checkout screen renders this verbatim and does no arithmetic of its own,
// and CreateOrderPayment re-runs the identical computation before charging — so
// what the customer is shown and what they are charged cannot diverge.
//
// POST /payments/order/:orderId/quote
func (h *PaymentHandler) QuoteOrderCredit(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req services.CreditRequest
	_ = c.ShouldBindJSON(&req) // an absent/!malformed body simply means "no credit"

	quote, err := services.BuildCreditQuote(database.DB, &order, userID, req, creditFlags())
	if err != nil {
		log.Printf("credit-quote failed order=%s: %v", order.OrderNumber, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not price this order"})
		return
	}
	c.JSON(http.StatusOK, creditQuoteResponse(quote))
}

// creditFlags snapshots the server feature gates for an allocation.
func creditFlags() services.CreditFlags {
	return services.CreditFlags{
		WalletCheckoutEnabled:  config.AppConfig.WalletCheckoutEnabled,
		LoyaltyCheckoutEnabled: config.AppConfig.LoyaltyCheckoutEnabled,
	}
}

// creditQuoteResponse renders a quote in rupees for the client. Paise stay
// server-side; the client only ever displays these figures.
//
// walletEnabled/loyaltyEnabled are served here rather than compiled into the app
// so a stale build can no longer disagree with the server about whether the
// feature exists — the previous build-time EXPO_PUBLIC_WALLET_CHECKOUT_ENABLED
// constant had to be moved in lockstep with the API env.
func creditQuoteResponse(q services.CreditQuote) gin.H {
	return gin.H{
		"redeemableCap":  services.FromPaise(q.RedeemableCapPaise),
		"nonRedeemable":  services.FromPaise(q.NonRedeemablePaise),
		"walletBalance":  services.FromPaise(q.WalletBalancePaise),
		"walletApplied":  services.FromPaise(q.WalletAppliedPaise),
		"walletMax":      services.FromPaise(q.WalletMaxPaise),
		"pointsBalance":  q.PointsBalance,
		"pointsApplied":  q.PointsAppliedPoints,
		"pointsValue":    services.FromPaise(q.PointsAppliedPaise),
		"pointsMax":      q.PointsMaxPoints,
		"payable":        services.FromPaise(q.PayablePaise),
		"loyaltyLimit":   q.LoyaltyLimitReason,
		"walletEnabled":  q.WalletEnabled,
		"loyaltyEnabled": q.LoyaltyEnabled,
	}
}
