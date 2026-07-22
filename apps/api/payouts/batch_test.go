package payouts

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBatchLegalTransitions(t *testing.T) {
	happy := []BatchState{BatchBuilding, BatchPendingApproval, BatchApproved, BatchExecuting, BatchPaid}
	for i := 0; i < len(happy)-1; i++ {
		if !happy[i].CanTransition(happy[i+1]) {
			t.Fatalf("%s -> %s should be legal", happy[i], happy[i+1])
		}
	}
	// A bank can return money days after a successful payout.
	if !BatchPaid.CanTransition(BatchReversed) {
		t.Fatal("paid -> reversed must be legal")
	}
	// Manual mode may approve straight out of building.
	if !BatchBuilding.CanTransition(BatchApproved) {
		t.Fatal("building -> approved must be legal for auto mode")
	}
}

func TestBatchCannotBeCancelledOnceHandedToTheProvider(t *testing.T) {
	// Once CreatePayout has been called the outcome must be established, never
	// assumed away — cancelling here would strand real money.
	if BatchExecuting.CanTransition(BatchCancelled) {
		t.Fatal("executing -> cancelled must be illegal")
	}
	if _, err := BatchExecuting.Transition(BatchCancelled); err == nil {
		t.Fatal("Transition must reject executing -> cancelled")
	}
}

func TestBatchIllegalTransitionsRejected(t *testing.T) {
	illegal := [][2]BatchState{
		{BatchBuilding, BatchExecuting}, // must not skip approval
		{BatchBuilding, BatchPaid},
		{BatchApproved, BatchPaid}, // must go through executing
		{BatchPaid, BatchExecuting},
		{BatchFailed, BatchExecuting}, // terminal
		{BatchCancelled, BatchApproved},
		{BatchReversed, BatchPaid},
	}
	for _, tc := range illegal {
		if tc[0].CanTransition(tc[1]) {
			t.Fatalf("%s -> %s must be illegal", tc[0], tc[1])
		}
	}
}

func TestBatchTerminalStates(t *testing.T) {
	for _, s := range []BatchState{BatchFailed, BatchReversed, BatchCancelled} {
		if !s.IsTerminal() {
			t.Fatalf("%s should be terminal", s)
		}
	}
	for _, s := range []BatchState{BatchBuilding, BatchPendingApproval, BatchApproved, BatchExecuting, BatchPaid} {
		if s.IsTerminal() {
			t.Fatalf("%s should not be terminal", s)
		}
	}
}

func TestIdempotencyKeyIsPureAndStable(t *testing.T) {
	ref := PayeeRef{Type: PayeeChef, ID: uuid.MustParse("11111111-2222-3333-4444-555555555555")}
	a := IdempotencyKeyFor("t1", ref, "2026-07-22")
	b := IdempotencyKeyFor("t1", ref, "2026-07-22")
	if a != b {
		t.Fatalf("key must be reproducible after a crash: %q vs %q", a, b)
	}
	if a != "payout:t1:chef:11111111-2222-3333-4444-555555555555:2026-07-22" {
		t.Fatalf("unexpected key format: %q", a)
	}

	// Every identity component must change the key, or two distinct payouts
	// would collide at the rail and one would silently not happen.
	other := PayeeRef{Type: PayeeDeliveryPartner, ID: ref.ID}
	if IdempotencyKeyFor("t1", other, "2026-07-22") == a {
		t.Fatal("payee type must affect the key")
	}
	if IdempotencyKeyFor("t2", ref, "2026-07-22") == a {
		t.Fatal("tenant must affect the key")
	}
	if IdempotencyKeyFor("t1", ref, "2026-07-23") == a {
		t.Fatal("business date must affect the key")
	}
}

func TestFormatBusinessDateUsesTheGivenLocation(t *testing.T) {
	ist := time.FixedZone("IST", 5*3600+1800)
	// 20:00 UTC on the 21st is already the 22nd in IST. A sweep keyed on UTC
	// would batch the wrong day's earnings for an Indian platform.
	utcEvening := time.Date(2026, 7, 21, 20, 0, 0, 0, time.UTC)
	if got := FormatBusinessDate(utcEvening, ist); got != "2026-07-22" {
		t.Fatalf("business date in IST = %q, want 2026-07-22", got)
	}
	if got := FormatBusinessDate(utcEvening, time.UTC); got != "2026-07-21" {
		t.Fatalf("business date in UTC = %q, want 2026-07-21", got)
	}
	if got := FormatBusinessDate(utcEvening, nil); got != "2026-07-21" {
		t.Fatalf("nil location must default to UTC, got %q", got)
	}
}
