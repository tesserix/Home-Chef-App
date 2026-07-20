package services

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// The customer's real-time order tracker keys off notification data
// {order_id, status}; every customer-facing transition must carry both so the
// client can update the right order optimistically.

func notifData(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &m))
	return m
}

func TestHandleOrderUpdated_NotifiesCustomerWithStatus(t *testing.T) {
	db := setupNotifDB(t)
	cust, order := uuid.New(), uuid.New()
	s := GetNotificationService()
	require.NoError(t, s.handleOrderUpdated(OrderEvent{
		OrderID: order, CustomerID: cust, Status: string(models.OrderStatusAccepted),
	}))
	var n models.Notification
	require.NoError(t, db.Where("user_id = ? AND type = ?", cust, "order_status").First(&n).Error)
	d := notifData(t, n.Data)
	require.Equal(t, order.String(), d["order_id"])
	require.Equal(t, string(models.OrderStatusAccepted), d["status"])
}

// Every customer-relevant transition must reach the customer, because the
// mobile tracker (#716) renders purely from these notifications — a stage that
// emits nothing leaves the customer stuck on the previous status until the
// fallback poll catches up.
//
// UpdateOrderStatus routes `delivered` to orders.delivered and everything else
// to orders.updated, so this covers the orders.updated half: the stages a chef
// (or driver) moves an order through between accept and hand-off, plus the two
// terminal states a customer must see immediately.
func TestHandleOrderUpdated_NotifiesCustomerForEveryStage(t *testing.T) {
	for _, status := range []models.OrderStatus{
		models.OrderStatusAccepted,
		models.OrderStatusPreparing,
		models.OrderStatusReady,
		models.OrderStatusPickedUp,
		models.OrderStatusDelivering,
		models.OrderStatusCancelled,
		models.OrderStatusRejected,
	} {
		t.Run(string(status), func(t *testing.T) {
			db := setupNotifDB(t)
			cust, order := uuid.New(), uuid.New()
			s := GetNotificationService()
			require.NoError(t, s.handleOrderUpdated(OrderEvent{
				OrderID: order, CustomerID: cust, Status: string(status),
			}))

			var n models.Notification
			require.NoError(t,
				db.Where("user_id = ? AND type = ?", cust, "order_status").First(&n).Error,
				"no customer notification emitted for %s", status)

			d := notifData(t, n.Data)
			require.Equal(t, order.String(), d["order_id"],
				"%s notification must identify the order", status)
			require.Equal(t, string(status), d["status"],
				"%s notification must carry the new status so the client can flip without a refetch", status)
			require.NotEmpty(t, n.Message, "%s notification needs a human-readable message", status)
			// The message is what the customer actually reads in the push and the
			// in-app list, so it must be written copy — not the raw-status
			// fallback, which reads like debug output.
			require.NotContains(t, n.Message, "status has been updated to",
				"%s has no written message; getOrderStatusMessage fell through to the raw-status fallback", status)
		})
	}
}

func TestHandleOrderDelivered_NotifiesCustomerWithStatus(t *testing.T) {
	db := setupNotifDB(t)
	cust, order := uuid.New(), uuid.New()
	s := GetNotificationService()
	require.NoError(t, s.handleOrderDelivered(OrderEvent{
		OrderID: order, CustomerID: cust, Status: string(models.OrderStatusDelivered),
	}))
	var n models.Notification
	require.NoError(t, db.Where("user_id = ? AND type = ?", cust, "order_delivered").First(&n).Error)
	d := notifData(t, n.Data)
	require.Equal(t, order.String(), d["order_id"])
	require.Equal(t, string(models.OrderStatusDelivered), d["status"], "delivered notification must carry status")
}
