package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// AdjustWallet lets an admin credit or debit a customer's store-credit wallet
// (goodwill credit, manual cashback, correction). Every adjustment is written to
// the audit log with the acting admin, amount, and reason. POST
// /admin/wallet/:userId/adjust
func (h *AdminHandler) AdjustWallet(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}

	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
		Reason string  `json:"reason" binding:"required,min=3"`
		Type   string  `json:"type" binding:"required,oneof=credit debit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount (>0), reason, and type (credit|debit) are required"})
		return
	}

	var txn *models.WalletTxn
	if req.Type == "credit" {
		txn, err = services.CreditWallet(database.DB, userID, req.Amount, models.WalletSourceAdminAdjust, nil, req.Reason, "", &adminID)
	} else {
		txn, err = services.DebitWallet(database.DB, userID, req.Amount, models.WalletSourceAdminAdjust, nil, req.Reason, "", &adminID)
	}
	if errors.Is(err, services.ErrInsufficientWalletBalance) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debit exceeds the customer's wallet balance"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to adjust wallet"})
		return
	}

	services.LogAudit(c, "wallet.admin_adjust", "wallet", userID.String(), nil, map[string]any{
		"type": req.Type, "amount": req.Amount, "reason": req.Reason, "balanceAfter": txn.BalanceAfter,
	})
	c.JSON(http.StatusOK, gin.H{"transaction": txn, "balance": txn.BalanceAfter})
}

// GetCustomerWallet returns a customer's wallet + recent ledger for the admin UI.
// GET /admin/wallet/:userId
func (h *AdminHandler) GetCustomerWallet(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user id"})
		return
	}
	w, err := services.WalletBalance(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load wallet"})
		return
	}
	txns, total, err := services.ListWalletTxns(database.DB, userID, 50, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load transactions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance": w.Balance, "currency": w.Currency, "transactions": txns, "count": total})
}
