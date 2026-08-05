package services

// chef_audience_notify_test.go — the subscriber fan-outs added alongside the
// menu drops: kitchen opened, dish got cheaper, ChefBook post published.
//
// The suppression cases carry the weight here. Each of these fires on a routine
// vendor action, so a handler that notified unconditionally would push the whole
// subscriber list every time a chef closed for the night or nudged a price up.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// notifyAudienceFixture gives one chef with one subscriber, on the audience
// schema (the daily-menu fixture has no chef_profiles to resolve a name from).
func notifyAudienceFixture(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, chefID := setupAudienceDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE notifications (id text PRIMARY KEY, user_id text, type text,
		title text, message text, data text, is_read integer DEFAULT 0, read_at datetime, created_at datetime)`).Error)

	userID := uuid.New()
	_, err := SubscribeToChef(userID, chefID)
	require.NoError(t, err)
	return db, chefID, userID
}

func notifCount(t *testing.T, db *gorm.DB, notifType string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.Notification{}).Where("type = ?", notifType).Count(&n).Error)
	return n
}

func TestChefOpenNotifiesSubscribers(t *testing.T) {
	db, chefID, _ := notifyAudienceFixture(t)

	s := GetNotificationService()
	require.NoError(t, s.handleChefAvailabilityChanged(Event{Data: map[string]any{
		"chefId": chefID.String(), "acceptingOrders": true,
	}}))

	require.Equal(t, int64(1), notifCount(t, db, "chef_open"))
}

// Closing must notify nobody: it sells nothing, and a kitchen toggling around a
// busy service would spam every subscriber.
func TestChefCloseNotifiesNobody(t *testing.T) {
	db, chefID, _ := notifyAudienceFixture(t)

	s := GetNotificationService()
	require.NoError(t, s.handleChefAvailabilityChanged(Event{Data: map[string]any{
		"chefId": chefID.String(), "acceptingOrders": false,
	}}))

	require.Zero(t, notifCount(t, db, "chef_open"))
}

func TestPriceDropNotifiesSubscribers(t *testing.T) {
	db, chefID, _ := notifyAudienceFixture(t)

	s := GetNotificationService()
	require.NoError(t, s.handleChefPriceChanged(Event{Data: map[string]any{
		"chef_id": chefID.String(), "chef_name": "Test Kitchen",
		"item_name": "Paneer Thali", "old_price": 220.0, "new_price": 180.0,
	}}))

	require.Equal(t, int64(1), notifCount(t, db, "chef_price_drop"))
}

// A price RISE must stay silent — nobody subscribed to be told a dish got
// dearer, and saying so is an argument against ordering.
func TestPriceRiseNotifiesNobody(t *testing.T) {
	db, chefID, _ := notifyAudienceFixture(t)

	s := GetNotificationService()
	require.NoError(t, s.handleChefPriceChanged(Event{Data: map[string]any{
		"chef_id": chefID.String(), "chef_name": "Test Kitchen",
		"item_name": "Paneer Thali", "old_price": 180.0, "new_price": 220.0,
	}}))

	require.Zero(t, notifCount(t, db, "chef_price_drop"))
}

func TestArticlePublishedNotifiesSubscribers(t *testing.T) {
	db, chefID, _ := notifyAudienceFixture(t)

	s := GetNotificationService()
	require.NoError(t, s.handleChefArticlePublished(Event{Data: map[string]any{
		"chef_id": chefID.String(), "chef_name": "Test Kitchen",
		"title": "How I fold a paratha", "slug": "how-i-fold-a-paratha",
	}}))

	require.Equal(t, int64(1), notifCount(t, db, "chef_article_published"))
}

// Turning one kind off must silence that kind and leave the others alone.
func TestPerKindOptOutSuppressesOnlyThatFanOut(t *testing.T) {
	db, chefID, userID := notifyAudienceFixture(t)
	require.NoError(t, UpdateSubscriptionNotifyPrefs(userID, chefID,
		map[string]bool{models.ChefNotifyAvailability: false}))

	s := GetNotificationService()
	require.NoError(t, s.handleChefAvailabilityChanged(Event{Data: map[string]any{
		"chefId": chefID.String(), "acceptingOrders": true,
	}}))
	require.NoError(t, s.handleChefArticlePublished(Event{Data: map[string]any{
		"chef_id": chefID.String(), "chef_name": "Test Kitchen", "title": "Still subscribed",
	}}))

	require.Zero(t, notifCount(t, db, "chef_open"))
	require.Equal(t, int64(1), notifCount(t, db, "chef_article_published"))
}

// Someone who merely favorited must not be pulled into the new kinds — only
// menu drops carry the favorites audience.
func TestFavoriteAloneGetsNoAudienceFanOut(t *testing.T) {
	db, chefID := setupAudienceDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE notifications (id text PRIMARY KEY, user_id text, type text,
		title text, message text, data text, is_read integer DEFAULT 0, read_at datetime, created_at datetime)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO favorite_chefs (id, user_id, chef_id) VALUES (?,?,?)`,
		uuid.NewString(), uuid.NewString(), chefID.String()).Error)

	s := GetNotificationService()
	require.NoError(t, s.handleChefAvailabilityChanged(Event{Data: map[string]any{
		"chefId": chefID.String(), "acceptingOrders": true,
	}}))

	require.Zero(t, notifCount(t, db, "chef_open"))
}

func TestAvailabilityEventRejectsBadChefID(t *testing.T) {
	notifyAudienceFixture(t)
	s := GetNotificationService()
	require.Error(t, s.handleChefAvailabilityChanged(Event{Data: map[string]any{
		"chefId": "not-a-uuid", "acceptingOrders": true,
	}}))
}
