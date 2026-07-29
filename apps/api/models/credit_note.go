package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreditNote is a GST credit note — the statutory reversal of tax already charged on an
// invoice (#834 item 2). Refund policy v3 returns the GST the platform collected on a
// cancelled meal-plan day, so without a matching credit note the output tax reported in the
// GSTR filing overstates what was actually retained, from the very first refund.
//
// One note per refunded day. SourceKey carries the natural key of what the note reverses
// ("mealplanday:<uuid>") and is UNIQUE, so a re-driven or concurrently-executed refund can
// never mint a second note for the same money.
type CreditNote struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	// CreditNoteNumber is the customer-visible reference, its own series (CN-…), never
	// reusing the invoice series.
	CreditNoteNumber string `gorm:"uniqueIndex;not null" json:"creditNoteNumber"`
	// SourceKey is the idempotency key of the refund this note reverses.
	SourceKey string `gorm:"uniqueIndex;not null" json:"sourceKey"`

	CustomerID uuid.UUID  `gorm:"type:uuid;not null;index" json:"customerId"`
	ChefID     *uuid.UUID `gorm:"type:uuid;index" json:"chefId,omitempty"`
	// What was refunded. Exactly one of these is set for a meal-plan day note.
	MealPlanID    *uuid.UUID `gorm:"type:uuid;index" json:"mealPlanId,omitempty"`
	MealPlanDayID *uuid.UUID `gorm:"type:uuid;index" json:"mealPlanDayId,omitempty"`
	OrderID       *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`
	// Reference is the human reference of the reversed document (the plan number).
	Reference string `gorm:"" json:"reference,omitempty"`

	// Amounts (₹). TotalAmount is what went back to the customer; TaxAmount is the GST slice
	// of it that this note reverses; TaxableValue is the rest (TotalAmount − TaxAmount).
	Currency     string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`
	TaxableValue float64 `gorm:"default:0" json:"taxableValue"`
	TaxAmount    float64 `gorm:"default:0" json:"taxAmount"`
	TotalAmount  float64 `gorm:"default:0" json:"totalAmount"`
	// RefundPercent records the agreed proportion the note was computed at, so a note can be
	// reconciled against the refund without re-deriving it.
	RefundPercent int    `gorm:"default:0" json:"refundPercent"`
	Reason        string `gorm:"type:text" json:"reason,omitempty"`

	IssuedAt  time.Time `gorm:"not null" json:"issuedAt"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// BeforeCreate assigns the id when the caller left it unset — see ChefPenalty.BeforeCreate.
func (n *CreditNote) BeforeCreate(*gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
