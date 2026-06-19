package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// referral.go — customer give-credit/get-credit referral program (#38). The
// reward is settled to the store-credit wallet (#33). Distinct from the
// driver-only DriverReferral in delivery.go.

// ReferralCode is a customer's unique, shareable referral code. One per user,
// minted lazily the first time they open the Refer & Earn screen.
type ReferralCode struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	Code      string    `gorm:"uniqueIndex;not null" json:"code"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// BeforeCreate mints the UUID in Go so the model works without the Postgres
// gen_random_uuid() default (e.g. in sqlite-backed unit tests).
func (r *ReferralCode) BeforeCreate(*gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// ReferralState is the lifecycle of a single redemption.
type ReferralState string

const (
	ReferralStatePending  ReferralState = "pending"  // referee signed up with a code; not yet rewarded
	ReferralStateRewarded ReferralState = "rewarded" // referee placed their first paid order; both credited
	ReferralStateRejected ReferralState = "rejected" // failed a fraud guard
)

// Referral is one redemption: a referee signed up with a referrer's code. A
// customer can be referred only ONCE, ever — enforced by the unique index on
// RefereeUserID. The reward fires on the referee's first paid order.
type Referral struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReferrerUserID uuid.UUID     `gorm:"type:uuid;not null;index" json:"referrerUserId"`
	RefereeUserID  uuid.UUID     `gorm:"type:uuid;uniqueIndex;not null" json:"refereeUserId"`
	Code           string        `gorm:"not null" json:"code"`
	Status         ReferralState `gorm:"type:varchar(12);not null;default:'pending'" json:"status"`
	// OrderID is the referee's first paid order that triggered the reward.
	OrderID        *uuid.UUID `gorm:"type:uuid" json:"orderId,omitempty"`
	ReferrerReward float64    `gorm:"default:0" json:"referrerReward"`
	RefereeReward  float64    `gorm:"default:0" json:"refereeReward"`
	RewardedAt     *time.Time `gorm:"" json:"rewardedAt,omitempty"`
	// Fraud snapshot captured at accept time (best-effort device/IP dedupe).
	RefereeDevice string    `gorm:"type:varchar(255)" json:"-"`
	RefereeIP     string    `gorm:"type:varchar(64)" json:"-"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// BeforeCreate mints the UUID in Go (see ReferralCode.BeforeCreate).
func (r *Referral) BeforeCreate(*gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
