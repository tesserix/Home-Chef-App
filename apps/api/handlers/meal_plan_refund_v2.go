package handlers

// meal_plan_refund_v2.go — HTTP endpoints for the v2 meal-plan refund workflow
// (docs/meal-plan-refund-flow-design.md): the CHEF decides a late (≤12h) skip/cancel (Full/Half/
// None/Decline), then an ADMIN pays it to the wallet or original method. Both gated by the flow flag.

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// ChefRefundDecision — POST /chef/meal-plans/days/:dayId/refund-decision. The chef resolves a day
// awaiting them (a ≤12h skip/cancel): {"choice":"full|half|none"} refunds that proportion of the
// fee/GST-excluded food (Full/Half go to the admin to pay; None resolves now, chef keeps payout),
// or {"decline":true} keeps the day (it will be cooked and delivered).
func (h *MealPlanHandler) ChefRefundDecision(c *gin.Context) {
	if !services.MealPlanRefundFlowV2Active() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not available"})
		return
	}
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	var req struct {
		Choice  models.RefundProportion `json:"choice"`
		Decline bool                    `json:"decline"`
	}
	_ = c.ShouldBindJSON(&req)

	// Authorize: the day's plan must belong to the authenticated chef.
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chef profile not found"})
		return
	}
	var day models.MealPlanDay
	if err := database.DB.Select("id", "meal_plan_id").First(&day, "id = ?", dayID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Day not found"})
		return
	}
	var plan models.MealPlan
	if err := database.DB.Select("id", "chef_id").First(&plan, "id = ?", day.MealPlanID).Error; err != nil || plan.ChefID != chef.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not your meal plan"})
		return
	}

	if err := services.ChefDecideMealPlanRefund(database.DB, dayID, req.Choice, req.Decline); err != nil {
		switch {
		case errors.Is(err, services.ErrRefundStageMismatch):
			c.JSON(http.StatusConflict, gin.H{"error": "This day is no longer awaiting your decision"})
		case errors.Is(err, services.ErrInvalidRefundChoice):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Choice must be full, half, or none (or decline)"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record decision"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// AdminPayMealPlanDayRefund — POST /admin/meal-plan-days/:dayId/pay-refund. Pay a day the chef
// approved (Full/Half) to {"destination":"wallet"} (instant) or {"destination":"source"} (original
// method, RBI ~5–7 days). Audited.
func (h *MealPlanHandler) AdminPayMealPlanDayRefund(c *gin.Context) {
	if !services.MealPlanRefundFlowV2Active() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not available"})
		return
	}
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	var req struct {
		Destination models.RefundDestination `json:"destination"`
	}
	_ = c.ShouldBindJSON(&req)
	dest := models.RefundDestinationWallet // default: instant wallet
	if req.Destination == models.RefundDestinationSource {
		dest = models.RefundDestinationSource
	}

	if err := services.AdminPayMealPlanRefund(database.DB, dayID, dest); err != nil {
		switch {
		case errors.Is(err, services.ErrRefundStageMismatch):
			c.JSON(http.StatusConflict, gin.H{"error": "This day is not awaiting an admin refund"})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Day not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to pay refund"})
		}
		return
	}
	services.LogAudit(c, "mealplan.refund.pay", "meal_plan_day", dayID.String(), nil, map[string]any{"destination": dest})
	c.JSON(http.StatusOK, gin.H{"status": "refunded", "destination": dest})
}
