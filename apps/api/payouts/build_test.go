package payouts

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// #741 — the sweep that turns ledger entries into one payout per payee per
// business date. Pure: no provider, no database, no clock of its own. Every
// input is passed in so the build-time and execution-time passes are
// reproducible from a recorded context.

var buildNow = time.Date(2026, 7, 22, 9, 0, 0, 0, time.UTC)

func inrM(minor int64) Money { return Money{Minor: minor, Currency: CurrencyINR} }

func defaultLimits() Limits {
	return Limits{
		MinPayout:          inrM(10_000), // ₹100
		PerPayeeDailyCap:   inrM(5_000_00),
		PlatformDailyCap:   inrM(50_000_00),
		MaturationDuration: 24 * time.Hour,
	}
}

func credit(minor int64, maturesAt time.Time) LedgerEntry {
	return LedgerEntry{
		ID: uuid.New(), TenantID: "t1", PayeeType: PayeeChef, PayeeID: uuid.Nil,
		Kind: EntryCreditOrderNet, AmountMinor: minor, Currency: CurrencyINR,
		SourceType: "order", SourceID: uuid.NewString(), MaturesAt: &maturesAt,
	}
}

func debit(kind EntryKind, minor int64) LedgerEntry {
	return LedgerEntry{
		ID: uuid.New(), TenantID: "t1", PayeeType: PayeeChef, PayeeID: uuid.Nil,
		Kind: kind, AmountMinor: minor, Currency: CurrencyINR,
		SourceType: "order_issue", SourceID: uuid.NewString(),
	}
}

func buildInput(entries []LedgerEntry) BuildInput {
	return BuildInput{
		TenantID:     "t1",
		Payee:        PayeeRef{Type: PayeeChef, ID: uuid.Nil},
		BusinessDate: "2026-07-22",
		Now:          buildNow,
		Entries:      entries,
		Limits:       defaultLimits(),
	}
}

func TestBuild_SweepsOnlyMaturedCredits(t *testing.T) {
	// 24h maturation is the window in which a late refund or fraud signal can
	// still cancel a credit before the money becomes irreversible.
	in := buildInput([]LedgerEntry{
		credit(30_000, buildNow.Add(-time.Hour)), // matured
		credit(20_000, buildNow.Add(time.Hour)),  // still maturing
	})

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out.Amount.Minor != 30_000 {
		t.Fatalf("amount = %d, want 30000 (maturing credit must be excluded)", out.Amount.Minor)
	}
	if len(out.EntryIDs) != 1 {
		t.Fatalf("swept %d entries, want 1", len(out.EntryIDs))
	}
}

func TestBuild_NetsOpenDebits(t *testing.T) {
	in := buildInput([]LedgerEntry{
		credit(50_000, buildNow.Add(-time.Hour)),
		debit(EntryDebitPenalty, 15_000),
	})

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out.Amount.Minor != 35_000 {
		t.Fatalf("amount = %d, want 35000", out.Amount.Minor)
	}
}

func TestBuild_NeverEmitsANegativePayout(t *testing.T) {
	// Debits exceeding credits become a recovery balance carried forward, not
	// a payment from the payee to the platform.
	in := buildInput([]LedgerEntry{
		credit(20_000, buildNow.Add(-time.Hour)),
		debit(EntryDebitClawback, 60_000),
	})

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out.Payable {
		t.Fatal("a payee in debt must not produce a payable batch")
	}
	if out.Amount.IsNegative() {
		t.Fatalf("amount = %v, must never be negative", out.Amount)
	}
	if out.Recovery.Minor != 40_000 {
		t.Fatalf("recovery = %d, want 40000 carried forward", out.Recovery.Minor)
	}
	if out.SkipReason != SkipRecoveryBalance {
		t.Fatalf("skip reason = %q, want %q", out.SkipReason, SkipRecoveryBalance)
	}
}

func TestBuild_HoldsAmountsBelowTheMinimum(t *testing.T) {
	// Paying ₹12 costs more in rail fees and reconciliation than it moves;
	// the credit stays in the ledger for tomorrow rather than being dropped.
	in := buildInput([]LedgerEntry{credit(1_200, buildNow.Add(-time.Hour))})

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out.Payable {
		t.Fatal("below-minimum amounts must not be paid out")
	}
	if out.SkipReason != SkipBelowMinimum {
		t.Fatalf("skip reason = %q, want %q", out.SkipReason, SkipBelowMinimum)
	}
	if out.Amount.Minor != 1_200 {
		t.Fatalf("the amount must still be reported for the admin view, got %d", out.Amount.Minor)
	}
}

func TestBuild_QueuesRatherThanTruncatingAboveTheDailyCap(t *testing.T) {
	// Silently paying the cap and dropping the rest would lose money from the
	// payee's ledger. The whole batch goes to review instead.
	lim := defaultLimits()
	lim.PerPayeeDailyCap = inrM(40_000)
	in := buildInput([]LedgerEntry{credit(90_000, buildNow.Add(-time.Hour))})
	in.Limits = lim

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out.Payable {
		t.Fatal("an over-cap batch must not auto-pay")
	}
	if out.SkipReason != SkipAboveDailyCap {
		t.Fatalf("skip reason = %q, want %q", out.SkipReason, SkipAboveDailyCap)
	}
	if out.Amount.Minor != 90_000 {
		t.Fatalf("amount must not be truncated to the cap, got %d", out.Amount.Minor)
	}
}

func TestBuild_IsIdempotentPerPayeePerDate(t *testing.T) {
	in := buildInput([]LedgerEntry{credit(30_000, buildNow.Add(-time.Hour))})

	a, _ := Build(in)
	b, _ := Build(in)
	if a.IdempotencyKey != b.IdempotencyKey {
		t.Fatalf("key must be reproducible: %q vs %q", a.IdempotencyKey, b.IdempotencyKey)
	}

	in.BusinessDate = "2026-07-23"
	c, _ := Build(in)
	if c.IdempotencyKey == a.IdempotencyKey {
		t.Fatal("a different business date must produce a different key")
	}
}

func TestBuild_SkipsAPayeeWithNothingMatured(t *testing.T) {
	in := buildInput([]LedgerEntry{credit(50_000, buildNow.Add(2*time.Hour))})

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out.Payable {
		t.Fatal("nothing matured — nothing to pay")
	}
	if out.SkipReason != SkipNothingMatured {
		t.Fatalf("skip reason = %q, want %q", out.SkipReason, SkipNothingMatured)
	}
	if len(out.EntryIDs) != 0 {
		t.Fatal("no entries should be claimed")
	}
}

func TestBuild_ClaimsExactlyTheEntriesItPaysFor(t *testing.T) {
	// The batch must reference every entry it settles and no others, or
	// reconciliation cannot reconstruct it later.
	matured := credit(30_000, buildNow.Add(-time.Hour))
	maturing := credit(20_000, buildNow.Add(time.Hour))
	pen := debit(EntryDebitPenalty, 5_000)
	in := buildInput([]LedgerEntry{matured, maturing, pen})

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	claimed := map[uuid.UUID]bool{}
	for _, id := range out.EntryIDs {
		claimed[id] = true
	}
	if !claimed[matured.ID] || !claimed[pen.ID] {
		t.Fatal("matured credit and applied debit must both be claimed")
	}
	if claimed[maturing.ID] {
		t.Fatal("a still-maturing credit must not be claimed")
	}
}

func TestBuild_MaturationBoundaryIsInclusive(t *testing.T) {
	// Off-by-one here silently delays every payout by a full sweep cycle.
	in := buildInput([]LedgerEntry{credit(30_000, buildNow)})

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !out.Payable {
		t.Fatal("a credit maturing exactly at the sweep instant is payable")
	}
}

func TestBuild_RejectsMixedCurrencies(t *testing.T) {
	usd := credit(10_000, buildNow.Add(-time.Hour))
	usd.Currency = "USD"
	in := buildInput([]LedgerEntry{credit(30_000, buildNow.Add(-time.Hour)), usd})

	if _, err := Build(in); err == nil {
		t.Fatal("a ledger mixing currencies must error rather than sum nonsense")
	}
}

func TestBuild_ConservesMoney(t *testing.T) {
	// The invariant the whole engine rests on: what is paid plus what is held
	// back plus what is recovered equals what the ledger said was owed.
	entries := []LedgerEntry{
		credit(80_000, buildNow.Add(-2*time.Hour)),
		credit(25_000, buildNow.Add(3*time.Hour)), // not yet due
		debit(EntryDebitPenalty, 12_000),
	}
	in := buildInput(entries)

	out, err := Build(in)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	bal, err := DeriveBalance(entries, buildNow)
	if err != nil {
		t.Fatalf("DeriveBalance: %v", err)
	}
	// paid-or-withheld now + still maturing == everything the ledger holds
	settled, err := out.Amount.Add(bal.Maturing)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if settled.Minor != bal.Total.Minor {
		t.Fatalf("conservation broken: batch %d + maturing %d != total %d",
			out.Amount.Minor, bal.Maturing.Minor, bal.Total.Minor)
	}
}
