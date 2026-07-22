package handlers

// chef_payout_gate.go — HTTP surface for the payout-setup gate (#739).
//
// The gate stops a chef switching on accepting_orders before there is a
// destination for their share of a customer's money. See
// services/payout_readiness.go for the policy; this file is only the
// request/response edge.

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// payoutGateBlocks enforces the gate for a handler that is about to switch
// accepting_orders on. It writes the error response itself and reports whether
// the caller should stop.
//
// The response is 409 with a machine-readable reasonCode rather than a bare
// 403, so the vendor app can deep-link the chef straight to the payout screen
// instead of showing a dead end.
func payoutGateBlocks(c *gin.Context, chef *models.ChefProfile, accepting bool) bool {
	err := services.RequirePayoutReadyToAcceptOrders(database.DB, chef, accepting, time.Now())
	if err == nil {
		return false
	}

	var required services.ErrPayoutSetupRequired
	if !errors.As(err, &required) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check payout setup"})
		return true
	}

	c.JSON(http.StatusConflict, gin.H{
		"error":      "Add your payout details before you start accepting orders",
		"reasonCode": required.ReasonCode,
		"action":     "setup_payout",
	})
	return true
}

// GetPayoutReadiness reports the chef's standing against the payout gate.
// GET /chef/payout/readiness
//
// The vendor app calls this to decide between nagging and blocking, so it
// returns readiness even when the gate is off or in its grace window.
func (h *ChefHandler) GetPayoutReadiness(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	c.JSON(http.StatusOK, services.ChefPayoutReadiness(database.DB, &chef, time.Now()))
}
