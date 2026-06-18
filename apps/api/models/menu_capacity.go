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

// ChefCapacitySettings holds a chef's per-meal order cutoffs, the auto-sold-out
// toggle (#48), and scheduled delivery-slot config (#51). One row per chef
// (mirrors ChefSettings). Cutoffs and slot windows are "HH:MM" in IST; "" means
// no cutoff/window for that meal.
type ChefCapacitySettings struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"chefId"`
	CutoffEnabled bool      `gorm:"default:false" json:"cutoffEnabled"`
	LunchCutoff   string    `gorm:"type:varchar(5)" json:"lunchCutoff"`  // "HH:MM" IST
	DinnerCutoff  string    `gorm:"type:varchar(5)" json:"dinnerCutoff"` // "HH:MM" IST
	AutoSoldOut   bool      `gorm:"default:true" json:"autoSoldOut"`

	// Scheduled delivery slots (#51). When SlotsEnabled, the customer picks a
	// lunch/dinner delivery window at checkout; each slot has a display window
	// (start–end "HH:MM" IST) and an optional per-day capacity (nil/0 =
	// unlimited). The existing Lunch/DinnerCutoff above doubles as that slot's
	// order cutoff. Booking counts live in ChefSlotDailyBookings.
	SlotsEnabled       bool   `gorm:"default:false" json:"slotsEnabled"`
	LunchSlotStart     string `gorm:"type:varchar(5)" json:"lunchSlotStart"`  // "HH:MM" IST
	LunchSlotEnd       string `gorm:"type:varchar(5)" json:"lunchSlotEnd"`    // "HH:MM" IST
	DinnerSlotStart    string `gorm:"type:varchar(5)" json:"dinnerSlotStart"` // "HH:MM" IST
	DinnerSlotEnd      string `gorm:"type:varchar(5)" json:"dinnerSlotEnd"`   // "HH:MM" IST
	LunchSlotCapacity  *int   `gorm:"" json:"lunchSlotCapacity"`              // nil/0 = unlimited
	DinnerSlotCapacity *int   `gorm:"" json:"dinnerSlotCapacity"`             // nil/0 = unlimited

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// ChefSlotDailyBookings is the per-chef, per-slot, per-day booked-order counter
// that backs scheduled-slot capacity (#51) — the slot analogue of
// MenuItemDailySales. Keyed by (chef_id, slot, booking_date) where slot is
// "lunch"|"dinner" and booking_date is the IST calendar day the order is
// scheduled to be delivered, so a slot's count resets each day with no cron.
// Reserve at order time with an atomic `UPDATE ... WHERE booked_qty + n <= cap`;
// release on cancellation/refund. remaining = cap - BookedQty; full = BookedQty >= cap.
type ChefSlotDailyBookings struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_chef_slot_day" json:"chefId"`
	// Slot is "lunch" or "dinner".
	Slot string `gorm:"type:varchar(8);not null;uniqueIndex:idx_chef_slot_day" json:"slot"`
	// BookingDate is the IST calendar day (midnight) the order is scheduled for.
	BookingDate time.Time `gorm:"type:date;not null;uniqueIndex:idx_chef_slot_day" json:"bookingDate"`
	BookedQty   int       `gorm:"not null;default:0" json:"bookedQty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
