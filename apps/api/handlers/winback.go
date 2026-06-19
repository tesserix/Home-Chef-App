package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// winback.go — customer-facing win-back offer surface (#42). Powers the in-app
// banner: the user's current active (open, non-expired) offer, or null.

type WinbackHandler struct{}

func NewWinbackHandler() *WinbackHandler { return &WinbackHandler{} }

// GetActiveWinback returns the authenticated user's active win-back offer, if any.
// GET /winback/active
func (h *WinbackHandler) GetActiveWinback(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	offer := services.GetActiveWinbackOffer(database.DB, userID)
	if offer == nil {
		c.JSON(http.StatusOK, gin.H{"offer": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"offer": gin.H{
		"code":            offer.Code,
		"discountPercent": offer.DiscountPercent,
		"expiresAt":       offer.ExpiresAt,
		"audienceType":    offer.AudienceType,
		"trigger":         offer.Trigger,
	}})
}
