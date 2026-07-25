package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoyaltyEarnBatch is a dated lot of earned points. Every point CREDIT writes one
// (with ExpiresAt = EarnedAt + expiry_days). Redeem/expiry/refund-reversal FIFO-consume
// PointsRemaining from the soonest-expiring lots. LoyaltyAccount.Balance stays the fast
// running total (= Σ PointsRemaining of the account's non-expired lots).
type LoyaltyEarnBatch struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID        `gorm:"type:uuid;not null;index" json:"userId"`
	Source          LoyaltyTxnSource `gorm:"type:varchar(20);not null" json:"source"`
	Points          float64          `gorm:"not null" json:"points"`          // originally earned
	PointsRemaining float64          `gorm:"not null" json:"pointsRemaining"` // unspent, un-expired
	EarnedAt        time.Time        `gorm:"not null;index" json:"earnedAt"`
	ExpiresAt       time.Time        `gorm:"not null;index" json:"expiresAt"`
	OrderID         *uuid.UUID       `gorm:"type:uuid;index" json:"orderId,omitempty"`
	IdempotencyKey  string           `gorm:"type:varchar(160);uniqueIndex;not null" json:"-"`
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"createdAt"`
}

func (b *LoyaltyEarnBatch) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
