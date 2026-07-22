package payouts

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeAdapter struct {
	typ   PayeeType
	rules []Rule
}

func (f fakeAdapter) Type() PayeeType { return f.typ }
func (f fakeAdapter) Resolve(context.Context, uuid.UUID) (PayeeProfile, error) {
	return PayeeProfile{}, nil
}
func (f fakeAdapter) MaturedCredits(context.Context, uuid.UUID, time.Time) ([]Credit, error) {
	return nil, nil
}
func (f fakeAdapter) OpenDebits(context.Context, uuid.UUID) ([]Debit, error) { return nil, nil }
func (f fakeAdapter) Rules() []Rule                                          { return f.rules }

func TestRegistryRoundTrip(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(fakeAdapter{typ: PayeeChef}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	got, err := r.Get(PayeeChef)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Type() != PayeeChef {
		t.Fatalf("Type = %s, want chef", got.Type())
	}
	if _, err := r.Get(PayeeDeliveryPartner); err == nil {
		t.Fatal("Get of an unregistered type must error rather than return nil")
	}
}

func TestRegisteringATypeTwiceIsRefused(t *testing.T) {
	// Silently overwriting would mean paying the wrong party from the wrong
	// ledger, and would surface as a money bug rather than a wiring bug.
	r := NewRegistry()
	if err := r.Register(fakeAdapter{typ: PayeeChef}); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := r.Register(fakeAdapter{typ: PayeeChef}); err == nil {
		t.Fatal("duplicate registration must be refused")
	}
}

func TestRegisterRejectsNilAndUntypedAdapters(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(nil); err == nil {
		t.Fatal("nil adapter must be refused")
	}
	if err := r.Register(fakeAdapter{typ: ""}); err == nil {
		t.Fatal("adapter with an empty payee type must be refused")
	}
}

func TestTypesIsSortedSoSweepsAreDeterministic(t *testing.T) {
	r := NewRegistry()
	// Register in reverse order; Types must still come back sorted.
	if err := r.Register(fakeAdapter{typ: PayeeDeliveryPartner}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := r.Register(fakeAdapter{typ: PayeeChef}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	types := r.Types()
	if len(types) != 2 || types[0] != PayeeChef || types[1] != PayeeDeliveryPartner {
		t.Fatalf("Types = %v, want [chef delivery_partner]", types)
	}
}

func TestPayeeRefString(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	ref := PayeeRef{Type: PayeeChef, ID: id}
	if got := ref.String(); got != "chef:11111111-2222-3333-4444-555555555555" {
		t.Fatalf("String = %q", got)
	}
}
