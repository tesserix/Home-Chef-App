package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// chef_loyalty.go — chef-side loyalty points. Distinct from the customer
// loyalty program (loyalty.go): chef points are earned per DELIVERED order,
// never spent at checkout, and convert into a rupee cashback (ChefBonus)
// netted onto the chef's weekly settlement once the balance crosses the
// conversion threshold.

// ChefLoyaltyAccount is the chef's live points balance. Kept as a row (not a
// SUM over txns) so conversion can decrement race-safely with a guarded UPDATE.
type ChefLoyaltyAccount struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"chefId"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`

	Points float64 `gorm:"default:0" json:"points"`
	// LifetimePoints only ever grows — the chef's all-time earned total.
	LifetimePoints float64 `gorm:"default:0" json:"lifetimePoints"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// BeforeCreate mints the UUID in Go (sqlite-backed unit tests).
func (a *ChefLoyaltyAccount) BeforeCreate(*gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// ChefLoyaltyTxnKind identifies why points moved.
type ChefLoyaltyTxnKind string

const (
	// ChefLoyaltyEarnOrder — points earned for a delivered order.
	ChefLoyaltyEarnOrder ChefLoyaltyTxnKind = "order_earn"
	// ChefLoyaltyConvert — points converted into a cashback ChefBonus (negative).
	ChefLoyaltyConvert ChefLoyaltyTxnKind = "conversion"
	// ChefLoyaltyAdminAdjust — manual admin correction (either sign).
	ChefLoyaltyAdminAdjust ChefLoyaltyTxnKind = "admin_adjustment"
)

// ChefLoyaltyTxn is the audit trail of every points movement.
type ChefLoyaltyTxn struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`

	Kind ChefLoyaltyTxnKind `gorm:"type:varchar(24);not null;index" json:"kind"`
	// Points is signed: positive earn, negative conversion.
	Points float64 `gorm:"not null" json:"points"`

	// SourceKey is the natural key of the movement ("cheforder:<orderID>",
	// "chefconv:<bonusID>"), UNIQUE so a re-stamped delivered status or a
	// retried conversion never double-moves points.
	SourceKey string `gorm:"uniqueIndex;not null" json:"sourceKey"`

	OrderID *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`
	// BonusID links a conversion txn to the cashback ChefBonus it funded.
	BonusID *uuid.UUID `gorm:"type:uuid;index" json:"bonusId,omitempty"`

	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// BeforeCreate mints the UUID in Go (sqlite-backed unit tests).
func (t *ChefLoyaltyTxn) BeforeCreate(*gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
