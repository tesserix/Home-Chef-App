package models

import (
	"time"

	"github.com/google/uuid"
)

// TipStatus tracks a post-delivery tip charge through the gateway.
type TipStatus string

const (
	TipPending TipStatus = "pending" // gateway order created, awaiting capture
	TipPaid    TipStatus = "paid"    // captured + split to the beneficiaries
	TipFailed  TipStatus = "failed"
)

// Tip is an optional, post-delivery, 100%-pass-through gratuity the customer pays
// to the chef and/or the delivery rider (#45). One Tip row = one Cashfree charge
// whose Easy Split allocates 100% to the beneficiaries' vendor accounts — no
// platform commission and no tax (mirrors the checkout-time tip's tax exclusion).
// The split is applied at capture, since delivery has already happened.
type Tip struct {
	// Live/test data partition. See models.ModePartition.
	ModePartition

	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID    uuid.UUID `gorm:"type:uuid;index;not null" json:"orderId"`
	CustomerID uuid.UUID `gorm:"type:uuid;index;not null" json:"customerId"`

	ChefAmount  float64 `gorm:"default:0" json:"chefAmount"`
	RiderAmount float64 `gorm:"default:0" json:"riderAmount"`
	Amount      float64 `gorm:"not null" json:"amount"` // chef + rider
	Currency    string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	// Beneficiary User IDs (for notifications + "tips received"). Nil when that
	// party isn't tipped or has no linked account.
	ChefUserID  *uuid.UUID `gorm:"type:uuid;index" json:"chefUserId,omitempty"`
	RiderUserID *uuid.UUID `gorm:"type:uuid;index" json:"riderUserId,omitempty"`

	Status            TipStatus `gorm:"type:varchar(12);index;default:'pending'" json:"status"`
	GatewayOrderID   string    `gorm:"uniqueIndex" json:"gatewayOrderId,omitempty"`
	GatewayPaymentID string    `gorm:"" json:"gatewayPaymentId,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Order *Order `gorm:"foreignKey:OrderID" json:"order,omitempty"`
}
