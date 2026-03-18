package models

import (
	"time"

	"github.com/google/uuid"
)

type Cart struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	ChefID    *uuid.UUID `gorm:"type:uuid;index" json:"chefId,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`

	// Relationships
	User  User        `gorm:"foreignKey:UserID" json:"-"`
	Chef  *ChefProfile `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
	Items []CartItem  `gorm:"foreignKey:CartID" json:"items,omitempty"`
}

type CartItem struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CartID     uuid.UUID `gorm:"type:uuid;not null;index" json:"cartId"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null" json:"menuItemId"`
	Quantity   int       `gorm:"not null;default:1" json:"quantity"`
	Notes      string    `gorm:"" json:"notes,omitempty"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Cart     Cart     `gorm:"foreignKey:CartID" json:"-"`
	MenuItem MenuItem `gorm:"foreignKey:MenuItemID" json:"menuItem,omitempty"`
}

// DTOs
type CartResponse struct {
	ID       uuid.UUID          `json:"id"`
	ChefID   *uuid.UUID         `json:"chefId,omitempty"`
	Chef     *ChefCartResponse  `json:"chef,omitempty"`
	Items    []CartItemResponse `json:"items"`
	Subtotal float64            `json:"subtotal"`
}

type ChefCartResponse struct {
	ID           uuid.UUID `json:"id"`
	BusinessName string    `json:"businessName"`
	MinimumOrder float64   `json:"minimumOrder"`
}

type CartItemResponse struct {
	ID         uuid.UUID        `json:"id"`
	MenuItemID uuid.UUID        `json:"menuItemId"`
	MenuItem   MenuItemResponse `json:"menuItem"`
	Quantity   int              `json:"quantity"`
	Notes      string           `json:"notes,omitempty"`
	Subtotal   float64          `json:"subtotal"`
}

func (c *Cart) ToResponse() CartResponse {
	items := make([]CartItemResponse, len(c.Items))
	var subtotal float64

	for i, item := range c.Items {
		itemSubtotal := item.MenuItem.Price * float64(item.Quantity)
		items[i] = CartItemResponse{
			ID:         item.ID,
			MenuItemID: item.MenuItemID,
			MenuItem:   item.MenuItem.ToResponse(),
			Quantity:   item.Quantity,
			Notes:      item.Notes,
			Subtotal:   itemSubtotal,
		}
		subtotal += itemSubtotal
	}

	response := CartResponse{
		ID:       c.ID,
		ChefID:   c.ChefID,
		Items:    items,
		Subtotal: subtotal,
	}

	if c.Chef != nil {
		response.Chef = &ChefCartResponse{
			ID:           c.Chef.ID,
			BusinessName: c.Chef.BusinessName,
			MinimumOrder: c.Chef.MinimumOrder,
		}
	}

	return response
}
