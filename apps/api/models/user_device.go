package models

import (
	"time"

	"github.com/google/uuid"
)

// UserDevice is one install of one app on one physical device.
//
// It replaces the single users.fcm_token column, which allowed exactly one
// deliverable device per account: a second sign-in overwrote the token and
// silently killed push on the first device (#1164). Push now fans out across
// every non-revoked row here.
//
// It doubles as the new-device-login detector: an unseen (user_id, device_id)
// pair is what triggers the security email.
//
// The schema itself lives in tesserix-k8s
// (charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql).
type UserDevice struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index:idx_user_devices_user_device,unique,priority:1" json:"userId"`

	// DeviceID is the client's stable per-install identifier, unique with
	// UserID. Two accounts on one handset are two rows, which is what lets a
	// shared device deliver to whichever account is signed in.
	DeviceID string `gorm:"type:text;not null;index:idx_user_devices_user_device,unique,priority:2" json:"deviceId"`

	// App is one of customer / vendor / delivery / admin.
	App      string `gorm:"type:text;not null" json:"app"`
	Platform string `gorm:"type:text" json:"platform"`
	Label    string `gorm:"type:text" json:"label"`

	FCMToken   string `gorm:"type:text;index" json:"-"`
	AppVersion string `gorm:"type:text" json:"appVersion,omitempty"`

	// Last known network origin, captured for the new-device email. Best-effort
	// and decorative — a failed geo lookup leaves country/city empty.
	LastIP      string `gorm:"type:text" json:"-"`
	LastCountry string `gorm:"type:text" json:"lastCountry,omitempty"`
	LastCity    string `gorm:"type:text" json:"lastCity,omitempty"`

	FirstSeenAt time.Time  `json:"firstSeenAt"`
	LastSeenAt  time.Time  `json:"lastSeenAt"`
	RevokedAt   *time.Time `json:"revokedAt,omitempty"`
}

func (UserDevice) TableName() string { return "user_devices" }

// Active reports whether this device should still receive push.
func (d UserDevice) Active() bool { return d.RevokedAt == nil }
