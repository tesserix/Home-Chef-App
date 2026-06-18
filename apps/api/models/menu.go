package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// MenuCategory represents a chef-scoped menu category (e.g., "Starters", "Main Course", "Desserts")
type MenuCategory struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID      uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex:idx_menu_categories_chef_name" json:"chefId"`
	Name        string         `gorm:"not null;uniqueIndex:idx_menu_categories_chef_name" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	SortOrder   int            `gorm:"default:0" json:"sortOrder"`
	IsActive    bool           `gorm:"default:true" json:"isActive"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Chef ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
}

type MenuItem struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"chefId"`
	CategoryID   *uuid.UUID     `gorm:"type:uuid;index" json:"categoryId,omitempty"`
	Name         string         `gorm:"not null" json:"name"`
	Description  string         `gorm:"type:text" json:"description,omitempty"`
	Price        float64        `gorm:"not null" json:"price"`
	ComparePrice float64        `gorm:"default:0" json:"comparePrice,omitempty"`
	ImageURL     string         `gorm:"" json:"imageUrl,omitempty"`
	DietaryTags  pq.StringArray `gorm:"type:text[]" json:"dietaryTags"`
	Allergens    pq.StringArray `gorm:"type:text[]" json:"allergens"`
	Ingredients  pq.StringArray `gorm:"type:text[]" json:"ingredients"`
	PrepTime     int            `gorm:"" json:"prepTime"`
	PortionSize  string         `gorm:"" json:"portionSize,omitempty"`
	Serves       int            `gorm:"default:1" json:"serves"`
	SpiceLevel   int            `gorm:"default:0" json:"spiceLevel"`
	// IsVeg is a nullable boolean: true = veg, false = non-veg, nil = not set
	// (legacy rows before the column was added). Frontend renders the FSSAI
	// green/brown dot only when the value is non-nil.
	IsVeg       *bool `gorm:"" json:"isVeg,omitempty"`
	IsAvailable bool  `gorm:"default:true" json:"isAvailable"`
	IsApproved  bool  `gorm:"default:false" json:"isApproved"`
	// DailyCapacity caps how many of this dish the chef will make per day (#48).
	// nil or <= 0 means unlimited. Remaining/sold-out are derived from the
	// MenuItemDailySales counter (IST calendar day), not a mutated flag.
	DailyCapacity *int `gorm:"" json:"dailyCapacity,omitempty"`
	// Transient capacity state (#48), populated by handlers, never persisted.
	RemainingToday *int `gorm:"-" json:"remainingToday,omitempty"`
	SoldOut        bool `gorm:"-" json:"soldOut"`
	// HSN code per HSN/SAC (GST harmonized system). Defaults to 996331
	// — the SAC for "Services provided by Restaurants … and others"
	// which covers prepared-food sales for our home-chef model. Chef
	// can override per item if their tax advisor wants a more specific
	// code. Printed on customer invoices per Wave 3 GSTIN flow.
	HSN          string         `gorm:"type:varchar(8);default:'996331'" json:"hsn,omitempty"`
	IsFeatured   bool           `gorm:"default:false" json:"isFeatured"`
	TotalOrders  int            `gorm:"default:0" json:"totalOrders"`
	Rating       float64        `gorm:"default:0" json:"rating"`
	TotalReviews int            `gorm:"default:0" json:"totalReviews"`
	SortOrder    int            `gorm:"default:0" json:"sortOrder"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Chef     ChefProfile     `gorm:"foreignKey:ChefID" json:"-"`
	Category *MenuCategory   `gorm:"foreignKey:CategoryID" json:"-"`
	Images   []MenuItemImage `gorm:"foreignKey:MenuItemID" json:"images,omitempty"`
}

type MenuItemImage struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null;index" json:"menuItemId"`
	URL        string    `gorm:"not null" json:"url"`
	IsPrimary  bool      `gorm:"default:false" json:"isPrimary"`
	SortOrder  int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`

	MenuItem MenuItem `gorm:"foreignKey:MenuItemID" json:"-"`
}

// DTOs

type MenuItemImageResponse struct {
	ID         uuid.UUID `json:"id"`
	MenuItemID uuid.UUID `json:"menuItemId"`
	URL        string    `json:"url"`
	IsPrimary  bool      `json:"isPrimary"`
	SortOrder  int       `json:"sortOrder"`
}

type MenuItemResponse struct {
	ID           uuid.UUID               `json:"id"`
	ChefID       uuid.UUID               `json:"chefId"`
	CategoryID   *uuid.UUID              `json:"categoryId,omitempty"`
	Name         string                  `json:"name"`
	Description  string                  `json:"description"`
	Price        float64                 `json:"price"`
	ComparePrice float64                 `json:"comparePrice,omitempty"`
	ImageURL     string                  `json:"imageUrl,omitempty"`
	Images       []MenuItemImageResponse `json:"images"`
	PrepTime     int                     `json:"prepTime"`
	PortionSize  string                  `json:"portionSize,omitempty"`
	Serves       int                     `json:"serves"`
	DietaryTags  []string                `json:"dietaryTags"`
	Allergens    []string                `json:"allergens"`
	SpiceLevel   int                     `json:"spiceLevel"`
	// IsVeg is omitted from JSON when nil (legacy items where the flag was not set).
	IsVeg       *bool   `json:"isVeg,omitempty"`
	IsAvailable bool    `json:"isAvailable"`
	IsFeatured  bool    `json:"isFeatured"`
	Rating      float64 `json:"rating"`
	HSN         string  `json:"hsn,omitempty"`
	// Capacity (#48). DailyCapacity nil = unlimited. RemainingToday + SoldOut are
	// populated by handlers that have DB access (the daily-sales counter); they're
	// omitted when the item is uncapped.
	DailyCapacity  *int `json:"dailyCapacity,omitempty"`
	RemainingToday *int `json:"remainingToday,omitempty"`
	SoldOut        bool `json:"soldOut"`
}

func (m *MenuItem) ToResponse() MenuItemResponse {
	dietaryTags := []string{}
	if m.DietaryTags != nil {
		dietaryTags = m.DietaryTags
	}

	allergens := []string{}
	if m.Allergens != nil {
		allergens = m.Allergens
	}

	images := make([]MenuItemImageResponse, len(m.Images))
	for i, img := range m.Images {
		images[i] = MenuItemImageResponse{
			ID:         img.ID,
			MenuItemID: img.MenuItemID,
			URL:        img.URL,
			IsPrimary:  img.IsPrimary,
			SortOrder:  img.SortOrder,
		}
	}

	return MenuItemResponse{
		ID:            m.ID,
		ChefID:        m.ChefID,
		CategoryID:    m.CategoryID,
		Name:          m.Name,
		Description:   m.Description,
		Price:         m.Price,
		ComparePrice:  m.ComparePrice,
		ImageURL:      m.ImageURL,
		Images:        images,
		PrepTime:      m.PrepTime,
		PortionSize:   m.PortionSize,
		Serves:        m.Serves,
		DietaryTags:   dietaryTags,
		Allergens:     allergens,
		SpiceLevel:    m.SpiceLevel,
		IsVeg:         m.IsVeg,
		IsAvailable:   m.IsAvailable,
		IsFeatured:    m.IsFeatured,
		Rating:        m.Rating,
		HSN:           m.HSN,
		DailyCapacity: m.DailyCapacity,
	}
}
