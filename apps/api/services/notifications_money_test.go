package services

import (
	"testing"

	"github.com/homechef/api/models"
)

// Money notifications must land in the Payment category, not Account. They used
// to fall through the default arm, so a user who muted account chatter also
// muted their own refunds and payouts.
func TestMoneyTypesMapToPaymentCategory(t *testing.T) {
	for _, notifType := range []string{
		"payment_success", "payment_failed", "payment_refunded",
		"delivery_fee_refund", "cancellation_refund",
		"payout_hold_release_eligible", "payout_hold_released", "payout_hold_disputed",
		"earnings_threshold_met", "tip_received",
		"loyalty_earned", "loyalty_redeemed", "referral_rewarded",
	} {
		if got := notificationTypeCategory(notifType); got != models.NotifCategoryPayment {
			t.Errorf("notificationTypeCategory(%q)=%q, want %q", notifType, got, models.NotifCategoryPayment)
		}
	}
}

// The categories that already worked must not have moved — a type silently
// changing category would flip which toggle mutes it.
func TestExistingCategoriesUnchanged(t *testing.T) {
	for notifType, want := range map[string]models.NotificationCategory{
		"order_status":          models.NotifCategoryOrder,
		"order_delivered":       models.NotifCategoryOrder,
		"chef_new_order":        models.NotifCategoryChef,
		"delivery_assigned":     models.NotifCategoryDelivery,
		"promo":                 models.NotifCategoryMarketing,
		"weekly_menu_published": models.NotifCategoryFavorites,
		"welcome":               models.NotifCategoryAccount,
	} {
		if got := notificationTypeCategory(notifType); got != want {
			t.Errorf("notificationTypeCategory(%q)=%q, want %q", notifType, got, want)
		}
	}
}

// The settings UI is driven off AllNotificationCategories, so a category the
// pipeline uses but the list omits is a toggle the user can never reach.
func TestPaymentCategoryIsSettable(t *testing.T) {
	var found bool
	for _, c := range models.AllNotificationCategories() {
		if c == models.NotifCategoryPayment {
			found = true
		}
	}
	if !found {
		t.Fatal("NotifCategoryPayment missing from AllNotificationCategories — no toggle would render")
	}
	// Money is transactional: it must default ON, like order and delivery.
	def := models.DefaultNotificationPreference(models.NotifCategoryPayment)
	if !def.PushEnabled {
		t.Error("payment pushes must default to enabled")
	}
}
