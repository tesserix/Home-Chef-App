package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoyaltyTxnType is the direction of a points ledger entry — mirrors the wallet
// ledger. Earns/bonuses add points; redeems/expiries/clawbacks remove them.
type LoyaltyTxnType string

const (
	LoyaltyCredit LoyaltyTxnType = "credit" // points into the account
	LoyaltyDebit  LoyaltyTxnType = "debit"  // points out of the account
)

// LoyaltyTxnSource classifies why a points ledger entry exists. Credits come
// from delivered orders or meal-subscription streak bonuses; debits come from
// redeeming to wallet store credit or an admin clawback.
type LoyaltyTxnSource string

const (
	LoyaltySourceOrder       LoyaltyTxnSource = "order"        // earned on a delivered order
	LoyaltySourceStreak      LoyaltyTxnSource = "streak_bonus" // meal-sub adherence streak
	LoyaltySourceRedeem      LoyaltyTxnSource = "redeem"       // converted to wallet credit
	LoyaltySourceAdminAdjust LoyaltyTxnSource = "admin_adjustment"
)

// Loyalty tiers, derived from lifetime points earned. Tiers are cosmetic by
// default (a status badge); perks are layered on later. Thresholds are
// admin-configurable via loyalty.* PlatformSettings.
const (
	LoyaltyTierBronze = "bronze"
	LoyaltyTierSilver = "silver"
	LoyaltyTierGold   = "gold"
)

// LoyaltyAccount is a customer's points balance — one per user. Like the Wallet,
// Balance is a cached aggregate kept consistent with the LoyaltyTransaction
// ledger inside the same DB transaction: the ledger is the source of truth,
// Balance is the fast read. Points are whole numbers stored as float64 to match
// the money math they convert into (RedeemRate ₹/point).
//
// Streak fields track meal-subscription adherence (#2): CurrentStreak is the
// run of consecutive delivered meal-sub days. LastStreakDay is the calendar date
// (truncated to midnight) of the most recent counted day; the next delivery
// continues the run when it lands within the configured grace window of it and
// resets the run when the gap is larger (a missed day produces such a gap).
type LoyaltyAccount struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	Balance        float64    `gorm:"not null;default:0" json:"balance"`        // redeemable points
	LifetimePoints float64    `gorm:"not null;default:0" json:"lifetimePoints"` // total ever earned — drives tier
	Tier           string     `gorm:"type:varchar(16);not null;default:'bronze'" json:"tier"`
	CurrentStreak  int        `gorm:"not null;default:0" json:"currentStreak"`
	LongestStreak  int        `gorm:"not null;default:0" json:"longestStreak"`
	LastStreakDay  *time.Time `gorm:"type:date" json:"lastStreakDay,omitempty"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

// LoyaltyTransaction is an immutable, append-only points ledger entry (no
// updates, no soft delete) — the exact shape of WalletTxn, in points instead of
// money. PointsAfter is the running balance snapshot after this entry.
// IdempotencyKey dedupes retried operations (a redelivered order-delivered
// event, a double-tapped redeem) so the same logical event can never
// double-earn/double-redeem: it is a unique index and every entry carries a
// non-empty key (semantic where one exists, e.g. "loyalty:order:<id>", else a
// generated UUID).
type LoyaltyTransaction struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	LoyaltyAccountID uuid.UUID        `gorm:"type:uuid;not null;index" json:"loyaltyAccountId"`
	UserID           uuid.UUID        `gorm:"type:uuid;not null;index" json:"userId"`
	Type             LoyaltyTxnType   `gorm:"type:varchar(10);not null" json:"type"`
	Source           LoyaltyTxnSource `gorm:"type:varchar(20);not null" json:"source"`
	Points           float64          `gorm:"not null" json:"points"`      // always positive
	PointsAfter      float64          `gorm:"not null" json:"pointsAfter"` // running snapshot
	OrderID          *uuid.UUID       `gorm:"type:uuid;index" json:"orderId,omitempty"`
	Reason           string           `gorm:"type:text" json:"reason,omitempty"`
	// CreatedBy is the admin user id for admin adjustments; nil for
	// system/customer-initiated entries.
	CreatedBy      *uuid.UUID `gorm:"type:uuid" json:"createdBy,omitempty"`
	IdempotencyKey string     `gorm:"type:varchar(160);uniqueIndex;not null" json:"-"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"createdAt"`
}

// BeforeCreate mints UUIDs for the sqlite test driver, which has no
// gen_random_uuid(). Postgres uses the column default in prod. Mirrors the hook
// pattern used across the models for test compatibility.
func (a *LoyaltyAccount) BeforeCreate(*gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (t *LoyaltyTransaction) BeforeCreate(*gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
