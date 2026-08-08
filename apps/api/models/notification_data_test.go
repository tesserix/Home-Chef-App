package models

// #1153. Notification.Data is a jsonb column held as a plain string, so a
// caller that sets no payload sends '' — which Postgres refuses outright, and
// the whole notice is lost with only a log line behind it.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

func notificationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: glogger.Default.LogMode(glogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE notifications (id TEXT PRIMARY KEY, user_id TEXT,
		type TEXT, title TEXT, message TEXT, data TEXT, is_read INTEGER DEFAULT 0,
		read_at DATETIME, created_at DATETIME)`).Error)
	return db
}

func TestNotificationWithNoPayloadStoresValidJSON(t *testing.T) {
	db := notificationDB(t)
	n := &Notification{
		ID: uuid.New(), UserID: uuid.New(),
		Type: "easy_split_vendor_rejected", Title: "Bank details rejected",
		Message: "Cashfree could not verify your account.",
	}

	require.NoError(t, db.Create(n).Error)

	var stored string
	require.NoError(t, db.Raw(`SELECT data FROM notifications WHERE id = ?`, n.ID.String()).
		Scan(&stored).Error)
	require.Equal(t, "{}", stored, "'' is not a value a jsonb column can hold")
}

func TestNotificationKeepsThePayloadItWasGiven(t *testing.T) {
	db := notificationDB(t)
	payload := `{"order_id":"abc","status":"delivered"}`
	n := &Notification{
		ID: uuid.New(), UserID: uuid.New(),
		Type: "order_status", Title: "Delivered", Data: payload,
	}

	require.NoError(t, db.Create(n).Error)

	var stored string
	require.NoError(t, db.Raw(`SELECT data FROM notifications WHERE id = ?`, n.ID.String()).
		Scan(&stored).Error)
	require.Equal(t, payload, stored)
}
