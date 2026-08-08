package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// adoptLegacyDeviceTokens carries a pre-#1164 users.fcm_token into the device
// registry that push now reads exclusively.
//
// Idempotent and re-run every boot, because the bootstrap CronJob does not
// replay DDL against a provisioned database (#1127) — without this, the accounts
// that had a token before the cutover receive nothing until they sign in again.
func adoptLegacyDeviceTokens(db *gorm.DB) error {
	if !db.Migrator().HasColumn("users", "fcm_token") || !db.Migrator().HasTable("user_devices") {
		return nil
	}
	res := db.Exec(`
		INSERT INTO user_devices (user_id, device_id, app, fcm_token, first_seen_at, last_seen_at)
		SELECT u.id, 'legacy-' || u.id, 'customer', u.fcm_token, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		FROM users u
		WHERE u.fcm_token IS NOT NULL AND u.fcm_token <> ''
		ON CONFLICT (user_id, device_id) DO NOTHING`)
	if res.Error != nil {
		return fmt.Errorf("adopting legacy device tokens: %w", res.Error)
	}
	if res.RowsAffected > 0 {
		log.Printf("device registry: adopted %d legacy push token(s)", res.RowsAffected)
	}
	return nil
}
