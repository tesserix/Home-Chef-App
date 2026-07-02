package services

// notifications_daily_menu_test.go — the per-date menu-drop follower fan-out
// (#419/#421), the daily counterpart of the weekly menu-drop. In-memory SQLite;
// the notifications.id column has no default so SQLite allows distinct NULLs
// across the fan-out (the model's gen_random_uuid() default can't run on SQLite).
// NATS is disconnected in tests, so saveNotification's real-time publish is a
// logged no-op and the DB row is the assertion target.

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

func setupNotifDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE favorite_chefs (id text PRIMARY KEY, user_id text, chef_id text, created_at datetime)`,
		`CREATE TABLE notifications (id text PRIMARY KEY, user_id text, type text, title text, message text,
			data text, is_read integer DEFAULT 0, read_at datetime, created_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// Publishing a daily menu fans a "menu drop" out to every follower (favorite).
func TestHandleDailyMenuPublished_FansOutToFollowers(t *testing.T) {
	db := setupNotifDB(t)
	chefID := uuid.New()
	for _, u := range []uuid.UUID{uuid.New(), uuid.New(), uuid.New()} {
		require.NoError(t, db.Exec(`INSERT INTO favorite_chefs (id, user_id, chef_id) VALUES (?,?,?)`,
			uuid.NewString(), u.String(), chefID.String()).Error)
	}
	// A follower of a DIFFERENT chef must not be notified.
	require.NoError(t, db.Exec(`INSERT INTO favorite_chefs (id, user_id, chef_id) VALUES (?,?,?)`,
		uuid.NewString(), uuid.NewString(), uuid.NewString()).Error)

	s := GetNotificationService()
	require.NoError(t, s.handleDailyMenuPublished(Event{Data: map[string]any{
		"chef_id":   chefID.String(),
		"chef_name": "Dum Alooo Kitchen",
		"date":      "2027-03-15",
	}}))

	var count int64
	require.NoError(t, db.Model(&models.Notification{}).
		Where("type = ?", "daily_menu_published").Count(&count).Error)
	require.Equal(t, int64(3), count, "only the chef's own 3 followers get the menu-drop")
}

func TestFollowerEventChef(t *testing.T) {
	id := uuid.New()
	got, name, err := followerEventChef(Event{Data: map[string]any{"chef_id": id.String()}}, "daily_menu_published")
	require.NoError(t, err)
	require.Equal(t, id, got)
	require.Equal(t, "A chef you follow", name) // empty name falls back

	_, _, err = followerEventChef(Event{Data: map[string]any{"chef_id": "not-a-uuid"}}, "daily_menu_published")
	require.Error(t, err)
}

// The daily menu-drop is gated by the customer's "favorites" preference.
func TestDailyMenuCategoryIsFavorites(t *testing.T) {
	require.Equal(t, models.NotifCategoryFavorites, notificationTypeCategory("daily_menu_published"))
}
