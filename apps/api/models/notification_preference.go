package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationCategory is the broad bucket a notification falls into so
// users can opt out of a whole class (e.g. "marketing") without muting
// critical transactional alerts (e.g. "order").
type NotificationCategory string

const (
	NotifCategoryOrder     NotificationCategory = "order"     // order confirmations + status
	NotifCategoryChef      NotificationCategory = "chef"      // chef-facing (new order, verified, etc.)
	NotifCategoryDelivery  NotificationCategory = "delivery"  // driver assignment, pickup, drop-off
	NotifCategoryAccount   NotificationCategory = "account"   // welcome, password, approvals
	NotifCategoryMarketing NotificationCategory = "marketing" // promos, newsletters
)

// AllNotificationCategories powers the admin/user settings UI so the list of
// toggles is driven by this single source of truth.
func AllNotificationCategories() []NotificationCategory {
	return []NotificationCategory{
		NotifCategoryOrder,
		NotifCategoryChef,
		NotifCategoryDelivery,
		NotifCategoryAccount,
		NotifCategoryMarketing,
	}
}

// NotificationPreference is one user's opt-in/out per category per channel.
// Missing rows fall through to DefaultNotificationPreference — we store only
// explicit overrides rather than seeding a row per user per category.
type NotificationPreference struct {
	ID          uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID      uuid.UUID            `gorm:"type:uuid;not null;index:idx_notif_pref_user_cat,unique,composite:user_cat" json:"userId"`
	Category    NotificationCategory `gorm:"type:varchar(32);not null;index:idx_notif_pref_user_cat,unique,composite:user_cat" json:"category"`
	EmailEnabled bool                `gorm:"default:true" json:"emailEnabled"`
	PushEnabled  bool                `gorm:"default:true" json:"pushEnabled"`
	SMSEnabled   bool                `gorm:"default:false" json:"smsEnabled"`
	UpdatedAt   time.Time            `gorm:"autoUpdateTime" json:"updatedAt"`
	CreatedAt   time.Time            `gorm:"autoCreateTime" json:"createdAt"`
}

// DefaultNotificationPreference is what the system assumes when a user
// hasn't set a per-category override. Order and delivery updates are
// opt-out (they're transactional); marketing is opt-in.
func DefaultNotificationPreference(cat NotificationCategory) NotificationPreference {
	switch cat {
	case NotifCategoryMarketing:
		return NotificationPreference{
			Category:     cat,
			EmailEnabled: false,
			PushEnabled:  false,
			SMSEnabled:   false,
		}
	default:
		return NotificationPreference{
			Category:     cat,
			EmailEnabled: true,
			PushEnabled:  true,
			SMSEnabled:   false,
		}
	}
}
