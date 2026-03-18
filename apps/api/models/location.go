package models

import (
	"time"

	"github.com/google/uuid"
)

type Country struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code           string    `gorm:"type:varchar(3);uniqueIndex;not null" json:"code"`
	Name           string    `gorm:"type:varchar(100);not null" json:"name"`
	PhoneCode      string    `gorm:"type:varchar(10);not null" json:"phoneCode"`
	CurrencyCode   string    `gorm:"type:varchar(3)" json:"currencyCode"`
	CurrencySymbol string    `gorm:"type:varchar(5)" json:"currencySymbol"`
	IsActive       bool      `gorm:"default:true" json:"isActive"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"createdAt"`

	States []State `gorm:"foreignKey:CountryID" json:"states,omitempty"`
}

type State struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CountryID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_state_country_code" json:"countryId"`
	Code      string    `gorm:"type:varchar(10);not null;uniqueIndex:idx_state_country_code" json:"code"`
	Name      string    `gorm:"type:varchar(100);not null" json:"name"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	Country Country `gorm:"foreignKey:CountryID" json:"-"`
	Cities  []City  `gorm:"foreignKey:StateID" json:"cities,omitempty"`
}

type City struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StateID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_city_state_name" json:"stateId"`
	Name      string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_city_state_name" json:"name"`
	IsMajor   bool      `gorm:"default:false" json:"isMajor"`
	Latitude  float64   `gorm:"" json:"latitude,omitempty"`
	Longitude float64   `gorm:"" json:"longitude,omitempty"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	State     State      `gorm:"foreignKey:StateID" json:"-"`
	Postcodes []Postcode `gorm:"foreignKey:CityID" json:"postcodes,omitempty"`
}

type Postcode struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CityID    uuid.UUID `gorm:"type:uuid;not null;index" json:"cityId"`
	Code      string    `gorm:"type:varchar(10);uniqueIndex;not null" json:"code"`
	AreaName  string    `gorm:"type:varchar(200)" json:"areaName"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	City City `gorm:"foreignKey:CityID" json:"-"`
}
