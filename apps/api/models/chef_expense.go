package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChefExpenseCategory buckets a chef's self-declared business expense.
type ChefExpenseCategory string

const (
	ExpenseIngredients ChefExpenseCategory = "ingredients"
	ExpenseGas         ChefExpenseCategory = "gas"
	ExpenseUtensils    ChefExpenseCategory = "utensils"
	ExpensePackaging   ChefExpenseCategory = "packaging"
	ExpenseTransport   ChefExpenseCategory = "transport"
	ExpenseEquipment   ChefExpenseCategory = "equipment"
	ExpenseUtilities   ChefExpenseCategory = "utilities"
	ExpenseOther       ChefExpenseCategory = "other"
)

// ValidExpenseCategory reports whether c is one of the known buckets.
func ValidExpenseCategory(c ChefExpenseCategory) bool {
	switch c {
	case ExpenseIngredients, ExpenseGas, ExpenseUtensils, ExpensePackaging,
		ExpenseTransport, ExpenseEquipment, ExpenseUtilities, ExpenseOther:
		return true
	}
	return false
}

// ChefExpense is one self-declared business expense a chef records for their
// own books (gas refills, ingredients, utensils, packaging …). These feed the
// expense analytics and the annual FY statement; they never touch settlement
// math — payouts are computed from orders alone.
type ChefExpense struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index:idx_chef_expense_chef_date" json:"chefId"`
	UserID uuid.UUID `gorm:"type:uuid;not null" json:"userId"`

	Category ChefExpenseCategory `gorm:"type:varchar(20);not null" json:"category"`
	Amount   float64             `gorm:"not null" json:"amount"`
	Currency string              `gorm:"type:varchar(3);default:'INR'" json:"currency"`
	Note     string              `gorm:"type:varchar(500)" json:"note,omitempty"`

	// OrderID optionally ties the expense to the order it was incurred for
	// (recorded from the order screen while cooking). Purely informational —
	// analytics and the FY statement sum expenses the same either way.
	OrderID *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`

	// ExpenseDate is the day the expense was incurred (date-only, stored UTC
	// midnight IST). FY bucketing uses this, not CreatedAt.
	ExpenseDate time.Time `gorm:"not null;index:idx_chef_expense_chef_date" json:"expenseDate"`

	// ReceiptPath is the private-bucket object path of an optional uploaded
	// bill/receipt image. Never a fetchable URL — readers (chef app, admin)
	// get a short-lived signed URL derived from it at response time. Column
	// name kept from the first cut of the schema.
	ReceiptPath string `gorm:"column:receipt_url;type:varchar(500)" json:"receiptPath,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// BeforeCreate assigns the ID app-side when the DB default can't (sqlite tests).
func (e *ChefExpense) BeforeCreate(_ *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
