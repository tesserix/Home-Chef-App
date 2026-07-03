package handlers

// payout_hold.go — customer confirm endpoints for the payout hold state machine
// (#387). A customer confirms they received a delivered order/day, advancing its
// hold awaiting_customer_confirmation -> release_eligible (or -> disputed when an
// open OrderIssue exists). All endpoints are owner-scoped and idempotent.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type PayoutHoldHandler struct{}

func NewPayoutHoldHandler() *PayoutHoldHandler { return &PayoutHoldHandler{} }

// confirmMessage is the user-facing copy for a resulting hold status (never a raw
// status code or error).
func confirmMessage(status models.PayoutHoldStatus) string {
	switch status {
	case models.PayoutHoldDisputed:
		return "Thanks — we've noted this while your reported issue is reviewed."
	default:
		return "Thanks for confirming you received your order."
	}
}

// ConfirmOrderReceived advances the caller's order hold to release_eligible.
// POST /orders/:id/confirm-received — owner-scoped, idempotent.
func (h *PayoutHoldHandler) ConfirmOrderReceived(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("id"))
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

	status, err := services.ConfirmOrderHold(database.DB, &order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not confirm receipt"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"payoutHoldStatus":    status,
		"customerConfirmedAt": order.CustomerConfirmedAt,
		"message":             confirmMessage(status),
	})
}

// ConfirmMealPlanDayReceived advances the caller's meal-plan day hold.
// POST /meal-plans/:id/days/:dayId/confirm-received — owner-scoped, idempotent.
func (h *PayoutHoldHandler) ConfirmMealPlanDayReceived(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	planID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid meal plan id"})
		return
	}
	dayID, err := uuid.Parse(c.Param("dayId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid meal plan day id"})
		return
	}

	var day models.MealPlanDay
	if err := database.DB.
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Where("meal_plan_days.id = ? AND meal_plan_days.meal_plan_id = ? AND meal_plans.customer_id = ?",
			dayID, planID, userID).
		First(&day).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Meal plan day not found"})
		return
	}

	status, err := services.ConfirmMealPlanDayHold(database.DB, &day)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not confirm receipt"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"payoutHoldStatus":    status,
		"customerConfirmedAt": day.CustomerConfirmedAt,
		"message":             confirmMessage(status),
	})
}

// ConfirmGroupOrderReceived advances a group/office order's hold to release_eligible
// on the HOST confirming receipt (or -> disputed on an open issue). POST
// /group-orders/:id/confirm-received — host-scoped, idempotent. The host is the
// payer/receiver of the consolidated order, so it mirrors CancelGroupOrder's guard.
func (h *PayoutHoldHandler) ConfirmGroupOrderReceived(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid id"})
		return
	}

	g, me, ok := loadGroupForParticipant(id, userID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group order not found"})
		return
	}
	if me.Role != models.GroupRoleHost {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the host can confirm receipt"})
		return
	}

	status, err := services.ConfirmGroupOrderHold(database.DB, &g)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not confirm receipt"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"payoutHoldStatus":    status,
		"customerConfirmedAt": g.CustomerConfirmedAt,
		"message":             confirmMessage(status),
	})
}

// ConfirmTodaysTiffin bulk-confirms all of the caller's delivered-today,
// still-awaiting tiffin days. POST /tiffin/confirm-today.
func (h *PayoutHoldHandler) ConfirmTodaysTiffin(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	confirmed, err := services.ConfirmTodaysTiffinForCustomer(database.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not confirm today's meals"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"confirmed": confirmed})
}
