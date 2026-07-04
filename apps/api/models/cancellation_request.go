package models

import (
	"time"

	"github.com/google/uuid"
)

// CancellationRequestStatus tracks a cancellation through the vendor-arbitration
// state machine (epic #475).
type CancellationRequestStatus string

const (
	// CancelReqPendingVendor: awaiting the vendor to confirm + pick a refund tier.
	CancelReqPendingVendor CancellationRequestStatus = "pending_vendor"
	// CancelReqAutoRefunded: a fast-path (chef not accepted / rejected) — full food
	// refund, no vendor needed.
	CancelReqAutoRefunded CancellationRequestStatus = "auto_refunded"
	// CancelReqApproved: the vendor confirmed and a tiered refund was issued.
	CancelReqApproved CancellationRequestStatus = "approved"
	// CancelReqDisputed: the customer disputed the vendor's tier → admin.
	CancelReqDisputed CancellationRequestStatus = "disputed"
	// CancelReqAdminReview: vendor timed out → admin decides.
	CancelReqAdminReview CancellationRequestStatus = "admin_review"
	// CancelReqResolved: an admin settled a dispute/timeout.
	CancelReqResolved CancellationRequestStatus = "resolved"
	// CancelReqNotAllowed: the order was already made/dispatched — not cancellable.
	CancelReqNotAllowed CancellationRequestStatus = "not_allowed"
)

// CancellationRequest is a customer's request to cancel an order, arbitrated by
// the vendor with a reason-based tiered refund (#475). The refund breakdown is a
// SNAPSHOT (paise) computed when the tier is decided; RefundExecuted flips true
// only once the money has actually moved (#477). Money invariant on every
// resolved request: order grand total == RefundTotalPaise + VendorKeptPaise +
// PlatformKeptPaise, and PlatformKeptPaise always includes the platform fee.
type CancellationRequest struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"orderId"`
	CustomerID uuid.UUID `gorm:"type:uuid;index;not null" json:"customerId"`
	ChefID     uuid.UUID `gorm:"type:uuid;index;not null" json:"chefId"`

	Status CancellationRequestStatus `gorm:"type:varchar(24);index;default:'pending_vendor'" json:"status"`

	CustomerReason string `gorm:"type:text" json:"customerReason,omitempty"`
	// VendorReason is the tier reason the vendor picked — one of the
	// services.CancellationReason values (not_started / materials_purchased /
	// in_preparation / ready). Stored as a string to keep models free of a
	// services dependency.
	VendorReason string `gorm:"type:varchar(32)" json:"vendorReason,omitempty"`
	// RefundDestination — "wallet" (instant) or "original" (gateway). Customer picks.
	RefundDestination string `gorm:"type:varchar(16)" json:"refundDestination,omitempty"`

	// Refund snapshot (paise), computed from the order + tier.
	FoodRefundPaise     int `gorm:"default:0" json:"foodRefundPaise"`
	DeliveryRefundPaise int `gorm:"default:0" json:"deliveryRefundPaise"`
	TaxRefundPaise      int `gorm:"default:0" json:"taxRefundPaise"`
	RefundTotalPaise    int `gorm:"default:0" json:"refundTotalPaise"`
	VendorKeptPaise     int `gorm:"default:0" json:"vendorKeptPaise"`
	PlatformKeptPaise   int `gorm:"default:0" json:"platformKeptPaise"`

	// RefundExecuted flips true only after the money has moved (#477); RefundRef
	// is the wallet-ledger reference or the gateway refund id.
	RefundExecuted bool   `gorm:"default:false" json:"refundExecuted"`
	RefundRef      string `gorm:"type:varchar(64)" json:"refundRef,omitempty"`

	// Dispute / admin arbitration.
	Disputed        bool       `gorm:"default:false" json:"disputed"`
	DisputeReason   string     `gorm:"type:text" json:"disputeReason,omitempty"`
	AdminResolvedBy *uuid.UUID `gorm:"type:uuid" json:"adminResolvedBy,omitempty"`
	AdminNote       string     `gorm:"type:text" json:"adminNote,omitempty"`

	// VendorRespondBy is the deadline; past it the request routes to admin.
	VendorRespondBy *time.Time `json:"vendorRespondBy,omitempty"`
	ResolvedAt      *time.Time `json:"resolvedAt,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
