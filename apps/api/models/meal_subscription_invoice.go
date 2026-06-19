package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// meal_subscription_invoice.go — a per-cycle invoice for a customer meal
// subscription (#281). Distinct from the chef/driver SubscriptionInvoice. The
// gross CycleAmount, the carried skip/missed CreditApplied, the net Amount charged,
// and the tax/total are frozen at generation so a later config change can't
// rewrite history.

// MealSubscriptionInvoice statuses.
const (
	MealInvoiceStatusPending = "pending"
	MealInvoiceStatusPaid    = "paid"
	MealInvoiceStatusFailed  = "failed"
)

type MealSubscriptionInvoice struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MealSubscriptionID uuid.UUID `gorm:"type:uuid;not null;index" json:"mealSubscriptionId"`
	InvoiceNumber      string    `gorm:"uniqueIndex;not null" json:"invoiceNumber"`
	Status             string    `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`

	CycleAmount   float64 `gorm:"not null" json:"cycleAmount"`   // gross for the cycle (before credits)
	CreditApplied float64 `gorm:"default:0" json:"creditApplied"` // skip/missed credit consumed
	Amount        float64 `gorm:"not null" json:"amount"`        // net = cycle − credit
	TaxAmount     float64 `gorm:"default:0" json:"taxAmount"`
	TotalAmount   float64 `gorm:"not null" json:"totalAmount"`
	Currency      string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	PeriodStart time.Time `gorm:"not null" json:"periodStart"`
	PeriodEnd   time.Time `gorm:"not null" json:"periodEnd"`

	GatewayPaymentID string     `gorm:"" json:"gatewayPaymentId,omitempty"`
	PaidAt           *time.Time `gorm:"" json:"paidAt,omitempty"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (i *MealSubscriptionInvoice) BeforeCreate(*gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
