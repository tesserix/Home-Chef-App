package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// TestModeHandler owns the admin surface for test-chef mode: flipping a kitchen
// between live and test, browsing and purging its debugging sessions, and
// editing who may see sandbox kitchens.
type TestModeHandler struct{}

func NewTestModeHandler() *TestModeHandler { return &TestModeHandler{} }

// SetChefMode flips a kitchen between live and test.
//
// PATCH /admin/chefs/:id/mode  {"mode":"live|test","reason":"…","orderWindowDays":30}
//
// A live→test flip opens a numbered session and clones the kitchen's setup plus
// recent orders into it; test→live closes the session and returns the kitchen to
// its untouched live data. Refused with 409 and a structured blocker list when
// the kitchen still has orders or payouts in flight, so the UI can say exactly
// what is in the way.
func (h *TestModeHandler) SetChefMode(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var req struct {
		Mode            string `json:"mode"`
		Reason          string `json:"reason"`
		OrderWindowDays int    `json:"orderWindowDays"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, _ := middleware.GetUserID(c)
	mode := models.NormalizeMode(req.Mode)

	if !models.IsTestMode(mode) {
		if err := services.CloseTestSession(database.DB, chefID, adminID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		services.LogAudit(c, "chef.mode.live", "chef", chefID.String(), nil, map[string]any{
			"reason": req.Reason,
		})
		c.JSON(http.StatusOK, gin.H{"mode": models.ChefModeLive})
		return
	}

	// A reason is required: six months later nobody remembers why a kitchen was
	// put into a sandbox, and the session list is the only record.
	if req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A reason is required to move a kitchen into test mode"})
		return
	}

	session, err := services.OpenTestSession(database.DB, chefID, adminID, req.Reason, req.OrderWindowDays)
	if err != nil {
		if errors.Is(err, services.ErrFlipBlocked) {
			c.JSON(http.StatusConflict, gin.H{
				"error":    "This kitchen still has work in flight",
				"blockers": services.TestFlipBlockers(database.DB, chefID),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.LogAudit(c, "chef.mode.test", "chef", chefID.String(), nil, map[string]any{
		"reason":    req.Reason,
		"sessionNo": session.SessionNo,
		"sessionId": session.ID.String(),
	})
	c.JSON(http.StatusOK, gin.H{"mode": models.ChefModeTest, "session": session})
}

// GetChefTestSessions lists a kitchen's debugging sessions, newest first.
// GET /admin/chefs/:id/test-sessions
func (h *TestModeHandler) GetChefTestSessions(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var sessions []models.ChefTestSession
	if err := database.DB.Where("chef_id = ?", chefID).
		Order("session_no DESC").Find(&sessions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// PurgeTestSession deletes every row a closed session produced.
// DELETE /admin/test-sessions/:id
func (h *TestModeHandler) PurgeTestSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session id"})
		return
	}
	deleted, err := services.PurgeTestSession(database.DB, sessionID)
	if err != nil {
		if errors.Is(err, services.ErrSessionOpen) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.LogAudit(c, "chef.test_session.purge", "test_session", sessionID.String(), nil,
		map[string]any{"deleted": deleted})
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

// GetTestModePolicy returns the list of accounts that may see test kitchens.
// GET /admin/test-mode-policy
func (h *TestModeHandler) GetTestModePolicy(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetTestModePolicy())
}

// UpdateTestModePolicy replaces the viewer allowlist.
// PUT /admin/test-mode-policy  {"viewerEmails":["…"]}
//
// An empty list is a legitimate choice — it means no customer account can see a
// sandbox kitchen at all — so it is saved as given rather than backfilled.
func (h *TestModeHandler) UpdateTestModePolicy(c *gin.Context) {
	var policy services.TestModePolicy
	if err := c.ShouldBindJSON(&policy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	adminID, _ := middleware.GetUserID(c)
	if err := services.SaveTestModePolicy(policy, &adminID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.LogAudit(c, "platform.test_mode_policy.update", "platform_settings", "test_mode_policy", nil,
		map[string]any{"viewerCount": len(policy.ViewerEmails)})
	c.JSON(http.StatusOK, policy)
}
