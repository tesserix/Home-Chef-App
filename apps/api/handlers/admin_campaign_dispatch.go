package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// admin_campaign_dispatch.go — #56. Schedule / send-now / test-send / metrics for
// marketing campaigns. The fan-out itself runs async on the CAMPAIGNS consumer;
// these endpoints just drive the lifecycle and read the delivery ledger.

// ScheduleCampaign sets a future send time.
// POST /admin/campaigns/:id/schedule  { "scheduledAt": "2026-07-01T09:00:00Z" }
func (h *AdminHandler) ScheduleCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	var req struct {
		ScheduledAt time.Time `json:"scheduledAt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A valid scheduledAt is required"})
		return
	}
	camp, err := services.ScheduleCampaign(database.DB, id, req.ScheduledAt)
	if err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, camp)
}

// SendCampaignNow queues a campaign for immediate dispatch.
// POST /admin/campaigns/:id/send
func (h *AdminHandler) SendCampaignNow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	camp, err := services.SendCampaignNow(database.DB, id)
	if err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, camp)
}

// TestSendCampaign sends the composed message to the requesting admin only.
// POST /admin/campaigns/:id/test
func (h *AdminHandler) TestSendCampaign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if err := services.TestSendCampaign(database.DB, id, userID); err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetCampaignMetrics returns the delivery/open overview for a campaign.
// GET /admin/campaigns/:id/metrics
func (h *AdminHandler) GetCampaignMetrics(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid campaign id"})
		return
	}
	m, err := services.GetCampaignMetrics(database.DB, id)
	if err != nil {
		campaignLifecycleError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}
