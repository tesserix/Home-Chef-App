package handlers

// chef_refunds.go — refund history for the chef dashboard.
//   GET /chef/refunds?limit=50 → orders the chef refunded, newest first.
//
// Order.RefundAmount is the authoritative cumulative refund per order — it is
// incremented by whole-order cancels, per-line cancels, AND post-delivery
// goodwill refunds alike. So we list ONE entry per order (using that total)
// and attach the cancelled line items as a breakdown; summing item amounts
// instead would double-count against the order total.

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

const (
	refundsDefaultLimit = 50
	refundsMaxLimit     = 200
)

// ChefRefundsHandler serves refund history.
type ChefRefundsHandler struct{}

// NewChefRefundsHandler constructs the handler.
func NewChefRefundsHandler() *ChefRefundsHandler {
	return &ChefRefundsHandler{}
}

type refundItemDetail struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Reason string  `json:"reason,omitempty"`
}

type refundEntry struct {
	OrderID     string             `json:"orderId"`
	OrderNumber string             `json:"orderNumber"`
	Amount      float64            `json:"amount"`
	Reason      string             `json:"reason,omitempty"`
	InitiatedBy string             `json:"initiatedBy,omitempty"`
	RefundedAt  time.Time          `json:"refundedAt"`
	Items       []refundItemDetail `json:"items,omitempty"`
}

// GetRefunds lists the chef's refunded orders, newest first.
func (h *ChefRefundsHandler) GetRefunds(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	limit := refundsDefaultLimit
	if raw := c.Query("limit"); raw != "" {
		if n, perr := strconv.Atoi(raw); perr == nil && n > 0 {
			limit = n
		}
	}
	if limit > refundsMaxLimit {
		limit = refundsMaxLimit
	}

	var orders []models.Order
	if err := database.DB.
		Preload("Items").
		Where("chef_id = ? AND refund_amount > 0 AND deleted_at IS NULL", chef.ID).
		Order("COALESCE(refunded_at, updated_at) DESC").
		Limit(limit).
		Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch refunds"})
		return
	}

	out := make([]refundEntry, 0, len(orders))
	for i := range orders {
		out = append(out, buildRefundEntry(&orders[i]))
	}
	c.JSON(http.StatusOK, gin.H{"refunds": out})
}

// buildRefundEntry shapes one order into a refund-history entry, resolving the
// refund timestamp and reason across the whole-order and per-line paths.
func buildRefundEntry(order *models.Order) refundEntry {
	items := make([]refundItemDetail, 0)
	var latestCancel time.Time
	for _, it := range order.Items {
		if !it.IsCancelled || it.RefundAmount <= 0 {
			continue
		}
		items = append(items, refundItemDetail{
			Name:   it.Name,
			Amount: it.RefundAmount,
			Reason: it.CancelledReason,
		})
		if it.CancelledAt != nil && it.CancelledAt.After(latestCancel) {
			latestCancel = *it.CancelledAt
		}
	}

	// Date: explicit order refund timestamp → latest item cancel → updated_at.
	refundedAt := order.UpdatedAt
	if order.RefundedAt != nil {
		refundedAt = *order.RefundedAt
	} else if !latestCancel.IsZero() {
		refundedAt = latestCancel
	}

	// Reason: explicit order reason → fall back to a line summary.
	reason := order.RefundReason
	if reason == "" && len(items) > 0 {
		if len(items) == 1 {
			reason = "Item cancelled"
		} else {
			reason = "Items cancelled"
		}
	}

	return refundEntry{
		OrderID:     order.ID.String(),
		OrderNumber: order.OrderNumber,
		Amount:      order.RefundAmount,
		Reason:      reason,
		InitiatedBy: order.RefundInitiatedBy,
		RefundedAt:  refundedAt,
		Items:       items,
	}
}
