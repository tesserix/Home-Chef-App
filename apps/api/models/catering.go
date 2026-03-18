package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type CateringRequestStatus string

const (
	CateringStatusOpen       CateringRequestStatus = "open"
	CateringStatusQuoted     CateringRequestStatus = "quoted"
	CateringStatusAccepted   CateringRequestStatus = "accepted"
	CateringStatusConfirmed  CateringRequestStatus = "confirmed"
	CateringStatusCompleted  CateringRequestStatus = "completed"
	CateringStatusCancelled  CateringRequestStatus = "cancelled"
)

type CateringQuoteStatus string

const (
	QuoteStatusPending  CateringQuoteStatus = "pending"
	QuoteStatusAccepted CateringQuoteStatus = "accepted"
	QuoteStatusRejected CateringQuoteStatus = "rejected"
	QuoteStatusExpired  CateringQuoteStatus = "expired"
)

type CateringRequest struct {
	ID            uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID    uuid.UUID             `gorm:"type:uuid;not null;index" json:"customerId"`
	Status        CateringRequestStatus `gorm:"type:varchar(20);default:'open'" json:"status"`

	// Event Details
	EventType     string    `gorm:"not null" json:"eventType"` // wedding, corporate, birthday, etc.
	EventDate     time.Time `gorm:"not null" json:"eventDate"`
	EventTime     string    `gorm:"" json:"eventTime"`
	GuestCount    int       `gorm:"not null" json:"guestCount"`
	Budget        float64   `gorm:"" json:"budget"`

	// Menu Preferences
	CuisineTypes  pq.StringArray `gorm:"type:text[]" json:"cuisineTypes"`
	DietaryNeeds  pq.StringArray `gorm:"type:text[]" json:"dietaryNeeds"`
	MenuStyle     string         `gorm:"" json:"menuStyle"` // buffet, plated, stations
	Description   string         `gorm:"type:text" json:"description"`

	// Location
	VenueName     string  `gorm:"" json:"venueName"`
	AddressLine1  string  `gorm:"" json:"addressLine1"`
	AddressLine2  string  `gorm:"" json:"addressLine2"`
	City          string  `gorm:"" json:"city"`
	State         string  `gorm:"" json:"state"`
	PostalCode    string  `gorm:"" json:"postalCode"`
	Latitude      float64 `gorm:"" json:"latitude"`
	Longitude     float64 `gorm:"" json:"longitude"`

	// Contact
	ContactName   string `gorm:"" json:"contactName"`
	ContactPhone  string `gorm:"" json:"contactPhone"`
	ContactEmail  string `gorm:"" json:"contactEmail"`

	// Deadline for quotes
	QuoteDeadline *time.Time `gorm:"" json:"quoteDeadline,omitempty"`

	// Selected Quote
	AcceptedQuoteID *uuid.UUID `gorm:"type:uuid" json:"acceptedQuoteId,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Customer      User            `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Quotes        []CateringQuote `gorm:"foreignKey:RequestID" json:"quotes,omitempty"`
	AcceptedQuote *CateringQuote  `gorm:"foreignKey:AcceptedQuoteID;references:ID" json:"acceptedQuote,omitempty"`
}

type CateringQuote struct {
	ID        uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	RequestID uuid.UUID           `gorm:"type:uuid;not null;index" json:"requestId"`
	ChefID    uuid.UUID           `gorm:"type:uuid;not null;index" json:"chefId"`
	Status    CateringQuoteStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	// Quote Details
	ProposedMenu   string         `gorm:"type:text" json:"proposedMenu"`
	MenuItems      pq.StringArray `gorm:"type:text[]" json:"menuItems"`
	PricePerPerson float64        `gorm:"not null" json:"pricePerPerson"`
	TotalPrice     float64        `gorm:"not null" json:"totalPrice"`
	Notes          string         `gorm:"type:text" json:"notes"`

	// Includes
	IncludesSetup     bool `gorm:"default:false" json:"includesSetup"`
	IncludesServing   bool `gorm:"default:false" json:"includesServing"`
	IncludesCleanup   bool `gorm:"default:false" json:"includesCleanup"`
	IncludesEquipment bool `gorm:"default:false" json:"includesEquipment"`

	// Validity
	ValidUntil *time.Time `gorm:"" json:"validUntil,omitempty"`

	// Response timestamps
	AcceptedAt  *time.Time `gorm:"" json:"acceptedAt,omitempty"`
	RejectedAt  *time.Time `gorm:"" json:"rejectedAt,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	// Relationships
	Request CateringRequest `gorm:"foreignKey:RequestID" json:"-"`
	Chef    ChefProfile     `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
}

// DTOs
type CateringRequestResponse struct {
	ID           uuid.UUID             `json:"id"`
	Status       CateringRequestStatus `json:"status"`
	EventType    string                `json:"eventType"`
	EventDate    time.Time             `json:"eventDate"`
	EventTime    string                `json:"eventTime"`
	GuestCount   int                   `json:"guestCount"`
	Budget       float64               `json:"budget"`
	CuisineTypes []string              `json:"cuisineTypes"`
	DietaryNeeds []string              `json:"dietaryNeeds"`
	MenuStyle    string                `json:"menuStyle"`
	Description  string                `json:"description"`
	VenueName    string                `json:"venueName"`
	City         string                `json:"city"`
	State        string                `json:"state"`
	QuotesCount  int                   `json:"quotesCount"`
	CreatedAt    time.Time             `json:"createdAt"`
}

type CateringQuoteResponse struct {
	ID                uuid.UUID           `json:"id"`
	RequestID         uuid.UUID           `json:"requestId"`
	ChefID            uuid.UUID           `json:"chefId"`
	Chef              ChefQuoteInfo       `json:"chef"`
	Status            CateringQuoteStatus `json:"status"`
	ProposedMenu      string              `json:"proposedMenu"`
	PricePerPerson    float64             `json:"pricePerPerson"`
	TotalPrice        float64             `json:"totalPrice"`
	Notes             string              `json:"notes"`
	IncludesSetup     bool                `json:"includesSetup"`
	IncludesServing   bool                `json:"includesServing"`
	IncludesCleanup   bool                `json:"includesCleanup"`
	IncludesEquipment bool                `json:"includesEquipment"`
	ValidUntil        *time.Time          `json:"validUntil,omitempty"`
	CreatedAt         time.Time           `json:"createdAt"`
}

type ChefQuoteInfo struct {
	ID           uuid.UUID `json:"id"`
	BusinessName string    `json:"businessName"`
	ProfileImage string    `json:"profileImage"`
	Rating       float64   `json:"rating"`
	TotalReviews int       `json:"totalReviews"`
	IsVerified   bool      `json:"verified"`
}

func (r *CateringRequest) ToResponse() CateringRequestResponse {
	cuisines := []string{}
	if r.CuisineTypes != nil {
		cuisines = r.CuisineTypes
	}
	dietary := []string{}
	if r.DietaryNeeds != nil {
		dietary = r.DietaryNeeds
	}

	return CateringRequestResponse{
		ID:           r.ID,
		Status:       r.Status,
		EventType:    r.EventType,
		EventDate:    r.EventDate,
		EventTime:    r.EventTime,
		GuestCount:   r.GuestCount,
		Budget:       r.Budget,
		CuisineTypes: cuisines,
		DietaryNeeds: dietary,
		MenuStyle:    r.MenuStyle,
		Description:  r.Description,
		VenueName:    r.VenueName,
		City:         r.City,
		State:        r.State,
		QuotesCount:  len(r.Quotes),
		CreatedAt:    r.CreatedAt,
	}
}

func (q *CateringQuote) ToResponse() CateringQuoteResponse {
	response := CateringQuoteResponse{
		ID:                q.ID,
		RequestID:         q.RequestID,
		ChefID:            q.ChefID,
		Status:            q.Status,
		ProposedMenu:      q.ProposedMenu,
		PricePerPerson:    q.PricePerPerson,
		TotalPrice:        q.TotalPrice,
		Notes:             q.Notes,
		IncludesSetup:     q.IncludesSetup,
		IncludesServing:   q.IncludesServing,
		IncludesCleanup:   q.IncludesCleanup,
		IncludesEquipment: q.IncludesEquipment,
		ValidUntil:        q.ValidUntil,
		CreatedAt:         q.CreatedAt,
	}

	if q.Chef.ID != uuid.Nil {
		response.Chef = ChefQuoteInfo{
			ID:           q.Chef.ID,
			BusinessName: q.Chef.BusinessName,
			ProfileImage: q.Chef.ProfileImage,
			Rating:       q.Chef.Rating,
			TotalReviews: q.Chef.TotalReviews,
			IsVerified:   q.Chef.IsVerified,
		}
	}

	return response
}
