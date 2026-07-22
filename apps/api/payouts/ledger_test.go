package payouts

import (
	"math/rand"
	"testing"
	"time"

	"github.com/google/uuid"
)

var refTime = time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)

func entry(kind EntryKind, minor int64, matures *time.Time) LedgerEntry {
	return LedgerEntry{
		ID:          uuid.New(),
		TenantID:    "t1",
		PayeeType:   PayeeChef,
		PayeeID:     uuid.New(),
		Kind:        kind,
		AmountMinor: minor,
		Currency:    CurrencyINR,
		SourceType:  "order",
		SourceID:    uuid.NewString(),
		MaturesAt:   matures,
	}
}

func at(d time.Duration) *time.Time { t := refTime.Add(d); return &t }

func TestEntryKindDirection(t *testing.T) {
	for _, k := range []EntryKind{EntryCreditOrderNet, EntryCreditAdjustment, EntryCreditPayoutReversed, EntryCreditReserveRelease} {
		if dir, err := k.Direction(); err != nil || dir != DirectionCredit {
			t.Fatalf("%s: got %v, %v; want credit", k, dir, err)
		}
	}
	for _, k := range []EntryKind{EntryDebitPayout, EntryDebitPenalty, EntryDebitClawback, EntryDebitReserve, EntryDebitAdjustment} {
		if dir, err := k.Direction(); err != nil || dir != DirectionDebit {
			t.Fatalf("%s: got %v, %v; want debit", k, dir, err)
		}
	}
	if _, err := EntryKind("mystery").Direction(); err == nil {
		t.Fatal("a kind without a direction prefix must error, not default to credit")
	}
}

func TestNegativeMagnitudeIsRejected(t *testing.T) {
	// A debit stored as a negative magnitude would flip into a credit and pay
	// the payee for being penalised. Refuse it rather than pay it out.
	e := entry(EntryDebitPenalty, -500, nil)
	if _, err := e.Amount(); err == nil {
		t.Fatal("negative magnitude must be rejected")
	}
}

func TestDeriveBalanceSplitsMaturedFromMaturing(t *testing.T) {
	entries := []LedgerEntry{
		entry(EntryCreditOrderNet, 10000, at(-48*time.Hour)), // matured
		entry(EntryCreditOrderNet, 2500, at(+6*time.Hour)),   // still maturing
		entry(EntryDebitPenalty, 1500, nil),                  // applies now
	}
	b, err := DeriveBalance(entries, refTime)
	if err != nil {
		t.Fatalf("DeriveBalance: %v", err)
	}
	if b.Matured.Minor != 8500 {
		t.Fatalf("Matured = %d, want 8500", b.Matured.Minor)
	}
	if b.Maturing.Minor != 2500 {
		t.Fatalf("Maturing = %d, want 2500", b.Maturing.Minor)
	}
	if b.Total.Minor != 11000 {
		t.Fatalf("Total = %d, want 11000", b.Total.Minor)
	}
	if b.Payable().Minor != 8500 {
		t.Fatalf("Payable = %d, want 8500", b.Payable().Minor)
	}
	if !b.Recovery().IsZero() {
		t.Fatalf("Recovery = %v, want zero", b.Recovery())
	}
}

func TestDebitsApplyEvenWithNothingOwed(t *testing.T) {
	// A penalty raised against a payee with no matured credit must not
	// silently vanish — it becomes a recovery balance carried forward.
	entries := []LedgerEntry{
		entry(EntryCreditOrderNet, 1000, at(+6*time.Hour)), // not yet sweepable
		entry(EntryDebitPenalty, 4000, nil),
	}
	b, err := DeriveBalance(entries, refTime)
	if err != nil {
		t.Fatalf("DeriveBalance: %v", err)
	}
	if b.Matured.Minor != -4000 {
		t.Fatalf("Matured = %d, want -4000", b.Matured.Minor)
	}
	if !b.Payable().IsZero() {
		t.Fatalf("Payable = %v, want zero — the engine must never emit a negative payout", b.Payable())
	}
	if b.Recovery().Minor != 4000 {
		t.Fatalf("Recovery = %d, want 4000", b.Recovery().Minor)
	}
}

func TestMaturationBoundaryIsInclusive(t *testing.T) {
	// A credit maturing exactly at asOf is sweepable. Off-by-one here would
	// silently delay every payout by a full sweep cycle.
	exact := refTime
	entries := []LedgerEntry{entry(EntryCreditOrderNet, 700, &exact)}
	b, err := DeriveBalance(entries, refTime)
	if err != nil {
		t.Fatalf("DeriveBalance: %v", err)
	}
	if b.Matured.Minor != 700 {
		t.Fatalf("credit maturing exactly at asOf must be matured, got %d", b.Matured.Minor)
	}
}

func TestDeriveBalanceIsOrderIndependent(t *testing.T) {
	base := []LedgerEntry{
		entry(EntryCreditOrderNet, 12345, at(-time.Hour)),
		entry(EntryCreditOrderNet, 6789, at(+time.Hour)),
		entry(EntryDebitPenalty, 500, nil),
		entry(EntryDebitClawback, 2200, nil),
		entry(EntryCreditPayoutReversed, 3000, nil),
		entry(EntryDebitReserve, 100, nil),
		entry(EntryCreditReserveRelease, 100, at(-2*time.Hour)),
	}
	want, err := DeriveBalance(base, refTime)
	if err != nil {
		t.Fatalf("DeriveBalance: %v", err)
	}

	rng := rand.New(rand.NewSource(20260722))
	for i := 0; i < 200; i++ {
		shuffled := append([]LedgerEntry(nil), base...)
		rng.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		got, err := DeriveBalance(shuffled, refTime)
		if err != nil {
			t.Fatalf("shuffle %d: %v", i, err)
		}
		if got != want {
			t.Fatalf("shuffle %d changed the balance: got %+v, want %+v", i, got, want)
		}
	}
}

func TestDeriveBalanceRejectsMixedCurrencies(t *testing.T) {
	a := entry(EntryCreditOrderNet, 100, nil)
	b := entry(EntryCreditOrderNet, 100, nil)
	b.Currency = "USD"
	if _, err := DeriveBalance([]LedgerEntry{a, b}, refTime); err == nil {
		t.Fatal("a ledger mixing currencies must error rather than sum nonsense")
	}
}

func TestDeriveBalanceOfNothingIsZero(t *testing.T) {
	b, err := DeriveBalance(nil, refTime)
	if err != nil {
		t.Fatalf("DeriveBalance: %v", err)
	}
	if !b.Total.IsZero() || !b.Payable().IsZero() || !b.Recovery().IsZero() {
		t.Fatalf("empty ledger = %+v, want all zero", b)
	}
}
