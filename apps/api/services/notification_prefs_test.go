package services

// notification_prefs_test.go — backend-testable slice of issue #61 (push
// notification matrix). Firing every category on real iOS/Android devices is
// manual (and gated on the APNs cert, #14). What we CAN guarantee here is the
// part that decides WHETHER a push goes out at all:
//   - the type→category mapping (so a push is gated by the right toggle),
//   - the opt-in/opt-out defaults (marketing off, transactional on),
//   - that a stored preference actually suppresses the channel.
// A regression here means users get notifications they muted (or miss critical
// ones), so it's worth pinning down.

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func TestNotificationTypeCategory(t *testing.T) {
	cases := map[string]models.NotificationCategory{
		"order_confirmation": models.NotifCategoryOrder,
		"order_status":       models.NotifCategoryOrder,
		"order_cancelled":    models.NotifCategoryOrder,
		"order_delivered":    models.NotifCategoryOrder,
		"chef_new_order":     models.NotifCategoryChef,
		"chef_verified":      models.NotifCategoryChef,
		"delivery_assigned":  models.NotifCategoryDelivery,
		"delivery_picked_up": models.NotifCategoryDelivery,
		"promo":              models.NotifCategoryMarketing,
		"marketing":          models.NotifCategoryMarketing,
		// Transactional/system types fall through to the account bucket.
		"fssai_expiring":   models.NotifCategoryAccount,
		"weekly_statement": models.NotifCategoryAccount,
		"":                 models.NotifCategoryAccount,
	}
	for typ, want := range cases {
		if got := notificationTypeCategory(typ); got != want {
			t.Errorf("notificationTypeCategory(%q) = %q, want %q", typ, got, want)
		}
	}
}

func TestDefaultNotificationPreference(t *testing.T) {
	// Marketing is opt-IN: every channel off by default.
	m := models.DefaultNotificationPreference(models.NotifCategoryMarketing)
	if m.EmailEnabled || m.PushEnabled || m.SMSEnabled {
		t.Fatalf("marketing must default all-off, got %+v", m)
	}
	// Transactional categories are opt-OUT: email+push on, sms off.
	for _, cat := range []models.NotificationCategory{
		models.NotifCategoryOrder, models.NotifCategoryChef,
		models.NotifCategoryDelivery, models.NotifCategoryAccount,
	} {
		p := models.DefaultNotificationPreference(cat)
		if !p.EmailEnabled || !p.PushEnabled || p.SMSEnabled {
			t.Errorf("%s default = %+v; want email+push on, sms off", cat, p)
		}
	}
}

func setupNotifPrefDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE notification_preferences (
		id text PRIMARY KEY, user_id text, category text,
		email_enabled integer, push_enabled integer, sms_enabled integer,
		created_at datetime, updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func insertPref(t *testing.T, db *gorm.DB, uid uuid.UUID, cat models.NotificationCategory, email, push, sms int) {
	t.Helper()
	if err := db.Exec(
		`INSERT INTO notification_preferences (id, user_id, category, email_enabled, push_enabled, sms_enabled) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), uid.String(), string(cat), email, push, sms,
	).Error; err != nil {
		t.Fatalf("insert pref: %v", err)
	}
}

func TestShouldSend_RespectsStoredPreference(t *testing.T) {
	db := setupNotifPrefDB(t)
	uid := uuid.New()
	// User muted order PUSH but kept order email.
	insertPref(t, db, uid, models.NotifCategoryOrder, 1, 0, 0)

	if ShouldSend(uid, models.NotifCategoryOrder, ChannelPush) {
		t.Error("push must be suppressed when the user disabled order push")
	}
	if !ShouldSend(uid, models.NotifCategoryOrder, ChannelEmail) {
		t.Error("email must still be allowed (only push was disabled)")
	}
}

func TestShouldSend_MissingRowUsesDefaults(t *testing.T) {
	setupNotifPrefDB(t)
	uid := uuid.New()
	// No stored row → category defaults apply.
	if !ShouldSend(uid, models.NotifCategoryOrder, ChannelPush) {
		t.Error("order push should default ON (transactional opt-out)")
	}
	if ShouldSend(uid, models.NotifCategoryMarketing, ChannelPush) {
		t.Error("marketing push should default OFF (opt-in)")
	}
}

func TestShouldSendForType_GatesByCategory(t *testing.T) {
	db := setupNotifPrefDB(t)
	uid := uuid.New()
	// Disable the whole chef category.
	insertPref(t, db, uid, models.NotifCategoryChef, 0, 0, 0)

	// chef_new_order maps to the chef category → suppressed.
	if ShouldSendForType(uid, "chef_new_order", ChannelPush) {
		t.Error("chef_new_order push must be suppressed when chef category disabled")
	}
	// A different category is unaffected (default on).
	if !ShouldSendForType(uid, "order_status", ChannelPush) {
		t.Error("order_status push should remain allowed")
	}
}
