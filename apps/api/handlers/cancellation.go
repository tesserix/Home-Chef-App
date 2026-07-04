package handlers

// cancellation.go — customer cancellation with vendor arbitration + tiered refund
// (epic #475). Flow: customer requests → auto fast-path (chef not engaged → full
// food refund) OR vendor review; vendor confirms a reason → tiered refund. The
// money math is the pure, tested calculator (services.ComputeCancellationRefund);
// the refund is issued via the hardened wallet-credit / gateway-refund + payout-
// hold cross-guard paths. Platform fee is always kept.

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type CancellationHandler struct{}

func NewCancellationHandler() *CancellationHandler { return &CancellationHandler{} }

// orderDispatched reports whether a driver is already carrying the order (delivery
// fee is then non-refundable — the driver is paid).
func orderDispatched(status models.OrderStatus) bool {
	switch status {
	case models.OrderStatusPickedUp, models.OrderStatusDelivering, models.OrderStatusDelivered:
		return true
	default:
		return false
	}
}

// snapshotFor computes the refund breakdown for an order at a food-refund %.
func snapshotFor(order *models.Order, pct int) services.CancellationRefund {
	return services.ComputeCancellationRefund(
		services.ToPaise(order.Subtotal),    // food (vendor)
		services.ToPaise(order.DeliveryFee), // delivery
		services.ToPaise(order.ServiceFee),  // platform fee — always kept
		services.ToPaise(order.Tax),         // tax
		orderDispatched(order.Status),
		pct,
	)
}

func applySnapshot(cr *models.CancellationRequest, s services.CancellationRefund) {
	cr.FoodRefundPaise = s.FoodRefund
	cr.DeliveryRefundPaise = s.DeliveryRefund
	cr.TaxRefundPaise = s.TaxRefund
	cr.RefundTotalPaise = s.Total
	cr.VendorKeptPaise = s.VendorKept
	cr.PlatformKeptPaise = s.PlatformKept
}

func validVendorReason(r string) bool {
	switch services.CancellationReason(r) {
	case services.CancelReasonNotStarted, services.CancelReasonMaterials,
		services.CancelReasonInPreparation, services.CancelReasonReady:
		return true
	default:
		return false
	}
}

// RequestCancellation — POST /orders/:id/cancel-request (customer).
func (h *CancellationHandler) RequestCancellation(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var body struct {
		Reason            string `json:"reason"`
		RefundDestination string `json:"refundDestination"` // wallet | original
	}
	_ = c.ShouldBindJSON(&body)
	dest := "wallet"
	if body.RefundDestination == "original" {
		dest = "original"
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderID, customerID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.PaymentStatus != models.PaymentCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "Only a paid order can be cancelled here"})
		return
	}
	// One request per order.
	var existing models.CancellationRequest
	if database.DB.Where("order_id = ?", orderID).First(&existing).Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A cancellation request already exists for this order", "request": existing})
		return
	}

	path, msg := services.ClassifyCancellation(order.Status)
	if path == services.CancelPathNotAllowed {
		c.JSON(http.StatusConflict, gin.H{"error": msg})
		return
	}

	cr := models.CancellationRequest{
		OrderID: orderID, CustomerID: customerID, ChefID: order.ChefID,
		CustomerReason: body.Reason, RefundDestination: dest,
	}

	if path == services.CancelPathFullRefund {
		// Chef not engaged (not accepted / rejected) → full food refund, no vendor.
		applySnapshot(&cr, snapshotFor(&order, 100))
		cr.Status = models.CancelReqAutoRefunded
		cr.VendorReason = string(services.CancelReasonNotStarted)
		now := time.Now()
		cr.ResolvedAt = &now
		if err := database.DB.Create(&cr).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create cancellation"})
			return
		}
		if err := h.executeRefund(&order, &cr); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "Cancelled, but the refund could not be issued — support will follow up", "request": cr})
			return
		}
		c.JSON(http.StatusOK, gin.H{"request": cr})
		return
	}

	// vendor_review — the chef confirms + picks a tier within the window.
	_, windowMin := services.ResolveCancellationTiers(database.DB)
	respondBy := time.Now().Add(time.Duration(windowMin) * time.Minute)
	cr.Status = models.CancelReqPendingVendor
	cr.VendorRespondBy = &respondBy
	if err := database.DB.Create(&cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create cancellation request"})
		return
	}
	// Tell the chef a cancellation is awaiting their confirmation.
	var chef models.ChefProfile
	if database.DB.Select("user_id").First(&chef, "id = ?", order.ChefID).Error == nil && chef.UserID != uuid.Nil {
		_ = services.EnqueueEvent(database.DB, services.SubjectChefNewOrder, "cancellation.requested", chef.UserID, map[string]any{
			"cancellation_request_id": cr.ID.String(), "order_id": orderID.String(),
		})
	}
	c.JSON(http.StatusCreated, gin.H{"request": cr})
}

// ConfirmCancellation — POST /chef/cancel-requests/:id/confirm (vendor).
func (h *CancellationHandler) ConfirmCancellation(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	reqID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request id"})
		return
	}
	var body struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || !validVendorReason(body.Reason) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reason must be one of not_started, materials_purchased, in_preparation, ready"})
		return
	}

	var cr models.CancellationRequest
	if err := database.DB.Where("id = ? AND chef_id = ? AND status = ?", reqID, chef.ID, models.CancelReqPendingVendor).
		First(&cr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No pending cancellation request found"})
		return
	}
	var order models.Order
	if err := database.DB.First(&order, "id = ?", cr.OrderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	tiers, _ := services.ResolveCancellationTiers(database.DB)
	pct := tiers.FoodRefundPct(services.CancellationReason(body.Reason))
	applySnapshot(&cr, snapshotFor(&order, pct))
	cr.VendorReason = body.Reason
	cr.Status = models.CancelReqApproved
	now := time.Now()
	cr.ResolvedAt = &now
	if err := database.DB.Save(&cr).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save the cancellation"})
		return
	}
	if err := h.executeRefund(&order, &cr); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Approved, but the refund could not be issued — support will follow up", "request": cr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": cr})
}

// executeRefund moves the money: it credits the customer's chosen destination for
// RefundTotalPaise, cancels the order, records the refund, and cross-guards the
// payout hold so the vendor is never paid the refunded slice (audit #10). The
// per-tier split is the SNAPSHOT already on cr; the platform fee is never in it.
func (h *CancellationHandler) executeRefund(order *models.Order, cr *models.CancellationRequest) error {
	refund := float64(cr.RefundTotalPaise) / 100.0

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if refund > 0 {
			if cr.RefundDestination == "original" {
				if order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == "" {
					return fmt.Errorf("original-method refund needs a razorpay payment")
				}
				rzp := services.GetRazorpay()
				if rzp == nil {
					return fmt.Errorf("razorpay unavailable")
				}
				resp, rErr := rzp.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
					Amount: cr.RefundTotalPaise, Speed: "normal",
					Notes: map[string]string{"order_id": order.ID.String(), "scope": "cancellation", "reason": cr.VendorReason},
				})
				if rErr != nil {
					return rErr
				}
				cr.RefundRef = resp.ID
			} else {
				if _, wErr := services.CreditWallet(tx, order.CustomerID, refund, models.WalletSourceRefund,
					&order.ID, "Cancellation refund", "cancel:"+cr.ID.String(), nil); wErr != nil {
					return wErr
				}
				cr.RefundRef = "wallet:cancel:" + cr.ID.String()
			}
		}
		now := time.Now()
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]any{
			"status":        models.OrderStatusCancelled,
			"refund_amount": order.RefundAmount + refund,
			"refunded_at":   now,
			"refund_reason": "customer cancellation",
		}).Error; err != nil {
			return err
		}
		cr.RefundExecuted = true
		// Key by order_id (one request per order, unique) so this works even when
		// the request id was assigned by the DB default and isn't on cr yet.
		return tx.Model(&models.CancellationRequest{}).Where("order_id = ?", order.ID).
			Updates(map[string]any{"refund_executed": true, "refund_ref": cr.RefundRef}).Error
	})
	if err != nil {
		return err
	}
	// Cross-guard the payout hold — the refunded slice must never reach the chef
	// (audit #10). Best-effort, after the refund committed.
	if hErr := services.WithholdOrReverseOrderHoldForRefund(database.DB, order.ID, "customer cancellation"); hErr != nil {
		log.Printf("cancellation payout cross-guard failed for order %s: %v", order.ID, hErr)
	}
	return nil
}

// ListChefCancellationRequests — GET /chef/cancel-requests?status= (vendor queue).
func (h *CancellationHandler) ListChefCancellationRequests(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}
	status := c.DefaultQuery("status", string(models.CancelReqPendingVendor))
	var reqs []models.CancellationRequest
	database.DB.Where("chef_id = ? AND status = ?", chef.ID, status).
		Order("created_at DESC").Find(&reqs)
	c.JSON(http.StatusOK, gin.H{"data": reqs})
}

// DisputeCancellation — POST /orders/:id/cancel-request/dispute (customer).
func (h *CancellationHandler) DisputeCancellation(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	var cr models.CancellationRequest
	if err := database.DB.Where("order_id = ? AND customer_id = ?", orderID, customerID).First(&cr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cancellation request not found"})
		return
	}
	// Only a vendor-decided refund can be disputed.
	if cr.Status != models.CancelReqApproved {
		c.JSON(http.StatusConflict, gin.H{"error": "This cancellation can't be disputed"})
		return
	}
	if err := database.DB.Model(&models.CancellationRequest{}).Where("id = ?", cr.ID).
		Updates(map[string]any{"status": models.CancelReqDisputed, "disputed": true, "dispute_reason": body.Reason}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not raise the dispute"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Dispute raised — our team will review it."})
}
