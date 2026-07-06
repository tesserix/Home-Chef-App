package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// admin_delivery_failure.go — #613. The admin delivery-failure RESOLUTION surface: one
// read-only queue aggregating every unresolved delivery failure across the three order
// shapes so an admin can SEE what needs a fault decision and resolve it via the existing
// resolve-delivery-failure endpoints. Read-only; the money seams live in the resolvers.
//
// The categories are DISJOINT by construction: a gateway order or a chef self-delivery
// opens a pending `delivery_failed` OrderIssue on its order; meal-plan days and group
// orders (shell orders with no razorpay id → no OrderIssue) instead terminalize to
// status=`failed`. So no row is double-surfaced.

type orderDeliveryFailureRow struct {
	IssueID        string    `json:"issueId"`
	OrderID        string    `json:"orderId"`
	OrderNumber    string    `json:"orderNumber"`
	CustomerID     string    `json:"customerId"`
	ChefID         string    `json:"chefId"`
	Total          float64   `json:"total"`
	HoldStatus     string    `json:"holdStatus"`
	Description    string    `json:"description"`
	Reason         string    `json:"reason"`         // parsed from Description
	SuggestedFault string    `json:"suggestedFault"` // parsed hint; admin still confirms
	ReportedBy     string    `json:"reportedBy"`     // courier vs chef_self_delivery
	CreatedAt      time.Time `json:"createdAt"`
}

type dayDeliveryFailureRow struct {
	DayID          string    `json:"dayId"`
	MealPlanID     string    `json:"mealPlanId"`
	MealPlanNumber string    `json:"mealPlanNumber"`
	CustomerID     string    `json:"customerId"`
	ChefID         string    `json:"chefId"`
	Date           time.Time `json:"date"`
	Price          float64   `json:"price"`
	HoldStatus     string    `json:"holdStatus"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type groupDeliveryFailureRow struct {
	GroupID    string    `json:"groupId"`
	HostID     string    `json:"hostId"`
	ChefID     string    `json:"chefId"`
	Subtotal   float64   `json:"subtotal"`
	Tax        float64   `json:"tax"`
	HoldStatus string    `json:"holdStatus"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// AdminListDeliveryFailures returns the read-only delivery-failure resolution queue (#613):
// pending `delivery_failed` OrderIssues (gateway + chef self-delivery), plus `failed`
// meal-plan days and `failed` group orders. Each row carries the resolve id
// (issueId / dayId / groupId), the money context, and the frozen payout-hold status, so the
// admin confirms a fault and calls the matching resolve-delivery-failure endpoint. Oldest
// first within each category, so the longest-frozen money is worked first.
func (h *OrderIssueHandler) AdminListDeliveryFailures(c *gin.Context) {
	orderIssues := []orderDeliveryFailureRow{}
	mealPlanDays := []dayDeliveryFailureRow{}
	groupOrders := []groupDeliveryFailureRow{}

	// 1) Gateway + chef self-delivery: pending delivery_failed OrderIssues + their order.
	if err := database.DB.Table("order_issues AS oi").
		Select("oi.id AS issue_id, oi.order_id, oi.customer_id, oi.chef_id, oi.description, oi.created_at, "+
			"o.order_number, o.total, o.payout_hold_status AS hold_status").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("oi.reason = ? AND oi.status = ?", models.IssueDeliveryFailed, models.IssuePending).
		Order("oi.created_at ASC").
		Scan(&orderIssues).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load delivery-failure order issues"})
		return
	}
	for i := range orderIssues {
		d := orderIssues[i].Description
		orderIssues[i].Reason = descField(d, "reason")
		orderIssues[i].SuggestedFault = descField(d, "suggested_fault")
		orderIssues[i].ReportedBy = descField(d, "reported_by")
	}

	// 2) Failed meal-plan days (shell order, no OrderIssue) + their plan.
	if err := database.DB.Table("meal_plan_days AS d").
		Select("d.id AS day_id, d.meal_plan_id, d.date, d.price, d.payout_hold_status AS hold_status, d.updated_at, "+
			"mp.meal_plan_number, mp.customer_id, mp.chef_id").
		Joins("JOIN meal_plans mp ON mp.id = d.meal_plan_id").
		Where("d.status = ?", models.MealPlanDayFailed).
		Order("d.updated_at ASC").
		Scan(&mealPlanDays).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load failed meal-plan days"})
		return
	}

	// 3) Failed group orders (consolidated shell, no OrderIssue).
	if err := database.DB.Table("group_orders").
		Select("id AS group_id, host_id, chef_id, subtotal, tax, payout_hold_status AS hold_status, updated_at").
		Where("status = ?", models.GroupOrderFailed).
		Order("updated_at ASC").
		Scan(&groupOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load failed group orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orderIssues":  orderIssues,
		"mealPlanDays": mealPlanDays,
		"groupOrders":  groupOrders,
		"count":        len(orderIssues) + len(mealPlanDays) + len(groupOrders),
	})
}

// descField extracts the value of a `key=value` token from a delivery_failed issue
// description ("delivery failed: reason=… suggested_fault=… reported_by=…", built by
// services.RecordDeliveryFailure). Values are single whitespace-delimited tokens; returns
// "" when the key is absent (defensive — the surface degrades to the raw description).
func descField(desc, key string) string {
	for _, tok := range strings.Fields(desc) {
		if v, ok := strings.CutPrefix(tok, key+"="); ok {
			return v
		}
	}
	return ""
}
