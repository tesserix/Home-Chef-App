package models

import (
	"time"

	"github.com/google/uuid"
)

// TaxRate is the ONE place a tax rate comes from. Order pricing, the checkout
// quote, meal plans, subscription billing, the stored invoice and the admin
// simulator all resolve through it, so none of them can drift from the others.
//
// A row with region="" is the country-wide fallback; a row with a specific region
// wins over it. Rows with is_active=false are excluded from lookup so admins can
// disable a rule without deleting history.
//
// Rates are per SUPPLY, not per order, because one bill can carry several: in
// India the food is restaurant service (5%, no ITC) while the platform's own fee
// is a standard-rated service, and delivery depends on WHO carries it. See
// docs/india-gst-model.md and docs/gst-refunds-and-gateway-costs.md.
type TaxRate struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CountryCode string    `gorm:"type:varchar(2);not null;index:idx_tax_lookup" json:"countryCode"`
	Region      string    `gorm:"type:varchar(10);default:'';index:idx_tax_lookup" json:"region"`
	// Human-readable name shown on invoices ("GST", "VAT", "Sales Tax").
	TaxName string `gorm:"type:varchar(40);not null" json:"taxName"`
	// Rate is the jurisdiction default, applied to any supply with no override.
	Rate float64 `gorm:"not null" json:"rate"`

	// Per-supply overrides. Leave one at 0 and that supply takes Rate.
	FoodPercent    float64 `gorm:"default:0" json:"foodPercent"`
	ServicePercent float64 `gorm:"default:0" json:"servicePercent"`
	// DeliverySelfPercent applies when the CHEF carries their own food: door
	// delivery by the person who cooked it is arguable as part of the composite
	// restaurant supply, taxed at the food rate.
	DeliverySelfPercent float64 `gorm:"default:0" json:"deliverySelfPercent"`
	// DeliveryPlatformPercent applies when the PLATFORM arranges a rider. Local
	// delivery through an e-commerce operator by a supplier not liable for
	// registration became a notified §9(5) supply at 18% on 22 September 2025
	// (56th GST Council, 3 Sep 2025).
	DeliveryPlatformPercent float64 `gorm:"default:0" json:"deliveryPlatformPercent"`
	SubscriptionPercent     float64 `gorm:"default:0" json:"subscriptionPercent"`

	// Inclusive=true means the quoted price already contains the tax (European
	// VAT); false means it is added on top (Indian GST on the food).
	Inclusive bool `gorm:"default:false" json:"inclusive"`
	// ServiceInclusive is the platform fee's own inclusivity, separate because the
	// fee can be quoted all-in while the food is quoted net. That is the whole of
	// the "Option B" pricing decision: the customer's fee does not move, and the
	// GST is backed out of it rather than added to it.
	ServiceInclusive bool `gorm:"default:false" json:"serviceInclusive"`

	// Registration identity for this jurisdiction, printed on the invoice: what
	// the number is called ("GSTIN", "ABN", "TIN") and the operator's own number
	// when it differs from the global company one.
	RegistrationIDLabel string `gorm:"type:varchar(20);default:''" json:"registrationIdLabel,omitempty"`
	CompanyTaxID        string `gorm:"type:varchar(40);default:''" json:"companyTaxId,omitempty"`
	// Notes: free-form, shown to admins only.
	Notes     string    `gorm:"type:text" json:"notes,omitempty"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName pins the Postgres table — GORM would otherwise pluralize to
// "tax_rates" which is what we want, but make it explicit for clarity.
func (TaxRate) TableName() string { return "tax_rates" }

// TaxRates is a resolved rule: what each supply on ONE order is taxed at, with
// the delivery leg already resolved to the carrier that will actually perform it.
type TaxRates struct {
	Name string `json:"name"`

	Food             float64 `json:"food"`
	FoodInclusive    bool    `json:"foodInclusive"`
	Service          float64 `json:"service"`
	ServiceInclusive bool    `json:"serviceInclusive"`
	Delivery         float64 `json:"delivery"`
	// DeliveryByPlatform records WHICH delivery rate was chosen, so the order can
	// freeze the reasoning and not just the number.
	DeliveryByPlatform bool    `json:"deliveryByPlatform"`
	Subscription       float64 `json:"subscription"`
}

// ComponentRates resolves the rates for one order. deliveryByPlatform selects
// between the two delivery rates — see DeliverySelfPercent / DeliveryPlatformPercent.
func (t *TaxRate) ComponentRates(deliveryByPlatform bool) TaxRates {
	if t == nil {
		return TaxRates{}
	}
	or := func(override float64) float64 {
		if override > 0 {
			return override
		}
		return t.Rate
	}
	delivery := or(t.DeliverySelfPercent)
	if deliveryByPlatform {
		delivery = or(t.DeliveryPlatformPercent)
	}
	return TaxRates{
		Name:               t.TaxName,
		Food:               or(t.FoodPercent),
		FoodInclusive:      t.Inclusive,
		Service:            or(t.ServicePercent),
		ServiceInclusive:   t.ServiceInclusive,
		Delivery:           delivery,
		DeliveryByPlatform: deliveryByPlatform,
		Subscription:       or(t.SubscriptionPercent),
	}
}
