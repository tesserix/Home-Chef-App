package payouts

import (
	"time"

	"github.com/google/uuid"
)

// build.go — the daily sweep (#741).
//
// Turns one payee's ledger into at most one payout for one business date.
// Deliberately pure: no provider, no database, no clock of its own. Guardrails
// are re-evaluated at build time AND again at execution, and both passes must
// reach the same answer from the same recorded inputs — which is only true if
// nothing here reads ambient state.

// SkipReason explains why a build produced no payable batch. Stable across
// releases: it is persisted, shown in the admin queue, and grouped in the
// failure analytics of #747.
type SkipReason string

const (
	// SkipNothingMatured — credits exist but none are past their maturation.
	SkipNothingMatured SkipReason = "nothing_matured"
	// SkipBelowMinimum — the net is real but too small to be worth a transfer.
	SkipBelowMinimum SkipReason = "below_minimum"
	// SkipAboveDailyCap — the net exceeds a cap, so a human decides.
	SkipAboveDailyCap SkipReason = "above_daily_cap"
	// SkipRecoveryBalance — debits exceed credits; the payee owes the platform.
	SkipRecoveryBalance SkipReason = "recovery_balance"
)

// BuildInput is everything the sweep is allowed to consider.
type BuildInput struct {
	TenantID string
	Payee    PayeeRef

	// BusinessDate is the sweep date (YYYY-MM-DD) in the platform's timezone.
	// A date rather than a timestamp so a retried sweep on the same day
	// collides by construction.
	BusinessDate string

	// Now is the sweep instant. Passed in, never read from the clock, so a
	// re-evaluation at execution time reproduces the build-time decision.
	Now time.Time

	// Entries are this payee's unsettled ledger rows.
	Entries []LedgerEntry

	Limits Limits
}

// BuildResult is the sweep's decision for one payee.
//
// Amount is always reported even when the batch is not payable: the admin
// queue and the analytics in #747 need to show what was withheld and why, and
// a silently-dropped amount is indistinguishable from an amount that never
// existed.
type BuildResult struct {
	Payee          PayeeRef
	BusinessDate   string
	IdempotencyKey string

	// Payable is true only when the batch should proceed to execution.
	Payable bool

	// Amount is the net for this sweep, floored at zero.
	Amount Money

	// Recovery is what the payee owes the platform, carried forward.
	Recovery Money

	// EntryIDs are exactly the ledger rows this batch settles — every one it
	// pays for and no others, so reconciliation (#746) can reconstruct it.
	EntryIDs []uuid.UUID

	// SkipReason is set when Payable is false.
	SkipReason SkipReason
}

// Build runs the sweep for a single payee and business date.
//
// It never emits a negative payout and never truncates to a cap: an over-cap
// batch goes to review whole, because paying the cap and dropping the
// remainder would quietly lose money from the payee's ledger.
func Build(in BuildInput) (BuildResult, error) {
	res := BuildResult{
		Payee:          in.Payee,
		BusinessDate:   in.BusinessDate,
		IdempotencyKey: IdempotencyKeyFor(in.TenantID, in.Payee, in.BusinessDate),
		Amount:         Zero(CurrencyINR),
		Recovery:       Zero(CurrencyINR),
	}

	balance, err := DeriveBalance(in.Entries, in.Now)
	if err != nil {
		return BuildResult{}, err
	}

	// Claim the rows this sweep settles: every matured credit, and every debit
	// (debits apply immediately — that is how a recovery balance arises).
	var maturedCredits int
	for _, e := range in.Entries {
		amt, err := e.Amount()
		if err != nil {
			return BuildResult{}, err
		}
		if amt.IsPositive() && e.MaturesAt != nil && e.MaturesAt.After(in.Now) {
			continue // still maturing
		}
		if amt.IsPositive() {
			maturedCredits++
		}
		res.EntryIDs = append(res.EntryIDs, e.ID)
	}

	res.Amount = balance.Payable()
	res.Recovery = balance.Recovery()

	switch {
	case balance.Matured.IsNegative():
		// Owed to the platform. Carry forward; never pay, never claim.
		res.SkipReason = SkipRecoveryBalance
		res.EntryIDs = nil
		return res, nil

	case maturedCredits == 0:
		res.SkipReason = SkipNothingMatured
		res.EntryIDs = nil
		return res, nil
	}

	if cmp, err := res.Amount.Cmp(in.Limits.MinPayout); err == nil && cmp < 0 {
		// Leave it in the ledger for tomorrow rather than paying a sum that
		// costs more in rail fees and reconciliation than it moves.
		res.SkipReason = SkipBelowMinimum
		res.EntryIDs = nil
		return res, nil
	}

	if !in.Limits.PerPayeeDailyCap.IsZero() {
		if cmp, err := res.Amount.Cmp(in.Limits.PerPayeeDailyCap); err == nil && cmp > 0 {
			res.SkipReason = SkipAboveDailyCap
			res.EntryIDs = nil
			return res, nil
		}
	}

	res.Payable = true
	return res, nil
}
