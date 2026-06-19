package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// LoyaltyHandler serves the customer-facing loyalty program (#40): points
// balance + tier + streak, the points ledger, and redeem-to-wallet.
type LoyaltyHandler struct{}

func NewLoyaltyHandler() *LoyaltyHandler { return &LoyaltyHandler{} }

// GetLoyalty returns the authenticated customer's points balance, tier, streak,
// and the live program config (so the app can show the earn/redeem rules and the
// next-tier target without hardcoding them).
// GET /customer/loyalty
func (h *LoyaltyHandler) GetLoyalty(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	cfg := services.GetLoyaltyConfig(database.DB)
	acct, err := services.LoyaltyBalance(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load loyalty account"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"balance":        acct.Balance,
		"lifetimePoints": acct.LifetimePoints,
		"tier":           acct.Tier,
		"currentStreak":  acct.CurrentStreak,
		"longestStreak":  acct.LongestStreak,
		"config": gin.H{
			"enabled":         cfg.Enabled,
			"redeemRate":      cfg.RedeemRate,
			"minRedeem":       cfg.MinRedeem,
			"streakThreshold": cfg.StreakThreshold,
			"streakBonus":     cfg.StreakBonus,
			"tierSilverAt":    cfg.TierSilverAt,
			"tierGoldAt":      cfg.TierGoldAt,
		},
	})
}

// GetLoyaltyTransactions returns the customer's points ledger, newest first.
// GET /customer/loyalty/transactions
func (h *LoyaltyHandler) GetLoyaltyTransactions(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	txns, total, err := services.ListLoyaltyTxns(database.DB, userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load transactions"})
		return
	}
	totalPages := (total + int64(limit) - 1) / int64(limit)
	c.JSON(http.StatusOK, gin.H{
		"data": txns,
		"pagination": gin.H{
			"page": page, "limit": limit, "total": total,
			"totalPages": totalPages, "hasNext": int64(page) < totalPages, "hasPrev": page > 1,
		},
	})
}

type redeemLoyaltyRequest struct {
	Points float64 `json:"points" binding:"required"`
}

// RedeemLoyalty converts points to wallet store credit.
// POST /customer/loyalty/redeem  { "points": 300 }
func (h *LoyaltyHandler) RedeemLoyalty(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req redeemLoyaltyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if req.Points <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Points must be positive"})
		return
	}

	lt, wt, err := services.RedeemLoyalty(database.DB, userID, req.Points)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrLoyaltyBelowMinRedeem):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Below the minimum redeemable points"})
		case errors.Is(err, services.ErrInsufficientLoyaltyPoints):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough points"})
		case errors.Is(err, services.ErrLoyaltyDisabled):
			c.JSON(http.StatusForbidden, gin.H{"error": "Loyalty program is currently unavailable"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to redeem points"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"pointsRedeemed": lt.Points,
		"pointsBalance":  lt.PointsAfter,
		"walletCredited": wt.Amount,
		"walletBalance":  wt.BalanceAfter,
	})
}
