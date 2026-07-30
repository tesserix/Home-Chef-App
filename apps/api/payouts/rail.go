package payouts

import (
	"context"
	"errors"
	"time"
)

// rail.go — the seam money actually leaves through.
//
// PayeeAdapter brings domain amounts INTO the engine; Rail takes settled amounts
// OUT of it. Keeping them as two interfaces is what lets the engine reason about
// what is owed without knowing whether it will be paid by Cashfree, RazorpayX or
// a bank file — and lets a rail be swapped without touching the ledger.
//
// Like PayeeAdapter, implementations live outside this package (in services/,
// wired at startup). This package must never import one.
//
// ── The ambiguity problem ────────────────────────────────────────────────────
//
// Everything below is shaped by one fact: a disbursement call can fail in a way
// that does not tell you whether money moved. A timeout, a dropped connection, a
// 502 from a load balancer — the request may have been executed in full, or never
// received. This is why Batch has an `executing` state that cannot transition to
// `cancelled`, and why the batch comment says GetPayoutByReference must be
// consulted before any retry.
//
// So the contract is:
//
//  1. Disburse is keyed by the batch's IdempotencyKey. Calling it twice with the
//     same key must never move money twice — the rail either dedupes it or the
//     implementation must make it so.
//  2. On ANY ambiguous error, the caller must NOT retry Disburse. It must call
//     GetPayoutByReference with the same key and let the rail state the truth.
//  3. A rail that cannot answer (2) is not safe to use for payouts, because the
//     only remaining options are to risk a double payment or to strand the payee.
type Rail interface {
	// Name identifies the rail, and is what Batch.Provider records. Stable —
	// it is persisted, so renaming it orphans historical batches.
	Name() string

	// EnsureBeneficiary registers (or re-registers) a destination and reports
	// the rail's opinion of it.
	//
	// MUST be idempotent on method.RailBeneficiaryID: called again for a
	// beneficiary that already exists, it returns the existing registration
	// rather than creating a second. This is the reason BeneficiaryIDFor is
	// deterministic.
	//
	// The instrument itself (account number / IFSC / VPA) is passed in
	// separately via Instrument rather than read from the method, because the
	// method deliberately does not hold it — see method.go.
	EnsureBeneficiary(ctx context.Context, req BeneficiaryRequest) (BeneficiaryResult, error)

	// Disburse sends one batch's amount to one destination.
	//
	// Returns ErrRailAmbiguous when the outcome is unknown. Callers MUST then
	// call GetPayoutByReference rather than retrying.
	Disburse(ctx context.Context, req DisburseRequest) (DisburseResult, error)

	// GetPayoutByReference resolves what actually happened to the disbursement
	// carrying idempotencyKey.
	//
	// Returns ErrRailNotFound when the rail has no record of it — which is the
	// ONLY safe proof that money did not move, and therefore the only condition
	// under which a caller may re-attempt Disburse.
	GetPayoutByReference(ctx context.Context, idempotencyKey string) (DisburseResult, error)
}

// Rail sentinel errors. Callers branch on these, so they are part of the contract.
var (
	// ErrRailAmbiguous — the call may or may not have moved money. Never retry
	// Disburse on this; resolve with GetPayoutByReference.
	ErrRailAmbiguous = errors.New("payouts: rail outcome ambiguous")

	// ErrRailNotFound — the rail has no record of this reference. Safe to
	// re-attempt.
	ErrRailNotFound = errors.New("payouts: no payout found for reference")

	// ErrRailNotConfigured — no usable credentials for this rail/mode.
	ErrRailNotConfigured = errors.New("payouts: rail not configured")

	// ErrBeneficiaryRejected — the rail refused the destination. Terminal until
	// the payee supplies new details; retrying cannot help.
	ErrBeneficiaryRejected = errors.New("payouts: beneficiary rejected by rail")
)

// Instrument is the actual destination, resolved at the moment of use.
//
// Deliberately a transient value, never persisted by the engine: it is read from
// Secret Manager, passed down, used, and dropped. Anything that holds one for
// longer than a single call is a leak of the most sensitive data the platform has.
type Instrument struct {
	Kind MethodKind

	// Bank — set when Kind is MethodBankAccount.
	AccountNumber string
	IFSC          string

	// UPI — set when Kind is MethodUPI.
	VPA string
}

// Valid reports whether the instrument carries what its kind requires. Checked
// before a rail call so a half-populated destination fails here, with a clear
// message, rather than as an opaque rail validation error.
func (i Instrument) Valid() bool {
	switch i.Kind {
	case MethodBankAccount:
		return i.AccountNumber != "" && i.IFSC != ""
	case MethodUPI:
		return i.VPA != ""
	default:
		return false
	}
}

// BeneficiaryRequest registers a destination with a rail.
type BeneficiaryRequest struct {
	// BeneficiaryID is the deterministic rail-side id (BeneficiaryIDFor).
	BeneficiaryID string
	// Name is the account holder's name. Rails match this against the account
	// and reject on mismatch, so it must be the bank's version, not a display
	// name the payee chose.
	Name       string
	Instrument Instrument

	// Contact details. Optional to the rail, but a phone or email is what makes
	// a failed transfer traceable, so pass them when known.
	Email string
	Phone string
}

// BeneficiaryResult is the rail's verdict on a destination.
type BeneficiaryResult struct {
	BeneficiaryID string
	Status        MethodStatus
	// Detail carries the rail's own wording on rejection, for the admin surface.
	Detail string
}

// DisburseRequest is one payout instruction.
type DisburseRequest struct {
	// IdempotencyKey is the batch's key. It is BOTH the dedupe key and the
	// handle GetPayoutByReference is later queried with, so it must be stable
	// for the life of the batch.
	IdempotencyKey string

	// BeneficiaryID must already be registered and verified.
	BeneficiaryID string

	Amount Money

	// Remarks appears on the payee's bank statement. Rails cap and sanitise
	// this; keep it short and free of anything sensitive.
	Remarks string

	// Kind selects the transfer instrument where a rail supports several.
	Kind MethodKind
}

// RailStatus is the rail's view of a disbursement, normalised across providers.
type RailStatus string

const (
	// RailPending — accepted and in flight. Not terminal; keep polling or wait
	// for the webhook.
	RailPending RailStatus = "pending"
	// RailSuccess — money reached the payee. Terminal.
	RailSuccess RailStatus = "success"
	// RailFailed — money never left. Terminal, and safe: nothing to claw back.
	RailFailed RailStatus = "failed"
	// RailReversed — money left and came back (dead account, bank reject). NOT
	// the same as failed: the ledger has to un-settle entries that were already
	// consumed, which is why Batch models paid -> reversed as a legal move.
	RailReversed RailStatus = "reversed"
)

// BatchState maps a rail status onto the engine's batch state machine.
//
// Centralised here so every rail lands on the same states and no adapter can
// invent its own mapping — an adapter that called a reversal "failed" would make
// the engine credit the payee back twice.
func (s RailStatus) BatchState() (BatchState, bool) {
	switch s {
	case RailSuccess:
		return BatchPaid, true
	case RailFailed:
		return BatchFailed, true
	case RailReversed:
		return BatchReversed, true
	case RailPending:
		return BatchExecuting, true
	default:
		return "", false
	}
}

// Terminal reports whether no further status change is expected.
func (s RailStatus) Terminal() bool {
	return s == RailSuccess || s == RailFailed || s == RailReversed
}

// DisburseResult is what a rail reports about one disbursement.
type DisburseResult struct {
	Status RailStatus

	// Reference is the rail's own payout id, recorded as Batch.ProviderRef.
	Reference string

	// UTR is the bank reference the payee can quote to their bank. Only
	// meaningful once settled, and the single most useful field in a support
	// conversation — surface it wherever a payout is shown.
	UTR string

	// FailureCode / FailureDetail carry the rail's reason. FailureCode is the
	// machine-readable one worth alerting on; FailureDetail is for humans.
	FailureCode   string
	FailureDetail string

	// SettledAt is when the rail says the money landed, where it reports it.
	SettledAt *time.Time
}
