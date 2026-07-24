package handlers

// meal_plan_refund_v2.go — HTTP endpoints for the v2 meal-plan refund workflow
// (docs/meal-plan-refund-flow-design.md): the CHEF decides a late (≤12h) skip/cancel (Full/Half/
// None/Decline), then an ADMIN pays it to the wallet or original method. Both gated by the flow flag.

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// chefRefundDecisionDay is one day awaiting the chef's Full/Half/None/Decline, with the
// fee/GST-excluded refund amounts so the chef can weigh prep-done vs refund.
type chefRefundDecisionDay struct {
	DayID          string  `json:"dayId"`
	Date           string  `json:"date"`
	Slot           string  `json:"slot"`
	DishName       string  `json:"dishName"`
	CustomerName   string  `json:"customerName"`
	MealPlanNumber string  `json:"mealPlanNumber"`
	FoodPrice      float64 `json:"foodPrice"`
	FullRefund     float64 `json:"fullRefund"`
	HalfRefund     float64 `json:"halfRefund"`
}

// GetChefPendingRefundDecisions — GET /chef/meal-plan-days/pending-refund-decisions. Lists the days
// awaiting THIS chef's decision on a ≤12h skip/cancel. Empty when the v2 flow is off.
func (h *MealPlanHandler) GetChefPendingRefundDecisions(c *gin.Context) {
	chef, ok := authedChef(c)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	if !services.MealPlanRefundFlowV2Active() {
		c.JSON(http.StatusOK, gin.H{"data": []chefRefundDecisionDay{}})
		return
	}
	var days []models.MealPlanDay
	database.DB.
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Where("meal_plans.chef_id = ? AND meal_plan_days.refund_stage = ?", chef.ID, models.MPRefundPendingChef).
		Order("meal_plan_days.date ASC").Find(&days)

	out := make([]chefRefundDecisionDay, 0, len(days))
	for i := range days {
		d := &days[i]
		var plan models.MealPlan
		if err := database.DB.Select("id", "meal_plan_number", "customer_id").First(&plan, "id = ?", d.MealPlanID).Error; err != nil {
			continue
		}
		var cust models.User
		database.DB.Select("first_name", "last_name").First(&cust, "id = ?", plan.CustomerID)
		out = append(out, chefRefundDecisionDay{
			DayID: d.ID.String(), Date: d.Date.Format("2006-01-02"), Slot: string(d.Slot),
			DishName: d.DishName, CustomerName: strings.TrimSpace(cust.FirstName + " " + cust.LastName),
			MealPlanNumber: plan.MealPlanNumber, FoodPrice: d.Price,
			FullRefund: services.MealPlanRefundAmount(&plan, d, models.RefundProportionFull),
			HalfRefund: services.MealPlanRefundAmount(&plan, d, models.RefundProportionHalf),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

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
