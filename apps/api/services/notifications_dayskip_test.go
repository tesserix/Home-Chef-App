package services

// notifications_dayskip_test.go — chef gets notified when a customer skips a
// plan day (#422), so they don't cook it. Reuses setupNotifDB from
// notifications_daily_menu_test.go (same package).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestHandleMealPlanDaySkippedChef_NotifiesChef(t *testing.T) {
	db := setupNotifDB(t)
	chefUserID := uuid.New()

	s := GetNotificationService()
	require.NoError(t, s.handleMealPlanDaySkippedChef(Event{
		UserID: chefUserID,
		Data: map[string]any{
			"meal_plan_id": uuid.NewString(),
			"day_id":       uuid.NewString(),
			"date":         "2027-03-15",
		},
	}))

	var n models.Notification
	require.NoError(t, db.Where("type = ?", "meal_plan_day_skipped_chef").First(&n).Error)
	require.Equal(t, chefUserID, n.UserID, "the skip notification targets the chef")
}

// The skip-chef notification is gated by the chef preference category.
func TestMealPlanDaySkippedChefCategoryIsChef(t *testing.T) {
	require.Equal(t, models.NotifCategoryChef, notificationTypeCategory("meal_plan_day_skipped_chef"))
}
