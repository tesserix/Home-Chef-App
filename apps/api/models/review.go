package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Review struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"orderId"`
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customerId"`
	ChefID     uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`

	// Ratings (1-5 stars)
	OverallRating  int `gorm:"not null" json:"overallRating"`
	FoodRating     int `gorm:"" json:"foodRating"`
	DeliveryRating int `gorm:"" json:"deliveryRating"`
	ValueRating    int `gorm:"" json:"valueRating"`
	// Ratings 2.0 sub-scores (#35): packaging + hygiene, 1-5, optional.
	PackagingRating int `gorm:"" json:"packagingRating"`
	HygieneRating   int `gorm:"" json:"hygieneRating"`

	// Content
	Title   string         `gorm:"" json:"title"`
	Comment string         `gorm:"type:text" json:"comment"`
	Images  pq.StringArray `gorm:"type:text[]" json:"images"`

	// Moderation
	IsApproved   bool   `gorm:"default:true" json:"isApproved"`
	IsHidden     bool   `gorm:"default:false" json:"isHidden"`
	HiddenReason string `gorm:"" json:"hiddenReason,omitempty"`

	// Response
	ChefResponse    string     `gorm:"type:text" json:"chefResponse,omitempty"`
	ChefRespondedAt *time.Time `gorm:"" json:"chefRespondedAt,omitempty"`

	// Helpful votes
	HelpfulCount int `gorm:"default:0" json:"helpfulCount"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Order    Order       `gorm:"foreignKey:OrderID" json:"-"`
	Customer User        `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Chef     ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

// DTOs
type ReviewResponse struct {
	ID              uuid.UUID  `json:"id"`
	OrderID         uuid.UUID  `json:"orderId"`
	OverallRating   int        `json:"overallRating"`
	FoodRating      int        `json:"foodRating"`
	DeliveryRating  int        `json:"deliveryRating"`
	ValueRating     int        `json:"valueRating"`
	PackagingRating int        `json:"packagingRating"`
	HygieneRating   int        `json:"hygieneRating"`
	Title           string     `json:"title,omitempty"`
	Comment         string     `json:"comment"`
	Images          []string   `json:"images"`
	ChefResponse    string     `json:"chefResponse,omitempty"`
	ChefRespondedAt *time.Time `json:"chefRespondedAt,omitempty"`
	HelpfulCount    int        `json:"helpfulCount"`
	CustomerName    string     `json:"customerName"`
	CustomerAvatar  string     `json:"customerAvatar,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
}

func (r *Review) ToResponse() ReviewResponse {
	images := []string{}
	if r.Images != nil {
		images = r.Images
	}

	customerName := ""
	customerAvatar := ""
	if r.Customer.ID != uuid.Nil {
		if len(r.Customer.LastName) > 0 {
			customerName = r.Customer.FirstName + " " + string(r.Customer.LastName[0]) + "."
		} else {
			customerName = r.Customer.FirstName
		}
		customerAvatar = r.Customer.Avatar
	}

	return ReviewResponse{
		ID:              r.ID,
		OrderID:         r.OrderID,
		OverallRating:   r.OverallRating,
		FoodRating:      r.FoodRating,
		DeliveryRating:  r.DeliveryRating,
		ValueRating:     r.ValueRating,
		PackagingRating: r.PackagingRating,
		HygieneRating:   r.HygieneRating,
		Title:           r.Title,
		Comment:         r.Comment,
		Images:          images,
		ChefResponse:    r.ChefResponse,
		ChefRespondedAt: r.ChefRespondedAt,
		HelpfulCount:    r.HelpfulCount,
		CustomerName:    customerName,
		CustomerAvatar:  customerAvatar,
		CreatedAt:       r.CreatedAt,
	}
}

// DishRating is a per-dish rating attached to a review (#35/#145) — a customer
// can rate individual menu items from their order, which rolls up into each
// dish's MenuItem.Rating. One row per (review, menu item).
type DishRating struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReviewID   uuid.UUID `gorm:"type:uuid;not null;index" json:"reviewId"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null;index" json:"menuItemId"`
	ChefID     uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	Rating     int       `gorm:"not null" json:"rating"` // 1-5
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
