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
		// subtotal/tax/total are needed so MealPlanRefundAmount can add the day's delivery fee.
		if err := database.DB.Select("id", "meal_plan_number", "customer_id", "subtotal", "tax", "total").First(&plan, "id = ?", d.MealPlanID).Error; err != nil {
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

// adminPendingRefundDay is one day whose refund the customer routed to their ORIGINAL method,
// awaiting an admin to execute the gateway reversal (wallet refunds resolve instantly, no admin).
type adminPendingRefundDay struct {
	DayID          string  `json:"dayId"`
	Date           string  `json:"date"`
	Slot           string  `json:"slot"`
	DishName       string  `json:"dishName"`
	CustomerName   string  `json:"customerName"`
	ChefName       string  `json:"chefName"`
	MealPlanNumber string  `json:"mealPlanNumber"`
	ChefChoice     string  `json:"chefChoice"`   // full | half
	RefundAmount   float64 `json:"refundAmount"` // fee/GST-excluded amount to pay the customer
}

// GetAdminPendingRefunds — GET /admin/meal-plan-days/pending-refunds. Days whose refund the
// customer routed to their ORIGINAL method (RBI); an admin executes the Razorpay reversal via the
// HMAC gateway. Wallet refunds never appear here (they resolve instantly). Empty when v2 is off.
func (h *MealPlanHandler) GetAdminPendingRefunds(c *gin.Context) {
	if !services.MealPlanRefundFlowV2Active() {
		c.JSON(http.StatusOK, gin.H{"data": []adminPendingRefundDay{}})
		return
	}
	var days []models.MealPlanDay
	database.DB.Where("refund_stage = ?", models.MPRefundPendingAdmin).Order("date ASC").Find(&days)

	out := make([]adminPendingRefundDay, 0, len(days))
	for i := range days {
		d := &days[i]
		var plan models.MealPlan
		if err := database.DB.Select("id", "meal_plan_number", "customer_id", "chef_id", "subtotal", "tax", "total").First(&plan, "id = ?", d.MealPlanID).Error; err != nil {
			continue
		}
		var cust models.User
		database.DB.Select("first_name", "last_name").First(&cust, "id = ?", plan.CustomerID)
		var chef models.ChefProfile
		database.DB.Select("business_name").First(&chef, "id = ?", plan.ChefID)
		out = append(out, adminPendingRefundDay{
			DayID: d.ID.String(), Date: d.Date.Format("2006-01-02"), Slot: string(d.Slot),
			DishName: d.DishName, CustomerName: strings.TrimSpace(cust.FirstName + " " + cust.LastName),
			ChefName: chef.BusinessName, MealPlanNumber: plan.MealPlanNumber,
			ChefChoice:   string(d.ChefRefundChoice),
			RefundAmount: services.MealPlanRefundAmount(&plan, d, d.ChefRefundChoice),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// CustomerChooseRefundMedium — POST /meal-plans/:id/days/:dayId/refund-medium. The customer makes
// the RBI-required medium choice for a refund awaiting them: {"medium":"wallet"} credits their
// HomeChef wallet instantly; {"medium":"source"} refunds their original card/UPI (5–7 days).
func (h *MealPlanHandler) CustomerChooseRefundMedium(c *gin.Context) {
	if !services.MealPlanRefundFlowV2Active() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not available"})
		return
	}
	customerID, _ := middleware.GetUserID(c)
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	var req struct {
		Medium models.RefundDestination `json:"medium"`
	}
	_ = c.ShouldBindJSON(&req)

	if err := services.CustomerChooseMealPlanRefundMedium(database.DB, dayID, customerID, req.Medium); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidRefundMedium):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Choose your wallet or your original payment method"})
		case errors.Is(err, services.ErrRefundStageMismatch):
			c.JSON(http.StatusConflict, gin.H{"error": "This refund has already been handled"})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Refund not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record your choice"})
		}
		return
	}
	instant := req.Medium == models.RefundDestinationWallet
	msg := "We’ll refund your original payment method in 5–7 business days."
	if instant {
		msg = "Refunded to your HomeChef wallet — ready to use on your next order."
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "instant": instant, "message": msg})
}

// customerRefundChoiceDay is one refund awaiting the customer's medium choice.
type customerRefundChoiceDay struct {
	DayID          string  `json:"dayId"`
	MealPlanID     string  `json:"mealPlanId"`
	MealPlanNumber string  `json:"mealPlanNumber"`
	Date           string  `json:"date"`
	Slot           string  `json:"slot"`
	DishName       string  `json:"dishName"`
	Amount         float64 `json:"amount"`
}

// GetCustomerPendingRefundChoices — GET /meal-plans/refund-choices. The customer's refunds that are
// agreed and awaiting their medium choice (wallet vs original). Empty when the v2 flow is off.
func (h *MealPlanHandler) GetCustomerPendingRefundChoices(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	if !services.MealPlanRefundFlowV2Active() {
		c.JSON(http.StatusOK, gin.H{"data": []customerRefundChoiceDay{}})
		return
	}
	var days []models.MealPlanDay
	database.DB.
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Where("meal_plans.customer_id = ? AND meal_plan_days.refund_stage = ?", customerID, models.MPRefundPendingCustomer).
		Order("meal_plan_days.date ASC").Find(&days)

	out := make([]customerRefundChoiceDay, 0, len(days))
	for i := range days {
		d := &days[i]
		var plan models.MealPlan
		if err := database.DB.Select("id", "meal_plan_number", "subtotal", "tax", "total").First(&plan, "id = ?", d.MealPlanID).Error; err != nil {
			continue
		}
		out = append(out, customerRefundChoiceDay{
			DayID: d.ID.String(), MealPlanID: d.MealPlanID.String(), MealPlanNumber: plan.MealPlanNumber,
			Date: d.Date.Format("2006-01-02"), Slot: string(d.Slot), DishName: d.DishName,
			Amount: services.MealPlanRefundAmount(&plan, d, d.ChefRefundChoice),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// AdminExecuteMealPlanDayRefund — POST /admin/meal-plan-days/:dayId/execute-refund. Runs the
// customer-chosen ORIGINAL (gateway) refund. The admin only EXECUTES — the customer already chose
// the medium (RBI). Audited.
func (h *MealPlanHandler) AdminExecuteMealPlanDayRefund(c *gin.Context) {
	if !services.MealPlanRefundFlowV2Active() {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not available"})
		return
	}
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid day id"})
		return
	}
	if err := services.AdminExecuteMealPlanRefund(database.DB, dayID); err != nil {
		switch {
		case errors.Is(err, services.ErrRefundStageMismatch):
			c.JSON(http.StatusConflict, gin.H{"error": "This day is not awaiting an admin refund"})
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Day not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute the refund"})
		}
		return
	}
	services.LogAudit(c, "mealplan.refund.execute", "meal_plan_day", dayID.String(), nil, map[string]any{"destination": "source"})
	c.JSON(http.StatusOK, gin.H{"status": "refunded", "destination": "source"})
}
