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
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID      uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex:idx_daily_menu_chef_date" json:"chefId"`
	Date        time.Time       `gorm:"type:date;not null;uniqueIndex:idx_daily_menu_chef_date" json:"date"`
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
	// SortOrder controls display order within a (date, slot).
	SortOrder int       `gorm:"default:0" json:"sortOrder"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
