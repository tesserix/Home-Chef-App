package handlers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// IssueReportWindow is how long after delivery a customer may still report an
// issue (#37). Beyond it the order is closed to disputes — a late-claim guard.
const IssueReportWindow = 48 * time.Hour

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
	if order.Status == models.OrderStatusCancelled || order.Status == models.OrderStatusRefunded {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This order is no longer eligible for a refund"})
		return
	}
	// Already fully refunded (status may still read 'completed') — nothing left.
	if order.RefundAmount >= order.Total {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This order has already been fully refunded"})
		return
	}
	// #37: reports are accepted only within IssueReportWindow of delivery. Beyond
	// that the order is closed to disputes — a late-claim / abuse guard. Only
	// enforced once the order has actually been delivered (DeliveredAt set); an
	// undelivered paid order isn't time-bound here.
	if order.DeliveredAt != nil && time.Since(*order.DeliveredAt) > IssueReportWindow {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "The 48-hour window to report an issue on this order has passed.",
		})
		return
	}

	// One open report per order. The UIs disable the submit button while a
	// request is in flight, but enforce it server-side too: this blocks duplicate
	// refunds from a retried/replayed request and stops a single order inflating
	// the chef's issue rate. A previously *rejected* report doesn't lock the order
	// (the customer may legitimately re-report with better evidence).
	var priorIssues int64
	database.DB.Model(&models.OrderIssue{}).
		Where("order_id = ? AND status <> ?", order.ID, models.IssueRejected).
		Count(&priorIssues)
	if priorIssues > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "You've already reported an issue on this order — our team is reviewing it."})
		return
	}

	reason := models.IssueReason(strings.TrimSpace(c.PostForm("reason")))
	if !models.ValidIssueReason(reason) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reason"})
		return
	}
	description := strings.TrimSpace(c.PostForm("description"))
	if len(description) > 1000 { // guard against oversized input from non-UI clients
		description = description[:1000]
	}

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
			// #622: a line already cancelled (and refunded) via CancelOrderItem must NOT be
			// counted as refundable here — RefundIssueToWallet would return that line's money a
			// SECOND time (the customer already got it back through the cancel gateway refund).
			// The symmetric guard to claimOrderItemForCancel's issue-refund check.
			if it.IsCancelled {
				continue
			}
			affectedSubtotals = append(affectedSubtotals, it.Subtotal)
			validAffected = append(validAffected, id)
		}
	}

	// Optional photos — up to maxIssuePhotos repeated `photo` fields. A single
	// `photo` field (the legacy single-photo client) still works unchanged.
	const maxIssuePhotos = 4
	var photoURLs []string
	if form, ferr := c.MultipartForm(); ferr == nil && form != nil {
		for i, header := range form.File["photo"] {
			if i >= maxIssuePhotos {
				break
			}
			if header.Size > 5*1024*1024 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Each photo must be 5 MB or smaller."})
				return
			}
			ct := header.Header.Get("Content-Type")
			if !services.IsImageContentType(ct) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid photo type. Allowed: JPEG, PNG, WebP."})
				return
			}
			file, oerr := header.Open()
			if oerr != nil {
				log.Printf("order issue photo open failed for order %s: %v", order.ID, oerr)
				continue
			}
			// Evidence photos can contain sensitive imagery, so store them in the
			// PRIVATE bucket (unguessable object key) and serve via short-lived
			// signed URLs at read time (see signIssuePhotoURLs). PhotoURLs holds
			// the object PATH, not a public URL.
			folder := fmt.Sprintf("order_issues/%s", order.ID.String())
			objectPath, uerr := services.UploadPrivateFile(c.Request.Context(), folder, header.Filename, file, ct)
			file.Close()
			if uerr != nil {
				log.Printf("order issue photo upload failed for order %s: %v", order.ID, uerr)
			} else {
				photoURLs = append(photoURLs, objectPath)
			}
		}
	}

	requested := services.ComputeIssueRefund(order.Subtotal, order.Tax, order.Total, order.RefundAmount, affectedSubtotals)

	// #618 slice 2: a quality issue on a delivered meal-plan DAY is reported against the
	// day's per-day fulfilment SHELL order. Link the issue to the day so the admin queue
	// can attribute it, and (below) freeze the day's payout hold so a pending report can't
	// release the day before it's resolved. The hold lives on meal_plan_days, not the shell
	// order. A normal food order has no linked day (record-not-found → nil, no freeze).
	var mealPlanDayID *uuid.UUID
	var mpDay models.MealPlanDay
	switch err := database.DB.Select("id", "status").Where("order_id = ?", order.ID).First(&mpDay).Error; {
	case err == nil:
		// #618 slice 2: a quality-issue refund is valid ONLY on a DELIVERED day. A day in
		// any other state — notably `failed` (a failed delivery already handled by the admin
		// delivery-failure resolver, whose RefundDay is the AUTHORITATIVE refunder on a
		// disjoint wallet key `mealplan-refund:<dayID>`) — must not also take this path's
		// `issue:<id>` refund, or the two independent writers double-refund the customer.
		// Rejecting keeps the issue-refund path and RefundDay on disjoint day populations.
		if mpDay.Status != models.MealPlanDayDelivered {
			c.JSON(http.StatusConflict, gin.H{"error": "This meal isn't eligible for a report right now — our team is handling it."})
			return
		}
		mealPlanDayID = &mpDay.ID
	case !errors.Is(err, gorm.ErrRecordNotFound):
		// Fail CLOSED: if this IS a meal-plan-day shell we must link + freeze it, so silently
		// falling through as a normal order would skip the day reconcile on the resulting
		// refund (a chef-overpay window). Ask the customer to retry rather than half-file it.
		log.Printf("order issue: meal-plan day lookup for order %s failed: %v", order.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not file your report"})
		return
	}

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
		MealPlanDayID:   mealPlanDayID,
	}
	if err := database.DB.Create(&issue).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not file your report"})
		return
	}

	// Issue rate per chef (#37) — every report counts toward the quality signal.
	database.DB.Model(&models.ChefProfile{}).Where("id = ?", order.ChefID).
		UpdateColumn("issue_count", gorm.Expr("issue_count + 1"))

	// #618 slice 2: freeze the linked meal-plan day's payout hold to `disputed` so a pending
	// report (or an admin reviewing it) blocks the day's release/auto-confirm before it's
	// resolved. Idempotent (none/awaiting→disputed only, #458) and money-safe — plain DB
	// state, runs regardless of the escrow flags. Best-effort: the report already committed.
	if mealPlanDayID != nil {
		if err := services.SetMealPlanDayHoldDisputed(database.DB, *mealPlanDayID); err != nil {
			log.Printf("order issue %s: freeze meal-plan day %s hold failed: %v", issue.ID, *mealPlanDayID, err)
		}
	}

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
		// Best-effort notification (matches the other notify-only enqueues); the
		// refund has already committed, so a notify failure must not fail the report.
		if err := services.EnqueueEvent(database.DB, services.SubjectOrderIssueReported, "order.issue.reported", chefUserID, map[string]any{
			"issue_id":     issue.ID.String(),
			"order_id":     order.ID.String(),
			"order_number": order.OrderNumber,
			"reason":       string(reason),
		}); err != nil {
			log.Printf("order issue %s: failed to enqueue chef notification: %v", issue.ID, err)
		}
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
	signIssuePhotoURLs(c.Request.Context(), issues)
	c.JSON(http.StatusOK, gin.H{"data": issues, "count": len(issues)})
}

// signIssuePhotoURLs replaces each stored private object path in an issue's
// PhotoURLs with a short-lived signed URL so the client can render evidence
// photos without the bucket being world-readable. The mutation is in-memory
// only (never persisted). Rows written before the private-bucket switch may
// already hold a full public URL — those are passed through unchanged.
func signIssuePhotoURLs(ctx context.Context, issues []models.OrderIssue) {
	for i := range issues {
		if len(issues[i].PhotoURLs) == 0 {
			continue
		}
		signed := make([]string, 0, len(issues[i].PhotoURLs))
		for _, p := range issues[i].PhotoURLs {
			if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
				signed = append(signed, p) // legacy public URL
				continue
			}
			u, err := services.GenerateSignedURL(ctx, p, 15*time.Minute)
			if err != nil {
				log.Printf("order issue: failed to sign photo %q: %v", p, err)
				continue
			}
			signed = append(signed, u)
		}
		issues[i].PhotoURLs = signed
	}
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
	signIssuePhotoURLs(c.Request.Context(), issues)
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
		// FaultPolicy (#618): chef_clawback (default) claws back the chef's payout;
		// platform_goodwill refunds the customer but lets the chef keep their payout.
		// Empty → the PlatformSettings default (order_issue.default_fault_policy).
		FaultPolicy string `json:"faultPolicy"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A positive amount is required"})
		return
	}

	// Resolve the fault policy: explicit choice, else the configured default (clawback).
	policy := models.IssueFaultPolicy(req.FaultPolicy)
	if policy == "" {
		policy = services.GetIssueConfig(database.DB).DefaultFaultPolicy
	}
	if policy != models.FaultChefClawback && policy != models.FaultPlatformGoodwill {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fault policy"})
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
	// Cap the assisted refund at the order's remaining refundable amount. Use
	// services.RemainingRefundable (#527/#560), NOT the naive Total − RefundAmount, which
	// under-states after a per-line cancel (the reduced Total minus the bumped RefundAmount
	// double-counts the cancelled line) and could falsely reject a legitimate refund on the
	// still-live items. RefundIssueToWallet re-caps authoritatively under the order lock —
	// including the #624 exclusion of affected lines cancelled between report and resolve — so
	// this is just the early "nothing left" guard.
	remaining := services.RemainingRefundable(&models.Order{ID: issue.OrderID})
	if req.Amount > remaining {
		req.Amount = remaining
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nothing left to refund on this order"})
		return
	}

	clawback := policy != models.FaultPlatformGoodwill
	if err := services.RefundIssueToWalletWithPolicy(database.DB, &issue, req.Amount, "admin", &adminID, clawback); err != nil {
		if errors.Is(err, services.ErrNothingToRefund) {
			// #624: every affected line named on this issue was already refunded via a per-line
			// cancel between the report and this resolve → the lock-time cap zeroed the credit.
			c.JSON(http.StatusConflict, gin.H{"error": "The items on this issue have already been refunded"})
			return
		}
		if errors.Is(err, services.ErrGoodwillFullRefund) {
			// #618: goodwill can't fully refund (refunded_at would block the chef payout). No
			// money moved — the admin lowers the amount or resolves with clawback instead.
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Platform goodwill applies to partial refunds only — lower the amount or claw back from the chef"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not issue the refund"})
		return
	}
	// Audit the fault decision (#618) — both clawback and goodwill, with the admin actor.
	services.LogAudit(c, "order_issue.resolved", "order_issue", issueID.String(), nil, gin.H{
		"policy": string(policy), "amount": req.Amount, "refundAmount": issue.RefundAmount,
	})
	c.JSON(http.StatusOK, gin.H{"status": string(issue.Status), "refundAmount": issue.RefundAmount, "faultPolicy": string(policy)})
}

// AdminRejectIssue declines an issue with no refund. A rejected dispute means the
// chef legitimately earned the payout, so — in the SAME transaction — it drives the
// order's disputed payout hold out of the dead-end back to release_eligible (#458).
// Folding both writes into one tx means the just-rejected issue is visible to the
// hold-drive's NOT EXISTS(pending) guard and closes any crash-stranding window.
func (h *OrderIssueHandler) AdminRejectIssue(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue id"})
		return
	}
	now := time.Now()

	var found, rejected bool
	if txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		// Load the issue's order so its payout hold can be driven in the same tx.
		var issue models.OrderIssue
		if err := tx.Select("id", "order_id").First(&issue, "id = ?", issueID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil // found stays false → 404
			}
			return err
		}
		found = true

		// #585: acquire the ORDER-row lock BEFORE the issue-row UPDATE, matching
		// RefundIssueToWallet's order-first ordering. Rejecting an issue then drives the
		// order hold (ReleaseDisputedHoldsForOrderIfCleared) — i.e. issue-then-order lock
		// order — while a concurrent RefundIssueToWallet locks order-then-issue; taking the
		// order lock first here makes both paths acquire (order, issue) in the same order,
		// so a concurrent reject + refund on the same order can't deadlock. Postgres-only
		// (sqlite has no row locks); a missing order has no hold to drive, so tolerate it.
		if issue.OrderID != uuid.Nil {
			lockTx := tx
			if tx.Dialector.Name() == "postgres" {
				lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
			}
			var o models.Order
			if err := lockTx.Select("id").First(&o, "id = ?", issue.OrderID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		res := tx.Model(&models.OrderIssue{}).
			Where("id = ? AND status = ?", issueID, models.IssuePending).
			Updates(map[string]any{"status": models.IssueRejected, "resolved_at": now, "resolved_by": adminID})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // already handled → rejected stays false → 409
		}
		rejected = true

		return services.ReleaseDisputedHoldsForOrderIfCleared(tx, issue.OrderID)
	}); txErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not reject the issue"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Issue not found"})
		return
	}
	if !rejected {
		c.JSON(http.StatusConflict, gin.H{"error": "This issue has already been handled"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": string(models.IssueRejected)})
}

// AdminResolveDeliveryFailure executes the admin-confirmed money policy for a
// `delivery_failed` issue (#393 slice 3, hybrid model). The driver's reason only
// SUGGESTED a fault; the admin confirms a concrete class and the matching outcome runs:
// customer-fault → no refund + release the vendor hold; platform/chef-fault → full
// customer refund + block the chef payout. Ambiguous is rejected — the admin must decide.
func (h *OrderIssueHandler) AdminResolveDeliveryFailure(c *gin.Context) {
	adminID, _ := middleware.GetUserID(c)
	issueID, err := uuid.Parse(c.Param("issueId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid issue id"})
		return
	}
	var req struct {
		Fault models.DeliveryFaultClass `json:"fault" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A fault class (customer, platform, or chef) is required"})
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

	switch err := services.ResolveDeliveryFailure(database.DB, &issue, req.Fault, adminID); {
	case errors.Is(err, services.ErrAmbiguousFault):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Confirm a concrete fault: customer, platform, or chef"})
	case errors.Is(err, services.ErrNotDeliveryFailure):
		c.JSON(http.StatusBadRequest, gin.H{"error": "This endpoint only resolves delivery-failure issues"})
	case errors.Is(err, services.ErrIssueAlreadyHandled):
		c.JSON(http.StatusConflict, gin.H{"error": "This issue has already been handled"})
	case errors.Is(err, services.ErrNothingToRefund):
		c.JSON(http.StatusConflict, gin.H{"error": "This order has already been fully refunded"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve the delivery failure"})
	default:
		c.JSON(http.StatusOK, gin.H{"status": "resolved", "fault": string(req.Fault)})
	}
}
