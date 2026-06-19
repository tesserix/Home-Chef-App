package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// meal_subscription_fulfillment.go — one row per generated delivery (sub × date ×
// slot) for the customer meal subscription (#282). The system places the order on
// the customer's behalf at the chef's cutoff; this tracks per-day status so the
// customer/chef see adherence and missed days drive next-cycle credits.

const (
	MealFulfillScheduled = "scheduled" // due but not yet placed
	MealFulfillPlaced    = "placed"    // order created on the customer's behalf
	MealFulfillDelivered = "delivered" // order delivered
	MealFulfillMissed    = "missed"    // chef no-show / cancelled → credited
	MealFulfillSkipped   = "skipped"   // customer skipped before cutoff → credited
)

type MealSubscriptionFulfillment struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MealSubscriptionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_meal_fulfill_cell;index" json:"mealSubscriptionId"`
	CustomerID         uuid.UUID `gorm:"type:uuid;not null;index" json:"customerId"`
	ChefID             uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	Date               time.Time `gorm:"not null;uniqueIndex:idx_meal_fulfill_cell" json:"date"`
	Slot               MealSlot  `gorm:"type:varchar(10);not null;uniqueIndex:idx_meal_fulfill_cell" json:"slot"`
	DishName           string    `gorm:"" json:"dishName"`
	Price              float64   `gorm:"default:0" json:"price"`
	Status             string    `gorm:"type:varchar(12);not null;default:'scheduled';index" json:"status"`
	OrderID            *uuid.UUID `gorm:"type:uuid" json:"orderId,omitempty"`
	CreatedAt          time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (f *MealSubscriptionFulfillment) BeforeCreate(*gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
