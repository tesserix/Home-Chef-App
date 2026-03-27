package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PromoCode struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code           string         `gorm:"uniqueIndex;not null" json:"code"`
	Description    string         `gorm:"type:text" json:"description"`
	DiscountType   string         `gorm:"type:varchar(20);not null" json:"discountType"` // "percentage" or "fixed"
	DiscountValue  float64        `gorm:"not null" json:"discountValue"`
	MinOrderAmount float64        `gorm:"default:0" json:"minOrderAmount"`
	MaxDiscount    float64        `gorm:"default:0" json:"maxDiscount"`
	UsageLimit     int            `gorm:"default:0" json:"usageLimit"`
	UsageCount     int            `gorm:"default:0" json:"usageCount"`
	PerUserLimit   int            `gorm:"default:0" json:"perUserLimit"`
	ValidFrom      time.Time      `gorm:"not null" json:"validFrom"`
	ValidUntil     *time.Time     `gorm:"" json:"validUntil,omitempty"`
	IsActive       bool           `gorm:"default:true" json:"isActive"`
	ApplicableTo   string         `gorm:"type:varchar(30);default:'all'" json:"applicableTo"` // "all", "new_users", "returning_users"
	CreatedByID    uuid.UUID      `gorm:"type:uuid;not null" json:"createdById"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
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
