package models

import (
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriberType distinguishes chef vs driver subscriptions
type SubscriberType string

const (
	SubscriberChef   SubscriberType = "chef"
	SubscriberDriver SubscriberType = "driver"
)

// BillingInterval defines plan cadence
type BillingInterval string

const (
	BillingMonthly   BillingInterval = "monthly"
	BillingQuarterly BillingInterval = "quarterly"
	BillingYearly    BillingInterval = "yearly"
)

// SubscriptionStatus tracks lifecycle
type SubscriptionStatus string

const (
	SubStatusTrial     SubscriptionStatus = "trial"
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusPastDue   SubscriptionStatus = "past_due"
	SubStatusSuspended SubscriptionStatus = "suspended"
	SubStatusCancelled SubscriptionStatus = "cancelled"
	SubStatusExpired   SubscriptionStatus = "expired"
)

// InvoiceStatus tracks invoice lifecycle
type InvoiceStatus string

const (
	InvoicePending    InvoiceStatus = "pending"
	InvoiceProcessing InvoiceStatus = "processing"
	InvoicePaid       InvoiceStatus = "paid"
	InvoiceFailed     InvoiceStatus = "failed"
	InvoiceRefunded   InvoiceStatus = "refunded"
	InvoiceVoided     InvoiceStatus = "voided"
)

// EarningSource categorises ledger entries
type EarningSource string

const (
	EarningOrderRevenue EarningSource = "order_revenue"
	EarningDeliveryFee  EarningSource = "delivery_fee"
	EarningTip          EarningSource = "tip"
)

// Subscription represents a chef or driver billing subscription
type Subscription struct {
	ID             uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID          `gorm:"type:uuid;not null;index" json:"userId"`
	SubscriberType SubscriberType     `gorm:"type:varchar(10);not null" json:"subscriberType"`
	CountryCode    string             `gorm:"type:varchar(3);not null" json:"countryCode"`
	Currency       string             `gorm:"type:varchar(3);not null" json:"currency"`
	BillingInterval BillingInterval   `gorm:"type:varchar(10);not null" json:"billingInterval"`
	Status         SubscriptionStatus `gorm:"type:varchar(20);default:'trial'" json:"status"`
	PlanAmount     float64            `gorm:"not null" json:"planAmount"`

	// Trial period
	TrialStartsAt time.Time `gorm:"not null" json:"trialStartsAt"`
	TrialEndsAt   time.Time `gorm:"not null" json:"trialEndsAt"`

	// Billing cycle
	CurrentPeriodStart *time.Time `gorm:"" json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time `gorm:"" json:"currentPeriodEnd,omitempty"`

	// Billing starts only after earnings threshold met
	BillingStartsAt *time.Time `gorm:"" json:"billingStartsAt,omitempty"`

	// Grace period after failed payment
	GraceEndsAt *time.Time `gorm:"" json:"graceEndsAt,omitempty"`

	// Cancellation
	CancelledAt  *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason string     `gorm:"" json:"cancelReason,omitempty"`
	RefundAmount float64    `gorm:"default:0" json:"refundAmount"`

	// Payment gateway integration
	PaymentGateway string `gorm:"type:varchar(20)" json:"paymentGateway"`
	GatewaySubID   string `gorm:"" json:"-"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User     User                  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Invoices []SubscriptionInvoice `gorm:"foreignKey:SubscriptionID" json:"invoices,omitempty"`
}

// SubscriptionInvoice tracks individual billing invoices
type SubscriptionInvoice struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SubscriptionID uuid.UUID     `gorm:"type:uuid;not null;index" json:"subscriptionId"`
	InvoiceNumber  string        `gorm:"uniqueIndex;not null" json:"invoiceNumber"`
	Status         InvoiceStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`

	// Amounts
	Amount      float64 `gorm:"not null" json:"amount"`
	Currency    string  `gorm:"type:varchar(3);not null" json:"currency"`
	TaxAmount   float64 `gorm:"default:0" json:"taxAmount"`
	TotalAmount float64 `gorm:"not null" json:"totalAmount"`

	// Period
	PeriodStart time.Time `gorm:"not null" json:"periodStart"`
	PeriodEnd   time.Time `gorm:"not null" json:"periodEnd"`

	// Earnings snapshot
	EarningsAtGeneration float64 `gorm:"default:0" json:"earningsAtGeneration"`

	// Payment gateway
	PaymentGateway   string `gorm:"type:varchar(20)" json:"paymentGateway"`
	GatewayPaymentID string `gorm:"" json:"-"`
	GatewayOrderID   string `gorm:"" json:"-"`

	// Retry tracking
	AttemptCount  int        `gorm:"default:0" json:"attemptCount"`
	MaxAttempts   int        `gorm:"default:3" json:"maxAttempts"`
	LastAttemptAt *time.Time `gorm:"" json:"lastAttemptAt,omitempty"`
	NextRetryAt   *time.Time `gorm:"" json:"nextRetryAt,omitempty"`

	// Settlement
	PaidAt       *time.Time `gorm:"" json:"paidAt,omitempty"`
	RefundedAt   *time.Time `gorm:"" json:"refundedAt,omitempty"`
	RefundAmount float64    `gorm:"default:0" json:"refundAmount"`
	FailureReason string   `gorm:"" json:"failureReason,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	// Relationships
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
}

// EarningsLedger records individual earning events for threshold tracking
type EarningsLedger struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null;index:idx_earnings_user_cycle" json:"userId"`
	SubscriptionID uuid.UUID      `gorm:"type:uuid;not null;index" json:"subscriptionId"`
	SubscriberType SubscriberType `gorm:"type:varchar(10);not null" json:"subscriberType"`

	// Cycle boundaries
	CycleStart time.Time `gorm:"not null;index:idx_earnings_user_cycle" json:"cycleStart"`
	CycleEnd   time.Time `gorm:"not null;index:idx_earnings_user_cycle" json:"cycleEnd"`

	// Earning details
	Source     EarningSource `gorm:"type:varchar(20);not null" json:"source"`
	OrderID    *uuid.UUID    `gorm:"type:uuid" json:"orderId,omitempty"`
	DeliveryID *uuid.UUID    `gorm:"type:uuid" json:"deliveryId,omitempty"`
	Amount     float64       `gorm:"not null" json:"amount"`
	Currency   string        `gorm:"type:varchar(3);not null" json:"currency"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

	// Relationships
	User         User         `gorm:"foreignKey:UserID" json:"-"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"-"`
}

// ---------- DTOs ----------

// SubscriptionResponse is the API response for a subscription
type SubscriptionResponse struct {
	ID              uuid.UUID          `json:"id"`
	UserID          uuid.UUID          `json:"userId"`
	SubscriberType  SubscriberType     `json:"subscriberType"`
	CountryCode     string             `json:"countryCode"`
	Currency        string             `json:"currency"`
	BillingInterval BillingInterval    `json:"billingInterval"`
	Status          SubscriptionStatus `json:"status"`
	PlanAmount      float64            `json:"planAmount"`
	TrialStartsAt   time.Time          `json:"trialStartsAt"`
	TrialEndsAt     time.Time          `json:"trialEndsAt"`
	CurrentPeriodStart *time.Time      `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd   *time.Time      `json:"currentPeriodEnd,omitempty"`
	BillingStartsAt    *time.Time      `json:"billingStartsAt,omitempty"`
	GraceEndsAt        *time.Time      `json:"graceEndsAt,omitempty"`
	CancelledAt        *time.Time      `json:"cancelledAt,omitempty"`
	CancelReason       string          `json:"cancelReason,omitempty"`
	RefundAmount       float64         `json:"refundAmount"`
	PaymentGateway     string          `json:"paymentGateway,omitempty"`
	CreatedAt          time.Time       `json:"createdAt"`
}

// SubscriptionInvoiceResponse is the API response for an invoice
type SubscriptionInvoiceResponse struct {
	ID                   uuid.UUID     `json:"id"`
	SubscriptionID       uuid.UUID     `json:"subscriptionId"`
	InvoiceNumber        string        `json:"invoiceNumber"`
	Status               InvoiceStatus `json:"status"`
	Amount               float64       `json:"amount"`
	Currency             string        `json:"currency"`
	TaxAmount            float64       `json:"taxAmount"`
	TotalAmount          float64       `json:"totalAmount"`
	PeriodStart          time.Time     `json:"periodStart"`
	PeriodEnd            time.Time     `json:"periodEnd"`
	EarningsAtGeneration float64       `json:"earningsAtGeneration"`
	AttemptCount         int           `json:"attemptCount"`
	PaidAt               *time.Time    `json:"paidAt,omitempty"`
	FailureReason        string        `json:"failureReason,omitempty"`
	CreatedAt            time.Time     `json:"createdAt"`
}

// EarningsSummary is the API response for current cycle earnings
type EarningsSummary struct {
	CycleStart       *time.Time `json:"cycleStart,omitempty"`
	CycleEnd         *time.Time `json:"cycleEnd,omitempty"`
	TotalEarnings    float64    `json:"totalEarnings"`
	OrderRevenue     float64    `json:"orderRevenue"`
	DeliveryFees     float64    `json:"deliveryFees"`
	Tips             float64    `json:"tips"`
	Threshold        float64    `json:"threshold"`
	ThresholdMet     bool       `json:"thresholdMet"`
	Currency         string     `json:"currency"`
}

// ToResponse converts Subscription to SubscriptionResponse
func (s *Subscription) ToResponse() SubscriptionResponse {
	return SubscriptionResponse{
		ID:                 s.ID,
		UserID:             s.UserID,
		SubscriberType:     s.SubscriberType,
		CountryCode:        s.CountryCode,
		Currency:           s.Currency,
		BillingInterval:    s.BillingInterval,
		Status:             s.Status,
		PlanAmount:         s.PlanAmount,
		TrialStartsAt:      s.TrialStartsAt,
		TrialEndsAt:        s.TrialEndsAt,
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		BillingStartsAt:   s.BillingStartsAt,
		GraceEndsAt:       s.GraceEndsAt,
		CancelledAt:       s.CancelledAt,
		CancelReason:      s.CancelReason,
		RefundAmount:      s.RefundAmount,
		PaymentGateway:    s.PaymentGateway,
		CreatedAt:         s.CreatedAt,
	}
}

// ToResponse converts SubscriptionInvoice to SubscriptionInvoiceResponse
func (i *SubscriptionInvoice) ToResponse() SubscriptionInvoiceResponse {
	return SubscriptionInvoiceResponse{
		ID:                   i.ID,
		SubscriptionID:       i.SubscriptionID,
		InvoiceNumber:        i.InvoiceNumber,
		Status:               i.Status,
		Amount:               i.Amount,
		Currency:             i.Currency,
		TaxAmount:            i.TaxAmount,
		TotalAmount:          i.TotalAmount,
		PeriodStart:          i.PeriodStart,
		PeriodEnd:            i.PeriodEnd,
		EarningsAtGeneration: i.EarningsAtGeneration,
		AttemptCount:         i.AttemptCount,
		PaidAt:               i.PaidAt,
		FailureReason:        i.FailureReason,
		CreatedAt:            i.CreatedAt,
	}
}

// IntervalMonths returns the number of months for a billing interval
func (b BillingInterval) IntervalMonths() int {
	switch b {
	case BillingQuarterly:
		return 3
	case BillingYearly:
		return 12
	default:
		return 1
	}
}

// RoundAmount rounds a float to 2 decimal places
func RoundAmount(amount float64) float64 {
	return math.Round(amount*100) / 100
}
