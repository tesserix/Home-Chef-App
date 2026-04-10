package models

import (
	"time"

	"github.com/google/uuid"
)

// EmailVerificationToken stores email verification tokens for user registration
type EmailVerificationToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	Token     string     `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	ExpiresAt time.Time  `gorm:"not null" json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}

// IsValid returns true if the token is not expired and not yet used
func (t *EmailVerificationToken) IsValid() bool {
	return t.UsedAt == nil && time.Now().Before(t.ExpiresAt)
}
