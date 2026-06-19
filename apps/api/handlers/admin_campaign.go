package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// admin_campaign.go — #56. Admin surface for marketing campaigns: compose +
// segment + preview + lifecycle. Dispatch/metrics endpoints live in
// admin_campaign_dispatch.go. All routes sit under the RequireAdmin group.

func campaignInputError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

// CreateCampaign composes a new draft campaign.
// POST /admin/campaigns
func (h *AdminHandler) CreateCampaign(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var in services.CampaignInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	uid := userID
	camp, err := services.CreateCampaign(database.DB, in, &uid)
	if err != nil {
		campaignInputError(c, err)
		return
	}
	c.JSON(http.StatusCreated, camp)
}

// UpdateCampaign edits a draft/scheduled campaign.
// PUT /admin/campaigns/:id
func (h *AdminHandler) UpdateCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	var in services.CampaignInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	camp, err := services.UpdateCampaign(database.DB, id, in)
	if err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, camp)
}

// ListCampaigns returns a page of campaigns, newest first.
// GET /admin/campaigns
func (h *AdminHandler) ListCampaigns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	camps, total, err := services.ListCampaigns(database.DB, limit, (page-1)*limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load campaigns"})
		return
	}
	totalPages := (total + int64(limit) - 1) / int64(limit)
	c.JSON(http.StatusOK, gin.H{
		"data": camps,
		"pagination": gin.H{
			"page": page, "limit": limit, "total": total,
			"totalPages": totalPages, "hasNext": int64(page) < totalPages, "hasPrev": page > 1,
		},
	})
}

// GetCampaign returns one campaign.
// GET /admin/campaigns/:id
func (h *AdminHandler) GetCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	camp, err := services.GetCampaign(database.DB, id)
	if err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, camp)
}

// DeleteCampaign soft-deletes a draft/cancelled campaign.
// DELETE /admin/campaigns/:id
func (h *AdminHandler) DeleteCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	if err := services.DeleteCampaign(database.DB, id); err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// CancelCampaign cancels a draft/scheduled/queued campaign.
// POST /admin/campaigns/:id/cancel
func (h *AdminHandler) CancelCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	camp, err := services.CancelCampaign(database.DB, id)
	if err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, camp)
}

// PreviewCampaignSegment counts the matched vs reachable audience for a segment
// without persisting anything — powers the live audience preview in the builder.
// POST /admin/campaigns/preview  { segment, sendPush, sendEmail }
func (h *AdminHandler) PreviewCampaignSegment(c *gin.Context) {
	var req struct {
		Segment   services.SegmentCriteria `json:"segment"`
		SendPush  bool                     `json:"sendPush"`
		SendEmail bool                     `json:"sendEmail"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	// Default to both channels for the preview if neither flagged, so the admin
	// always sees the reachable breakdown while composing.
	if !req.SendPush && !req.SendEmail {
		req.SendPush, req.SendEmail = true, true
	}
	res, err := services.SegmentPreview(database.DB, req.Segment, req.SendPush, req.SendEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to preview segment"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// campaignLifecycleError maps service errors to status codes.
func campaignLifecycleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrCampaignNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
	case errors.Is(err, services.ErrCampaignNotEditable):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrCampaignNoName), errors.Is(err, services.ErrCampaignNoChannel),
		errors.Is(err, services.ErrCampaignPushContent), errors.Is(err, services.ErrCampaignEmailContent):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
	}
}
