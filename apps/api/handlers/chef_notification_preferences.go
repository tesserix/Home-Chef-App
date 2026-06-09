package handlers

import (
	"log"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"gorm.io/gorm"
)

// ChefNotificationPreferencesHandler exposes the chef's per-channel
// notification gating. A chef has at most one preferences row; GET
// returns defaults when no row exists so the mobile client always
// receives a usable shape.
type ChefNotificationPreferencesHandler struct{}

func NewChefNotificationPreferencesHandler() *ChefNotificationPreferencesHandler {
	return &ChefNotificationPreferencesHandler{}
}

// HH:MM 24-hour wall clock — what the mobile picker emits.
var hhmmRegex = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// GetPreferences returns the chef's current notification preferences,
// falling back to the system defaults (transactional on, promo off)
// if no row has been written yet.
// GET /chef/notification-preferences
func (h *ChefNotificationPreferencesHandler) GetPreferences(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var prefs models.ChefNotificationPreferences
	if err := database.DB.Where("chef_id = ?", chef.ID).First(&prefs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, models.DefaultNotificationPreferences(chef.ID))
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load preferences"})
		return
	}
	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences upserts the chef's preferences. Whenever the row
// changes we mirror the change to FCM topic subscriptions so the
// chef's device stops/starts receiving topic-broadcast pushes
// immediately, not on next app launch.
// PUT /chef/notification-preferences
func (h *ChefNotificationPreferencesHandler) UpdatePreferences(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var req struct {
		NewOrders         *bool   `json:"newOrders"`
		Payouts           *bool   `json:"payouts"`
		CustomerMessages  *bool   `json:"customerMessages"`
		Promo             *bool   `json:"promo"`
		QuietHoursEnabled *bool   `json:"quietHoursEnabled"`
		QuietHoursStart   *string `json:"quietHoursStart"`
		QuietHoursEnd     *string `json:"quietHoursEnd"`
		Timezone          *string `json:"timezone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.QuietHoursStart != nil && !hhmmRegex.MatchString(*req.QuietHoursStart) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quietHoursStart must be HH:MM 24h"})
		return
	}
	if req.QuietHoursEnd != nil && !hhmmRegex.MatchString(*req.QuietHoursEnd) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quietHoursEnd must be HH:MM 24h"})
		return
	}

	// Load or initialize. Defaults make the partial-update semantics
	// safe — fields the client doesn't send keep their current value.
	var prefs models.ChefNotificationPreferences
	err = database.DB.Where("chef_id = ?", chef.ID).First(&prefs).Error
	if err == gorm.ErrRecordNotFound {
		prefs = models.DefaultNotificationPreferences(chef.ID)
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load preferences"})
		return
	}

	// Snapshot the current state of the topic-driven categories so we
	// only call FCM for the ones that actually flipped. Network calls
	// to the firebase admin API aren't free and rate limits exist.
	prev := prefs

	if req.NewOrders != nil {
		prefs.NewOrders = *req.NewOrders
	}
	if req.Payouts != nil {
		prefs.Payouts = *req.Payouts
	}
	if req.CustomerMessages != nil {
		prefs.CustomerMessages = *req.CustomerMessages
	}
	if req.Promo != nil {
		prefs.Promo = *req.Promo
	}
	if req.QuietHoursEnabled != nil {
		prefs.QuietHoursEnabled = *req.QuietHoursEnabled
	}
	if req.QuietHoursStart != nil {
		prefs.QuietHoursStart = *req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		prefs.QuietHoursEnd = *req.QuietHoursEnd
	}
	if req.Timezone != nil {
		prefs.Timezone = *req.Timezone
	}

	if err := database.DB.Save(&prefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save preferences"})
		return
	}

	// Best-effort FCM topic reconcile. We deliberately do NOT fail the
	// HTTP request on FCM error — the prefs are authoritative, and the
	// in-app notification renderer ALSO reads them so the chef sees
	// the desired behavior even if a topic update lags.
	go reconcileChefFCMTopics(chef.UserID, prev, prefs)

	c.JSON(http.StatusOK, prefs)
}

// reconcileChefFCMTopics subscribes / unsubscribes the chef's device
// token from the per-category topic when a category bit flips. Skips
// any category that hasn't changed.
func reconcileChefFCMTopics(userID interface{}, prev, next models.ChefNotificationPreferences) {
	// Topics map 1:1 to category names so admin tooling and analytics
	// can refer to them by the same string everywhere.
	type topicSync struct {
		topic    string
		wasOn    bool
		nowOn    bool
	}
	syncs := []topicSync{
		{topic: "chef-new-orders", wasOn: prev.NewOrders, nowOn: next.NewOrders},
		{topic: "chef-payouts", wasOn: prev.Payouts, nowOn: next.Payouts},
		{topic: "chef-customer-messages", wasOn: prev.CustomerMessages, nowOn: next.CustomerMessages},
		{topic: "chef-promo", wasOn: prev.Promo, nowOn: next.Promo},
	}

	// Look up the user's FCM token. If they haven't registered one
	// yet (e.g. notification permission denied) there's nothing to do.
	uid, ok := userID.(interface{ String() string })
	if !ok {
		return
	}
	var user models.User
	if err := database.DB.Where("id = ?", uid.String()).First(&user).Error; err != nil || user.FCMToken == "" {
		return
	}

	for _, s := range syncs {
		if s.wasOn == s.nowOn {
			continue
		}
		var err error
		if s.nowOn {
			err = services.SubscribeToFCMTopic(user.FCMToken, s.topic)
		} else {
			err = services.UnsubscribeFromFCMTopic(user.FCMToken, s.topic)
		}
		if err != nil {
			log.Printf("fcm topic reconcile failed (token=…%s topic=%s on=%v): %v",
				lastN(user.FCMToken, 6), s.topic, s.nowOn, err)
		}
	}
}

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
