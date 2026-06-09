package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"gorm.io/gorm"
)

// ChefOrderCancelHandler holds the chef-side cancellation routes.
// Whole-order and per-line cancel both refund through Razorpay; per-
// line additionally recomputes the order totals so subsequent
// statements + invoices reflect the smaller scope.
//
// Razorpay-only for now — orders paid via Stripe Connect return 422
// with a hint to use the Stripe-specific flow (TODO once that ships).
type ChefOrderCancelHandler struct{}

func NewChefOrderCancelHandler() *ChefOrderCancelHandler {
	return &ChefOrderCancelHandler{}
}

// cancellableStatuses lists the order states where a chef can still
// pull the plug. Once the order is out the door (picked_up onward)
// cancellation becomes a customer-support problem, not a chef
// problem — we don't expose it to mobile then.
var cancellableStatuses = map[models.OrderStatus]bool{
	models.OrderStatusAccepted:  true,
	models.OrderStatusPreparing: true,
	models.OrderStatusReady:     true,
}

type cancelRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CancelOrder cancels the whole order, issues a full Razorpay refund,
// and notifies the customer. Idempotent on a re-call: if the order
// is already cancelled with a refund ID we return 200 + the same
// payload so retries from the mobile client don't double-refund.
// POST /chef/orders/:orderId/cancel
func (h *ChefOrderCancelHandler) CancelOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderID, chef.ID).
		Preload("Items").First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req cancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := models.CancelReason(req.Reason)
	if !reason.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reason; expected out_of_ingredient|equipment_failure|customer_request|other"})
		return
	}

	// Idempotency — same order cancelled already with a refund?
	// Return the existing state instead of re-refunding.
	if order.Status == models.OrderStatusCancelled && order.RefundID != "" {
		c.JSON(http.StatusOK, order.ToResponse())
		return
	}

	if !cancellableStatuses[order.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order cannot be cancelled at this stage"})
		return
	}

	if order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "only Razorpay-paid orders can be cancelled from the chef app today; reach out to support for Stripe orders",
		})
		return
	}

	// Already-refunded amount (e.g. per-line cancels) reduces what we
	// can refund now. amountToRefund is in paise — Razorpay's native
	// unit. Floor + min 1 paise; if there's nothing left to refund we
	// flip the status and skip the gateway call.
	refundable := order.Total - order.RefundAmount
	amountPaise := int(roundPaise(refundable))
	var refundID string
	var refundedAt time.Time
	if amountPaise > 0 {
		rzp := services.GetRazorpay()
		if rzp == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "razorpay client unavailable; refund deferred"})
			return
		}
		refundResp, err := rzp.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
			Amount: amountPaise,
			Speed:  "normal",
			Notes: map[string]string{
				"order_id":    order.ID.String(),
				"order_no":    order.OrderNumber,
				"chef_id":     chef.ID.String(),
				"reason":      string(reason),
				"initiator":   "chef",
			},
		})
		if err != nil {
			services.CaptureSentryError(c, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "refund failed at gateway; please retry"})
			return
		}
		refundID = refundResp.ID
		refundedAt = time.Now().UTC()
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":              models.OrderStatusCancelled,
		"cancelled_at":        now,
		"cancel_reason":       string(reason),
		"refund_amount":       order.Total,
		"refund_reason":       string(reason),
		"refund_initiated_by": "chef",
	}
	if refundID != "" {
		updates["refund_id"] = refundID
		updates["refunded_at"] = refundedAt
	}
	if err := database.DB.Model(&order).Updates(updates).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}
	// Refresh the in-memory copy so the response reflects the saved state.
	_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error

	go publishOrderCancelled(order)

	c.JSON(http.StatusOK, order.ToResponse())
}

// CancelOrderItem marks a single line as unfulfillable, refunds only
// that line (subtotal + proportional tax share), and recomputes the
// order totals atomically. The remaining items continue prep.
// POST /chef/orders/:orderId/items/:itemId/cancel
func (h *ChefOrderCancelHandler) CancelOrderItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")
	itemID := c.Param("itemId")

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderID, chef.ID).
		Preload("Items").First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req cancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := models.CancelReason(req.Reason)
	if !reason.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reason"})
		return
	}

	if !cancellableStatuses[order.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order is no longer mid-prep; cannot cancel a single line"})
		return
	}

	// Locate the target line in the preloaded slice. Looping is fine —
	// orders cap at a handful of lines in practice.
	var target *models.OrderItem
	for i := range order.Items {
		if order.Items[i].ID.String() == itemID {
			target = &order.Items[i]
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found on this order"})
		return
	}
	if target.IsCancelled {
		// Idempotent — return the current order state without re-refunding.
		c.JSON(http.StatusOK, order.ToResponse())
		return
	}

	if order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "per-line refunds only supported on Razorpay orders today",
		})
		return
	}

	// Tax share for this line = order.Tax * (line.Subtotal / order.Subtotal).
	// Computed against the ORIGINAL subtotal/tax so concurrent partial
	// cancels can't drift the proportional split.
	lineRefund := target.Subtotal
	if order.Subtotal > 0 {
		lineRefund += order.Tax * (target.Subtotal / order.Subtotal)
	}
	amountPaise := int(roundPaise(lineRefund))
	if amountPaise <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "computed refund amount is zero; nothing to do"})
		return
	}

	rzp := services.GetRazorpay()
	if rzp == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "razorpay client unavailable; refund deferred"})
		return
	}
	refundResp, err := rzp.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
		Amount: amountPaise,
		Speed:  "normal",
		Notes: map[string]string{
			"order_id":      order.ID.String(),
			"order_item_id": target.ID.String(),
			"chef_id":       chef.ID.String(),
			"reason":        string(reason),
			"initiator":     "chef",
			"scope":         "line",
		},
	})
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "refund failed at gateway; please retry"})
		return
	}

	now := time.Now().UTC()
	// Single transaction so the item flip + order recompute land or
	// roll back together. The chef cannot end up with a refund that
	// the order totals don't reflect.
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(target).Updates(map[string]interface{}{
			"is_cancelled":     true,
			"cancelled_reason": string(reason),
			"cancelled_at":     now,
			"refund_id":        refundResp.ID,
			"refund_amount":    lineRefund,
		}).Error; err != nil {
			return err
		}

		// Re-read fresh order + items inside the txn so we don't race
		// a concurrent per-line cancel.
		var fresh models.Order
		if err := tx.Where("id = ?", order.ID).Preload("Items").First(&fresh).Error; err != nil {
			return err
		}

		newSubtotal := 0.0
		for _, it := range fresh.Items {
			if !it.IsCancelled {
				newSubtotal += it.Subtotal
			}
		}
		var newTax float64
		if fresh.Subtotal > 0 {
			newTax = fresh.Tax * (newSubtotal / fresh.Subtotal)
		}
		newTotal := newSubtotal + fresh.DeliveryFee + fresh.ServiceFee + newTax + fresh.Tip - fresh.Discount

		return tx.Model(&fresh).Updates(map[string]interface{}{
			"subtotal":      newSubtotal,
			"tax":           newTax,
			"total":         newTotal,
			"refund_amount": fresh.RefundAmount + lineRefund,
		}).Error
	})
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}

	// Refresh + emit. Skip status flip even if every line is now
	// cancelled — chef may intend to re-add via the customer-support
	// path; an explicit CancelOrder call is the right way to flip
	// status. (Could revisit later if zero-item-orders prove confusing.)
	_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error
	go publishOrderUpdated(order)

	c.JSON(http.StatusOK, order.ToResponse())
}

// loadChefForUser returns the ChefProfile owned by the given user.
// Wrapped so both cancel handlers can share the same not-found path.
func loadChefForUser(userID uuid.UUID) (models.ChefProfile, error) {
	var chef models.ChefProfile
	err := database.DB.Where("user_id = ?", userID).First(&chef).Error
	return chef, err
}

// roundPaise converts a rupee amount to paise (integer Razorpay unit)
// using round-half-away-from-zero so refunds match what the customer
// sees on their statement. Returns float64 only so callers can pick
// their own int width.
func roundPaise(rupees float64) float64 {
	paise := rupees * 100
	if paise >= 0 {
		return float64(int64(paise + 0.5))
	}
	return float64(int64(paise - 0.5))
}

// publishOrderCancelled emits the NATS event the customer + driver
// services consume to push notifications + flip dashboards.
func publishOrderCancelled(order models.Order) {
	if err := services.PublishOrderEvent(services.SubjectOrderCancelled, services.OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	}); err != nil {
		log.Printf("failed to publish order.cancelled: %v", err)
	}
}

// publishOrderUpdated emits the generic update event — used after
// per-line cancels since the order itself stays alive but totals
// have changed.
func publishOrderUpdated(order models.Order) {
	if err := services.PublishOrderEvent(services.SubjectOrderUpdated, services.OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	}); err != nil {
		log.Printf("failed to publish order.updated: %v", err)
	}
}

