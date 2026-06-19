package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// winback.go — subscription / customer win-back offers (#42). When a customer
// lapses (stops ordering) or a chef/driver cancels/suspends their subscription,
// the platform auto-issues a targeted, time-limited, single-use discount promo to
// win them back. Each offer mints its own unique promo code (reusing the #39
// promo engine); reactivation = that code being redeemed.

// Win-back triggers — what caused the offer.
const (
	WinbackTriggerLapsed           = "lapsed"                      // customer inactivity (no recent order)
	WinbackTriggerSubCancelled     = "subscription_cancelled"      // chef/driver subscriber cancelled
	WinbackTriggerSubSuspended     = "subscription_suspended"      // payment failure → suspended
	WinbackTriggerMealSubCancelled = "meal_subscription_cancelled" // customer cancelled their tiffin subscription (#278)
)

// Win-back offer lifecycle.
const (
	WinbackStatusOffered     = "offered"
	WinbackStatusReactivated = "reactivated"
	WinbackStatusExpired     = "expired"
)

// Win-back audiences — who the offer targets (drives notification copy + where the
// code applies: customer → order checkout, chef/driver → re-subscription).
const (
	WinbackAudienceCustomer = "customer"
	WinbackAudienceChef     = "chef"
	WinbackAudienceDriver   = "driver"
)

type WinbackOffer struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null;index" json:"userId"`
	AudienceType    string         `gorm:"type:varchar(16);not null" json:"audienceType"`
	Trigger         string         `gorm:"type:varchar(32);not null" json:"trigger"`
	PromoCodeID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"promoCodeId"`
	Code            string         `gorm:"type:varchar(32);not null" json:"code"`
	DiscountPercent float64        `gorm:"default:0" json:"discountPercent"`
	Status          string         `gorm:"type:varchar(16);not null;default:'offered';index" json:"status"`
	// SubscriptionID links the offer to the cancelled/suspended subscription that
	// triggered it (nil for a lapse-triggered customer offer).
	SubscriptionID *uuid.UUID     `gorm:"type:uuid" json:"subscriptionId,omitempty"`
	OfferedAt      time.Time      `gorm:"autoCreateTime;index" json:"offeredAt"`
	ExpiresAt      time.Time      `gorm:"not null" json:"expiresAt"`
	ReactivatedAt  *time.Time     `gorm:"" json:"reactivatedAt,omitempty"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate mints a UUID on drivers (e.g. the sqlite test driver) that don't
// honour the gen_random_uuid() default.
func (w *WinbackOffer) BeforeCreate(*gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}
