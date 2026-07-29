package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// daily_menu.go — a chef's per-CALENDAR-DATE tiffin menu (#405). Unlike the fixed
// weekly template (weekly_menu.go, one dish per weekday×slot×variant), each date
// carries MULTIPLE dishes per slot (rice + dal + sabji + curry…), so a home chef
// can cook different things on different days. The meal plan (#406) resolves a
// booked (date, slot) to these dishes / the day's thali.

// DailyMenu is the per-(chef, date) header holding publish state.
type DailyMenu struct {
	// Live/test data partition. See models.ModePartition.
	ModePartition

	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	// The (chef, date) pair is unique per MODE — see the
	// idx_daily_menu_chef_date_live / _test pair in database.go's postMigrate
	// block. Mode-blind uniqueness here would reject the live→test clone.
	// Named _lookup so it does not collide with the legacy index postMigrate drops.
	ChefID      uuid.UUID       `gorm:"type:uuid;not null;index:idx_daily_menu_chef_date_lookup" json:"chefId"`
	Date        time.Time       `gorm:"type:date;not null;index:idx_daily_menu_chef_date_lookup" json:"date"`
	IsPublished bool            `gorm:"default:false" json:"isPublished"`
	PublishedAt *time.Time      `gorm:"" json:"publishedAt,omitempty"`
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time       `gorm:"autoUpdateTime" json:"updatedAt"`
	Items       []DailyMenuItem `gorm:"foreignKey:DailyMenuID" json:"items,omitempty"`
}

// DailyMenuItem is one dish on a date's menu. MULTIPLE per (date, slot) are
// allowed — there is deliberately NO unique-cell constraint (the key difference
// from WeeklyMenuItem), so a chef can list several dishes for the same slot.
type DailyMenuItem struct {
	// Live/test data partition. See models.ModePartition.
	ModePartition

	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DailyMenuID uuid.UUID      `gorm:"type:uuid;not null;index" json:"dailyMenuId"`
	ChefID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_daily_item_chef_date" json:"chefId"`
	Date        time.Time      `gorm:"type:date;not null;index:idx_daily_item_chef_date" json:"date"`
	Slot        MealSlot       `gorm:"type:varchar(10);not null" json:"slot"`
	Variant     MealVariant    `gorm:"type:varchar(10);not null" json:"variant"`
	Name        string         `gorm:"not null" json:"name"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Price       float64        `gorm:"default:0" json:"price"`
	ImageURL    string         `gorm:"" json:"imageUrl,omitempty"`
	DietaryTags pq.StringArray `gorm:"type:text[]" json:"dietaryTags"`
	Allergens   pq.StringArray `gorm:"type:text[]" json:"allergens"`
	// MenuItemID optionally links the dish to an à-la-carte MenuItem (reuse image).
	MenuItemID *uuid.UUID `gorm:"type:uuid" json:"menuItemId,omitempty"`
	// IsCombo marks this entry as a bundled set for the (date, slot): one set
	// Price covering ComboComponents (the dishes it includes, e.g. rice + dal +
	// sabji + curry). The API term is intentionally NEUTRAL ("combo"); clients
	// LOCALIZE the label by country/locale — "Thali" in India, "Combo" elsewhere.
	// A combo is the default plan choice for its slot; other items are à-la-carte.
	// (#406)
	IsCombo         bool           `gorm:"default:false" json:"isCombo"`
	ComboComponents pq.StringArray `gorm:"type:text[]" json:"comboComponents"`
	// How much food this entry actually is — see WeeklyMenuItem.PortionSize for
	// why it lives on the entry rather than being read through MenuItemID.
	PortionSize string `gorm:"" json:"portionSize,omitempty"`
	Serves      int    `gorm:"default:1" json:"serves"`
	// SortOrder controls display order within a (date, slot).
	SortOrder int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
