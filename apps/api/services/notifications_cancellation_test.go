package services

// notifications_cancellation_test.go — the cancellation flow's notifications
// (#475): the chef is told of a request, the customer of the refund. Reuses
// setupNotifDB from notifications_daily_menu_test.go (same package).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestHandleCancellationRequested_NotifiesChef(t *testing.T) {
	db := setupNotifDB(t)
	chefUserID := uuid.New()
	s := GetNotificationService()
	require.NoError(t, s.handleCancellationRequested(Event{
		UserID: chefUserID,
		Data:   map[string]any{"order_id": uuid.NewString()},
	}))
	var n models.Notification
	require.NoError(t, db.Where("type = ?", "cancellation_requested").First(&n).Error)
	require.Equal(t, chefUserID, n.UserID)
}

func TestHandleCancellationResolved_NotifiesCustomerWithRefund(t *testing.T) {
	db := setupNotifDB(t)
	custUserID := uuid.New()
	s := GetNotificationService()
	require.NoError(t, s.handleCancellationResolved(Event{
		UserID: custUserID,
		Data:   map[string]any{"order_id": uuid.NewString(), "refund": float64(252)},
	}))
	var n models.Notification
	require.NoError(t, db.Where("type = ?", "cancellation_resolved").First(&n).Error)
	require.Equal(t, custUserID, n.UserID)
	require.Contains(t, n.Message, "252", "the refund amount is in the message")
}

func TestCancellationNotificationCategories(t *testing.T) {
	require.Equal(t, models.NotifCategoryChef, notificationTypeCategory("cancellation_requested"))
	require.Equal(t, models.NotifCategoryOrder, notificationTypeCategory("cancellation_resolved"))
}
