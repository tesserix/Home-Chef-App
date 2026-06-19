package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// meal_subscription.go — the customer tiffin meal subscription (#2/#3). A customer
// subscribes to a chef's published weekly fixed menu (#1) and is billed weekly or
// monthly via Razorpay Subscriptions + UPI Autopay; orders are auto-generated each
// day at the chef's cutoff. DISTINCT from MealPlan (one-time advance booking) and
// the chef/driver Subscription (platform SaaS billing) — do not overload those.

// MealCadence — billing/delivery cadence.
const (
	MealCadenceWeekly  = "weekly"
	MealCadenceMonthly = "monthly"
)

// MealSubscription lifecycle.
const (
	MealSubStatusTrialing  = "trialing"  // sampling a paid trial before subscribing
	MealSubStatusActive    = "active"    // recurring, generating orders
	MealSubStatusPaused    = "paused"    // customer paused; no orders generated
	MealSubStatusPastDue   = "past_due"  // a charge failed; generation halted
	MealSubStatusCancelled = "cancelled" // terminal
)

// MealTrial lifecycle (a one-time PAID sampler, one per chef, no mandate).
const (
	MealTrialStatusPending   = "pending"
	MealTrialStatusPaid      = "paid"
	MealTrialStatusConverted = "converted"
	MealTrialStatusExpired   = "expired"
)

// ChefSubscriptionConfig is a chef's tiffin-subscription offer (#4): which slots,
// per-meal price, cadences, capacity, cutoff, flat delivery fee, and trial config.
type ChefSubscriptionConfig struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID        uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"chefId"`
	Enabled       bool           `gorm:"default:false" json:"enabled"`
	Slots         pq.StringArray `gorm:"type:text[]" json:"slots"`     // offered: ["lunch","dinner"]
	Cadences      pq.StringArray `gorm:"type:text[]" json:"cadences"`  // ["weekly","monthly"]
	PerMealPrice  float64        `gorm:"default:0" json:"perMealPrice"`
	DeliveryFee   float64        `gorm:"default:0" json:"deliveryFee"` // flat, per cycle
	DailyCapacity int            `gorm:"default:0" json:"dailyCapacity"`
	CutoffTime    string         `gorm:"type:varchar(5);default:'21:00'" json:"cutoffTime"` // HH:MM, IST
	// Paid trial (#4).
	TrialEnabled      bool      `gorm:"default:false" json:"trialEnabled"`
	TrialDurationDays int       `gorm:"default:3" json:"trialDurationDays"`
	TrialPrice        float64   `gorm:"default:0" json:"trialPrice"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// MealTrial is a one-time paid sampler of a chef before subscribing. Unique per
// (customer, chef) so a customer can't trial the same chef twice.
type MealTrial struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID      uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_meal_trial_cust_chef" json:"customerId"`
	ChefID          uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_meal_trial_cust_chef" json:"chefId"`
	Price           float64    `gorm:"default:0" json:"price"`
	DurationDays    int        `gorm:"default:0" json:"durationDays"`
	Status          string     `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`
	RazorpayOrderID string     `gorm:"" json:"razorpayOrderId,omitempty"`
	StartsAt        *time.Time `gorm:"" json:"startsAt,omitempty"`
	EndsAt          *time.Time `gorm:"" json:"endsAt,omitempty"`
	CreatedAt       time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (t *MealTrial) BeforeCreate(*gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

type MealSubscription struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customerId"`
	ChefID     uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`

	// Selection — the customer picks slots, days and veg/nonveg; price is computed.
	Slots   pq.StringArray `gorm:"type:text[]" json:"slots"`        // ["lunch","dinner"]
	Days    pq.Int64Array  `gorm:"type:integer[]" json:"days"`      // day-of-week 0=Sun..6=Sat
	Variant MealVariant    `gorm:"type:varchar(10)" json:"variant"` // veg/nonveg, applied to every day
	Cadence string         `gorm:"type:varchar(10)" json:"cadence"` // weekly|monthly

	// Pricing — frozen at subscribe so later chef price edits don't change an active plan.
	PerMealPrice float64 `gorm:"default:0" json:"perMealPrice"`
	DeliveryFee  float64 `gorm:"default:0" json:"deliveryFee"`
	CycleAmount  float64 `gorm:"not null" json:"cycleAmount"` // charged per cycle
	Currency     string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	Status             string     `gorm:"type:varchar(16);not null;default:'trialing';index" json:"status"`
	CurrentPeriodStart *time.Time `gorm:"" json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time `gorm:"" json:"currentPeriodEnd,omitempty"`

	// CreditBalance carries skip/missed-day credits to the next cycle's charge (#42 owner decision).
	CreditBalance float64 `gorm:"default:0" json:"creditBalance"`

	TrialID          *uuid.UUID `gorm:"type:uuid" json:"trialId,omitempty"`
	DefaultAddressID *uuid.UUID `gorm:"type:uuid" json:"defaultAddressId,omitempty"`

	// Razorpay recurring (Phase 2). GatewaySubID is the Razorpay subscription id.
	PaymentGateway string `gorm:"type:varchar(20);default:'razorpay'" json:"paymentGateway"`
	GatewaySubID   string `gorm:"" json:"-"`

	PausedAt     *time.Time     `gorm:"" json:"pausedAt,omitempty"`
	CancelledAt  *time.Time     `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason string         `gorm:"" json:"cancelReason,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (s *MealSubscription) BeforeCreate(*gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// MealSubscriptionSkip records a single skipped delivery date (before cutoff) so
// the auto-order cron skips it and the cycle is credited.
type MealSubscriptionSkip struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MealSubscriptionID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_meal_skip_date" json:"mealSubscriptionId"`
	Date               time.Time `gorm:"not null;uniqueIndex:idx_meal_skip_date" json:"date"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (s *MealSubscriptionSkip) BeforeCreate(*gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
