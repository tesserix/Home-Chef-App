package models

import (
	"time"

	"github.com/google/uuid"
)

// The two public signals a customer can give a kitchen, kept separate from
// FavoriteChef: that one is a curated shortlist capped at 7, so it can be
// neither an uncapped popularity signal nor the notification audience.
//
// A like is one tap and says "this is good" — it feeds ranking only.
// A subscription is a standing request to hear from the kitchen, so it feeds
// ranking AND is the list that menu drops, price changes, open/close and
// ChefBook posts fan out to.

// ChefLike is one customer's like of one kitchen. Uncapped.
type ChefLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_chef_likes_user_chef" json:"userId"`
	ChefID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_chef_likes_user_chef;index" json:"chefId"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	User User        `gorm:"foreignKey:UserID" json:"-"`
	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

// ChefSubscription is one customer's standing subscription to one kitchen.
//
// The per-kind flags default ON so subscribing is a single tap, and exist so a
// customer who only wants menu drops is not forced to unsubscribe entirely to
// stop the rest. They are ANDed with the user's global notification
// preferences, which still win.
type ChefSubscription struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_chef_subscriptions_user_chef" json:"userId"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_chef_subscriptions_user_chef;index" json:"chefId"`

	NotifyMenu         bool `gorm:"default:true" json:"notifyMenu"`
	NotifyPriceChange  bool `gorm:"default:true" json:"notifyPriceChange"`
	NotifyAvailability bool `gorm:"default:true" json:"notifyAvailability"`
	NotifyArticles     bool `gorm:"default:true" json:"notifyArticles"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	User User        `gorm:"foreignKey:UserID" json:"-"`
	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

// Fan-out kinds. The column driving each is resolved by SubscriptionNotifyColumn
// so a caller cannot silently notify against a column that does not exist.
const (
	ChefNotifyMenu         = "menu"
	ChefNotifyPriceChange  = "price_change"
	ChefNotifyAvailability = "availability"
	ChefNotifyArticles     = "articles"
)

// SubscriptionNotifyColumn maps a fan-out kind to its opt-out column, and
// reports whether the kind is known.
func SubscriptionNotifyColumn(kind string) (string, bool) {
	switch kind {
	case ChefNotifyMenu:
		return "notify_menu", true
	case ChefNotifyPriceChange:
		return "notify_price_change", true
	case ChefNotifyAvailability:
		return "notify_availability", true
	case ChefNotifyArticles:
		return "notify_articles", true
	}
	return "", false
}

// ChefAudienceState is what the customer app renders the like/subscribe controls
// from — the viewer's own state plus the public totals.
type ChefAudienceState struct {
	ChefID          uuid.UUID `json:"chefId"`
	Liked           bool      `json:"liked"`
	Subscribed      bool      `json:"subscribed"`
	LikeCount       int       `json:"likeCount"`
	SubscriberCount int       `json:"subscriberCount"`
}

// ChefSubscriptionResponse is one row of the customer's "kitchens you follow".
type ChefSubscriptionResponse struct {
	ID                 uuid.UUID           `json:"id"`
	ChefID             uuid.UUID           `json:"chefId"`
	Chef               ChefProfileResponse `json:"chef"`
	NotifyMenu         bool                `json:"notifyMenu"`
	NotifyPriceChange  bool                `json:"notifyPriceChange"`
	NotifyAvailability bool                `json:"notifyAvailability"`
	NotifyArticles     bool                `json:"notifyArticles"`
	CreatedAt          time.Time           `json:"createdAt"`
}
