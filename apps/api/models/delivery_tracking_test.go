package models

// delivery_tracking_test.go — backend-testable slice of issue #63 (live
// delivery tracking). The on-device map render/poll is verified manually; what
// we CAN pin down here is Delivery.ToResponse()'s live-position contract, which
// the customer app's tracking screen depends on:
//   - prefer the 3PL rider coords (fed by provider webhooks) when present,
//   - else fall back to the own-fleet partner's last-known position,
//   - else 0,0 (the "no location yet" state the app shows fallback copy for).
// Getting this wrong silently strands the rider marker at the equator, so it's
// worth a regression guard. Also documents which statuses are terminal (the app
// stops polling on those).

import (
	"testing"

	"github.com/google/uuid"
)

func TestDeliveryToResponse_PrefersRiderCoords(t *testing.T) {
	d := &Delivery{
		ID:             uuid.New(),
		Status:         DeliveryInTransit,
		RiderLatitude:  12.9716,
		RiderLongitude: 77.5946,
		RiderName:      "Asha K.",
		RiderPhone:     "+91900000000",
		// A partner is also present, but the 3PL rider coords must win.
		DeliveryPartner: DeliveryPartner{ID: uuid.New(), CurrentLatitude: 1.1, CurrentLongitude: 2.2},
	}
	r := d.ToResponse()
	if r.CurrentLatitude != 12.9716 || r.CurrentLongitude != 77.5946 {
		t.Fatalf("expected rider coords, got (%v,%v)", r.CurrentLatitude, r.CurrentLongitude)
	}
	if r.RiderName != "Asha K." || r.RiderPhone != "+91900000000" {
		t.Fatalf("rider identity not surfaced: %q / %q", r.RiderName, r.RiderPhone)
	}
}

func TestDeliveryToResponse_FallsBackToPartner(t *testing.T) {
	d := &Delivery{
		ID:              uuid.New(),
		Status:          DeliveryPickedUp,
		DeliveryPartner: DeliveryPartner{ID: uuid.New(), CurrentLatitude: 19.07, CurrentLongitude: 72.87},
	}
	r := d.ToResponse()
	if r.CurrentLatitude != 19.07 || r.CurrentLongitude != 72.87 {
		t.Fatalf("expected partner coords fallback, got (%v,%v)", r.CurrentLatitude, r.CurrentLongitude)
	}
	if r.RiderName != "" || r.RiderPhone != "" {
		t.Fatalf("own-fleet leg must not expose 3PL rider identity: %q / %q", r.RiderName, r.RiderPhone)
	}
}

func TestDeliveryToResponse_NoLocationYet_ZeroFallback(t *testing.T) {
	// Fresh 3PL assignment, no webhook yet, no own-fleet partner attached.
	d := &Delivery{ID: uuid.New(), Status: DeliveryAssigned}
	r := d.ToResponse()
	if r.CurrentLatitude != 0 || r.CurrentLongitude != 0 {
		t.Fatalf("expected 0,0 'no location yet' fallback, got (%v,%v)", r.CurrentLatitude, r.CurrentLongitude)
	}
}

// TestDeliveryTerminalStatuses documents the lifecycle states the customer app
// treats as terminal (polling must stop). If a new non-terminal status is added
// without updating the app's stop condition it would keep polling forever; this
// keeps the contract explicit alongside the enum.
func TestDeliveryTerminalStatuses(t *testing.T) {
	terminal := map[DeliveryStatus]bool{
		DeliveryDelivered: true,
		DeliveryFailed:    true,
		DeliveryReturned:  true,
		DeliveryCancelled: true,
	}
	nonTerminal := []DeliveryStatus{
		DeliveryPending, DeliveryAssigned, DeliveryAtPickup,
		DeliveryPickedUp, DeliveryInTransit, DeliveryAtDropoff,
	}
	for s := range terminal {
		if s == "" {
			t.Fatalf("terminal status must be a non-empty constant")
		}
	}
	for _, s := range nonTerminal {
		if terminal[s] {
			t.Fatalf("%q must not be terminal", s)
		}
	}
}
