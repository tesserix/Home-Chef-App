package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestParseBorzoCallback(t *testing.T) {
	// Top-level shape with courier coords.
	id, st, cb, ok := parseBorzoCallback([]byte(`{"order_id":99001,"status":"active","courier":{"latitude":"12.97","longitude":"77.59","phone":"9000000000"}}`))
	if !ok || id != "99001" || st != models.DeliveryInTransit || cb.Courier.Phone != "9000000000" {
		t.Fatalf("top-level: id=%q st=%q ok=%v cb=%+v", id, st, ok, cb.Courier)
	}
	// "order"-nested shape.
	if id, st, _, ok := parseBorzoCallback([]byte(`{"event":"order","order":{"order_id":42,"status":"completed"}}`)); !ok || id != "42" || st != models.DeliveryDelivered {
		t.Fatalf("nested: id=%q st=%q ok=%v", id, st, ok)
	}
	// Unknown status → ok=false.
	if _, _, _, ok := parseBorzoCallback([]byte(`{"order_id":7,"status":"weird"}`)); ok {
		t.Fatal("unknown status must be ok=false")
	}
	// Missing order id → ok=false.
	if _, _, _, ok := parseBorzoCallback([]byte(`{"status":"completed"}`)); ok {
		t.Fatal("missing order id must be ok=false")
	}
	// Garbage → ok=false, no panic.
	if _, _, _, ok := parseBorzoCallback([]byte(`nope`)); ok {
		t.Fatal("garbage must be ok=false")
	}
}
