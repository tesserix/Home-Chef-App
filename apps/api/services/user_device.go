package services

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// The device registry behind multi-device sign-in (#1164).
//
// Before this, push lived in a single users.fcm_token column, so a second
// sign-in overwrote the first device's token and silently stopped delivering
// to it. Every device now owns a row, and push fans out across all of them.

// RecordDeviceInput is one sighting of a device: who, which install, and where
// from. Everything but UserID and DeviceID is best-effort decoration.
type RecordDeviceInput struct {
	UserID     uuid.UUID
	DeviceID   string
	App        string
	Platform   string
	Label      string
	AppVersion string
	IP         string
	Country    string
	City       string
}

// DeviceSighting is what the caller needs to decide whether to warn the user.
type DeviceSighting struct {
	// Known is false the first time this account is seen on this device, which
	// is what triggers the new-device email.
	Known  bool
	Device models.UserDevice
}

// RecordDevice upserts the device and reports whether it had been seen before.
//
// Insert-first rather than select-then-insert: two devices signing in at once
// would both read "absent" and race to insert, and the loser would either error
// or send a second email. The unique (user_id, device_id) index arbitrates.
func RecordDevice(db *gorm.DB, in RecordDeviceInput) (DeviceSighting, error) {
	if db == nil || in.UserID == uuid.Nil || strings.TrimSpace(in.DeviceID) == "" {
		return DeviceSighting{Known: true}, nil
	}
	now := time.Now().UTC()
	row := models.UserDevice{
		ID:          uuid.New(),
		UserID:      in.UserID,
		DeviceID:    in.DeviceID,
		App:         in.App,
		Platform:    in.Platform,
		Label:       in.Label,
		AppVersion:  in.AppVersion,
		LastIP:      in.IP,
		LastCountry: in.Country,
		LastCity:    in.City,
		FirstSeenAt: now,
		LastSeenAt:  now,
	}

	res := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"app":          in.App,
			"platform":     in.Platform,
			"label":        in.Label,
			"app_version":  in.AppVersion,
			"last_ip":      in.IP,
			"last_country": in.Country,
			"last_city":    in.City,
			"last_seen_at": now,
			"revoked_at":   nil,
		}),
	}).Create(&row)
	if res.Error != nil {
		return DeviceSighting{}, res.Error
	}

	// RowsAffected can't distinguish insert from update portably, so re-read and
	// compare the two stored timestamps: an insert writes both from the same
	// value, an update moves only last_seen_at. Deliberately NOT compared
	// against the Go `now` above — Postgres truncates to microseconds, so a
	// freshly inserted row would never match it and no device would ever look
	// new. Both sides of this comparison come back through the same driver.
	var stored models.UserDevice
	if err := db.Where("user_id = ? AND device_id = ?", in.UserID, in.DeviceID).
		First(&stored).Error; err != nil {
		return DeviceSighting{}, err
	}
	return DeviceSighting{Known: !stored.FirstSeenAt.Equal(stored.LastSeenAt), Device: stored}, nil
}

// SetDeviceToken records (or clears, when token is empty) the FCM token for one
// device, creating the row if the client registers a token before any sighting.
func SetDeviceToken(db *gorm.DB, userID uuid.UUID, deviceID, app, token string) error {
	if db == nil || userID == uuid.Nil || strings.TrimSpace(deviceID) == "" {
		return nil
	}
	now := time.Now().UTC()
	row := models.UserDevice{
		ID: uuid.New(), UserID: userID, DeviceID: deviceID, App: app,
		FCMToken: token, FirstSeenAt: now, LastSeenAt: now,
	}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"fcm_token":    token,
			"last_seen_at": now,
			"revoked_at":   nil,
		}),
	}).Create(&row).Error; err != nil {
		return err
	}

	// One FCM token addresses one app install, so a token that has resurfaced
	// under another account belongs to whoever just registered it. Clearing the
	// stale copies stops a re-installed handset delivering a previous owner's
	// notifications. Scoped by token, so it never touches the user's own
	// other devices.
	if token == "" {
		return nil
	}
	return db.Model(&models.UserDevice{}).
		Where("fcm_token = ? AND NOT (user_id = ? AND device_id = ?)", token, userID, deviceID).
		Update("fcm_token", "").Error
}

// ActiveDeviceTokens returns every deliverable FCM token for the account. An
// empty result is normal (no app installed, notifications declined) and callers
// treat it as "nothing to send", never as an error.
func ActiveDeviceTokens(db *gorm.DB, userID uuid.UUID) ([]string, error) {
	if db == nil {
		return nil, nil
	}
	var tokens []string
	err := db.Model(&models.UserDevice{}).
		Where("user_id = ? AND revoked_at IS NULL AND fcm_token <> ''", userID).
		Order("last_seen_at DESC").
		Pluck("fcm_token", &tokens).Error
	return tokens, err
}

// SharesDevice reports whether two accounts have been signed in on the same
// handset — the self-referral signal that used to be a shared users.fcm_token.
// The device id is the better fingerprint: it survives token rotation and a
// notifications-declined install, both of which left the old check blind.
func SharesDevice(db *gorm.DB, a, b uuid.UUID) (bool, error) {
	if db == nil || a == uuid.Nil || b == uuid.Nil || a == b {
		return false, nil
	}
	var count int64
	err := db.Model(&models.UserDevice{}).
		Where("user_id = ? AND device_id IN (?)", a,
			db.Model(&models.UserDevice{}).Select("device_id").Where("user_id = ?", b)).
		Count(&count).Error
	return count > 0, err
}

// DropDeviceToken clears a token FCM has permanently rejected, leaving the
// account's other devices deliverable. The row survives so the device keeps its
// history and re-arms when the app registers a fresh token.
func DropDeviceToken(db *gorm.DB, userID uuid.UUID, token string) error {
	if db == nil || token == "" {
		return nil
	}
	return db.Model(&models.UserDevice{}).
		Where("user_id = ? AND fcm_token = ?", userID, token).
		Update("fcm_token", "").Error
}
