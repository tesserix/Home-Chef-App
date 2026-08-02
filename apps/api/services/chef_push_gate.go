package services

// chef_push_gate.go — enforce ChefNotificationPreferences on the push path.
//
// The model has documented its own gating rules since Wave 2 ("quiet hours only
// suppress payout / customer-message / promo categories") and the vendor app has
// shipped the toggles, but nothing on the send path ever read the row: the
// screen wrote preferences the pipeline ignored. A chef who turned Payouts off,
// or set quiet hours, still got every push.
//
// This is the missing half. It runs AFTER the per-user category gate in
// sendPushNotification, so a recipient must pass both — the customer-side
// NotificationPreference grid and, when they are a chef, their own topic
// toggles.

import (
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// chefPushTopic is the ChefNotificationPreferences column a push maps to.
type chefPushTopic int

const (
	chefTopicNone chefPushTopic = iota // not chef-gated; pass through
	chefTopicNewOrders
	chefTopicPayouts
	chefTopicCustomerMessages
	chefTopicPromo
)

// chefTopicForType maps a notification type to the chef toggle that governs it.
// Anything unmapped is chefTopicNone and is never suppressed — a new chef-facing
// notification defaults to being delivered rather than silently dropped.
func chefTopicForType(notifType string) chefPushTopic {
	switch notifType {
	case "chef_new_order", "accept_reminder":
		return chefTopicNewOrders
	case "payout_hold_release_eligible", "payout_hold_released", "payout_hold_disputed",
		"earnings_threshold_met", "tip_received", "weekly_statement", "payout_reminder":
		return chefTopicPayouts
	case "customer_message", "order_message":
		return chefTopicCustomerMessages
	case "promo", "marketing":
		return chefTopicPromo
	default:
		return chefTopicNone
	}
}

// ChefAllowsPush reports whether this user, IF they are a chef, wants this push.
// Non-chefs and unmapped types always pass. Errors resolve to "allow": a lookup
// failure must not silently mute a chef's own money.
func ChefAllowsPush(userID uuid.UUID, notifType string) bool {
	topic := chefTopicForType(notifType)
	if topic == chefTopicNone {
		return true
	}

	var chef models.ChefProfile
	if err := database.DB.Select("id").First(&chef, "user_id = ?", userID).Error; err != nil {
		return true // not a chef (or unreadable) — nothing for this gate to say
	}

	prefs := models.DefaultNotificationPreferences(chef.ID)
	if err := database.DB.Where("chef_id = ?", chef.ID).First(&prefs).Error; err != nil {
		prefs = models.DefaultNotificationPreferences(chef.ID)
	}

	switch topic {
	case chefTopicNewOrders:
		// A new order is a revenue event the chef opted into by being open, so it
		// bypasses quiet hours entirely — the model has always said so.
		return prefs.NewOrders
	case chefTopicPayouts:
		if !prefs.Payouts {
			return false
		}
	case chefTopicCustomerMessages:
		if !prefs.CustomerMessages {
			return false
		}
	case chefTopicPromo:
		if !prefs.Promo {
			return false
		}
	}
	return !insideQuietHours(prefs, time.Now())
}

// insideQuietHours resolves the chef's wall-clock window in their own timezone.
// An unparseable window or zone is treated as "not quiet" — a misconfigured
// preference should degrade to delivering, never to silence.
func insideQuietHours(prefs models.ChefNotificationPreferences, now time.Time) bool {
	if !prefs.QuietHoursEnabled {
		return false
	}
	loc, err := time.LoadLocation(strings.TrimSpace(prefs.Timezone))
	if err != nil {
		log.Printf("chef-push-gate: bad timezone %q for chef %s: %v", prefs.Timezone, prefs.ChefID, err)
		return false
	}
	start, okStart := minutesOfDay(prefs.QuietHoursStart)
	end, okEnd := minutesOfDay(prefs.QuietHoursEnd)
	if !okStart || !okEnd || start == end {
		return false
	}
	local := now.In(loc)
	cur := local.Hour()*60 + local.Minute()
	if start < end {
		return cur >= start && cur < end
	}
	// The window crosses midnight (e.g. 22:00 → 07:00): inside means either side.
	return cur >= start || cur < end
}

// minutesOfDay parses "HH:MM" into minutes past midnight.
func minutesOfDay(hhmm string) (int, bool) {
	t, err := time.Parse("15:04", strings.TrimSpace(hhmm))
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}
