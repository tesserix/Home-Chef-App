package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type CustomerProfile struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID              uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	DateOfBirth         *time.Time     `gorm:"" json:"dateOfBirth"`
	DietaryPreferences  pq.StringArray `gorm:"type:text[]" json:"dietaryPreferences"`
	FoodAllergies       pq.StringArray `gorm:"type:text[]" json:"foodAllergies"`
	CuisinePreferences  pq.StringArray `gorm:"type:text[]" json:"cuisinePreferences"`
	SpiceTolerance      string         `gorm:"type:varchar(20)" json:"spiceTolerance"`
	HouseholdSize       string         `gorm:"type:varchar(10)" json:"householdSize"`
	OnboardingCompleted bool           `gorm:"default:false" json:"onboardingCompleted"`
	OnboardingStep      int            `gorm:"default:0" json:"onboardingStep"`
	PreferredCurrency   string         `gorm:"type:varchar(3);default:'INR'" json:"preferredCurrency"`
	CreatedAt           time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt           time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

type CustomerProfileResponse struct {
	ID                  uuid.UUID  `json:"id"`
	UserID              uuid.UUID  `json:"userId"`
	FirstName           string     `json:"firstName"`
	LastName            string     `json:"lastName"`
	Email               string     `json:"email"`
	Phone               string     `json:"phone,omitempty"`
	Avatar              string     `json:"avatar,omitempty"`
	DateOfBirth         *time.Time `json:"dateOfBirth,omitempty"`
	DietaryPreferences  []string   `json:"dietaryPreferences"`
	FoodAllergies       []string   `json:"foodAllergies"`
	CuisinePreferences  []string   `json:"cuisinePreferences"`
	SpiceTolerance      string     `json:"spiceTolerance"`
	HouseholdSize       string     `json:"householdSize"`
	OnboardingCompleted bool         `json:"onboardingCompleted"`
	OnboardingStep      int          `json:"onboardingStep"`
	PreferredCurrency   string       `json:"preferredCurrency"`
	AuthProvider        AuthProvider `json:"authProvider"`
}

func (cp *CustomerProfile) ToResponse(user *User) CustomerProfileResponse {
	dietary := make([]string, 0)
	if cp.DietaryPreferences != nil {
		dietary = cp.DietaryPreferences
	}
	allergies := make([]string, 0)
	if cp.FoodAllergies != nil {
		allergies = cp.FoodAllergies
	}
	cuisines := make([]string, 0)
	if cp.CuisinePreferences != nil {
		cuisines = cp.CuisinePreferences
	}

	return CustomerProfileResponse{
		ID:                  cp.ID,
		UserID:              cp.UserID,
		FirstName:           user.FirstName,
		LastName:            user.LastName,
		Email:               user.Email,
		Phone:               user.Phone,
		Avatar:              user.Avatar,
		DateOfBirth:         cp.DateOfBirth,
		DietaryPreferences:  dietary,
		FoodAllergies:       allergies,
		CuisinePreferences:  cuisines,
		SpiceTolerance:      cp.SpiceTolerance,
		HouseholdSize:       cp.HouseholdSize,
		OnboardingCompleted: cp.OnboardingCompleted,
		OnboardingStep:      cp.OnboardingStep,
		PreferredCurrency:   cp.PreferredCurrency,
		AuthProvider:        user.AuthProvider,
	}
}
