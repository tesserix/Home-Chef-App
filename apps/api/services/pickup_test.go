package services

// pickup_test.go — the guard that keeps the ready-to-collect flow honest.
//
// Every message in pickup.go re-reads the order and no-ops unless it is still
// genuinely awaiting collection. That is what makes an at-least-once Temporal
// activity retry safe, and it is the only thing stopping the flow from nagging a
// customer about an order they already collected or cancelled.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupPickupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(
		`CREATE TABLE platform_settings (id text PRIMARY KEY, key text, value text, created_at datetime, updated_at datetime)`,
	).Error)
	return db
}

func TestAwaitingCollection_OnlyAPickupOrderStillAtReady(t *testing.T) {
	cases := []struct {
		name       string
		fulfilment models.FulfillmentType
		status     models.OrderStatus
		want       bool
	}{
		{"pickup, cooked and waiting", models.FulfillmentPickup, models.OrderStatusReady, true},
		// `delivered` is what the chef sets when the customer walks out with it —
		// the pickup equivalent of "collected".
		{"pickup, already collected", models.FulfillmentPickup, models.OrderStatusDelivered, false},
		{"pickup, cancelled while waiting", models.FulfillmentPickup, models.OrderStatusCancelled, false},
		{"pickup, refunded", models.FulfillmentPickup, models.OrderStatusRefunded, false},
		{"pickup, still cooking", models.FulfillmentPickup, models.OrderStatusPreparing, false},
		// A delivery order at `ready` is waiting for a CARRIER, not a customer.
		// Chasing its customer to come and collect would be nonsense.
		{"delivery at ready", models.FulfillmentDelivery, models.OrderStatusReady, false},
		{"chef delivery at ready", models.FulfillmentChefDelivery, models.OrderStatusReady, false},
		// Orders predating the column default to delivery server-side, so an empty
		// fulfilment type must never be chased for collection.
		{"legacy order with no fulfilment type", "", models.OrderStatusReady, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, awaitingCollection(models.Order{
				FulfillmentType: tc.fulfilment,
				Status:          tc.status,
			}))
		})
	}
}

func TestPickupReminderSettings_DefaultsAndOverrides(t *testing.T) {
	db := setupPickupDB(t)

	// Defaults: ~1 hour of chasing before the chef is told (3 × 20 min).
	require.Equal(t, 20, PickupReminderIntervalMinutes(db))
	require.Equal(t, 3, PickupReminderMaxCount(db))

	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value) VALUES (?, ?, ?), (?, ?, ?)`,
		uuid.NewString(), "pickup.reminder_interval_minutes", "45",
		uuid.NewString(), "pickup.reminder_max_count", "2",
	).Error)
	require.Equal(t, 45, PickupReminderIntervalMinutes(db))
	require.Equal(t, 2, PickupReminderMaxCount(db))
}

func TestPickupReminderSettings_IgnoreUnusableValues(t *testing.T) {
	db := setupPickupDB(t)

	// A garbled or non-positive setting must fall back to the default rather than
	// producing a zero interval — a 0s timer would spin the reminder loop through
	// its whole budget instantly and burn all three nudges in one second.
	require.NoError(t, db.Exec(
		`INSERT INTO platform_settings (id, key, value) VALUES (?, ?, ?), (?, ?, ?)`,
		uuid.NewString(), "pickup.reminder_interval_minutes", "not-a-number",
		uuid.NewString(), "pickup.reminder_max_count", "0",
	).Error)
	require.Equal(t, 20, PickupReminderIntervalMinutes(db))
	require.Equal(t, 3, PickupReminderMaxCount(db))
}

func TestGetOrderStatusMessage_ReadyMeansDifferentThingsPerMode(t *testing.T) {
	// The old single "Your order is ready for pickup/delivery" string was aimed at
	// nobody: it told a pickup customer nothing about having to go and fetch it,
	// and told a delivery customer to go and fetch it.
	pickup := getOrderStatusMessage(string(models.OrderStatusReady), models.FulfillmentPickup)
	require.Equal(t, "Your order is ready to collect", pickup)

	for _, mode := range []models.FulfillmentType{models.FulfillmentDelivery, models.FulfillmentChefDelivery, ""} {
		msg := getOrderStatusMessage(string(models.OrderStatusReady), mode)
		require.Equal(t, "Your order is ready and will be on its way shortly", msg)
		// Never tell a delivery customer to go and collect.
		require.NotContains(t, msg, "collect")
	}
}

func TestGetOrderStatusMessage_TerminalStatusPerMode(t *testing.T) {
	require.Equal(t,
		"Order collected. Enjoy!",
		getOrderStatusMessage(string(models.OrderStatusDelivered), models.FulfillmentPickup))
	require.Equal(t,
		"Your order has been delivered. Enjoy!",
		getOrderStatusMessage(string(models.OrderStatusDelivered), models.FulfillmentDelivery))
}

func TestGetOrderStatusMessage_SharedStatusesAreModeIndependent(t *testing.T) {
	// Everything that isn't `ready` or `delivered` means the same thing either
	// way, and must not have drifted while the two special cases were carved out.
	for _, status := range []models.OrderStatus{
		models.OrderStatusAccepted,
		models.OrderStatusPreparing,
		models.OrderStatusCancelled,
		models.OrderStatusRejected,
		models.OrderStatusRefunded,
	} {
		require.Equal(t,
			getOrderStatusMessage(string(status), models.FulfillmentDelivery),
			getOrderStatusMessage(string(status), models.FulfillmentPickup),
			"status %s should read the same for both modes", status)
	}
}
