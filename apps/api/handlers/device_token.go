package handlers

import (
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

// fcmTokenRe matches the allowed charset of an FCM registration token (long,
// URL-safe strings). Length is bounded separately (100–4096) because Go's RE2
// caps inline repeat counts at 1000. Anything outside this charset/length is
// rejected so a malicious or malformed value can't be persisted (and later
// spliced into the FCM IID URL). An empty token is allowed separately
// (logout-clear).
var fcmTokenRe = regexp.MustCompile(`^[A-Za-z0-9:_-]+$`)

const (
	fcmTokenMinLen = 100
	fcmTokenMaxLen = 4096
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

	// Empty = logout-clear. A non-empty token must be well-formed.
	token := *req.Token
	if token != "" && (len(token) < fcmTokenMinLen || len(token) > fcmTokenMaxLen || !fcmTokenRe.MatchString(token)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device token"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := database.DB.Model(&user).Update("fcm_token", token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save device token"})
		return
	}

	// A device only ever belongs to one account. Clear this token from any
	// other user row so a re-installed / re-used device (same FCM token) never
	// keeps delivering pushes to a previous account.
	if token != "" {
		database.DB.Model(&models.User{}).
			Where("fcm_token = ? AND id <> ?", token, userID).
			Update("fcm_token", "")
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
