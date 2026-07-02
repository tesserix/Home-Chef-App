package services

import (
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// Channel identifies which transport we're considering for a given dispatch.
type Channel string

const (
	ChannelEmail Channel = "email"
	ChannelPush  Channel = "push"
	ChannelSMS   Channel = "sms"
)

// notificationTypeCategory maps a NotificationEvent.Data["type"] value to
// the coarse category the user toggles in their settings. New event types
// default to "account" if not listed here — adjust as new categories
// emerge rather than falling through silently.
func notificationTypeCategory(notifType string) models.NotificationCategory {
	switch notifType {
	case "order_confirmation", "order_status", "order_cancelled", "order_delivered":
		return models.NotifCategoryOrder
	case "chef_new_order", "chef_verified", "chef_rejected":
		return models.NotifCategoryChef
	case "delivery_assigned", "delivery_picked_up", "delivery_arrived":
		return models.NotifCategoryDelivery
	case "promo", "marketing":
		return models.NotifCategoryMarketing
	case "weekly_menu_published", "daily_menu_published":
		return models.NotifCategoryFavorites
	default:
		return models.NotifCategoryAccount
	}
}

// ShouldSend looks up the user's NotificationPreference for the given
// category + channel. Missing row = default (see models.DefaultNotificationPreference).
// Returns true when the channel is enabled for the user.
func ShouldSend(userID uuid.UUID, category models.NotificationCategory, channel Channel) bool {
	var pref models.NotificationPreference
	err := database.DB.
		Where("user_id = ? AND category = ?", userID, category).
		First(&pref).Error
	if err != nil {
		pref = models.DefaultNotificationPreference(category)
	}
	switch channel {
	case ChannelEmail:
		return pref.EmailEnabled
	case ChannelPush:
		return pref.PushEnabled
	case ChannelSMS:
		return pref.SMSEnabled
	}
	return false
}

// ShouldSendForType is a convenience that maps a NotificationEvent.Data["type"]
// to its category and evaluates ShouldSend in one call.
func ShouldSendForType(userID uuid.UUID, notifType string, channel Channel) bool {
	return ShouldSend(userID, notificationTypeCategory(notifType), channel)
}

// GetUserPreferences returns every category with the effective settings
// (DB row if present, defaults otherwise). Handy for the profile UI so the
// client can show the whole grid in one payload.
func GetUserPreferences(userID uuid.UUID) []models.NotificationPreference {
	var stored []models.NotificationPreference
	database.DB.Where("user_id = ?", userID).Find(&stored)
	byCat := make(map[models.NotificationCategory]models.NotificationPreference, len(stored))
	for _, p := range stored {
		byCat[p.Category] = p
	}

	out := make([]models.NotificationPreference, 0, len(models.AllNotificationCategories()))
	for _, cat := range models.AllNotificationCategories() {
		if p, ok := byCat[cat]; ok {
			out = append(out, p)
			continue
		}
		def := models.DefaultNotificationPreference(cat)
		def.UserID = userID
		out = append(out, def)
	}
	return out
}

// UpsertUserPreference creates or updates a single category's row for a user.
func UpsertUserPreference(userID uuid.UUID, cat models.NotificationCategory, email, push, sms bool) error {
	var pref models.NotificationPreference
	err := database.DB.
		Where("user_id = ? AND category = ?", userID, cat).
		First(&pref).Error
	if err != nil {
		pref = models.NotificationPreference{
			UserID:       userID,
			Category:     cat,
			EmailEnabled: email,
			PushEnabled:  push,
			SMSEnabled:   sms,
		}
		return database.DB.Create(&pref).Error
	}
	pref.EmailEnabled = email
	pref.PushEnabled = push
	pref.SMSEnabled = sms
	return database.DB.Save(&pref).Error
}
