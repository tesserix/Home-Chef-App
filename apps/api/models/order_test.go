package models

import (
	"testing"
	"time"
)

func TestToResponse_FulfillmentTypeDefaultsToDelivery(t *testing.T) {
	o := Order{} // unset
	if got := o.ToResponse().FulfillmentType; got != FulfillmentDelivery {
		t.Fatalf("empty FulfillmentType should normalize to delivery, got %q", got)
	}
	o2 := Order{FulfillmentType: FulfillmentPickup}
	if got := o2.ToResponse().FulfillmentType; got != FulfillmentPickup {
		t.Fatalf("want pickup, got %q", got)
	}
}

// #617: the customer app needs the escrow hold state to render "Confirm received".
// ToResponse must surface PayoutHoldStatus + CustomerConfirmedAt (they were dropped
// by the DTO before this change).
func TestToResponse_SurfacesPayoutHoldState(t *testing.T) {
	confirmedAt := time.Now()
	o := Order{PayoutHoldStatus: PayoutHoldAwaitingConfirmation, CustomerConfirmedAt: &confirmedAt}
	r := o.ToResponse()
	if r.PayoutHoldStatus != PayoutHoldAwaitingConfirmation {
		t.Fatalf("want payoutHoldStatus awaiting_customer_confirmation, got %q", r.PayoutHoldStatus)
	}
	if r.CustomerConfirmedAt == nil || !r.CustomerConfirmedAt.Equal(confirmedAt) {
		t.Fatalf("customerConfirmedAt not surfaced: %v", r.CustomerConfirmedAt)
	}

	// No hold (escrow flags off) → empty status is omitted, so the CTA never renders.
	noHold := Order{}
	if got := noHold.ToResponse().PayoutHoldStatus; got != PayoutHoldNone {
		t.Fatalf("no-hold order should surface empty payoutHoldStatus, got %q", got)
	}
}
