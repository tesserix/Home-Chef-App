package handlers

// admin_cancellation.go — admin arbitration of cancellation requests (#475/#480).
// Two things land here: customer DISPUTES of a vendor's refund tier, and vendor
// TIMEOUTS (pending → admin_review by the sweep). The admin picks the correct
// tier; the refund is executed (timeout, never refunded) or topped up to the new
// amount (dispute, already refunded). Platform fee stays nonrefundable throughout.

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// GetAdminCancellationRequests — GET /admin/cancel-requests?status= .
func (h *CancellationHandler) GetAdminCancellationRequests(c *gin.Context) {
	q := database.DB.Order("created_at DESC").Limit(200)
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	} else {
		q = q.Where("status IN ?", []models.CancellationRequestStatus{
			models.CancelReqDisputed, models.CancelReqAdminReview,
		})
	}
	var reqs []models.CancellationRequest
	q.Find(&reqs)
	c.JSON(http.StatusOK, gin.H{"data": reqs})
}

// ResolveCancellationAdmin — POST /admin/cancel-requests/:id/resolve.
func (h *CancellationHandler) ResolveCancellationAdmin(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request id"})
		return
	}
	var body struct {
		Reason string `json:"reason" binding:"required"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !validVendorReason(body.Reason) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason must be one of not_started, materials_purchased, in_preparation, ready"})
		return
	}

	var cr models.CancellationRequest
	if err := database.DB.Where("id = ? AND status IN ?", reqID,
		[]models.CancellationRequestStatus{models.CancelReqDisputed, models.CancelReqAdminReview}).
		First(&cr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No cancellation request awaiting admin review"})
		return
	}
	var order models.Order
	if err := database.DB.First(&order, "id = ?", cr.OrderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	tiers, _ := services.ResolveCancellationTiers(database.DB)
	pct := tiers.FoodRefundPct(services.CancellationReason(body.Reason))
	now := time.Now()

	if cr.Status == models.CancelReqAdminReview {
		// Timed out — never refunded. Cap at the remaining balance (no prior refund of its own),
		// set the admin's tier, and issue the refund.
		newSnap := snapshotFor(&order, pct)
		applySnapshot(&cr, newSnap)
		cr.VendorReason = body.Reason
		if err := services.ExecuteCancellationRefund(&order, &cr); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Resolved, but the refund could not be issued — see ops"})
			return
		}
	} else {
		// Disputed — cr.RefundTotalPaise was ALREADY refunded and is folded into order.RefundAmount.
		// This re-snapshot SUPERSEDES that earlier refund (the top-up below is the difference), so
		// the cap basis is Total − OTHER refunds = RemainingRefundable + this cancellation's own
		// prior refund. Capping at the bare RemainingRefundable would double-subtract
		// cr.RefundTotalPaise and silently shortchange the admin's ruling (#642 verify). Still
		// money-safe: cumulative ≤ Total, and we never claw back.
		newSnap := uncappedSnapshot(&order, pct).CappedAt(remainingRefundablePaise(&order) + cr.RefundTotalPaise)
		delta := newSnap.Total - cr.RefundTotalPaise
		if delta > 0 {
			if _, err := services.CreditWallet(database.DB, order.CustomerID, float64(delta)/100.0,
				models.WalletSourceRefund, &order.ID, "Cancellation dispute adjustment",
				"cancel-adjust:"+cr.ID.String(), &adminID); err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": "Could not issue the adjustment"})
				return
			}
			_ = services.EnqueueEvent(database.DB, services.SubjectCancellationResolved, "cancellation.resolved", order.CustomerID, map[string]any{
				"order_id": order.ID.String(), "refund": float64(newSnap.Total) / 100.0,
			})
		}
		applySnapshot(&cr, newSnap)
		cr.VendorReason = body.Reason
	}

	if err := database.DB.Model(&models.CancellationRequest{}).Where("id = ?", cr.ID).
		Updates(map[string]any{
			"status": models.CancelReqResolved, "admin_resolved_by": adminID,
			"admin_note": body.Note, "resolved_at": now, "vendor_reason": cr.VendorReason,
			"food_refund_paise": cr.FoodRefundPaise, "delivery_refund_paise": cr.DeliveryRefundPaise,
			"tax_refund_paise": cr.TaxRefundPaise, "refund_total_paise": cr.RefundTotalPaise,
			"vendor_kept_paise": cr.VendorKeptPaise, "platform_kept_paise": cr.PlatformKeptPaise,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the resolution"})
		return
	}
	services.LogAudit(c, "cancellation.resolved", "cancellation_request", cr.ID.String(),
		nil, gin.H{"reason": body.Reason, "note": body.Note})
	cr.Status = models.CancelReqResolved
	c.JSON(http.StatusOK, gin.H{"request": cr})
}
