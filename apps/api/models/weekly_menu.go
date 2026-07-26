package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// weekly_menu.go — a chef's fixed weekly menu (#1/#192). Each cell is a dish for a
// (dayOfWeek × slot × variant), so a customer's per-day veg/nonveg choice in a
// meal plan (#193) resolves to a real dish. The chef authors the dishes; the
// customer only picks veg-or-nonveg + which days.

// WeeklyMenu is the per-chef header (publish state).
type WeeklyMenu struct {
	// Live/test data partition. See models.ModePartition.
	ModePartition

	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	// Uniqueness is mode-aware and lives in database.go's postMigrate block —
	// one live menu per chef, one test menu per chef PER SESSION. A plain
	// uniqueIndex tag here would be mode-blind and reject the live→test clone.
	ChefID      uuid.UUID        `gorm:"type:uuid;index;not null" json:"chefId"`
	IsPublished bool             `gorm:"default:false" json:"isPublished"`
	PublishedAt *time.Time       `gorm:"" json:"publishedAt,omitempty"`
	CreatedAt   time.Time        `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time        `gorm:"autoUpdateTime" json:"updatedAt"`
	Items       []WeeklyMenuItem `gorm:"foreignKey:ChefID;references:ChefID" json:"items,omitempty"`
}

// WeeklyMenuItem is one cell: the dish for a (dayOfWeek 0=Sun..6=Sat, slot, variant).
// The (chef, day, slot, variant) tuple is unique — one dish per cell.
type WeeklyMenuItem struct {
	// Live/test data partition. See models.ModePartition.
	ModePartition

	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	// The (chef, day, slot, variant) cell is unique per MODE — see the
	// idx_weekly_cell_live / idx_weekly_cell_test pair in database.go's
	// postMigrate block. Mode-blind uniqueness here would reject the clone.
	// Named _lookup so it does not collide with the legacy mode-blind
	// idx_weekly_cell that postMigrate drops.
	ChefID      uuid.UUID   `gorm:"type:uuid;not null;index:idx_weekly_cell_lookup" json:"chefId"`
	DayOfWeek   int         `gorm:"not null;index:idx_weekly_cell_lookup" json:"dayOfWeek"`
	Slot        MealSlot    `gorm:"type:varchar(10);not null;index:idx_weekly_cell_lookup" json:"slot"`
	Variant     MealVariant `gorm:"type:varchar(10);not null;index:idx_weekly_cell_lookup" json:"variant"`
	Name        string      `gorm:"not null" json:"name"`
	Description string      `gorm:"type:text" json:"description,omitempty"`
	Price       float64     `gorm:"default:0" json:"price"`
	ImageURL    string      `gorm:"" json:"imageUrl,omitempty"`
	// Dietary & allergen tags (#41). The veg/non-veg signal is the cell's Variant;
	// these add finer diet tags (e.g. jain, gluten-free) and declared allergens so
	// the tiffin menu can render badges + warn like the à-la-carte menu.
	DietaryTags pq.StringArray `gorm:"type:text[]" json:"dietaryTags"`
	Allergens   pq.StringArray `gorm:"type:text[]" json:"allergens"`
	// MenuItemID optionally links the cell to an à-la-carte MenuItem (reuse its image).
	MenuItemID *uuid.UUID `gorm:"type:uuid" json:"menuItemId,omitempty"`
	// IsCombo marks this cell as a bundled thali/combo — one set Price covering
	// ComboComponents (the dishes it includes, e.g. rice + dal + sabji + roti).
	// Neutral API term ("combo"); clients localize the label — "Thali" in India,
	// "Combo" elsewhere. Mirrors DailyMenuItem (#406).
	IsCombo         bool           `gorm:"default:false" json:"isCombo"`
	ComboComponents pq.StringArray `gorm:"type:text[]" json:"comboComponents"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
}
