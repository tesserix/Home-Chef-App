package services

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createUserDevicesTable mirrors the user_devices DDL in tesserix-k8s. Shared
// by every fixture whose subject now reads the device registry (#1164), so the
// shape is stated once rather than drifting across six setup helpers.
func createUserDevicesTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS user_devices (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, device_id TEXT NOT NULL,
		app TEXT NOT NULL DEFAULT '', platform TEXT, label TEXT, fcm_token TEXT,
		app_version TEXT, last_ip TEXT, last_country TEXT, last_city TEXT,
		first_seen_at DATETIME, last_seen_at DATETIME, revoked_at DATETIME,
		UNIQUE (user_id, device_id))`).Error)
}
