package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

// DeviceTokenHandler manages the authenticated user's push (FCM) device token.
// All three mobile apps (customer, vendor, delivery) register their token here
// after login so the server can target order/lifecycle pushes at the user.
type DeviceTokenHandler struct{}

// NewDeviceTokenHandler constructs a DeviceTokenHandler.
func NewDeviceTokenHandler() *DeviceTokenHandler {
	return &DeviceTokenHandler{}
}

// UpdateDeviceToken upserts (or clears) the caller's FCM device token.
//
// PUT /api/v1/profile/device-token  body: { "token": "<fcm token>" }
//
// A non-empty token is stored on the user row; an empty token clears it (used on
// logout / when the OS revokes notification permission). Push delivery reads
// user.fcm_token directly (services.SendToUser), so without this the server logs
// "Push skipped: user … has no FCM token" and no notification is ever delivered.
func (h *DeviceTokenHandler) UpdateDeviceToken(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		// Pointer so we can distinguish "field omitted" from an explicit "" clear.
		Token *string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if req.Token == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := database.DB.Model(&user).Update("fcm_token", *req.Token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save device token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
