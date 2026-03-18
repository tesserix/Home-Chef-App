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

type AuthProvider string

const (
	ProviderEmail    AuthProvider = "email"
	ProviderGoogle   AuthProvider = "google"
	ProviderFacebook AuthProvider = "facebook"
	ProviderApple    AuthProvider = "apple"
)

type User struct {
	ID            uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email         string       `gorm:"uniqueIndex;not null" json:"email"`
	Password      string       `gorm:"" json:"-"`
	FirstName     string       `gorm:"not null" json:"firstName"`
	LastName      string       `gorm:"not null" json:"lastName"`
	Phone         string       `gorm:"" json:"phone"`
	Avatar        string       `gorm:"" json:"avatar"`
	Role          UserRole     `gorm:"type:varchar(20);default:'customer'" json:"role"`
	AuthProvider  AuthProvider `gorm:"type:varchar(20);default:'email'" json:"authProvider"`
	ProviderID    string       `gorm:"" json:"-"`
	IsActive      bool         `gorm:"default:true" json:"isActive"`
	EmailVerified bool         `gorm:"default:false" json:"emailVerified"`
	PhoneVerified bool         `gorm:"default:false" json:"phoneVerified"`
	LastLoginAt   *time.Time   `gorm:"" json:"lastLoginAt"`
	CreatedAt     time.Time    `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time    `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

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

type RefreshToken struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	Token     string     `gorm:"uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"not null" json:"expiresAt"`
	RevokedAt *time.Time `gorm:"" json:"revokedAt"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"createdAt"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

type Address struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	Label      string    `gorm:"" json:"label"` // Home, Work, etc.
	Line1      string    `gorm:"not null" json:"line1"`
	Line2      string    `gorm:"" json:"line2"`
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
	EmailVerified       bool      `json:"emailVerified"`
	OnboardingCompleted bool      `json:"onboardingCompleted"`
	CreatedAt           time.Time `json:"createdAt"`
}

func (u *User) ToResponse() UserResponse {
	resp := UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		Phone:         u.Phone,
		Avatar:        u.Avatar,
		Role:          u.Role,
		IsActive:      u.IsActive,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
	if u.CustomerProfile != nil {
		resp.OnboardingCompleted = u.CustomerProfile.OnboardingCompleted
	}
	return resp
}
