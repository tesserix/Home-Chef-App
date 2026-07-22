package payouts

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BatchState is where a payout batch sits in its lifecycle.
type BatchState string

const (
	// BatchBuilding — items are being gathered; nothing is committed.
	BatchBuilding BatchState = "building"
	// BatchPendingApproval — guardrails said queue, or the mode is manual.
	BatchPendingApproval BatchState = "pending_approval"
	// BatchApproved — cleared to execute, but no money has moved.
	BatchApproved BatchState = "approved"
	// BatchExecuting — handed to the provider; outcome not yet known. This is
	// the only state where an ambiguous provider response is possible, and the
	// only one from which GetPayoutByReference must be consulted before retry.
	BatchExecuting BatchState = "executing"
	// BatchPaid — provider confirmed terminal success.
	BatchPaid BatchState = "paid"
	// BatchFailed — provider confirmed terminal failure; funds never left.
	BatchFailed BatchState = "failed"
	// BatchReversed — money left and came back (dead account, bank reject).
	BatchReversed BatchState = "reversed"
	// BatchCancelled — abandoned before execution.
	BatchCancelled BatchState = "cancelled"
)

// batchTransitions is the whole state machine. Anything not listed is refused.
//
// Two properties are deliberate:
//   - paid -> reversed is legal, because a bank can return money days later.
//   - executing -> cancelled is NOT legal. Once the provider has been called,
//     the outcome must be established, never assumed away.
var batchTransitions = map[BatchState][]BatchState{
	BatchBuilding:        {BatchPendingApproval, BatchApproved, BatchCancelled},
	BatchPendingApproval: {BatchApproved, BatchCancelled},
	BatchApproved:        {BatchExecuting, BatchCancelled},
	BatchExecuting:       {BatchPaid, BatchFailed},
	BatchPaid:            {BatchReversed},
	BatchFailed:          {},
	BatchReversed:        {},
	BatchCancelled:       {},
}

// IsTerminal reports whether no further transition is possible.
func (s BatchState) IsTerminal() bool {
	next, ok := batchTransitions[s]
	return ok && len(next) == 0
}

// CanTransition reports whether s -> to is a legal move.
func (s BatchState) CanTransition(to BatchState) bool {
	for _, allowed := range batchTransitions[s] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Transition validates a move and returns the new state.
func (s BatchState) Transition(to BatchState) (BatchState, error) {
	if !s.CanTransition(to) {
		return s, fmt.Errorf("payouts: illegal batch transition %s -> %s", s, to)
	}
	return to, nil
}

// Batch is one payee's payout for one business date.
//
// The (tenant, payee, business date) uniqueness is enforced by a database
// index, not only by Temporal's workflow-ID dedupe. Two independent guarantees
// are warranted here: a duplicate batch is a duplicate payment.
type Batch struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_payout_batch_unique" json:"tenantId"`

	PayeeType PayeeType `gorm:"type:varchar(32);not null;uniqueIndex:idx_payout_batch_unique" json:"payeeType"`
	PayeeID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_payout_batch_unique" json:"payeeId"`

	// BusinessDate is the sweep date in the platform's timezone, stored as
	// YYYY-MM-DD. A date rather than a timestamp, so a retried sweep on the
	// same day collides by construction.
	BusinessDate string `gorm:"type:date;not null;uniqueIndex:idx_payout_batch_unique" json:"businessDate"`

	State BatchState `gorm:"type:varchar(24);not null;index" json:"state"`

	AmountMinor int64    `gorm:"not null" json:"amountMinor"`
	Currency    Currency `gorm:"type:varchar(3);not null" json:"currency"`

	MethodID *uuid.UUID `gorm:"type:uuid" json:"methodId,omitempty"`

	// Provider is the adapter that executed (or will execute) this batch.
	Provider string `gorm:"type:varchar(32)" json:"provider,omitempty"`
	// ProviderRef is the rail's payout id; ProviderUTR the bank reference.
	ProviderRef string `gorm:"type:varchar(128);index" json:"providerRef,omitempty"`
	ProviderUTR string `gorm:"type:varchar(128)" json:"providerUtr,omitempty"`

	// IdempotencyKey is derived from the batch identity and sent to the rail.
	// It is what GetPayoutByReference is queried with after an ambiguous call.
	IdempotencyKey string `gorm:"type:varchar(64);uniqueIndex" json:"idempotencyKey"`

	// Decision records why the batch went where it did.
	Decision    Decision `gorm:"type:smallint" json:"decision"`
	ReasonCodes string   `gorm:"type:text" json:"reasonCodes,omitempty"`

	ApprovedBy *uuid.UUID `gorm:"type:uuid" json:"approvedBy,omitempty"`
	ApprovedAt *time.Time `json:"approvedAt,omitempty"`

	FailureCode   string `gorm:"type:varchar(64)" json:"failureCode,omitempty"`
	FailureDetail string `gorm:"type:text" json:"failureDetail,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (Batch) TableName() string { return "payout_batches" }

// Amount is the batch total.
func (b Batch) Amount() Money {
	return Money{Minor: b.AmountMinor, Currency: b.Currency}
}

// BatchItem links a batch to the ledger entries it settles. Keeping the link
// explicit means a batch can be reconstructed and re-verified long after the
// fact, which reconciliation (#746) depends on.
type BatchItem struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BatchID uuid.UUID `gorm:"type:uuid;not null;index" json:"batchId"`

	// LedgerEntryID is unique across items: one ledger entry can only ever be
	// settled by one batch.
	LedgerEntryID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"ledgerEntryId"`

	AmountMinor int64    `gorm:"not null" json:"amountMinor"`
	Currency    Currency `gorm:"type:varchar(3);not null" json:"currency"`

	CreatedAt time.Time `json:"createdAt"`
}

func (BatchItem) TableName() string { return "payout_batch_items" }

// BusinessDateFormat is the layout for Batch.BusinessDate.
const BusinessDateFormat = "2006-01-02"

// FormatBusinessDate renders a time as a business date in the given location.
func FormatBusinessDate(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format(BusinessDateFormat)
}

// IdempotencyKeyFor derives the rail idempotency key for a batch identity.
// It is a pure function of identity, so a retry after a crash reproduces the
// exact same key without reading anything back.
func IdempotencyKeyFor(tenantID string, ref PayeeRef, businessDate string) string {
	return fmt.Sprintf("payout:%s:%s:%s:%s", tenantID, ref.Type, ref.ID, businessDate)
}
