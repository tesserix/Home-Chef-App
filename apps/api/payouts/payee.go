package payouts

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PayeeType names a kind of party the engine can pay. Adding one must not
// require changing anything in this package — that is the whole point of the
// adapter registry, and #748 (delivery partners) is the proof.
type PayeeType string

const (
	PayeeChef            PayeeType = "chef"
	PayeeDeliveryPartner PayeeType = "delivery_partner"
)

// PayeeRef identifies a payee uniquely across types.
type PayeeRef struct {
	Type PayeeType `json:"type"`
	ID   uuid.UUID `json:"id"`
}

func (p PayeeRef) String() string { return string(p.Type) + ":" + p.ID.String() }

// PayeeProfile is everything the engine needs to know about a payee to decide
// whether it may pay them. Domain adapters project their own models onto this;
// the engine never sees a ChefProfile or a DeliveryPartner.
type PayeeProfile struct {
	Ref PayeeRef

	// DisplayName is used in statements, notifications and the admin queue.
	DisplayName string

	// Active is false when the payee is suspended, deactivated or otherwise
	// barred from trading. A false here blocks payment outright.
	Active bool

	// KYCVerified reports whether identity verification has completed. Money
	// must not leave to an unverified party.
	KYCVerified bool

	// TaxID is the PAN (or equivalent) used for withholding and for matching
	// against the payout method's beneficiary name.
	TaxID string

	// TenantID scopes every ledger, method and batch row this payee produces.
	TenantID string
}

// Credit is a domain-sourced amount owed to a payee, projected into the ledger.
// MaturesAt is when it becomes sweepable — the window during which a late
// refund or fraud signal can still cancel it before the money is irreversible.
type Credit struct {
	Kind        EntryKind
	Amount      Money
	SourceType  string
	SourceID    string
	MaturesAt   time.Time
	Description string
}

// Debit is an amount to be withheld from a payee: a penalty, a clawback, or a
// manual adjustment.
type Debit struct {
	Kind        EntryKind
	Amount      Money
	SourceType  string
	SourceID    string
	Description string
}

// PayeeAdapter is the single seam through which domain logic enters the engine.
//
// An implementation lives outside this package (in services/, wired at
// startup) and is free to import whatever domain models it needs. This package
// must never import an implementation.
type PayeeAdapter interface {
	// Type is the payee type this adapter serves.
	Type() PayeeType

	// Resolve returns the payee's current standing.
	Resolve(ctx context.Context, id uuid.UUID) (PayeeProfile, error)

	// MaturedCredits returns amounts owed that are sweepable as of asOf.
	MaturedCredits(ctx context.Context, id uuid.UUID, asOf time.Time) ([]Credit, error)

	// OpenDebits returns unsettled penalties and clawbacks. Debits under
	// appeal must be withheld here until the appeal window closes (#745).
	OpenDebits(ctx context.Context, id uuid.UUID) ([]Debit, error)

	// Rules returns payee-type-specific guardrails, evaluated after the
	// engine's own. FSSAI expiry for chefs, document expiry for drivers.
	Rules() []Rule
}

// Registry maps payee types to adapters. It is written once at startup and
// read concurrently by workflow activities, so reads take a read lock.
type Registry struct {
	mu       sync.RWMutex
	adapters map[PayeeType]PayeeAdapter
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{adapters: make(map[PayeeType]PayeeAdapter)}
}

// Register adds an adapter. Registering the same type twice is a wiring bug
// and is reported rather than silently overwriting: a shadowed adapter would
// mean paying the wrong party from the wrong ledger.
func (r *Registry) Register(a PayeeAdapter) error {
	if a == nil {
		return fmt.Errorf("payouts: cannot register a nil adapter")
	}
	t := a.Type()
	if t == "" {
		return fmt.Errorf("payouts: adapter has an empty payee type")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[t]; exists {
		return fmt.Errorf("payouts: adapter for payee type %q already registered", t)
	}
	r.adapters[t] = a
	return nil
}

// Get returns the adapter for a payee type.
func (r *Registry) Get(t PayeeType) (PayeeAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[t]
	if !ok {
		return nil, fmt.Errorf("payouts: no adapter registered for payee type %q", t)
	}
	return a, nil
}

// Types lists registered payee types in a stable order, so a sweep covers them
// deterministically and test output does not depend on map iteration.
func (r *Registry) Types() []PayeeType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PayeeType, 0, len(r.adapters))
	for t := range r.adapters {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
