package handlers

// cancellation.go — customer cancellation with vendor arbitration + tiered refund
// (epic #475). Flow: customer requests → auto fast-path (chef not engaged → full
// food refund) OR vendor review; vendor confirms a reason → tiered refund. The
// money math is the pure, tested calculator (services.ComputeCancellationRefund);
// the refund is issued via the hardened wallet-credit / gateway-refund + payout-
// hold cross-guard paths. Platform fee is always kept.

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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

// uncappedSnapshot is the pure tiered refund breakdown at a food-refund % (no remaining-balance
// cap). Callers apply the cap with the basis appropriate to their path (see snapshotFor and the
// admin disputed-adjust in admin_cancellation.go).
func uncappedSnapshot(order *models.Order, pct int) services.CancellationRefund {
	return services.ComputeCancellationRefund(
		services.ToPaise(order.Subtotal),    // food (vendor)
		services.ToPaise(order.DeliveryFee), // delivery
		services.ToPaise(order.ServiceFee),  // platform fee — always kept
		services.ToPaise(order.Tax),         // tax
		orderDispatched(order.Status),
		pct,
	)
}

// remainingRefundablePaise is the order's still-refundable balance (Total − already-refunded), the
// #642 cap basis.
func remainingRefundablePaise(order *models.Order) int {
	return services.ToPaise(services.RemainingRefundable(order))
}

// snapshotFor computes the refund breakdown, capped at what's STILL owed (Total − already-refunded)
// so a cancellation after a prior partial refund (e.g. a customer-issue refund) can't return more
// than the remaining balance (#642). For a FRESH cancellation (no prior refund OF ITS OWN yet):
// request, vendor-confirm, and the admin-review-timeout path. The admin DISPUTED-adjust path uses
// a different basis (it supersedes its own already-issued refund) — see admin_cancellation.go.
func snapshotFor(order *models.Order, pct int) services.CancellationRefund {
	return uncappedSnapshot(order, pct).CappedAt(remainingRefundablePaise(order))
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

// resolveRefundDestination decides where a cancellation refund lands.
//
// Refunds go back to the ORIGINAL payment method. The customer used to pick
// wallet-vs-original and the picker defaulted to "wallet", which quietly made
// store credit the normal outcome of a cancellation — and store credit is
// currently UNSPENDABLE: WalletCheckoutEnabled (WALLET_CHECKOUT_ENABLED, #141)
// is off in production, so wallet credit cannot be applied at checkout. Money
// refunded there is money the customer cannot use or withdraw, and no refund
// ever reaches the gateway. Original-method is the only destination a customer
// can actually realise, so it is no longer a choice.
//
// Wallet remains the fallback for the one case where the gateway genuinely
// cannot be re-credited — no Razorpay payment to refund against (e.g. an order
// settled entirely from store credit back when wallet checkout was on).
// ExecuteCancellationRefund hard-errors on "original" without a Razorpay
// payment, so resolving it here keeps that path unreachable.
//
// If wallet checkout is ever re-enabled AND the wallet is proven spendable
// end-to-end, reintroducing the choice is a deliberate product decision — not
// a default.
func resolveRefundDestination(order *models.Order) string {
	if strings.EqualFold(order.PaymentProvider, "razorpay") && order.RazorpayPaymentID != "" {
		return "original"
	}
	return "wallet"
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
		Reason string `json:"reason"`
		// RefundDestination is accepted for compatibility with older clients but is
		// no longer customer-selectable — see resolveRefundDestination.
		RefundDestination string `json:"refundDestination"`
	}
	_ = c.ShouldBindJSON(&body)

	var order models.Order
	if err := database.DB.Where("id = ? AND customer_id = ?", orderID, customerID).First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.PaymentStatus != models.PaymentCompleted {
		c.JSON(http.StatusConflict, gin.H{"error": "Only a paid order can be cancelled here"})
		return
	}
	dest := resolveRefundDestination(&order)
	// #544/#394: an order spawned by a typed escrow flow (meal-plan day / group order) is
	// refund-managed by THAT flow on a disjoint idempotency keyspace; its held chef payout is a
	// DIRECT transfer this arbitration flow's generic refund (ExecuteCancellationRefund, wallet key
	// cancel:<crID>) can't reverse. Refuse to open a cancellation request on one — cancel it via
	// the meal-plan / group flow instead. This is the SOLE CancellationRequest creation site, so
	// the guard here keeps ExecuteCancellationRefund (and the retry sweep) off typed orders
	// entirely. Mirrors the InitiateRefund guard (handlers/payment.go).
	switch kind, kErr := services.TypedRefundOrderKind(database.DB, orderID); {
	case kErr != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check order type"})
		return
	case kind == services.TypedRefundMealPlanDay:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "This order is part of a meal plan; cancel it through the meal-plan cancellation flow, not the generic order cancellation"})
		return
	case kind == services.TypedRefundGroupOrder:
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "This is a group order; cancel it through the group-order cancellation flow, not the generic order cancellation"})
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
		// If this inline attempt fails, the request stays refund_executed=false and
		// the cancellation-sweep retries it — the refund is never lost.
		if err := services.ExecuteCancellationRefund(&order, &cr); err != nil {
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
		_ = services.EnqueueEvent(database.DB, services.SubjectCancellationRequested, "cancellation.requested", chef.UserID, map[string]any{
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
	// A failure here leaves refund_executed=false; the cancellation-sweep retries.
	if err := services.ExecuteCancellationRefund(&order, &cr); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Approved, but the refund could not be issued — support will follow up", "request": cr})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": cr})
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

// GetCancellationRequest — GET /orders/:id/cancel-request (customer). Returns the
// request (status + refund snapshot) so the customer app can show progress.
func (h *CancellationHandler) GetCancellationRequest(c *gin.Context) {
	customerID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var cr models.CancellationRequest
	if err := database.DB.Where("order_id = ? AND customer_id = ?", orderID, customerID).First(&cr).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No cancellation request for this order"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"request": cr})
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
