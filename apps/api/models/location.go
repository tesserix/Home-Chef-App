// Package-level note: location models follow mark8ly platform-api's clean
// reference-data design — natural ISO-coded PKs (no synthetic UUIDs), no
// IsActive flag (obsolete rows are deleted, not soft-disabled), simple
// relationships expressed through string FKs. See
// `mark8ly/services/platform-api/internal/location/` for the original.
//
// Scope today is India-only: the seeder writes a single country (IN), all
// 36 Indian states + UTs, ~50 major Indian cities, and a representative
// set of PIN codes. The schema itself is country-agnostic so a second
// country can be added by extending the seed.
package models

import "time"

// Country is an ISO 3166-1 country.
type Country struct {
	// Code is the ISO 3166-1 alpha-2 country code (e.g. "IN").
	Code string `gorm:"primaryKey;column:code;type:char(2)" json:"code"`
	Name string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// NativeName is the country's name in its primary local language.
	NativeName string `gorm:"column:native_name;type:varchar(100)" json:"nativeName"`
	// CallingCode is the international dialing prefix including the leading "+".
	CallingCode string `gorm:"column:calling_code;type:varchar(10)" json:"callingCode"`
	// CurrencyCode is the ISO 4217 code of the country's primary currency.
	CurrencyCode string `gorm:"column:currency_code;type:char(3)" json:"currencyCode"`
	FlagEmoji    string `gorm:"column:flag_emoji;type:varchar(10)" json:"flagEmoji"`
	Region       string `gorm:"column:region;type:varchar(50)" json:"region"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"-"`
}

func (Country) TableName() string { return "countries" }

// State is a country's first-level administrative subdivision.
type State struct {
	// ID is the country code + state code joined by "-" (e.g. "IN-MH").
	ID string `gorm:"primaryKey;column:id;type:varchar(10)" json:"id"`
	// CountryCode is the parent country's ISO code.
	CountryCode string `gorm:"column:country_code;type:char(2);not null;index" json:"countryCode"`
	// Code is the state's local code (e.g. "MH", "KA").
	Code string `gorm:"column:code;type:varchar(10);not null" json:"code"`
	Name string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// Type is one of: state, territory.
	Type string `gorm:"column:type;type:varchar(20);not null;default:'state'" json:"type"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"-"`
}

func (State) TableName() string { return "states" }

// City is a populated place within a State.
//
// Cities don't have a global ISO standard, so the PK is a deterministic
// slug shaped as `{state_id}-{kebab-name}` (e.g. "IN-KA-bengaluru"). That
// stays human-debuggable in logs and lets us seed idempotently.
type City struct {
	ID      string `gorm:"primaryKey;column:id;type:varchar(80)" json:"id"`
	StateID string `gorm:"column:state_id;type:varchar(10);not null;index" json:"stateId"`
	Name    string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	// IsMajor flags top-tier population centers so the UI can surface them
	// first in the city autocomplete picker.
	IsMajor   bool    `gorm:"column:is_major;default:false" json:"isMajor"`
	Latitude  float64 `gorm:"column:latitude" json:"latitude,omitempty"`
	Longitude float64 `gorm:"column:longitude" json:"longitude,omitempty"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"-"`
}

func (City) TableName() string { return "cities" }

// Postcode is a postal code (PIN code in India) anchored to a City.
//
// The code itself is the PK — India PIN codes are 6 digits and globally
// unique within India. If we ever support a second country we'll need a
// composite (country_code, code) PK, but for now this stays simple.
type Postcode struct {
	Code     string `gorm:"primaryKey;column:code;type:varchar(10)" json:"code"`
	CityID   string `gorm:"column:city_id;type:varchar(80);not null;index" json:"cityId"`
	AreaName string `gorm:"column:area_name;type:varchar(200)" json:"areaName"`

	CreatedAt time.Time `gorm:"column:created_at;not null;default:now()" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null;default:now()" json:"-"`
}

func (Postcode) TableName() string { return "postcodes" }
