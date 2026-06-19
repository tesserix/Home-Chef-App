package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// admin_order_issue.go — #37/#262. Admin config for the order-issue refund policy
// (auto-approve cap + enable toggle), stored as PlatformSettings `order_issue.*`
// keys. The report handler + reward engine read the same keys via GetIssueConfig.

// GetOrderIssueConfig returns the effective order-issue refund policy.
func (h *AdminHandler) GetOrderIssueConfig(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetIssueConfig(database.DB))
}

// UpdateOrderIssueConfig upserts the provided fields (partial PUT supported).
func (h *AdminHandler) UpdateOrderIssueConfig(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		Enabled        *bool    `json:"enabled"`
		AutoApproveCap *float64 `json:"autoApproveCap"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.Enabled != nil {
		setPlatformSetting("order_issue.enabled", strconv.FormatBool(*req.Enabled), userID)
	}
	if req.AutoApproveCap != nil {
		setPlatformSetting("order_issue.auto_approve_cap", strconv.FormatFloat(*req.AutoApproveCap, 'f', -1, 64), userID)
	}
	c.JSON(http.StatusOK, services.GetIssueConfig(database.DB))
}
