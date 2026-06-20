package models

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type ChefProfile struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	BusinessName string    `gorm:"uniqueIndex;not null" json:"businessName"`
	// Slug is the URL-safe identifier for SEO chef pages + app universal links
	// (#58). Derived from BusinessName by BeforeSave; resolvable via GetChef.
	Slug           string         `gorm:"index" json:"slug"`
	Description    string         `gorm:"type:text" json:"description"`
	ProfileImage   string         `gorm:"" json:"profileImage"`
	BannerImage    string         `gorm:"" json:"bannerImage"`
	Cuisines       pq.StringArray `gorm:"type:text[]" json:"cuisines"`
	Specialties    pq.StringArray `gorm:"type:text[]" json:"specialties"`
	PrepTime       string         `gorm:"" json:"prepTime"` // e.g., "30-45 min"
	MinimumOrder   float64        `gorm:"default:0" json:"minimumOrder"`
	DeliveryRadius float64        `gorm:"default:10" json:"deliveryRadius"` // in km
	ServiceRadius  float64        `gorm:"default:10" json:"serviceRadius"`  // in km
	Rating         float64        `gorm:"default:0" json:"rating"`
	TotalReviews   int            `gorm:"default:0" json:"totalReviews"`
	TotalOrders    int            `gorm:"default:0" json:"totalOrders"`
	// IssueCount is the number of customer-reported order issues (#37); the issue
	// rate (issues/orders) feeds the chef's quality signal.
	IssueCount      int        `gorm:"default:0" json:"issueCount"`
	IsVerified      bool       `gorm:"default:false" json:"verified"`
	VerifiedAt      *time.Time `gorm:"" json:"verifiedAt"`
	IsActive        bool       `gorm:"default:true" json:"isActive"`
	AcceptingOrders bool       `gorm:"default:true" json:"acceptingOrders"`
	// PausedUntil powers "Back in {15,30,60} min": when set in the future the
	// kitchen is temporarily closed (AcceptingOrders is flipped false alongside
	// it). The auto-resume cron clears it + reopens once the time passes.
	PausedUntil   *time.Time     `gorm:"" json:"pausedUntil,omitempty"`
	KitchenPhotos pq.StringArray `gorm:"type:text[]" json:"kitchenPhotos"`

	// KitchenType is always home_kitchen. Fe3dr is exclusively for individual
	// home chefs cooking from their own home — it does NOT onboard commercial
	// vendors (cloud kitchens, shared/commercial kitchens, restaurants). The
	// column is kept queryable so admin tooling can audit it and the
	// onboarding-approval gate can hard-block anything that isn't a home kitchen
	// (defense in depth). Defaults to home_kitchen so chefs created before the
	// column existed read as home.
	KitchenType string `gorm:"type:varchar(20);default:'home_kitchen'" json:"kitchenType"`

	// Address
	AddressLine1 string  `gorm:"" json:"addressLine1"`
	AddressLine2 string  `gorm:"" json:"addressLine2"`
	City         string  `gorm:"" json:"city"`
	State        string  `gorm:"" json:"state"`
	PostalCode   string  `gorm:"" json:"postalCode"`
	Latitude     float64 `gorm:"" json:"latitude"`
	Longitude    float64 `gorm:"" json:"longitude"`

	// Featured/Promoted
	IsFeatured    bool       `gorm:"default:false" json:"isFeatured"`
	FeaturedUntil *time.Time `gorm:"" json:"featuredUntil,omitempty"`

	// Payment gateway linked accounts
	StripeAccountID   string `gorm:"" json:"-"`
	RazorpayAccountID string `gorm:"" json:"-"` // Razorpay Route linked account ID
	// PaymentProvider picks which gateway a customer's order gets routed
	// through. "razorpay" (India) or "stripe" (international). Defaults to
	// razorpay so existing chefs keep working after the column is added.
	PaymentProvider string `gorm:"type:varchar(20);default:'razorpay'" json:"paymentProvider"`
	// PayoutCountry is the ISO-3166 alpha-2 country for Stripe Connect
	// onboarding (US, GB, AE, …). Unused for Razorpay chefs.
	PayoutCountry string `gorm:"type:varchar(2);default:'IN'" json:"payoutCountry"`
	// StripeChargesEnabled / StripePayoutsEnabled mirror the Connect
	// account flags — we cache them here so the UI doesn't have to round-
	// trip Stripe on every page view. Refreshed on webhook + on demand.
	StripeChargesEnabled bool `gorm:"default:false" json:"stripeChargesEnabled"`
	StripePayoutsEnabled bool `gorm:"default:false" json:"stripePayoutsEnabled"`

	// Payout details
	PayoutMethod      string `gorm:"default:''" json:"-"`
	BankAccountNumber string `gorm:"default:''" json:"-"`
	BankIFSC          string `gorm:"default:''" json:"-"`
	BankAccountName   string `gorm:"default:''" json:"-"`
	UpiID             string `gorm:"default:''" json:"-"`

	// Regulatory IDs. Stored as plain strings (not -) because admin
	// tooling + the customer-facing invoice (Wave 3) need to render
	// them. PAN remains internal-only (`json:"-"`) per existing
	// pattern; FSSAI license number is intentionally exposed because
	// the invoice prints it for the customer's records.
	//
	// FSSAILicenseNumber: 14-digit numeric Food Safety license id.
	// Validated client-side on submit; backend just stores the value
	// so admins can pivot the field for FoSCoS API verification later.
	PanNumber          string `gorm:"type:varchar(10)" json:"-"`
	FSSAILicenseNumber string `gorm:"type:varchar(14)" json:"fssaiLicenseNumber,omitempty"`
	// GSTIN — 15-character Goods & Services Tax Identification Number.
	// Optional (chefs below the threshold of ~₹20L turnover don't need
	// one). When set, printed on customer invoices alongside the FSSAI
	// number, and the chef can claim input tax credit. Stored as
	// varchar(15) without format enforcement at the DB level —
	// validation lives in the handler so future format changes don't
	// require a migration.
	GSTIN string `gorm:"type:varchar(15)" json:"gstin,omitempty"`

	// FSSAI lockout override (#93). For genuine edge cases — e.g. a government
	// renewal backlog where a chef's paperwork is filed but not yet processed —
	// an admin can grant a time-boxed, reason-logged reprieve from the FSSAI
	// expiry lockout. While FSSAIOverrideUntil is in the future the lockout is
	// suspended; once it passes the expiry gate re-applies automatically with no
	// cleanup. Every grant/clear is written to the audit log for compliance
	// evidence. Reason + granting admin are retained for that paper trail.
	FSSAIOverrideUntil  *time.Time `gorm:"index" json:"fssaiOverrideUntil,omitempty"`
	FSSAIOverrideReason string     `gorm:"type:text" json:"fssaiOverrideReason,omitempty"`
	FSSAIOverrideBy     *uuid.UUID `gorm:"type:uuid" json:"fssaiOverrideBy,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	// Relationships
	User      User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	MenuItems []MenuItem     `gorm:"foreignKey:ChefID" json:"menuItems,omitempty"`
	Schedules []ChefSchedule `gorm:"foreignKey:ChefID" json:"schedules,omitempty"`
	Orders    []Order        `gorm:"foreignKey:ChefID" json:"orders,omitempty"`
	Reviews   []Review       `gorm:"foreignKey:ChefID" json:"reviews,omitempty"`
	Posts     []Post         `gorm:"foreignKey:ChefID" json:"posts,omitempty"`
}

type ChefSchedule struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID    uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	DayOfWeek int       `gorm:"not null" json:"dayOfWeek"` // 0-6, Sunday-Saturday
	OpenTime  string    `gorm:"" json:"openTime"`          // HH:MM format
	CloseTime string    `gorm:"" json:"closeTime"`         // HH:MM format
	IsClosed  bool      `gorm:"default:false" json:"isClosed"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

// ChefSettings stores vendor preferences (notifications, auto-accept, etc.)
type ChefSettings struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID              uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"chefId"`
	AutoAcceptOrders    bool      `gorm:"default:false" json:"autoAcceptOrders"`
	AutoAcceptThreshold float64   `gorm:"default:0" json:"autoAcceptThreshold"`
	PushNewOrder        bool      `gorm:"default:true" json:"pushNewOrder"`
	PushOrderUpdate     bool      `gorm:"default:true" json:"pushOrderUpdate"`
	EmailDailySummary   bool      `gorm:"default:true" json:"emailDailySummary"`
	EmailWeeklyReport   bool      `gorm:"default:true" json:"emailWeeklyReport"`
	SmsNewOrder         bool      `gorm:"default:false" json:"smsNewOrder"`
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

// slugApostrophes are stripped (not dashed) so "amma's" → "ammas"; slugNonWord
// collapses every other run of non-alphanumerics to a single dash.
var (
	slugApostrophes = regexp.MustCompile(`['’]`)
	slugNonWord     = regexp.MustCompile(`[^a-z0-9]+`)
)

// ChefSlug builds a URL-safe slug from a chef's business name for SEO landing
// pages and app universal links (#58). e.g. "Amma's Kitchen!" → "ammas-kitchen".
func ChefSlug(businessName string) string {
	s := strings.ToLower(strings.TrimSpace(businessName))
	s = slugApostrophes.ReplaceAllString(s, "")
	s = slugNonWord.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// EffectiveSlug returns the stored slug, falling back to one derived from the
// business name so API responses always carry a slug even before backfill.
func (c *ChefProfile) EffectiveSlug() string {
	if c.Slug != "" {
		return c.Slug
	}
	return ChefSlug(c.BusinessName)
}

// BeforeSave stamps a slug on create/update when one isn't set, so SEO pages and
// app universal links can resolve a chef by slug (#58). Stored (not derived on
// read) so the lookup is a simple indexed equality. Collisions are rare
// (BusinessName is unique) and resolve to the oldest chef; uniqueness hardening
// is a follow-up.
func (c *ChefProfile) BeforeSave(*gorm.DB) error {
	if c.Slug == "" && c.BusinessName != "" {
		c.Slug = ChefSlug(c.BusinessName)
	}
	return nil
}

// KitchenTypeHome is the only kitchen type the platform accepts — an individual
// home kitchen. Commercial kitchen types (cloud/shared) are intentionally not
// defined as constants: Fe3dr onboards home chefs only.
const KitchenTypeHome = "home_kitchen"

// IsHomeKitchen reports whether this profile is an individual home kitchen — the
// only kind Fe3dr accepts. A blank value (rows created before the column existed)
// is treated as a home kitchen for backward compatibility.
func (c *ChefProfile) IsHomeKitchen() bool {
	return c.KitchenType == "" || c.KitchenType == KitchenTypeHome
}

// DTOs
type ChefProfileResponse struct {
	ID            uuid.UUID `json:"id"`
	UserID        uuid.UUID `json:"userId"`
	BusinessName  string    `json:"businessName"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	ProfileImage  string    `json:"profileImage"`
	BannerImage   string    `json:"bannerImage"`
	Cuisines      []string  `json:"cuisines"`
	Specialties   []string  `json:"specialties"`
	PrepTime      string    `json:"prepTime"`
	MinimumOrder  float64   `json:"minimumOrder"`
	DeliveryFee   float64   `json:"deliveryFee"`
	PriceRange    string    `json:"priceRange"`
	ServiceRadius float64   `json:"serviceRadius"`
	Rating        float64   `json:"rating"`
	TotalReviews  int       `json:"totalReviews"`
	TotalOrders   int       `json:"totalOrders"`
	IsVerified    bool      `json:"verified"`
	// FoodSafetyBadge: chef holds a verified, non-expired FSSAI licence (#35).
	// Set by the handler (needs a DB lookup), so it's false on the bare model.
	FoodSafetyBadge bool `json:"foodSafetyBadge"`
	// ProBadge: chef has an active premium subscription — the Verified-Pro badge
	// (#44). Like FoodSafetyBadge it needs a DB lookup, so the handler populates it.
	ProBadge        bool                   `json:"proBadge"`
	IsFeatured      bool                   `json:"isFeatured"`
	IsOnline        bool                   `json:"isOnline"`
	AcceptingOrders bool                   `json:"acceptingOrders"`
	PausedUntil     *time.Time             `json:"pausedUntil,omitempty"`
	KitchenPhotos   []string               `json:"kitchenPhotos"`
	KitchenType     string                 `json:"kitchenType"`
	City            string                 `json:"city"`
	State           string                 `json:"state"`
	Country         string                 `json:"country"`  // chef's PayoutCountry (ISO alpha-2)
	Currency        string                 `json:"currency"` // ISO-4217, derived from country
	Latitude        float64                `json:"latitude"`
	Longitude       float64                `json:"longitude"`
	OperatingHours  map[string]interface{} `json:"operatingHours,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
}

// priceRangeFromMinOrder derives a display-friendly price range string.
func priceRangeFromMinOrder(min float64) string {
	if min >= 500 {
		return "$$$$"
	}
	if min >= 200 {
		return "$$$"
	}
	if min >= 100 {
		return "$$"
	}
	return "$"
}

func (c *ChefProfile) ToResponse() ChefProfileResponse {
	cuisines := []string{}
	if c.Cuisines != nil {
		cuisines = c.Cuisines
	}
	specialties := []string{}
	if c.Specialties != nil {
		specialties = c.Specialties
	}

	kitchenPhotos := []string{}
	if c.KitchenPhotos != nil {
		kitchenPhotos = c.KitchenPhotos
	}

	// Always render a kitchen type; blank legacy rows are home kitchens.
	kitchenType := c.KitchenType
	if kitchenType == "" {
		kitchenType = KitchenTypeHome
	}

	country := c.PayoutCountry
	if country == "" {
		country = "IN"
	}
	// currencyForCountryLocal keeps this model file free of service-layer
	// imports (avoids a circular dep). The full map lives in
	// services.CurrencyForCountry and should stay in sync — do NOT diverge.
	currency := strings.ToUpper(currencyForCountryLocal(country))

	return ChefProfileResponse{
		ID:              c.ID,
		UserID:          c.UserID,
		BusinessName:    c.BusinessName,
		Slug:            c.EffectiveSlug(),
		Description:     c.Description,
		ProfileImage:    c.ProfileImage,
		BannerImage:     c.BannerImage,
		Cuisines:        cuisines,
		Specialties:     specialties,
		PrepTime:        c.PrepTime,
		MinimumOrder:    c.MinimumOrder,
		DeliveryFee:     0, // TODO: populate when delivery fee model is added
		PriceRange:      priceRangeFromMinOrder(c.MinimumOrder),
		ServiceRadius:   c.ServiceRadius,
		Rating:          c.Rating,
		TotalReviews:    c.TotalReviews,
		TotalOrders:     c.TotalOrders,
		IsVerified:      c.IsVerified,
		IsFeatured:      c.IsFeatured && c.FeaturedUntil != nil && c.FeaturedUntil.After(time.Now()),
		IsOnline:        c.AcceptingOrders,
		AcceptingOrders: c.AcceptingOrders,
		PausedUntil:     c.PausedUntil,
		KitchenPhotos:   kitchenPhotos,
		KitchenType:     kitchenType,
		City:            c.City,
		State:           c.State,
		Country:         country,
		Currency:        currency,
		Latitude:        c.Latitude,
		Longitude:       c.Longitude,
		CreatedAt:       c.CreatedAt,
	}
}

// currencyForCountryLocal mirrors services.CurrencyForCountry. It exists
// here only so the model's ToResponse can derive a currency without
// importing the services package (which would create a cycle —
// services imports models).
func currencyForCountryLocal(country string) string {
	switch strings.ToUpper(country) {
	case "US":
		return "usd"
	case "GB":
		return "gbp"
	case "CA":
		return "cad"
	case "AU":
		return "aud"
	case "NZ":
		return "nzd"
	case "SG":
		return "sgd"
	case "HK":
		return "hkd"
	case "AE":
		return "aed"
	case "JP":
		return "jpy"
	case "KR":
		return "krw"
	case "MY":
		return "myr"
	case "TH":
		return "thb"
	case "CH":
		return "chf"
	case "SE":
		return "sek"
	case "NO":
		return "nok"
	case "DK":
		return "dkk"
	case "DE", "FR", "IT", "ES", "NL", "BE", "AT", "PT", "IE", "FI", "GR", "LU", "SK", "SI", "EE", "LV", "LT", "CY", "MT":
		return "eur"
	case "IN", "":
		return "inr"
	default:
		return "inr"
	}
}

// ToPublicResponse builds a response that includes operating hours from schedules.
func (c *ChefProfile) ToPublicResponse(schedules []ChefSchedule) ChefProfileResponse {
	resp := c.ToResponse()

	dayNames := map[int]string{
		0: "sunday", 1: "monday", 2: "tuesday", 3: "wednesday",
		4: "thursday", 5: "friday", 6: "saturday",
	}

	operatingHours := make(map[string]interface{})
	for _, s := range schedules {
		name, ok := dayNames[s.DayOfWeek]
		if !ok || s.IsClosed {
			continue
		}
		operatingHours[name] = map[string]string{
			"open":  s.OpenTime,
			"close": s.CloseTime,
		}
	}
	resp.OperatingHours = operatingHours
	return resp
}
