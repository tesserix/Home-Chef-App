package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// chef_referral.go — chef-refers-chef program. Codes are minted from the shared
// referral_codes table (one code per user, same as the customer program); this
// file holds the chef-side redemption + the settlement-credit mechanism.

// ChefReferralState is the lifecycle of one kitchen referral.
type ChefReferralState string

const (
	// ChefReferralPending — referee onboarded with a code; milestone not reached.
	ChefReferralPending ChefReferralState = "pending"
	// ChefReferralRewarded — referee hit the delivered-orders milestone; both
	// sides' bonuses were raised.
	ChefReferralRewarded ChefReferralState = "rewarded"
	// ChefReferralRejected — an admin voided it (fraud / policy).
	ChefReferralRejected ChefReferralState = "rejected"
)

// ChefReferral is one redemption: a new kitchen onboarded with an existing
// chef's code. A kitchen can be referred only once — unique on RefereeChefID.
// Both rewards VEST when the referee kitchen completes MilestoneOrders
// delivered orders, so a referral only pays once the new kitchen is real.
type ChefReferral struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReferrerChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"referrerChefId"`
	ReferrerUserID uuid.UUID `gorm:"type:uuid;not null;index" json:"referrerUserId"`
	RefereeChefID  uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"refereeChefId"`
	RefereeUserID  uuid.UUID `gorm:"type:uuid;not null;index" json:"refereeUserId"`
	Code           string    `gorm:"not null" json:"code"`

	Status ChefReferralState `gorm:"type:varchar(12);not null;default:'pending';index" json:"status"`

	// MilestoneOrders is FROZEN from config at accept time so an admin retuning
	// the program never moves the goalposts on an in-flight referral.
	MilestoneOrders int `gorm:"not null;default:10" json:"milestoneOrders"`

	// Amounts (₹) frozen at reward time; 0 while pending.
	ReferrerAmount float64    `gorm:"default:0" json:"referrerAmount"`
	RefereeAmount  float64    `gorm:"default:0" json:"refereeAmount"`
	RewardedAt     *time.Time `gorm:"" json:"rewardedAt,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// BeforeCreate mints the UUID in Go so the model works without the Postgres
// gen_random_uuid() default (sqlite-backed unit tests).
func (r *ChefReferral) BeforeCreate(*gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// ChefBonusStatus is the lifecycle of a settlement credit.
type ChefBonusStatus string

const (
	// ChefBonusPending — raised, not yet added onto a settlement.
	ChefBonusPending ChefBonusStatus = "pending"
	// ChefBonusCredited — added onto a weekly settlement statement (CreditedStatementID).
	ChefBonusCredited ChefBonusStatus = "credited"
	// ChefBonusVoided — an admin cancelled it; never credited.
	ChefBonusVoided ChefBonusStatus = "voided"
)

// ChefBonusKind identifies why a credit was raised.
type ChefBonusKind string

const (
	ChefBonusReferralReferrer ChefBonusKind = "referral_referrer"
	ChefBonusReferralReferee  ChefBonusKind = "referral_referee"
	// ChefBonusLoyaltyCashback — a chef converted loyalty points into cashback.
	ChefBonusLoyaltyCashback ChefBonusKind = "loyalty_cashback"
	// ChefBonusCancellationRetained — the share of a CANCELLED order the chef
	// retained under the epic #475 tier model (the food the customer was not
	// refunded because the chef had already started work). Computed and shown to
	// the customer since #475 but never paid until #947; it settles through the
	// weekly statement like any other bonus rather than through the payout-hold
	// path, which must keep excluding cancelled orders.
	ChefBonusCancellationRetained ChefBonusKind = "cancellation_retained"
	// ChefBonusTipCatchup — a customer tip that was charged but never reached the
	// chef (#964), on an order already billed on a FROZEN weekly statement that can
	// no longer absorb it. Only ever raised by the backfill; once checkout writes
	// chef_tip the statement includes tips directly and no catch-up is needed.
	ChefBonusTipCatchup ChefBonusKind = "tip_catchup"
	// ChefBonusStatementCatchup — an order whose payout was still HELD (awaiting
	// customer confirmation, or disputed) when its weekly statement closed, and so
	// could not be billed on it (#927). Statements are frozen and windowed on
	// delivered_at, so no later statement would ever pick it up; this credit is
	// what makes excluding a transient hold state safe rather than a silent loss.
	ChefBonusStatementCatchup ChefBonusKind = "statement_catchup"
)

// ChefBonus is a rupee credit owed to a chef, added onto their next weekly
// settlement — the exact credit mirror of ChefPenalty (which is netted OFF).
type ChefBonus struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	// UserID is denormalized so the settlement join (by user) and the
	// notification path don't need a second lookup — same as ChefPenalty.
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`

	Kind   ChefBonusKind   `gorm:"type:varchar(24);not null;index" json:"kind"`
	Status ChefBonusStatus `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`

	// SourceKey is the natural key of the event that raised the credit
	// ("chefref-referrer:<referralID>"), UNIQUE so retries credit once.
	SourceKey string `gorm:"uniqueIndex;not null" json:"sourceKey"`

	ReferralID *uuid.UUID `gorm:"type:uuid;index" json:"referralId,omitempty"`

	Currency string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`
	Amount   float64 `gorm:"default:0" json:"amount"`
	Reason   string  `gorm:"type:text" json:"reason,omitempty"`

	// Settlement linkage, set when the credit is added onto a weekly statement.
	CreditedStatementID *uuid.UUID `gorm:"type:uuid;index" json:"creditedStatementId,omitempty"`
	CreditedAt          *time.Time `gorm:"" json:"creditedAt,omitempty"`

	// Void audit.
	VoidedBy   *uuid.UUID `gorm:"type:uuid" json:"voidedBy,omitempty"`
	VoidedAt   *time.Time `gorm:"" json:"voidedAt,omitempty"`
	VoidReason string     `gorm:"type:text" json:"voidReason,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName pins the plural GORM's inflector gets wrong ("chef_bonus").
func (ChefBonus) TableName() string { return "chef_bonuses" }

// BeforeCreate assigns the id when the caller left it unset (sqlite tests).
func (b *ChefBonus) BeforeCreate(*gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
