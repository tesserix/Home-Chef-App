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
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CountryCode string    `gorm:"type:varchar(2);not null;index:idx_tax_lookup" json:"countryCode"`
	Region      string    `gorm:"type:varchar(10);default:'';index:idx_tax_lookup" json:"region"`
	// Human-readable name shown on invoices ("GST", "VAT", "Sales Tax"). Also
	// used as the fallback label when a jurisdiction has no better label.
	TaxName string `gorm:"type:varchar(40);not null" json:"taxName"`
	// Rate expressed as a percent (5.0 for 5%). The jurisdiction's default —
	// applied to every component that has no override below.
	Rate float64 `gorm:"not null" json:"rate"`
	// Per-component overrides, because one supply is not one rate: in India the
	// food is restaurant service (5%, no ITC) while the platform's own fee and a
	// platform-arranged delivery are standard-rated services. Leave them 0 and the
	// component takes Rate, which is exactly the uniform behaviour these rows have
	// today — so separating a component later is one admin edit, not a deploy.
	// See docs/india-gst-model.md §8.
	FoodPercent         float64 `gorm:"default:0" json:"foodPercent"`
	ServicePercent      float64 `gorm:"default:0" json:"servicePercent"`
	DeliveryPercent     float64 `gorm:"default:0" json:"deliveryPercent"`
	SubscriptionPercent float64 `gorm:"default:0" json:"subscriptionPercent"`
	// Inclusive=true means prices already contain the tax (common in
	// Europe); inclusive=false means tax is added on top (US sales tax,
	// India GST on takeaway). Display logic differs so the invoice can
	// show either "incl. VAT" or "+ 5% GST" correctly.
	Inclusive bool `gorm:"default:false" json:"inclusive"`
	// Registration identity for this jurisdiction, printed on the invoice:
	// what the number is called ("GSTIN", "ABN", "TIN") and the operator's own
	// number when it differs from the global company one.
	RegistrationIDLabel string `gorm:"type:varchar(20);default:''" json:"registrationIdLabel,omitempty"`
	CompanyTaxID        string `gorm:"type:varchar(40);default:''" json:"companyTaxId,omitempty"`
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

// ComponentRates resolves the rate each part of an order is taxed at, filling
// any unset override from Rate. This is the ONE place a rate comes from: order
// pricing, the checkout quote, meal plans, subscription billing and the stored
// invoice all read it, so none of them can drift from the others again.
func (t *TaxRate) ComponentRates() TaxRates {
	if t == nil {
		return TaxRates{}
	}
	or := func(override float64) float64 {
		if override > 0 {
			return override
		}
		return t.Rate
	}
	return TaxRates{
		Name:         t.TaxName,
		Inclusive:    t.Inclusive,
		Food:         or(t.FoodPercent),
		Service:      or(t.ServicePercent),
		Delivery:     or(t.DeliveryPercent),
		Subscription: or(t.SubscriptionPercent),
	}
}

// TaxRates is a resolved rule — what each component of an order is taxed at.
type TaxRates struct {
	Name      string `json:"name"`
	Inclusive bool   `json:"inclusive"`

	Food         float64 `json:"food"`
	Service      float64 `json:"service"`
	Delivery     float64 `json:"delivery"`
	Subscription float64 `json:"subscription"`
}

// Uniform reports whether every order component carries the same rate — the
// case today, and the one where an invoice shows a single CGST/SGST pair rather
// than a pair per rate.
func (r TaxRates) Uniform() bool {
	return r.Food == r.Service && r.Food == r.Delivery
}
