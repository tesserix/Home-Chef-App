package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Promo funding sources (#39). Platform-funded promos are absorbed by the
// platform's margin; chef-funded promos are billed to the chef at settlement
// (the discount is subtracted from their Route payout + earnings).
const (
	PromoFundingPlatform = "platform"
	PromoFundingChef     = "chef"
)

type PromoCode struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code           string     `gorm:"uniqueIndex;not null" json:"code"`
	Description    string     `gorm:"type:text" json:"description"`
	DiscountType   string     `gorm:"type:varchar(20);not null" json:"discountType"` // "percentage" or "fixed"
	DiscountValue  float64    `gorm:"not null" json:"discountValue"`
	MinOrderAmount float64    `gorm:"default:0" json:"minOrderAmount"`
	MaxDiscount    float64    `gorm:"default:0" json:"maxDiscount"`
	UsageLimit     int        `gorm:"default:0" json:"usageLimit"`
	UsageCount     int        `gorm:"default:0" json:"usageCount"`
	PerUserLimit   int        `gorm:"default:0" json:"perUserLimit"`
	ValidFrom      time.Time  `gorm:"not null" json:"validFrom"`
	ValidUntil     *time.Time `gorm:"" json:"validUntil,omitempty"`
	IsActive       bool       `gorm:"default:true" json:"isActive"`
	ApplicableTo   string     `gorm:"type:varchar(30);default:'all'" json:"applicableTo"` // "all", "new_users", "returning_users"
	// FundingSource (#39): "platform" (default) or "chef". ChefID is required and
	// scopes the promo to that chef's orders when funding is "chef".
	FundingSource string     `gorm:"type:varchar(16);default:'platform'" json:"fundingSource"`
	ChefID        *uuid.UUID `gorm:"type:uuid;index" json:"chefId,omitempty"`
	// BudgetCap (#39): max total discount spend across all redemptions (0 =
	// unlimited). BudgetSpent is the running total; a redemption is blocked once
	// BudgetSpent + discount would exceed BudgetCap.
	BudgetCap   float64        `gorm:"default:0" json:"budgetCap"`
	BudgetSpent float64        `gorm:"default:0" json:"budgetSpent"`
	CreatedByID uuid.UUID      `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type PromoCodeUsage struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PromoCodeID uuid.UUID `gorm:"type:uuid;not null;index" json:"promoCodeId"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	OrderID     uuid.UUID `gorm:"type:uuid;not null" json:"orderId"`
	Discount    float64   `gorm:"not null" json:"discount"`
	UsedAt      time.Time `gorm:"autoCreateTime" json:"usedAt"`

	PromoCode PromoCode `gorm:"foreignKey:PromoCodeID" json:"promoCode,omitempty"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
