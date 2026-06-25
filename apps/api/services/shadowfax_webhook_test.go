package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestParseShadowfaxCallback(t *testing.T) {
	// ofd → in_transit, order_id carried through verbatim.
	cid, st, ok := parseShadowfaxCallback([]byte(`{"order_id":"HC-1","status":"ofd","event_timestamp":"2026-06-25 10:00:00"}`))
	if !ok || cid != "HC-1" || st != models.DeliveryInTransit {
		t.Fatalf("got cid=%q st=%q ok=%v; want HC-1/in_transit/true", cid, st, ok)
	}
	// delivered → delivered.
	if _, st, ok := parseShadowfaxCallback([]byte(`{"order_id":"HC-2","status":"delivered"}`)); !ok || st != models.DeliveryDelivered {
		t.Fatalf("delivered map failed: %q %v", st, ok)
	}
	// Unknown status → ok=false (caller acks + ignores).
	if _, _, ok := parseShadowfaxCallback([]byte(`{"order_id":"HC-3","status":"recd_at_rev_hub_unknownx"}`)); ok {
		t.Fatal("unknown status must be ok=false")
	}
	// Missing order_id → ok=false.
	if _, _, ok := parseShadowfaxCallback([]byte(`{"status":"delivered"}`)); ok {
		t.Fatal("missing order_id must be ok=false")
	}
	// Garbage payload → ok=false, no panic.
	if _, _, ok := parseShadowfaxCallback([]byte(`not json`)); ok {
		t.Fatal("garbage must be ok=false")
	}
}
