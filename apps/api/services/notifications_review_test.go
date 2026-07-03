package services

// notifications_review_test.go — chef gets notified when a customer posts a
// review on one of their orders (#422). Reuses setupNotifDB from
// notifications_daily_menu_test.go (same package).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestHandleReviewPosted_NotifiesChef(t *testing.T) {
	db := setupNotifDB(t)
	chefUserID := uuid.New()

	s := GetNotificationService()
	require.NoError(t, s.handleReviewPosted(Event{
		UserID: chefUserID,
		Data: map[string]any{
			"review_id": uuid.NewString(),
			"order_id":  uuid.NewString(),
			"rating":    float64(5),
		},
	}))

	var n models.Notification
	require.NoError(t, db.Where("type = ?", "review_posted").First(&n).Error)
	require.Equal(t, chefUserID, n.UserID, "the review notification targets the chef")
}

func TestHandleReviewPosted_NoChefUserIsNoop(t *testing.T) {
	db := setupNotifDB(t)
	s := GetNotificationService()
	require.NoError(t, s.handleReviewPosted(Event{UserID: uuid.Nil}))

	var count int64
	require.NoError(t, db.Model(&models.Notification{}).Count(&count).Error)
	require.Equal(t, int64(0), count)
}

// The review notification is gated by the chef's "chef" preference category.
func TestReviewPostedCategoryIsChef(t *testing.T) {
	require.Equal(t, models.NotifCategoryChef, notificationTypeCategory("review_posted"))
}
