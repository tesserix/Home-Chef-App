package models

import (
	"time"

	"github.com/google/uuid"
)

// Two-factor authentication state.
//
// The schema itself lives in tesserix-k8s
// (charts/apps/db-schema-bootstrap/schemas/homechef/homechef/homechef_db.sql);
// these are the ORM views of it, per the repo rule that app repos hold models
// and never raw SQL.

// UserMFASettings is one row per user who has interacted with two-factor.
// Absence of a row means two-factor has never been touched, which is the same
// as disabled — the gate treats a missing row as "off" rather than erroring.
type UserMFASettings struct {
	UserID uuid.UUID `gorm:"type:uuid;primaryKey" json:"userId"`

	// Enabled flips true only once a channel is enrolled AND the backup codes
	// have been acknowledged. Enrolling alone must not arm the gate, or a user
	// who closes the tab mid-setup is locked out with no recovery codes.
	Enabled bool `gorm:"default:false" json:"enabled"`

	EmailEnrolled bool `gorm:"default:false" json:"emailEnrolled"`
	PhoneEnrolled bool `gorm:"default:false" json:"phoneEnrolled"`

	// The MFA phone is held separately from User.Phone on purpose. Sharing the
	// column would mean editing a profile silently relocates the second factor,
	// which is an account-takeover path.
	PhoneE164Enc  EncryptedString `gorm:"column:phone_e164_enc;type:text" json:"-"`
	PhoneE164Bidx string          `gorm:"column:phone_e164_bidx;type:text;index" json:"-"`

	EnrolledAt *time.Time `json:"enrolledAt,omitempty"`
	DisabledAt *time.Time `json:"disabledAt,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (UserMFASettings) TableName() string { return "user_mfa_settings" }

// MFABackupCode is one single-use recovery code, stored as an HMAC so a
// database leak does not hand over working codes.
type MFABackupCode struct {
	ID       uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID   uuid.UUID  `gorm:"type:uuid;index;not null" json:"userId"`
	CodeHMAC string     `gorm:"type:text;not null;index" json:"-"`
	UsedAt   *time.Time `json:"usedAt,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

func (MFABackupCode) TableName() string { return "mfa_backup_codes" }

// TrustedDevice records a device the user chose to remember, so the challenge
// does not repeat on every login.
//
// Scoped to one user AND one app: a token lifted from the vendor app must not
// vouch for the admin app. By product decision there is no time-based expiry —
// trust ends only on explicit revocation, on password change, or when
// two-factor is switched off.
type TrustedDevice struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;index;not null" json:"userId"`

	// App is one of customer / vendor / delivery / admin.
	App string `gorm:"type:text;not null;index" json:"app"`

	// TokenHash is SHA-256 of the token. The token itself is returned to the
	// client exactly once, at mint time, and never stored.
	TokenHash string `gorm:"type:text;not null;uniqueIndex" json:"-"`

	// Label is what the user sees in the device list ("Pixel 9", "Safari on Mac").
	Label    string `gorm:"type:text" json:"label"`
	Platform string `gorm:"type:text" json:"platform"`

	CreatedAt  time.Time  `json:"createdAt"`
	LastSeenAt time.Time  `json:"lastSeenAt"`
	RevokedAt  *time.Time `json:"revokedAt,omitempty"`
}

func (TrustedDevice) TableName() string { return "trusted_devices" }

// Active reports whether this device may still skip the challenge.
func (d TrustedDevice) Active() bool { return d.RevokedAt == nil }
