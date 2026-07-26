package handlers

// moderation.go — the HTTP surface for reporting content and blocking users.
//
// App Review guideline 1.2 requires an app with user-generated content to offer
// a way to report objectionable content, a way to block abusive users, a filter
// on the content itself, and published contact details. This file is the first
// two; services/moderation.go holds the filter (AutoHideThreshold), and the
// contact details live in the apps' Legal screen and on fe3dr.com.
//
// A reviewer WILL exercise these. They must work on a fresh account with no
// orders, so nothing here requires any relationship to the reported party.

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type ModerationHandler struct{}

func NewModerationHandler() *ModerationHandler { return &ModerationHandler{} }

// CreateReport files a report against a piece of content.
//
// POST /v1/reports
//
//	{ "targetType": "review", "targetId": "...", "reason": "harassment",
//	  "details": "optional free text" }
func (h *ModerationHandler) CreateReport(c *gin.Context) {
	var req struct {
		TargetType string `json:"targetType" binding:"required"`
		TargetID   string `json:"targetId" binding:"required"`
		Reason     string `json:"reason" binding:"required"`
		Details    string `json:"details"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetType := models.ReportableType(req.TargetType)
	if !models.ValidReportableType(targetType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported targetType"})
		return
	}
	reason := models.ReportReason(req.Reason)
	if !models.ValidReportReason(reason) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported reason"})
		return
	}
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "targetId must be a UUID"})
		return
	}

	// Cap the free-text field. It reaches a human triage screen, and an
	// unbounded blob is both a storage and a rendering problem.
	details := req.Details
	if len(details) > 2000 {
		details = details[:2000]
	}

	reporterID, _ := middleware.GetUserID(c)
	report, err := services.CreateReport(database.DB, reporterID, targetType, targetID, reason, details)
	switch {
	case err == nil:
		// Deliberately not surfaced to the reporter: whether this particular
		// report tipped the content over the auto-hide threshold is information
		// about OTHER users' reports.
		if _, hideErr := services.ApplyAutoHide(database.DB, targetType, targetID); hideErr != nil {
			log.Printf("moderation: auto-hide check failed for %s/%s: %v", targetType, targetID, hideErr)
		}
		services.LogAudit(c, "moderation.report.create", string(targetType), targetID.String(),
			nil, gin.H{"reportId": report.ID, "reason": reason})
		c.JSON(http.StatusCreated, gin.H{
			"status":   "received",
			"reportId": report.ID,
			"message":  "Thanks — our team will review this within 24 hours.",
		})

	case errors.Is(err, services.ErrDuplicateReport):
		// Same answer as success. The user tapped Report twice; telling them
		// off for it serves nobody.
		c.JSON(http.StatusOK, gin.H{
			"status":   "received",
			"reportId": report.ID,
			"message":  "Thanks — our team will review this within 24 hours.",
		})

	case errors.Is(err, services.ErrTargetNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "That content no longer exists"})

	default:
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not file the report; please try again"})
	}
}

// BlockUser blocks another user.
//
// POST /v1/blocks   { "userId": "...", "reason": "harassment" }
func (h *ModerationHandler) BlockUser(c *gin.Context) {
	var req struct {
		UserID string `json:"userId" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	blockedID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId must be a UUID"})
		return
	}

	reason := models.ReportReason(req.Reason)
	if req.Reason != "" && !models.ValidReportReason(reason) {
		reason = models.ReasonOther
	}

	blockerID, _ := middleware.GetUserID(c)
	switch err := services.BlockUser(database.DB, blockerID, blockedID, reason); {
	case err == nil:
		services.LogAudit(c, "moderation.block.create", "user", blockedID.String(), nil, nil)
		c.JSON(http.StatusOK, gin.H{"status": "blocked"})
	case errors.Is(err, services.ErrSelfBlock):
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot block yourself"})
	default:
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not block this user; please try again"})
	}
}

// UnblockUser reverses a block.
//
// DELETE /v1/blocks/:userId
func (h *ModerationHandler) UnblockUser(c *gin.Context) {
	blockedID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId must be a UUID"})
		return
	}
	blockerID, _ := middleware.GetUserID(c)
	if err := services.UnblockUser(database.DB, blockerID, blockedID); err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not unblock; please try again"})
		return
	}
	services.LogAudit(c, "moderation.block.delete", "user", blockedID.String(), nil, nil)
	c.JSON(http.StatusOK, gin.H{"status": "unblocked"})
}

// ListBlocks returns who the caller has blocked, so the app can render a
// manage-blocks screen. Apple expects blocking to be reversible.
//
// GET /v1/blocks
func (h *ModerationHandler) ListBlocks(c *gin.Context) {
	blockerID, _ := middleware.GetUserID(c)

	var blocks []models.UserBlock
	if err := database.DB.Where("blocker_id = ?", blockerID).
		Order("created_at DESC").Find(&blocks).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load your blocked list"})
		return
	}

	type blockedUser struct {
		UserID    string `json:"userId"`
		Name      string `json:"name"`
		Reason    string `json:"reason,omitempty"`
		CreatedAt string `json:"createdAt"`
	}

	out := make([]blockedUser, 0, len(blocks))
	for _, b := range blocks {
		// Look the name up per row rather than preloading: the list is short by
		// nature, and a Preload here would pull the whole User row (including
		// encrypted PII columns) for every entry.
		var u models.User
		name := "Removed user"
		if err := database.DB.Select("id, first_name, last_name").
			First(&u, "id = ?", b.BlockedID).Error; err == nil {
			if n := u.FirstName + " " + u.LastName; len(n) > 1 {
				name = n
			}
		}
		out = append(out, blockedUser{
			UserID:    b.BlockedID.String(),
			Name:      name,
			Reason:    string(b.Reason),
			CreatedAt: b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	c.JSON(http.StatusOK, gin.H{"blocks": out})
}

// ─── Admin triage ────────────────────────────────────────────────────────────

// AdminListReports returns the triage queue, newest first.
//
// GET /v1/admin/reports?status=pending
func (h *ModerationHandler) AdminListReports(c *gin.Context) {
	q := database.DB.Model(&models.ContentReport{})
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}

	var reports []models.ContentReport
	if err := q.Order("created_at DESC").Limit(200).Find(&reports).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not load reports"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": reports, "count": len(reports)})
}

// AdminResolveReport records a triage decision.
//
// POST /v1/admin/reports/:id/resolve   { "status": "upheld", "note": "..." }
//
// Resolving does not itself remove content — the admin review/message tooling
// already does that, and conflating the two would make it impossible to record
// "upheld, content already deleted by its author".
func (h *ModerationHandler) AdminResolveReport(c *gin.Context) {
	reportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id must be a UUID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status := models.ReportStatus(req.Status)
	if status != models.ReportUpheld && status != models.ReportRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be upheld or rejected"})
		return
	}

	var report models.ContentReport
	if err := database.DB.First(&report, "id = ?", reportID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	adminID, _ := middleware.GetUserID(c)
	now := time.Now().UTC()
	if err := database.DB.Model(&report).Updates(map[string]any{
		"status":          status,
		"reviewed_by":     adminID,
		"reviewed_at":     now,
		"resolution_note": req.Note,
	}).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not resolve the report"})
		return
	}

	services.LogAudit(c, "moderation.report.resolve", string(report.TargetType),
		report.TargetID.String(), gin.H{"status": report.Status}, gin.H{"status": status})
	c.JSON(http.StatusOK, gin.H{"status": "resolved", "reportStatus": status})
}
