package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// order_issue.go — live order issue reporting → instant refund (#37). A customer
// reports a problem (missing item / quality issue) on their order; small, clear
// cases are auto-refunded to the store-credit wallet (#33), larger ones go to
// assisted (admin) review. Distinct from the generic SupportTicket.

// IssueReason is why the customer is reporting.
type IssueReason string

const (
	IssueMissingItem  IssueReason = "missing_item"
	IssueQualityIssue IssueReason = "quality_issue"
	IssueWrongItem    IssueReason = "wrong_item"
	IssueDamaged      IssueReason = "damaged"
	IssueOther        IssueReason = "other"
	// IssueDeliveryFailed is a SYSTEM-opened issue (not customer-reported) marking a
	// terminally-failed delivery that needs admin fault resolution (#393). It is the
	// dispute signal that freezes the order's payout hold and surfaces it in the admin
	// payout/dispute queue; the reported reason + suggested fault ride in Description.
	IssueDeliveryFailed IssueReason = "delivery_failed"
)

// ValidIssueReason reports whether r is a customer-reportable reason. Note:
// IssueDeliveryFailed is deliberately excluded — it is system-opened only, so the
// customer issue-report endpoint cannot forge one.
func ValidIssueReason(r IssueReason) bool {
	switch r {
	case IssueMissingItem, IssueQualityIssue, IssueWrongItem, IssueDamaged, IssueOther:
		return true
	}
	return false
}

// IssueStatus tracks the lifecycle of a reported issue.
type IssueStatus string

const (
	IssuePending      IssueStatus = "pending"       // awaiting assisted (admin) review
	IssueAutoRefunded IssueStatus = "auto_refunded" // small/clear case refunded instantly
	IssueResolved     IssueStatus = "resolved"      // admin approved a refund
	IssueRejected     IssueStatus = "rejected"      // admin declined
)

// IssueFaultPolicy decides who bears a resolved issue's refund (#618). Default is
// full clawback from the chef (a customer issue is the chef's fault by default);
// the admin can override to platform goodwill (refund the customer, chef keeps
// their payout, the platform absorbs the cost).
type IssueFaultPolicy string

const (
	// FaultChefClawback — refund the customer AND claw back the chef's payout (default).
	FaultChefClawback IssueFaultPolicy = "chef_clawback"
	// FaultPlatformGoodwill — refund the customer but do NOT claw back the chef's payout.
	FaultPlatformGoodwill IssueFaultPolicy = "platform_goodwill"
)

// OrderIssue is a customer-reported problem with one of their orders.
type OrderIssue struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID    uuid.UUID `gorm:"type:uuid;not null;index" json:"orderId"`
	ChefID     uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customerId"`

	Reason      IssueReason    `gorm:"type:varchar(20);not null" json:"reason"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	PhotoURLs   pq.StringArray `gorm:"type:text[]" json:"photoUrls"`
	// AffectedItemIDs are the order-item IDs the customer flagged (for missing /
	// damaged / wrong item); empty for a whole-order quality complaint.
	AffectedItemIDs pq.StringArray `gorm:"type:text[]" json:"affectedItemIds"`

	// RequestedAmount is the system-computed refundable amount from the affected
	// items; RefundAmount is what was actually credited (after caps / admin).
	RequestedAmount float64     `gorm:"default:0" json:"requestedAmount"`
	RefundAmount    float64     `gorm:"default:0" json:"refundAmount"`
	Status          IssueStatus `gorm:"type:varchar(16);not null;default:'pending'" json:"status"`

	ResolvedBy  *uuid.UUID `gorm:"type:uuid" json:"-"`
	ResolvedAt  *time.Time `gorm:"" json:"resolvedAt,omitempty"`
	RefundTxnID *uuid.UUID `gorm:"type:uuid" json:"-"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// BeforeCreate mints the UUID in Go so the model works without the Postgres
// gen_random_uuid() default (e.g. in sqlite-backed unit tests).
func (i *OrderIssue) BeforeCreate(*gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}
