package models

import (
	"time"

	"github.com/google/uuid"
)

// modifier.go — add-on / modifier groups (#232) and combo components (#233).
//
// A ModifierGroup is a per-menu-item set of choices (e.g. "Spice level" — single,
// required; or "Extras" — multi, optional) with per-option price deltas. The
// customer picks options at add-to-cart; the order snapshots the picks + price.
//
// A ComboItem is one included item in a combo MenuItem (IsCombo). The combo's own
// Price is the bundle price; the components are informational (shown to the
// customer + on the invoice). A combo otherwise flows like any line item.

// ModifierGroup belongs to one MenuItem. Replace-all on item save.
type ModifierGroup struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null;index" json:"menuItemId"`
	Name       string    `gorm:"not null" json:"name"`
	Required   bool      `gorm:"default:false" json:"required"`
	// MinSelect/MaxSelect bound the picks. MaxSelect == 1 → single choice; > 1 or
	// 0 (unlimited) → multi. When Required, at least MinSelect (≥1) must be picked.
	MinSelect int       `gorm:"default:0" json:"minSelect"`
	MaxSelect int       `gorm:"default:1" json:"maxSelect"`
	SortOrder int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"-"`

	Options []ModifierOption `gorm:"foreignKey:GroupID" json:"options"`
}

// ModifierOption is one choice within a group, with a price delta in the chef's
// currency (0 for free add-ons; can be negative for a discount).
type ModifierOption struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	GroupID     uuid.UUID `gorm:"type:uuid;not null;index" json:"groupId"`
	Name        string    `gorm:"not null" json:"name"`
	PriceDelta  float64   `gorm:"default:0" json:"priceDelta"`
	IsAvailable bool      `gorm:"default:true" json:"isAvailable"`
	SortOrder   int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"-"`
}

// ComboItem is one component of a combo MenuItem (ComboID is the combo's ID).
type ComboItem struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ComboID    uuid.UUID `gorm:"type:uuid;not null;index" json:"comboId"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null" json:"menuItemId"`
	Name       string    `gorm:"" json:"name"` // snapshot of the included item's name
	Quantity   int       `gorm:"default:1" json:"quantity"`
	SortOrder  int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"-"`
}

// OrderItemModifier is the snapshot of one selected modifier on an order line,
// stored as JSON on OrderItem.Modifiers (immutable history).
type OrderItemModifier struct {
	GroupName  string  `json:"groupName"`
	OptionName string  `json:"optionName"`
	PriceDelta float64 `json:"priceDelta"`
}
