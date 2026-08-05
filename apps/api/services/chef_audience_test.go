package services

// chef_audience_test.go — likes/subscriptions and the counters discovery ranks
// by. In-memory SQLite; the counter assertions are the point, since an inflated
// like_count would quietly promote a chef in search.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupAudienceDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE chef_profiles (id text PRIMARY KEY, business_name text,
			like_count integer DEFAULT 0, subscriber_count integer DEFAULT 0)`,
		`CREATE TABLE chef_likes (id text PRIMARY KEY, user_id text, chef_id text, created_at datetime,
			UNIQUE(user_id, chef_id))`,
		`CREATE TABLE chef_subscriptions (id text PRIMARY KEY, user_id text, chef_id text,
			notify_menu integer DEFAULT 1, notify_price_change integer DEFAULT 1,
			notify_availability integer DEFAULT 1, notify_articles integer DEFAULT 1,
			created_at datetime, updated_at datetime, UNIQUE(user_id, chef_id))`,
		`CREATE TABLE favorite_chefs (id text PRIMARY KEY, user_id text, chef_id text, created_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, business_name) VALUES (?, ?)`,
		chefID.String(), "Test Kitchen").Error)

	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db, chefID
}

func chefCounts(t *testing.T, db *gorm.DB, chefID uuid.UUID) (int, int) {
	t.Helper()
	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	return chef.LikeCount, chef.SubscriberCount
}

// Liking twice must leave one row and a count of one — otherwise a customer
// could rank a chef up by tapping repeatedly.
func TestLikeChefIsIdempotent(t *testing.T) {
	db, chefID := setupAudienceDB(t)
	userID := uuid.New()

	state, err := LikeChef(userID, chefID)
	require.NoError(t, err)
	require.True(t, state.Liked)
	require.Equal(t, 1, state.LikeCount)

	state, err = LikeChef(userID, chefID)
	require.NoError(t, err)
	require.Equal(t, 1, state.LikeCount)

	likes, _ := chefCounts(t, db, chefID)
	require.Equal(t, 1, likes)
}

// Unliking what was never liked must not drive the public count negative.
func TestUnlikeChefClampsAtZero(t *testing.T) {
	db, chefID := setupAudienceDB(t)

	state, err := UnlikeChef(uuid.New(), chefID)
	require.NoError(t, err)
	require.False(t, state.Liked)
	require.Equal(t, 0, state.LikeCount)

	likes, _ := chefCounts(t, db, chefID)
	require.Equal(t, 0, likes)
}

func TestLikeThenUnlikeReturnsToZero(t *testing.T) {
	db, chefID := setupAudienceDB(t)
	userID := uuid.New()

	_, err := LikeChef(userID, chefID)
	require.NoError(t, err)
	_, err = UnlikeChef(userID, chefID)
	require.NoError(t, err)

	likes, _ := chefCounts(t, db, chefID)
	require.Equal(t, 0, likes)
}

func TestSubscribeIsIdempotentAndCounts(t *testing.T) {
	db, chefID := setupAudienceDB(t)
	userID := uuid.New()

	for i := 0; i < 3; i++ {
		state, err := SubscribeToChef(userID, chefID)
		require.NoError(t, err)
		require.True(t, state.Subscribed)
		require.Equal(t, 1, state.SubscriberCount)
	}
	_, subs := chefCounts(t, db, chefID)
	require.Equal(t, 1, subs)
}

// Re-subscribing must not silently re-enable a kind the customer turned off.
func TestResubscribeKeepsNotifyPrefs(t *testing.T) {
	_, chefID := setupAudienceDB(t)
	userID := uuid.New()

	_, err := SubscribeToChef(userID, chefID)
	require.NoError(t, err)
	require.NoError(t, UpdateSubscriptionNotifyPrefs(userID, chefID,
		map[string]bool{models.ChefNotifyPriceChange: false}))

	_, err = SubscribeToChef(userID, chefID)
	require.NoError(t, err)

	ids, err := ChefAudienceUserIDs(chefID, models.ChefNotifyPriceChange)
	require.NoError(t, err)
	require.Empty(t, ids, "price-change opt-out must survive a re-subscribe")
}

func TestUnsubscribeRemovesFromAudience(t *testing.T) {
	_, chefID := setupAudienceDB(t)
	userID := uuid.New()

	_, err := SubscribeToChef(userID, chefID)
	require.NoError(t, err)
	_, err = UnsubscribeFromChef(userID, chefID)
	require.NoError(t, err)

	ids, err := ChefAudienceUserIDs(chefID, models.ChefNotifyMenu)
	require.NoError(t, err)
	require.Empty(t, ids)
}

// Menu drops must still reach people who only favorited the chef — that is the
// pre-existing #239/#405 audience and dropping it would silently unsubscribe
// every current customer.
func TestMenuAudienceIncludesFavoritesAndDedupes(t *testing.T) {
	db, chefID := setupAudienceDB(t)
	both := uuid.New()
	favOnly := uuid.New()
	subOnly := uuid.New()

	for _, u := range []uuid.UUID{both, favOnly} {
		require.NoError(t, db.Exec(`INSERT INTO favorite_chefs (id, user_id, chef_id) VALUES (?,?,?)`,
			uuid.NewString(), u.String(), chefID.String()).Error)
	}
	for _, u := range []uuid.UUID{both, subOnly} {
		_, err := SubscribeToChef(u, chefID)
		require.NoError(t, err)
	}

	ids, err := ChefAudienceUserIDs(chefID, models.ChefNotifyMenu)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{both, favOnly, subOnly}, ids)
}

// The kinds added since favorites existed go to subscribers only — a shortlist
// entry is not consent to a new class of notification.
func TestNonMenuAudienceExcludesFavorites(t *testing.T) {
	db, chefID := setupAudienceDB(t)
	favOnly := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO favorite_chefs (id, user_id, chef_id) VALUES (?,?,?)`,
		uuid.NewString(), favOnly.String(), chefID.String()).Error)

	for _, kind := range []string{
		models.ChefNotifyPriceChange, models.ChefNotifyAvailability, models.ChefNotifyArticles,
	} {
		ids, err := ChefAudienceUserIDs(chefID, kind)
		require.NoError(t, err)
		require.Empty(t, ids, kind)
	}
}

func TestPerKindOptOutRemovesOnlyThatKind(t *testing.T) {
	_, chefID := setupAudienceDB(t)
	userID := uuid.New()
	_, err := SubscribeToChef(userID, chefID)
	require.NoError(t, err)

	require.NoError(t, UpdateSubscriptionNotifyPrefs(userID, chefID,
		map[string]bool{models.ChefNotifyAvailability: false}))

	off, err := ChefAudienceUserIDs(chefID, models.ChefNotifyAvailability)
	require.NoError(t, err)
	require.Empty(t, off)

	on, err := ChefAudienceUserIDs(chefID, models.ChefNotifyArticles)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{userID}, on)
}

func TestUnknownNotifyKindIsRejected(t *testing.T) {
	_, chefID := setupAudienceDB(t)
	_, err := ChefAudienceUserIDs(chefID, "not_a_kind")
	require.Error(t, err)

	err = UpdateSubscriptionNotifyPrefs(uuid.New(), chefID, map[string]bool{"not_a_kind": true})
	require.Error(t, err)
}

func TestLikeUnknownChefIsRejected(t *testing.T) {
	setupAudienceDB(t)
	_, err := LikeChef(uuid.New(), uuid.New())
	require.ErrorIs(t, err, ErrChefNotFound)
}
