package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestMapShadowfaxStatus(t *testing.T) {
	cases := map[string]models.DeliveryStatus{
		"new":                   models.DeliveryAssigned,
		"ofp":                   models.DeliveryAssigned,
		"picked":                models.DeliveryPickedUp,
		"assigned_for_delivery": models.DeliveryInTransit,
		"ofd":                   models.DeliveryInTransit,
		"delivered":             models.DeliveryDelivered,
		"cancelled_by_seller":   models.DeliveryCancelled,
		"cancelled_by_customer": models.DeliveryCancelled,
		"rts":                   models.DeliveryReturned,
		"rto":                   models.DeliveryReturned,
		"lost":                  models.DeliveryFailed,
		"on_hold":               models.DeliveryFailed,
	}
	for raw, want := range cases {
		got, ok := mapShadowfaxStatus(raw)
		if !ok || got != want {
			t.Fatalf("mapShadowfaxStatus(%q) = %q,%v; want %q,true", raw, got, ok, want)
		}
	}
	if _, ok := mapShadowfaxStatus("totally_unknown"); ok {
		t.Fatal("unknown status must return ok=false")
	}
	// Case-insensitive + whitespace tolerant (providers vary casing).
	if got, ok := mapShadowfaxStatus("  DELIVERED "); !ok || got != models.DeliveryDelivered {
		t.Fatalf("normalised match failed: %q %v", got, ok)
	}
}
