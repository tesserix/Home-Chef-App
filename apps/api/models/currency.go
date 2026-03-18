package models

import (
	"time"

	"github.com/google/uuid"
)

type Currency struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code          string    `gorm:"type:varchar(3);uniqueIndex;not null" json:"code"`
	Name          string    `gorm:"type:varchar(100);not null" json:"name"`
	Symbol        string    `gorm:"type:varchar(10);not null" json:"symbol"`
	DecimalPlaces int       `gorm:"default:2" json:"decimalPlaces"`
	IsActive      bool      `gorm:"default:true" json:"isActive"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

type ExchangeRate struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BaseCurrency   string    `gorm:"type:varchar(3);not null;uniqueIndex:idx_rate_pair" json:"baseCurrency"`
	TargetCurrency string    `gorm:"type:varchar(3);not null;uniqueIndex:idx_rate_pair" json:"targetCurrency"`
	Rate           float64   `gorm:"not null" json:"rate"`
	FetchedAt      time.Time `gorm:"not null" json:"fetchedAt"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
