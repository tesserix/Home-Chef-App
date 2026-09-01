package models

import (
	"time"

	"github.com/google/uuid"
)

// UserIdentity links an upstream identity (provider + subject, e.g. "zitadel"
// + OIDC sub) to a local user. A user may hold several rows — the legacy GIP
// identity and the Zitadel one — which is what makes the GIP→Zitadel migration
// seamless: the first verified-email Zitadel login links a new row to the
// existing user instead of creating a duplicate account.
type UserIdentity struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	Provider    string     `gorm:"type:varchar(32);not null;uniqueIndex:idx_user_identities_provider_subject" json:"provider"`
	Subject     string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_user_identities_provider_subject;index:idx_user_identities_subject" json:"subject"`
	Email       string     `gorm:"type:varchar(255)" json:"email,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
}
