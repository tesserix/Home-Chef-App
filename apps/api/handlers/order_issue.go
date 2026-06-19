package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// order_issue.go — live order issue reporting → instant refund (#37). The
// customer reports a problem on their order with an optional photo; small clear
// cases auto-refund to the wallet, larger ones await assisted (admin) review.
// Routes the issue to the chef via the outbox/NATS notification pipeline.

type OrderIssueHandler struct{}

func NewOrderIssueHandler() *OrderIssueHandler { return &OrderIssueHandler{} }

// ReportIssue creates an order issue and, when eligible, instantly refunds the
// affected items to the customer's wallet.
// POST /orders/:id/report-issue — multipart: reason, description, affectedItemIds[], photo?
func (h *OrderIssueHandler) ReportIssue(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}

	var order models.Order
	if err := database.DB.Preload("Items").
		Where("id = ? AND customer_id = ?", orderID, userID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Reportable only on a paid order that isn't cancelled/fully refunded.
	if order.PaymentStatus != models.PaymentCompleted {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You can only report an issue on a paid order"})
		return
	}
	if order.Status == models.OrderStatusCancelled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This order was cancelled"})
		return
	}

	reason := models.IssueReason(strings.TrimSpace(c.PostForm("reason")))
	if !models.ValidIssueReason(reason) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reason"})
		return
	}
	description := strings.TrimSpace(c.PostForm("description"))

	// Affected items: repeated `affectedItemIds` fields and/or one comma-joined
	// value. Validate they belong to this order; collect their subtotals.
	affectedIDs := normalizeIDList(c.PostFormArray("affectedItemIds"))
	itemByID := map[string]models.OrderItem{}
	for _, it := range order.Items {
		itemByID[it.ID.String()] = it
	}
	var affectedSubtotals []float64
	var validAffected []string
	for _, id := range affectedIDs {
		if it, ok := itemByID[id]; ok {
			affectedSubtotals = append(affectedSubtotals, it.Subtotal)
			validAffected = append(validAffected, id)
		}
	}

	// Optional photo.
	var photoURLs []string
	if file, header, ferr := c.Request.FormFile("photo"); ferr == nil {
		defer file.Close()
		if header.Size > 5*1024*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Photo too large. Maximum 5 MB."})
			return
		}
		ct := header.Header.Get("Content-Type")
		if !services.IsImageContentType(ct) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo type. Allowed: JPEG, PNG, WebP."})
			return
		}
		folder := fmt.Sprintf("order_issues/%s", order.ID.String())
		url, uerr := services.UploadPublicFile(c.Request.Context(), folder, header.Filename, file, ct)
		if uerr != nil {
			log.Printf("order issue photo upload failed for order %s: %v", order.ID, uerr)
		} else {
			photoURLs = append(photoURLs, url)
		}
	}

	requested := services.ComputeIssueRefund(order.Subtotal, order.Tax, order.Total, order.RefundAmount, affectedSubtotals)

	issue := models.OrderIssue{
		OrderID:         order.ID,
		ChefID:          order.ChefID,
		CustomerID:      userID,
		Reason:          reason,
		Description:     description,
		PhotoURLs:       photoURLs,
		AffectedItemIDs: validAffected,
		RequestedAmount: requested,
		Status:          models.IssuePending,
	}
	if err := database.DB.Create(&issue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not file your report"})
		return
	}

	// Issue rate per chef (#37) — every report counts toward the quality signal.
	database.DB.Model(&models.ChefProfile{}).Where("id = ?", order.ChefID).
		UpdateColumn("issue_count", gorm.Expr("issue_count + 1"))

	// Auto-refund small/clear cases instantly; otherwise leave pending for admin.
	cfg := services.GetIssueConfig(database.DB)
	if services.ShouldAutoRefund(cfg, requested) {
		if err := services.RefundIssueToWallet(database.DB, &issue, requested, "system", nil); err != nil {
			log.Printf("order issue auto-refund failed for issue %s: %v", issue.ID, err)
		}
	}

	// Route to the chef (in-app + push) via the outbox.
	var chefUserID uuid.UUID
	database.DB.Model(&models.ChefProfile{}).Select("user_id").Where("id = ?", order.ChefID).Scan(&chefUserID)
	if chefUserID != uuid.Nil {
		_ = services.EnqueueEvent(database.DB, services.SubjectOrderIssueReported, "order.issue.reported", chefUserID, map[string]any{
			"issue_id":     issue.ID.String(),
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"reason":       string(reason),
		})
	}

	message := "Thanks for letting us know — our team will review this."
	if issue.Status == models.IssueAutoRefunded {
		message = "We've refunded you to your wallet. Sorry about that!"
	}
	c.JSON(http.StatusCreated, gin.H{
		"issueId":      issue.ID,
		"status":       string(issue.Status),
		"refundAmount": issue.RefundAmount,
		"message":      message,
	})
}

// GetMyOrderIssues lists the caller's issues for an order.
func (h *OrderIssueHandler) GetMyOrderIssues(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var issues []models.OrderIssue
	database.DB.Where("order_id = ? AND customer_id = ?", orderID, userID).
		Order("created_at DESC").Find(&issues)
	c.JSON(http.StatusOK, gin.H{"data": issues, "count": len(issues)})
}

// normalizeIDList flattens repeated + comma-joined form values into a de-duped,
// trimmed slice.
func normalizeIDList(vals []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range vals {
		for _, part := range strings.Split(v, ",") {
			p := strings.TrimSpace(part)
			if p != "" && !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

// ── Admin (#262): list + resolve/reject assisted issues ─────────────────────

// AdminListIssues returns order issues, newest first, optionally filtered by status.
func (h *OrderIssueHandler) AdminListIssues(c *gin.Context) {
	q := database.DB.Model(&models.OrderIssue{}).Order("created_at DESC")
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	var issues []models.OrderIssue
	q.Limit(200).Find(&issues)
	c.JSON(http.StatusOK, gin.H{"data": issues, "count": len(issues)})
}

// AdminResolveIssue approves an assisted refund (amount provided by the admin),
// crediting the wallet via the same exactly-once path as the auto-refund.
func (h *OrderIssueHandler) AdminResolveIssue(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue id"})
		return
	}
	var req struct {
		Amount float64 `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A positive amount is required"})
		return
	}

	var issue models.OrderIssue
	if err := database.DB.First(&issue, "id = ?", issueID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		return
	}
	if issue.Status != models.IssuePending {
		c.JSON(http.StatusConflict, gin.H{"error": "This issue has already been handled"})
		return
	}
	// Cap the assisted refund at the order's remaining refundable amount.
	var order models.Order
	if err := database.DB.Select("total, refund_amount").First(&order, "id = ?", issue.OrderID).Error; err == nil {
		if remaining := order.Total - order.RefundAmount; req.Amount > remaining {
			req.Amount = remaining
		}
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nothing left to refund on this order"})
		return
	}

	if err := services.RefundIssueToWallet(database.DB, &issue, req.Amount, "admin", &adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not issue the refund"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": string(issue.Status), "refundAmount": issue.RefundAmount})
}

// AdminRejectIssue declines an issue with no refund.
func (h *OrderIssueHandler) AdminRejectIssue(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue id"})
		return
	}
	now := time.Now()
	res := database.DB.Model(&models.OrderIssue{}).
		Where("id = ? AND status = ?", issueID, models.IssuePending).
		Updates(map[string]any{"status": models.IssueRejected, "resolved_at": now, "resolved_by": adminID})
	if res.RowsAffected == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "This issue has already been handled"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": string(models.IssueRejected)})
}
