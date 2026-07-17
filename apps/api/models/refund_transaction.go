package models

// refund_transaction.go — the refund ledger (#689, part of #687).
//
// One row per gateway refund ATTEMPT. HomeChef had no ledger: refund_reserve.go
// reserves on the ORDER row, so there was no per-attempt audit trail and no way
// to count in-flight refunds when validating a cap. mark8ly's coordinator has
// one (payment.RefundTransaction) and that is what lets it answer "how much is
// already in flight for this order" before calling the gateway.
//
// The row is written BEFORE the gateway call (status=pending) and finalized
// after (succeeded/failed). That ordering is the point: a pending row is a claim
// that survives a crash mid-gateway-call, so a sweep can reconcile it instead of
// the money silently vanishing from our books.

import (
	"time"

	"github.com/google/uuid"
)

// RefundTxnStatus is the ledger row's lifecycle.
type RefundTxnStatus string

const (
	// RefundTxnPending — reserved, gateway call not yet known to have landed.
	// A row stuck here after a crash is exactly what a reconcile sweep looks for.
	RefundTxnPending RefundTxnStatus = "pending"
	// RefundTxnSucceeded — the gateway accepted it; ProviderRefundID is set.
	RefundTxnSucceeded RefundTxnStatus = "succeeded"
	// RefundTxnFailed — the gateway rejected it; the reservation was released.
	RefundTxnFailed RefundTxnStatus = "failed"
)

// RefundTransaction is one attempt to move refund money for an order.
type RefundTransaction struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID uuid.UUID `gorm:"type:uuid;not null;index" json:"orderId"`

	// Provider is where the money went: "razorpay" | "stripe" | "wallet".
	// Wallet is a legitimate destination when there is no gateway payment to
	// refund against — it is NOT the odd case; refunding a wallet-paid order to
	// the wallet is correct. (The bug in #691 is a path that ignores this and
	// always picks wallet.)
	Provider string `gorm:"type:varchar(20);not null" json:"provider"`

	// ProviderPaymentID is the payment being refunded (e.g. pay_xxx). Empty for
	// a wallet refund.
	ProviderPaymentID string `gorm:"type:varchar(64)" json:"-"`

	// ProviderRefundID is the gateway's id for the refund (e.g. rfnd_xxx), or
	// "wallet:<txn-id>". Set on success. Never exposed — the gateway ids stay
	// server-side (models.Order tags RazorpayPaymentID `json:"-"` for the same
	// reason).
	ProviderRefundID string `gorm:"type:varchar(64)" json:"-"`

	// Amount is in major units (rupees) to match models.Order.Total and the rest
	// of the refund paths. The gateway call converts to paise at the boundary
	// (services.ToPaise) — mixing units inside the ledger is how rounding bugs
	// start.
	Amount       float64 `gorm:"not null" json:"amount"`
	CurrencyCode string  `gorm:"type:char(3);not null;default:'INR'" json:"currencyCode"`

	Status RefundTxnStatus `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`
	Reason string          `gorm:"type:varchar(200)" json:"reason,omitempty"`

	// IdempotencyKey is the gateway-facing key. UNIQUE: this is the ledger's
	// half of double-refund protection. The gateway dedups a retry with the same
	// key; this stops us even reserving twice for one logical refund.
	IdempotencyKey string `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`

	// ScopeID is the logical refund's identity — "cancel", "issue:<id>",
	// "line:<id>", "admin:<id>". Charset-restricted [A-Za-z0-9_-] because
	// Razorpay's X-Refund-Idempotency requires it (validated in the coordinator,
	// not here — a model shouldn't reject data the DB accepts).
	ScopeID string `gorm:"type:varchar(80);not null;index" json:"scopeId"`

	// Actor is who triggered it, for audit: "customer", "chef", "admin",
	// "system", or "user:<id>".
	Actor string `gorm:"type:varchar(64)" json:"actor,omitempty"`

	// FailureReason records why the gateway rejected it — the thing you actually
	// want when reconciling a failed refund.
	FailureReason string `gorm:"type:text" json:"failureReason,omitempty"`

	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

func (RefundTransaction) TableName() string { return "refund_transactions" }

// IsTerminal reports whether the row is finished — a pending row is either in
// flight or stranded by a crash, and either way must still count against the
// order's refundable balance.
func (r RefundTransaction) IsTerminal() bool {
	return r.Status == RefundTxnSucceeded || r.Status == RefundTxnFailed
}
