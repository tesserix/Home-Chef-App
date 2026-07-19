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
