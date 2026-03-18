package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeliveryProvider represents a third-party delivery service integration
type DeliveryProvider struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string    `gorm:"not null" json:"name"`                           // e.g. "Dunzo", "Porter", "Shadowfax"
	Code        string    `gorm:"uniqueIndex;not null" json:"code"`               // unique, lowercase slug: "dunzo", "porter", "shadowfax"
	Description string    `gorm:"" json:"description"`
	LogoURL     string    `gorm:"" json:"logoUrl"`                                // provider logo for admin UI

	// API Configuration
	APIBaseURL    string `gorm:"" json:"apiBaseUrl"`             // e.g. "https://api.dunzo.com/v1"
	APIKey        string `gorm:"" json:"-"`                      // encrypted, never exposed
	APISecret     string `gorm:"" json:"-"`
	WebhookSecret string `gorm:"" json:"-"`                      // for verifying inbound webhooks

	// Status mapping — maps provider statuses to Fe3dr DeliveryStatus
	// e.g. {"PICKED_UP": "picked_up", "DELIVERED": "delivered", "CANCELLED": "cancelled"}
	StatusMapping string `gorm:"type:jsonb;default:'{}'" json:"statusMapping"`

	// Configuration
	SupportedCities    string  `gorm:"type:jsonb;default:'[]'" json:"supportedCities"`       // JSON array of city names
	SupportedCountries string  `gorm:"type:jsonb;default:'[\"IN\"]'" json:"supportedCountries"` // JSON array of country codes
	MaxDistance        float64 `gorm:"default:20" json:"maxDistance"`                        // max delivery distance in km
	AvgPickupTime      int     `gorm:"default:15" json:"avgPickupTime"`                     // avg minutes to pickup

	// Pricing — what Fe3dr pays the provider per delivery
	PricingModel string  `gorm:"type:varchar(20);default:'per_delivery'" json:"pricingModel"` // per_delivery, per_km, flat_rate
	BaseCost     float64 `gorm:"default:0" json:"baseCost"`                                  // base cost per delivery
	PerKmCost    float64 `gorm:"default:0" json:"perKmCost"`                                 // additional per km
	Currency     string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	// Priority & Control
	Priority  int  `gorm:"default:1" json:"priority"`      // lower = higher priority (tried first)
	IsEnabled bool `gorm:"default:false" json:"isEnabled"`
	IsActive  bool `gorm:"default:true" json:"isActive"`   // soft disable without deleting

	// Rate limiting
	MaxConcurrentDeliveries int `gorm:"default:100" json:"maxConcurrentDeliveries"` // max concurrent active deliveries
	DailyLimit              int `gorm:"default:0" json:"dailyLimit"`                // 0 = unlimited

	// Stats (updated periodically)
	TotalDeliveries int        `gorm:"default:0" json:"totalDeliveries"`
	SuccessRate     float64    `gorm:"default:0" json:"successRate"`     // percentage
	AvgDeliveryTime int        `gorm:"default:0" json:"avgDeliveryTime"` // minutes
	LastUsedAt      *time.Time `gorm:"" json:"lastUsedAt,omitempty"`

	// Metadata
	ContactName  string `gorm:"" json:"contactName"`
	ContactEmail string `gorm:"" json:"contactEmail"`
	ContactPhone string `gorm:"" json:"contactPhone"`
	Notes        string `gorm:"type:text" json:"notes"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// DeliveryProviderResponse is the DTO for API responses (excludes API keys/secrets)
type DeliveryProviderResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	LogoURL     string    `json:"logoUrl"`

	APIBaseURL    string `json:"apiBaseUrl"`
	StatusMapping string `json:"statusMapping"`

	SupportedCities    string  `json:"supportedCities"`
	SupportedCountries string  `json:"supportedCountries"`
	MaxDistance        float64 `json:"maxDistance"`
	AvgPickupTime      int     `json:"avgPickupTime"`

	PricingModel string  `json:"pricingModel"`
	BaseCost     float64 `json:"baseCost"`
	PerKmCost    float64 `json:"perKmCost"`
	Currency     string  `json:"currency"`

	Priority  int  `json:"priority"`
	IsEnabled bool `json:"isEnabled"`
	IsActive  bool `json:"isActive"`

	MaxConcurrentDeliveries int `json:"maxConcurrentDeliveries"`
	DailyLimit              int `json:"dailyLimit"`

	TotalDeliveries int        `json:"totalDeliveries"`
	SuccessRate     float64    `json:"successRate"`
	AvgDeliveryTime int        `json:"avgDeliveryTime"`
	LastUsedAt      *time.Time `json:"lastUsedAt,omitempty"`

	ContactName  string `json:"contactName"`
	ContactEmail string `json:"contactEmail"`
	ContactPhone string `json:"contactPhone"`
	Notes        string `json:"notes"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ToResponse converts a DeliveryProvider to its safe response DTO
func (p *DeliveryProvider) ToResponse() DeliveryProviderResponse {
	return DeliveryProviderResponse{
		ID:                      p.ID,
		Name:                    p.Name,
		Code:                    p.Code,
		Description:             p.Description,
		LogoURL:                 p.LogoURL,
		APIBaseURL:              p.APIBaseURL,
		StatusMapping:           p.StatusMapping,
		SupportedCities:         p.SupportedCities,
		SupportedCountries:      p.SupportedCountries,
		MaxDistance:              p.MaxDistance,
		AvgPickupTime:           p.AvgPickupTime,
		PricingModel:            p.PricingModel,
		BaseCost:                p.BaseCost,
		PerKmCost:               p.PerKmCost,
		Currency:                p.Currency,
		Priority:                p.Priority,
		IsEnabled:               p.IsEnabled,
		IsActive:                p.IsActive,
		MaxConcurrentDeliveries: p.MaxConcurrentDeliveries,
		DailyLimit:              p.DailyLimit,
		TotalDeliveries:         p.TotalDeliveries,
		SuccessRate:             p.SuccessRate,
		AvgDeliveryTime:         p.AvgDeliveryTime,
		LastUsedAt:              p.LastUsedAt,
		ContactName:             p.ContactName,
		ContactEmail:            p.ContactEmail,
		ContactPhone:            p.ContactPhone,
		Notes:                   p.Notes,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}
