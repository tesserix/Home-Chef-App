package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChefPenaltyStatus is the lifecycle of a levy.
type ChefPenaltyStatus string

const (
	// ChefPenaltyPending — levied, not yet taken out of a settlement.
	ChefPenaltyPending ChefPenaltyStatus = "pending"
	// ChefPenaltyDeducted — netted off a weekly settlement statement (DeductedStatementID).
	ChefPenaltyDeducted ChefPenaltyStatus = "deducted"
	// ChefPenaltyWaived — an admin cancelled it; never deducted.
	ChefPenaltyWaived ChefPenaltyStatus = "waived"
)

// ChefPenaltyKind identifies why a levy was raised. One value today; typed so a second
// accountability rule doesn't have to reinterpret a free-text reason.
type ChefPenaltyKind string

// ChefPenaltyCancelLate — the chef cancelled a confirmed order close to service (#834).
const ChefPenaltyCancelLate ChefPenaltyKind = "cancel_late"

// ChefPenalty is a charge raised against a chef and netted off their next settlement
// (#834 item 6). Nothing of the kind existed before v3 — services/payout_recovery.go recovers
// FAILED payouts, it does not levy — so this is the whole mechanism: the ledger row, the
// deduction, the statement line, and the audit trail of who waived what.
//
// GUARDS, deliberate: a levy is skipped entirely for the first few cancellations in a rolling
// window (config), and an admin can waive any levy. Without both, a genuine emergency is
// auto-fined with no recourse — which costs chefs faster than the penalty saves.
type ChefPenalty struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChefID uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	// UserID is the chef's user id, denormalized so the settlement join (which is by user)
	// and the notification path don't need a second lookup.
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`

	Kind   ChefPenaltyKind   `gorm:"type:varchar(24);not null;index" json:"kind"`
	Status ChefPenaltyStatus `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`

	// SourceKey is the natural key of the event that raised the levy
	// ("chefcancel:<orderID>"), UNIQUE so a retried or concurrent cancel levies once.
	SourceKey string `gorm:"uniqueIndex;not null" json:"sourceKey"`

	OrderID *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`
	// Reference is the human reference of the cancelled document (the order number).
	Reference string `gorm:"" json:"reference,omitempty"`

	// Arithmetic, frozen at levy time so a later config change never restates a raised
	// penalty: Amount = round2(BasisAmount × RatePercent / 100).
	Currency    string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`
	BasisAmount float64 `gorm:"default:0" json:"basisAmount"`
	RatePercent float64 `gorm:"default:0" json:"ratePercent"`
	Amount      float64 `gorm:"default:0" json:"amount"`
	// LeadHours is how much notice the customer got — the reason this levy applied.
	LeadHours float64 `gorm:"default:0" json:"leadHours"`
	Reason    string  `gorm:"type:text" json:"reason,omitempty"`

	// Settlement linkage, set when the levy is netted off a weekly statement.
	DeductedStatementID *uuid.UUID `gorm:"type:uuid;index" json:"deductedStatementId,omitempty"`
	DeductedAt          *time.Time `gorm:"" json:"deductedAt,omitempty"`

	// Waiver audit.
	WaivedBy    *uuid.UUID `gorm:"type:uuid" json:"waivedBy,omitempty"`
	WaivedAt    *time.Time `gorm:"" json:"waivedAt,omitempty"`
	WaiveReason string     `gorm:"type:text" json:"waiveReason,omitempty"`
	OccurredAt  time.Time  `gorm:"not null;index" json:"occurredAt"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

// BeforeCreate assigns the id when the caller left it unset — the sqlite test harness has no
// gen_random_uuid(). Postgres uses the column default in prod. Mirrors the hook pattern used
// across the models. It also means a caller can use p.ID (waiver, audit) straight after Create.
func (p *ChefPenalty) BeforeCreate(*gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
