package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestMapBorzoStatus(t *testing.T) {
	cases := map[string]models.DeliveryStatus{
		"new":       models.DeliveryAssigned,
		"available": models.DeliveryAssigned,
		"active":    models.DeliveryInTransit,
		"delayed":   models.DeliveryInTransit,
		"completed": models.DeliveryDelivered,
		"canceled":  models.DeliveryCancelled,
		"cancelled": models.DeliveryCancelled, // tolerate both spellings
	}
	for raw, want := range cases {
		got, ok := mapBorzoStatus(raw)
		if !ok || got != want {
			t.Fatalf("mapBorzoStatus(%q) = %q,%v; want %q,true", raw, got, ok, want)
		}
	}
	if _, ok := mapBorzoStatus("unknown_status_x"); ok {
		t.Fatal("unknown status must return ok=false")
	}
	if got, ok := mapBorzoStatus("  COMPLETED "); !ok || got != models.DeliveryDelivered {
		t.Fatalf("normalised match failed: %q %v", got, ok)
	}
}
