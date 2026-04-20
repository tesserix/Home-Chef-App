package models

import (
	"time"

	"github.com/google/uuid"
)

// TaxRate captures the tax rule applied to orders delivered to a given
// country (and optionally a specific region/state for countries like the
// US and Canada where the rate varies below the country level). The rule
// is picked at order creation based on the delivery address — US orders
// to CA use California's rate, Indian orders get GST, EU orders get VAT.
//
// A row with region="" is the country-wide fallback; a row with a specific
// region wins over it. Rows with is_active=false are excluded from lookup
// so admins can disable a rule without deleting history.
type TaxRate struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CountryCode string     `gorm:"type:varchar(2);not null;index:idx_tax_lookup" json:"countryCode"`
	Region      string     `gorm:"type:varchar(10);default:'';index:idx_tax_lookup" json:"region"`
	// Human-readable name shown on invoices ("GST", "VAT", "Sales Tax"). Also
	// used as the fallback label when a jurisdiction has no better label.
	TaxName string `gorm:"type:varchar(40);not null" json:"taxName"`
	// Rate expressed as a percent (5.0 for 5%). Applied to (subtotal +
	// deliveryFee + serviceFee) unless more granular rules are added later.
	Rate float64 `gorm:"not null" json:"rate"`
	// Inclusive=true means prices already contain the tax (common in
	// Europe); inclusive=false means tax is added on top (US sales tax,
	// India GST on takeaway). Display logic differs so the invoice can
	// show either "incl. VAT" or "+ 5% GST" correctly.
	Inclusive bool `gorm:"default:false" json:"inclusive"`
	// Notes: free-form description shown to admins; not rendered to
	// customers. Useful for "GST on restaurants — small scheme 5% only".
	Notes     string    `gorm:"type:text" json:"notes,omitempty"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName pins the Postgres table — GORM would otherwise pluralize to
// "tax_rates" which is what we want, but make it explicit for clarity.
func (TaxRate) TableName() string { return "tax_rates" }
