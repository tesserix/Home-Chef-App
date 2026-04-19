package models

import (
	"time"

	"github.com/google/uuid"
)

// ApiKey is a platform-level integration credential that third parties
// (partner services, scripts, dashboards) can use to call the HomeChef API
// without going through a user login.
//
// Only the SHA-256 hash of the secret is stored — the full key is shown to
// the admin exactly once at creation time, the same way GitHub does.
type ApiKey struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name       string     `gorm:"not null" json:"name"`
	Prefix     string     `gorm:"index;not null" json:"prefix"` // e.g. "hc_abcd1234" — shown in UI
	KeyHash    string     `gorm:"uniqueIndex;not null" json:"-"` // SHA-256(full key)
	Scopes     string     `gorm:"type:text" json:"scopes"`       // comma-separated: "read,write,admin"
	CreatedBy  uuid.UUID  `gorm:"type:uuid;index" json:"createdBy"`
	LastUsedAt *time.Time `gorm:"" json:"lastUsedAt,omitempty"`
	ExpiresAt  *time.Time `gorm:"" json:"expiresAt,omitempty"`
	RevokedAt  *time.Time `gorm:"" json:"revokedAt,omitempty"`
	CreatedAt  time.Time  `gorm:"autoCreateTime" json:"createdAt"`
}

// IsActive reports whether the key can currently be used to authenticate.
func (k *ApiKey) IsActive(now time.Time) bool {
	if k.RevokedAt != nil {
		return false
	}
	if k.ExpiresAt != nil && now.After(*k.ExpiresAt) {
		return false
	}
	return true
}
