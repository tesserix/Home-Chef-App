package models

import (
	"time"

	"github.com/google/uuid"
)

// menu_capacity.go — chef capacity & cutoff controls (#48).

// MenuItemDailySales is the per-dish, per-day sold counter that backs the daily
// capacity cap. Keyed by (menu_item_id, sale_date) where sale_date is the IST
// calendar day, so caps reset automatically each day with no cron. Reserve at
// order time with an atomic `UPDATE ... WHERE sold_qty + n <= cap`; release on
// cancellation. remaining = cap - SoldQty; soldOut = SoldQty >= cap.
type MenuItemDailySales struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_item_day" json:"menuItemId"`
	ChefID     uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	// SaleDate is the IST calendar day (midnight) this counter is for.
	SaleDate  time.Time `gorm:"type:date;not null;uniqueIndex:idx_item_day" json:"saleDate"`
	SoldQty   int       `gorm:"not null;default:0" json:"soldQty"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// ChefCapacitySettings holds a chef's per-meal order cutoffs and the auto-sold-out
// toggle (#48). One row per chef (mirrors ChefSettings). Cutoffs are "HH:MM" in
// IST; "" means no cutoff for that meal.
type ChefCapacitySettings struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"chefId"`
	CutoffEnabled bool      `gorm:"default:false" json:"cutoffEnabled"`
	LunchCutoff   string    `gorm:"type:varchar(5)" json:"lunchCutoff"`  // "HH:MM" IST
	DinnerCutoff  string    `gorm:"type:varchar(5)" json:"dinnerCutoff"` // "HH:MM" IST
	AutoSoldOut   bool      `gorm:"default:true" json:"autoSoldOut"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
