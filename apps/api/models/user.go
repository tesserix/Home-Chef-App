package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleCustomer     UserRole = "customer"
	RoleChef         UserRole = "chef"
	RoleDelivery     UserRole = "delivery"
	RoleAdmin        UserRole = "admin"
	RoleFleetManager UserRole = "fleet_manager"
)

// AuthPool identifies which Google Identity Platform tenant pool a user
// belongs to. Multiple users can share the same email across different
// pools (e.g., a chef who also orders as a customer).
type AuthPool string

const (
	PoolCustomer AuthPool = "customer"
	PoolBusiness AuthPool = "business"
	PoolInternal AuthPool = "internal"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"not null" json:"email"`
	FirstName string    `gorm:"not null" json:"firstName"`
	LastName  string    `gorm:"not null" json:"lastName"`
	Phone     string    `gorm:"" json:"phone"`

	// PII companions (#710 P1). Written alongside the plaintext columns above;
	// nothing reads them until P2. json:"-" so they never reach an API response.
	// text, not varchar — prefixed base64 ciphertext outgrows varchar(255).
	//
	// email_bidx is deliberately NOT uniqueIndex here: the live constraint is
	// partial and per-pool, UNIQUE (lower(email), auth_pool), and a plain gorm
	// uniqueIndex would be neither. Its blind-index replacement is raw DDL in
	// database.go alongside the original, added in P2 when reads switch over.
	EmailEnc     EncryptedString `gorm:"column:email_enc;type:text" json:"-"`
	EmailBidx    string          `gorm:"column:email_bidx;type:text;index" json:"-"`
	FirstNameEnc EncryptedString `gorm:"column:first_name_enc;type:text" json:"-"`
	LastNameEnc  EncryptedString `gorm:"column:last_name_enc;type:text" json:"-"`
	PhoneEnc     EncryptedString `gorm:"column:phone_enc;type:text" json:"-"`
	PhoneBidx    string          `gorm:"column:phone_bidx;type:text;index" json:"-"`
	Avatar    string    `gorm:"" json:"avatar"`
	Role      UserRole  `gorm:"type:varchar(20);default:'customer'" json:"role"`

	// GIP identity. Populated when apps/auth-bff upserts a user after a
	// successful Google Identity Platform sign-in. See migration
	// 20260514000002_add_gip_identity_to_users.up.sql.
	GIPUid      string   `gorm:"column:gip_uid;uniqueIndex" json:"gipUid,omitempty"`
	GIPTenantID string   `gorm:"column:gip_tenant_id" json:"gipTenantId,omitempty"`
	GIPProvider string   `gorm:"column:gip_provider" json:"gipProvider,omitempty"`
	AuthPool    AuthPool `gorm:"column:auth_pool;type:varchar(16)" json:"authPool,omitempty"`

	IsActive      bool   `gorm:"default:true" json:"isActive"`
	PhoneVerified bool   `gorm:"default:false" json:"phoneVerified"`
	FCMToken      string `gorm:"column:fcm_token" json:"-"`

	// DPDP §6 marketing-consent state. MarketingConsent records the user's
	// opt-in for promotional emails; MarketingConsentAt is the timestamp the
	// consent was granted. See migration 20260515000001 (CW-01b).
	MarketingConsent   bool       `gorm:"column:marketing_consent;not null;default:false" json:"marketingConsent"`
	MarketingConsentAt *time.Time `gorm:"column:marketing_consent_at" json:"marketingConsentAt,omitempty"`

	LastLoginAt *time.Time     `gorm:"column:last_login_at" json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	CustomerProfile *CustomerProfile `gorm:"foreignKey:UserID" json:"customerProfile,omitempty"`
	ChefProfile     *ChefProfile     `gorm:"foreignKey:UserID" json:"chefProfile,omitempty"`
	DeliveryPartner *DeliveryPartner `gorm:"foreignKey:UserID" json:"deliveryPartner,omitempty"`
	Addresses       []Address        `gorm:"foreignKey:UserID" json:"addresses,omitempty"`
	PaymentMethods  []PaymentMethod  `gorm:"foreignKey:UserID" json:"paymentMethods,omitempty"`
	Orders          []Order          `gorm:"foreignKey:CustomerID" json:"orders,omitempty"`
	Reviews         []Review         `gorm:"foreignKey:CustomerID" json:"reviews,omitempty"`
	Notifications   []Notification   `gorm:"foreignKey:UserID" json:"notifications,omitempty"`
}

// BeforeSave mirrors the plaintext PII columns into their encrypted and
// blind-index companions (#710 P1). Hook rather than per-call-site so a new
// handler that sets Email/Phone/name cannot forget it.
//
// Does not fire for map-based Updates — see models.PIIUpdates.
func (u *User) BeforeSave(*gorm.DB) error {
	u.EmailEnc = encOf(u.Email)
	u.EmailBidx = bidxOf(u.Email)
	u.FirstNameEnc = encOf(u.FirstName)
	u.LastNameEnc = encOf(u.LastName)
	u.PhoneEnc = encOf(u.Phone)
	u.PhoneBidx = bidxOf(u.Phone)
	return nil
}

type Address struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	Label      string    `gorm:"" json:"label"` // Home, Work, etc.
	Line1      string    `gorm:"not null" json:"line1"`
	Line2      string    `gorm:"" json:"line2"`
	// PII companions (#710 P1) — not searched, so ciphertext only.
	Line1Enc EncryptedString `gorm:"column:line1_enc;type:text" json:"-"`
	Line2Enc EncryptedString `gorm:"column:line2_enc;type:text" json:"-"`
	City       string    `gorm:"not null" json:"city"`
	State      string    `gorm:"not null" json:"state"`
	PostalCode string    `gorm:"not null" json:"postalCode"`
	Country    string    `gorm:"default:'US'" json:"country"`
	Latitude   float64   `gorm:"" json:"latitude"`
	Longitude  float64   `gorm:"" json:"longitude"`
	IsDefault  bool      `gorm:"default:false" json:"isDefault"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

// BeforeCreate mints the UUID in Go so the model works without the Postgres
// gen_random_uuid() default (e.g. in sqlite-backed unit tests).
func (a *Address) BeforeCreate(*gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// BeforeSave mirrors the address lines into their encrypted companions (#710 P1).
func (a *Address) BeforeSave(*gorm.DB) error {
	a.Line1Enc = encOf(a.Line1)
	a.Line2Enc = encOf(a.Line2)
	return nil
}

type PaymentMethod struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	StripePaymentID string    `gorm:"not null" json:"-"`
	Type            string    `gorm:"not null" json:"type"` // card, bank_account
	Last4           string    `gorm:"" json:"last4"`
	Brand           string    `gorm:"" json:"brand"`
	ExpMonth        int       `gorm:"" json:"expMonth"`
	ExpYear         int       `gorm:"" json:"expYear"`
	IsDefault       bool      `gorm:"default:false" json:"isDefault"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"createdAt"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

// DTOs for API responses
type UserResponse struct {
	ID                  uuid.UUID `json:"id"`
	Email               string    `json:"email"`
	FirstName           string    `json:"firstName"`
	LastName            string    `json:"lastName"`
	Phone               string    `json:"phone,omitempty"`
	Avatar              string    `json:"avatar,omitempty"`
	Role                UserRole  `json:"role"`
	IsActive            bool      `json:"isActive"`
	OnboardingCompleted bool      `json:"onboardingCompleted"`
	CreatedAt           time.Time `json:"createdAt"`
}

func (u *User) ToResponse() UserResponse {
	resp := UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Phone:     u.Phone,
		Avatar:    u.Avatar,
		Role:      u.Role,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
	if u.CustomerProfile != nil {
		resp.OnboardingCompleted = u.CustomerProfile.OnboardingCompleted
	}
	return resp
}
