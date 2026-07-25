package payouts

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EntryKind names why an amount entered the ledger. The prefix carries the
// direction so a new kind cannot be added without declaring its sign.
type EntryKind string

const (
	// Credits — amounts owed to the payee.
	EntryCreditOrderNet       EntryKind = "credit.order_net"
	EntryCreditAdjustment     EntryKind = "credit.adjustment"
	EntryCreditPayoutReversed EntryKind = "credit.payout_reversed"
	EntryCreditReserveRelease EntryKind = "credit.reserve_release"

	// Debits — amounts withheld from the payee.
	EntryDebitPayout     EntryKind = "debit.payout"
	EntryDebitPenalty    EntryKind = "debit.penalty"
	EntryDebitClawback   EntryKind = "debit.clawback"
	EntryDebitReserve    EntryKind = "debit.reserve"
	EntryDebitAdjustment EntryKind = "debit.adjustment"
)

// Direction is the sign an entry kind contributes to a balance.
type Direction int

const (
	DirectionCredit Direction = 1
	DirectionDebit  Direction = -1
)

// Direction derives the sign from the kind's prefix.
func (k EntryKind) Direction() (Direction, error) {
	switch {
	case strings.HasPrefix(string(k), "credit."):
		return DirectionCredit, nil
	case strings.HasPrefix(string(k), "debit."):
		return DirectionDebit, nil
	default:
		return 0, fmt.Errorf("payouts: entry kind %q has no direction prefix", k)
	}
}

// LedgerEntry is one immutable movement in a payee's payable ledger.
//
// Entries are append-only: there is no UpdatedAt, no soft delete and no update
// path. A mistake is corrected by writing a compensating entry, so the history
// of what the platform believed it owed is always reconstructable. Balances are
// always derived (DeriveBalance) and never stored in a mutable column.
type LedgerEntry struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID string    `gorm:"type:varchar(64);not null;index:idx_payout_ledger_payee" json:"tenantId"`

	PayeeType PayeeType `gorm:"type:varchar(32);not null;index:idx_payout_ledger_payee" json:"payeeType"`
	PayeeID   uuid.UUID `gorm:"type:uuid;not null;index:idx_payout_ledger_payee" json:"payeeId"`

	Kind        EntryKind `gorm:"type:varchar(48);not null;index" json:"kind"`
	AmountMinor int64     `gorm:"not null" json:"amountMinor"`
	Currency    Currency  `gorm:"type:varchar(3);not null" json:"currency"`

	// SourceType/SourceID point back at the domain object that caused this
	// entry (an order, a meal-plan day, an order-issue, a batch). Together
	// with Kind they form the natural dedupe key for the ledger projector.
	SourceType string `gorm:"type:varchar(48);not null;index:idx_payout_ledger_source" json:"sourceType"`
	SourceID   string `gorm:"type:varchar(64);not null;index:idx_payout_ledger_source" json:"sourceId"`

	// MaturesAt is when a credit becomes sweepable. Debits apply immediately
	// and leave this nil.
	MaturesAt *time.Time `gorm:"index" json:"maturesAt,omitempty"`

	// BatchID is set when the entry is consumed by (or produced by) a batch.
	BatchID *uuid.UUID `gorm:"type:uuid;index" json:"batchId,omitempty"`

	// ActorID is the admin who caused a manual entry; nil for automatic ones.
	ActorID *uuid.UUID `gorm:"type:uuid" json:"actorId,omitempty"`
	Reason  string     `gorm:"type:text" json:"reason,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

func (LedgerEntry) TableName() string { return "payout_ledger_entries" }

// Amount returns the entry's signed contribution to a balance: credits
// positive, debits negative, regardless of how the amount was stored.
func (e LedgerEntry) Amount() (Money, error) {
	dir, err := e.Kind.Direction()
	if err != nil {
		return Money{}, err
	}
	// Amounts are stored as magnitudes. Storing a negative magnitude would
	// make a debit behave as a credit, so reject it rather than pay it out.
	if e.AmountMinor < 0 {
		return Money{}, fmt.Errorf("payouts: entry %s has negative magnitude %d", e.ID, e.AmountMinor)
	}
	return Money{Minor: e.AmountMinor * int64(dir), Currency: e.Currency}, nil
}

// Balance is a payee's position at a point in time.
type Balance struct {
	// Matured is what may be swept now: matured credits less all debits.
	// It goes negative when debits exceed credits — see Recovery.
	Matured Money `json:"matured"`

	// Maturing is credited but not yet sweepable.
	Maturing Money `json:"maturing"`

	// Total is Matured + Maturing: everything the platform owes, or is owed.
	Total Money `json:"total"`
}

// Payable is what a batch may actually pay: Matured, floored at zero.
func (b Balance) Payable() Money { return b.Matured.ClampAtZero() }

// Recovery is the amount the payee owes the platform, as a positive number.
// Non-zero recovery blocks payouts and flags manual review (#745); the engine
// never emits a negative payout.
func (b Balance) Recovery() Money {
	if b.Matured.IsNegative() {
		return b.Matured.Neg()
	}
	return Zero(b.Matured.Currency)
}

// DeriveBalance folds entries into a balance as of a moment in time.
//
// Credits maturing after asOf count as Maturing; everything else, including
// every debit, applies immediately. Debits apply even when they exceed matured
// credits, which is precisely how a recovery balance arises: a penalty raised
// against a payee with nothing owed must not silently vanish.
//
// The fold is order-independent — the property test asserts that any shuffling
// of the same entries yields the same balance.
func DeriveBalance(entries []LedgerEntry, asOf time.Time) (Balance, error) {
	matured := Money{}
	maturing := Money{}

	for _, e := range entries {
		amt, err := e.Amount()
		if err != nil {
			return Balance{}, err
		}
		isMaturingCredit := amt.IsPositive() && e.MaturesAt != nil && e.MaturesAt.After(asOf)
		if isMaturingCredit {
			if maturing, err = maturing.Add(amt); err != nil {
				return Balance{}, err
			}
			continue
		}
		if matured, err = matured.Add(amt); err != nil {
			return Balance{}, err
		}
	}

	total, err := matured.Add(maturing)
	if err != nil {
		return Balance{}, err
	}
	return Balance{Matured: matured, Maturing: maturing, Total: total}, nil
}
