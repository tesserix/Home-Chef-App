package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// WalletHandler serves the customer-facing store-credit wallet (#33).
type WalletHandler struct{}

func NewWalletHandler() *WalletHandler { return &WalletHandler{} }

// GetWallet returns the authenticated customer's store-credit balance.
// GET /customer/wallet
func (h *WalletHandler) GetWallet(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	w, err := services.WalletBalance(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load wallet"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance": w.Balance, "currency": w.Currency})
}

// GetWalletTransactions returns the customer's ledger, newest first (paginated).
// GET /customer/wallet/transactions
func (h *WalletHandler) GetWalletTransactions(c *gin.Context) {
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

	txns, total, err := services.ListWalletTxns(database.DB, userID, limit, offset)
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
