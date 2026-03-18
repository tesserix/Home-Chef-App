package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeliveryZone struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name    string    `gorm:"type:varchar(100);not null" json:"name"`
	City    string    `gorm:"type:varchar(100);not null;index" json:"city"`
	State   string    `gorm:"type:varchar(100)" json:"state"`
	Country string    `gorm:"type:varchar(2);default:'IN';index" json:"country"` // ISO 3166-1 alpha-2
	Tier    string    `gorm:"type:varchar(20);default:'standard'" json:"tier"`    // metro, mid_tier, standard, regional

	// Bounding box for quick filtering
	MinLatitude  float64 `gorm:"" json:"minLatitude"`
	MaxLatitude  float64 `gorm:"" json:"maxLatitude"`
	MinLongitude float64 `gorm:"" json:"minLongitude"`
	MaxLongitude float64 `gorm:"" json:"maxLongitude"`

	// Detailed boundary (GeoJSON)
	Boundary string `gorm:"type:jsonb;default:'{}'" json:"boundary"` // GeoJSON polygon

	// Pricing — all amounts in the zone's local currency
	Currency        string  `gorm:"type:varchar(3);default:'INR'" json:"currency"` // ISO 4217 currency code
	BaseFare        float64 `gorm:"default:0" json:"baseFare"`
	PerKmRate       float64 `gorm:"default:0" json:"perKmRate"`
	MinimumFare     float64 `gorm:"default:0" json:"minimumFare"`
	SurgeMultiplier float64 `gorm:"default:1.0" json:"surgeMultiplier"`

	// Tipping
	TipEnabled       bool    `gorm:"default:true" json:"tipEnabled"`
	DefaultTipPercent float64 `gorm:"default:10" json:"defaultTipPercent"` // Suggested tip %
	MaxTipAmount     float64 `gorm:"default:0" json:"maxTipAmount"`       // 0 = no cap

	// Driver payout share — 100% (subscription model, no platform commission)
	DriverPayoutPercent float64 `gorm:"default:100" json:"driverPayoutPercent"` // % of delivery fee to driver

	IsActive  bool           `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type DeliveryZoneResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	City                string    `json:"city"`
	State               string    `json:"state"`
	Country             string    `json:"country"`
	Tier                string    `json:"tier"`
	MinLatitude         float64   `json:"minLatitude"`
	MaxLatitude         float64   `json:"maxLatitude"`
	MinLongitude        float64   `json:"minLongitude"`
	MaxLongitude        float64   `json:"maxLongitude"`
	Currency            string    `json:"currency"`
	BaseFare            float64   `json:"baseFare"`
	PerKmRate           float64   `json:"perKmRate"`
	MinimumFare         float64   `json:"minimumFare"`
	SurgeMultiplier     float64   `json:"surgeMultiplier"`
	TipEnabled          bool      `json:"tipEnabled"`
	DefaultTipPercent   float64   `json:"defaultTipPercent"`
	MaxTipAmount        float64   `json:"maxTipAmount"`
	DriverPayoutPercent float64   `json:"driverPayoutPercent"`
	IsActive            bool      `json:"isActive"`
	CreatedAt           time.Time `json:"createdAt"`
}

func (z *DeliveryZone) ToResponse() DeliveryZoneResponse {
	return DeliveryZoneResponse{
		ID:                  z.ID,
		Name:                z.Name,
		City:                z.City,
		State:               z.State,
		Country:             z.Country,
		Tier:                z.Tier,
		MinLatitude:         z.MinLatitude,
		MaxLatitude:         z.MaxLatitude,
		MinLongitude:        z.MinLongitude,
		MaxLongitude:        z.MaxLongitude,
		Currency:            z.Currency,
		BaseFare:            z.BaseFare,
		PerKmRate:           z.PerKmRate,
		MinimumFare:         z.MinimumFare,
		SurgeMultiplier:     z.SurgeMultiplier,
		TipEnabled:          z.TipEnabled,
		DefaultTipPercent:   z.DefaultTipPercent,
		MaxTipAmount:        z.MaxTipAmount,
		DriverPayoutPercent: z.DriverPayoutPercent,
		IsActive:            z.IsActive,
		CreatedAt:           z.CreatedAt,
	}
}
