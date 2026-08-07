package handlers

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestTrackingAwaitsDriver(t *testing.T) {
	t.Run("a pickup order never gets a driver", func(t *testing.T) {
		if trackingAwaitsDriver(models.FulfillmentPickup) {
			t.Fatal("pickup subscribed to delivery.assigned; the customer collects, so no assignment can ever arrive")
		}
	})

	t.Run("a 3PL delivery waits for one", func(t *testing.T) {
		if !trackingAwaitsDriver(models.FulfillmentDelivery) {
			t.Fatal("delivery must wait for the driver to be assigned")
		}
	})

	t.Run("a chef delivery waits for one", func(t *testing.T) {
		if !trackingAwaitsDriver(models.FulfillmentChefDelivery) {
			t.Fatal("chef delivery still creates a delivery leg to track")
		}
	})

	t.Run("an unset fulfillment waits, matching the delivery default", func(t *testing.T) {
		if !trackingAwaitsDriver("") {
			t.Fatal("an unset type must fall back to the delivery default, not silently stop tracking")
		}
	})
}
