// chef_audience.go — likes and subscriptions: who is listening to a kitchen,
// and how popular it is.
//
// Every mutation moves the denormalised counter on chef_profiles in the SAME
// transaction as the row, and only when the row actually changed — so a double
// tap or a retried request cannot inflate a chef's ranking.

package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrChefNotFound is returned when the target kitchen does not exist.
var ErrChefNotFound = errors.New("chef not found")

// LikeChef records a like. Idempotent: liking twice leaves one row and one count.
func LikeChef(userID, chefID uuid.UUID) (models.ChefAudienceState, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := chefMustExist(tx, chefID); err != nil {
			return err
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&models.ChefLike{UserID: userID, ChefID: chefID})
		if res.Error != nil {
			return res.Error
		}
		return bumpChefCounter(tx, chefID, "like_count", res.RowsAffected)
	})
	if err != nil {
		return models.ChefAudienceState{}, err
	}
	return GetChefAudienceState(userID, chefID)
}

// UnlikeChef removes a like. Idempotent: unliking what was never liked is a no-op.
func UnlikeChef(userID, chefID uuid.UUID) (models.ChefAudienceState, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND chef_id = ?", userID, chefID).
			Delete(&models.ChefLike{})
		if res.Error != nil {
			return res.Error
		}
		return bumpChefCounter(tx, chefID, "like_count", -res.RowsAffected)
	})
	if err != nil {
		return models.ChefAudienceState{}, err
	}
	return GetChefAudienceState(userID, chefID)
}

// SubscribeToChef starts a subscription, with every notification kind on.
// Idempotent, and deliberately does NOT reset the per-kind flags of an existing
// subscription — re-tapping subscribe must not silently re-enable what the
// customer turned off.
func SubscribeToChef(userID, chefID uuid.UUID) (models.ChefAudienceState, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := chefMustExist(tx, chefID); err != nil {
			return err
		}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ChefSubscription{
			UserID:             userID,
			ChefID:             chefID,
			NotifyMenu:         true,
			NotifyPriceChange:  true,
			NotifyAvailability: true,
			NotifyArticles:     true,
		})
		if res.Error != nil {
			return res.Error
		}
		return bumpChefCounter(tx, chefID, "subscriber_count", res.RowsAffected)
	})
	if err != nil {
		return models.ChefAudienceState{}, err
	}
	return GetChefAudienceState(userID, chefID)
}

// UnsubscribeFromChef ends a subscription. Idempotent.
func UnsubscribeFromChef(userID, chefID uuid.UUID) (models.ChefAudienceState, error) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND chef_id = ?", userID, chefID).
			Delete(&models.ChefSubscription{})
		if res.Error != nil {
			return res.Error
		}
		return bumpChefCounter(tx, chefID, "subscriber_count", -res.RowsAffected)
	})
	if err != nil {
		return models.ChefAudienceState{}, err
	}
	return GetChefAudienceState(userID, chefID)
}

// UpdateSubscriptionNotifyPrefs flips the per-kind flags on an existing
// subscription. Only the kinds present in prefs are touched.
func UpdateSubscriptionNotifyPrefs(userID, chefID uuid.UUID, prefs map[string]bool) error {
	updates := map[string]any{}
	for kind, on := range prefs {
		col, ok := models.SubscriptionNotifyColumn(kind)
		if !ok {
			return fmt.Errorf("unknown notification kind %q", kind)
		}
		updates[col] = on
	}
	if len(updates) == 0 {
		return nil
	}
	res := database.DB.Model(&models.ChefSubscription{}).
		Where("user_id = ? AND chef_id = ?", userID, chefID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetChefAudienceState returns the viewer's own like/subscribe state plus the
// public totals. userID may be uuid.Nil for an anonymous viewer.
func GetChefAudienceState(userID, chefID uuid.UUID) (models.ChefAudienceState, error) {
	state := models.ChefAudienceState{ChefID: chefID}

	var chef models.ChefProfile
	if err := database.DB.Select("id", "like_count", "subscriber_count").
		First(&chef, "id = ?", chefID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return state, ErrChefNotFound
		}
		return state, err
	}
	state.LikeCount = chef.LikeCount
	state.SubscriberCount = chef.SubscriberCount

	if userID == uuid.Nil {
		return state, nil
	}
	var likes, subs int64
	if err := database.DB.Model(&models.ChefLike{}).
		Where("user_id = ? AND chef_id = ?", userID, chefID).Count(&likes).Error; err != nil {
		return state, err
	}
	if err := database.DB.Model(&models.ChefSubscription{}).
		Where("user_id = ? AND chef_id = ?", userID, chefID).Count(&subs).Error; err != nil {
		return state, err
	}
	state.Liked = likes > 0
	state.Subscribed = subs > 0
	return state, nil
}

// ChefAudienceUserIDs is the distinct set of customers to notify for one kind.
//
// Menu drops additionally include people who merely favorited the chef, because
// that is who #239/#405 has always notified and dropping them would silently
// unsubscribe every existing customer. The kinds added since are subscribers
// only — a shortlist entry is not consent to a new class of notification.
func ChefAudienceUserIDs(chefID uuid.UUID, kind string) ([]uuid.UUID, error) {
	col, ok := models.SubscriptionNotifyColumn(kind)
	if !ok {
		return nil, fmt.Errorf("unknown notification kind %q", kind)
	}

	var ids []uuid.UUID
	if err := database.DB.Model(&models.ChefSubscription{}).
		Where("chef_id = ? AND "+col+" = ?", chefID, true).
		Pluck("user_id", &ids).Error; err != nil {
		return nil, fmt.Errorf("load subscribers: %w", err)
	}

	if kind == models.ChefNotifyMenu {
		var favIDs []uuid.UUID
		if err := database.DB.Model(&models.FavoriteChef{}).
			Where("chef_id = ?", chefID).Pluck("user_id", &favIDs).Error; err != nil {
			return nil, fmt.Errorf("load favorites: %w", err)
		}
		ids = append(ids, favIDs...)
	}

	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup || id == uuid.Nil {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

// ListChefSubscriptions returns the customer's subscribed kitchens, newest first.
func ListChefSubscriptions(userID uuid.UUID) ([]models.ChefSubscription, error) {
	var subs []models.ChefSubscription
	err := database.DB.Preload("Chef").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&subs).Error
	return subs, err
}

// bumpChefCounter moves a counter by delta, clamped at zero so a double-delete
// or a manually removed row can never drive a public count negative.
func bumpChefCounter(tx *gorm.DB, chefID uuid.UUID, column string, delta int64) error {
	if delta == 0 {
		return nil
	}
	// CASE rather than GREATEST: the same clamp has to run on Postgres in prod
	// and on SQLite under test, and GREATEST does not exist on SQLite.
	expr := gorm.Expr(column+" + ?", delta)
	if delta < 0 {
		expr = gorm.Expr(
			"CASE WHEN "+column+" + ? < 0 THEN 0 ELSE "+column+" + ? END", delta, delta)
	}
	return tx.Model(&models.ChefProfile{}).
		Where("id = ?", chefID).
		UpdateColumn(column, expr).Error
}

// chefDisplayName is chefBusinessName with wording that still reads correctly
// in a notification when the lookup misses — that one returns "" by design,
// because its reference-prefixing callers depend on the empty fallback.
func chefDisplayName(chefID uuid.UUID) string {
	if name := chefBusinessName(chefID); name != "" {
		return name
	}
	return "A kitchen you follow"
}

func chefMustExist(tx *gorm.DB, chefID uuid.UUID) error {
	var count int64
	if err := tx.Model(&models.ChefProfile{}).Where("id = ?", chefID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrChefNotFound
	}
	return nil
}
