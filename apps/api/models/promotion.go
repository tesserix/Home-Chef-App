package models

import (
	"time"

	"github.com/google/uuid"
)

type PromotionStatus string

const (
	PromotionActive    PromotionStatus = "active"
	PromotionExpired   PromotionStatus = "expired"
	PromotionCancelled PromotionStatus = "cancelled"
	PromotionPending   PromotionStatus = "pending" // Payment not yet captured
)

// ChefPromotion tracks a chef's featured ad purchase
type ChefPromotion struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID    uuid.UUID       `gorm:"type:uuid;not null;index" json:"chefId"`
	Status    PromotionStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Amount    float64         `gorm:"not null" json:"amount"`
	Currency  string          `gorm:"type:varchar(3);not null" json:"currency"`
	Duration  int             `gorm:"not null" json:"duration"` // Days (30 = monthly)
	StartsAt  time.Time       `gorm:"not null" json:"startsAt"`
	ExpiresAt time.Time       `gorm:"not null" json:"expiresAt"`

	// Payment
	RazorpayOrderID   string `gorm:"" json:"-"`
	RazorpayPaymentID string `gorm:"" json:"-"`
	PaymentMethod     string `gorm:"" json:"paymentMethod,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
}

// PromotionPricing holds per-country pricing for featured ads
type PromotionPricing struct {
	MonthlyPrice float64
	Currency     string
}

// DefaultPromotionPricing returns featured ad pricing by country code
func DefaultPromotionPricing(countryCode string) PromotionPricing {
	switch countryCode {
	case "IN":
		return PromotionPricing{MonthlyPrice: 499, Currency: "INR"}
	case "US":
		return PromotionPricing{MonthlyPrice: 9.99, Currency: "USD"}
	case "GB":
		return PromotionPricing{MonthlyPrice: 7.99, Currency: "GBP"}
	case "AU":
		return PromotionPricing{MonthlyPrice: 12.99, Currency: "AUD"}
	case "PK":
		return PromotionPricing{MonthlyPrice: 999, Currency: "PKR"}
	case "BD":
		return PromotionPricing{MonthlyPrice: 999, Currency: "BDT"}
	case "LK":
		return PromotionPricing{MonthlyPrice: 1999, Currency: "LKR"}
	case "NP":
		return PromotionPricing{MonthlyPrice: 999, Currency: "NPR"}
	case "AE":
		return PromotionPricing{MonthlyPrice: 29.99, Currency: "AED"}
	case "SA":
		return PromotionPricing{MonthlyPrice: 29.99, Currency: "SAR"}
	case "SG":
		return PromotionPricing{MonthlyPrice: 9.99, Currency: "SGD"}
	case "MY":
		return PromotionPricing{MonthlyPrice: 29.99, Currency: "MYR"}
	case "CA":
		return PromotionPricing{MonthlyPrice: 9.99, Currency: "CAD"}
	case "DE", "FR", "IT", "ES", "NL":
		return PromotionPricing{MonthlyPrice: 7.99, Currency: "EUR"}
	default:
		return PromotionPricing{MonthlyPrice: 9.99, Currency: "USD"}
	}
}
