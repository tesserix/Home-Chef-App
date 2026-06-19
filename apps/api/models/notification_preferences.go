package models

import (
	"time"

	"github.com/google/uuid"
)

// ChefNotificationPreferences captures per-chef notification gating —
// which categories the chef wants to receive pushes for, plus
// optional quiet hours during which no non-critical pushes are sent.
//
// "New order" pushes are always sent regardless of quiet hours because
// missing one is a revenue event; the chef chose to be open. Quiet
// hours only suppress payout / customer-message / promo categories.
type ChefNotificationPreferences struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"-"`
	ChefID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"-"`

	// Category gates. true = chef wants this kind of push.
	NewOrders        bool `gorm:"default:true" json:"newOrders"`
	Payouts          bool `gorm:"default:true" json:"payouts"`
	CustomerMessages bool `gorm:"default:true" json:"customerMessages"`
	Promo            bool `gorm:"default:false" json:"promo"`

	// Quiet hours. Strings instead of time.Time so the schema doesn't
	// drag in a date — only the wall-clock window matters. "HH:MM"
	// 24-hour. End < Start means the window crosses midnight.
	QuietHoursEnabled bool   `gorm:"default:false" json:"quietHoursEnabled"`
	QuietHoursStart   string `gorm:"type:varchar(5);default:'22:00'" json:"quietHoursStart"`
	QuietHoursEnd     string `gorm:"type:varchar(5);default:'07:00'" json:"quietHoursEnd"`
	// IANA timezone string for resolving the wall-clock window into
	// real timestamps when the cron / push pipeline checks "is the
	// chef inside quiet hours right now?".
	Timezone string `gorm:"type:varchar(64);default:'Asia/Kolkata'" json:"timezone"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"-"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"-"`
}

// DefaultNotificationPreferences returns the default prefs returned by
// GET when a chef has no stored row yet. New chefs are opted IN to
// transactional pushes (new orders, payouts, customer messages) and
// OUT of promo. Matches Wave 2 design intent.
func DefaultNotificationPreferences(chefID uuid.UUID) ChefNotificationPreferences {
	return ChefNotificationPreferences{
		ChefID:            chefID,
		NewOrders:         true,
		Payouts:           true,
		CustomerMessages:  true,
		Promo:             false,
		QuietHoursEnabled: false,
		QuietHoursStart:   "22:00",
		QuietHoursEnd:     "07:00",
		Timezone:          "Asia/Kolkata",
	}
}
