package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/services"
)

func assignedFrame(t *testing.T, orderID, deliveryID string) []byte {
	t.Helper()
	raw, err := json.Marshal(services.Event{
		ID:        "evt-1",
		Type:      "delivery.assigned",
		Timestamp: time.Now().UTC(),
		UserID:    uuid.New(),
		Data: map[string]any{
			"order_id":    orderID,
			"delivery_id": deliveryID,
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

func TestAssignedDeliveryForOrder(t *testing.T) {
	mine := uuid.New().String()
	delivery := uuid.New().String()

	t.Run("returns the delivery when the event is for this order", func(t *testing.T) {
		got, ok := assignedDeliveryForOrder(assignedFrame(t, mine, delivery), mine)
		if !ok || got != delivery {
			t.Fatalf("got (%q, %v), want (%q, true)", got, ok, delivery)
		}
	})

	t.Run("ignores an assignment for somebody else's order", func(t *testing.T) {
		if _, ok := assignedDeliveryForOrder(assignedFrame(t, uuid.New().String(), delivery), mine); ok {
			t.Fatal("matched an unrelated order")
		}
	})

	t.Run("ignores an event with no delivery id", func(t *testing.T) {
		if _, ok := assignedDeliveryForOrder(assignedFrame(t, mine, ""), mine); ok {
			t.Fatal("matched an event carrying no delivery")
		}
	})

	t.Run("ignores a malformed frame", func(t *testing.T) {
		if _, ok := assignedDeliveryForOrder([]byte("not json"), mine); ok {
			t.Fatal("matched malformed JSON")
		}
	})
}
