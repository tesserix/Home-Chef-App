package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

func prefsAt(tz, start, end string, enabled bool) models.ChefNotificationPreferences {
	p := models.DefaultNotificationPreferences(uuid.New())
	p.QuietHoursEnabled = enabled
	p.QuietHoursStart = start
	p.QuietHoursEnd = end
	p.Timezone = tz
	return p
}

// The default window (22:00 → 07:00) crosses midnight, so "inside" has to mean
// either side of it. Treating it as a plain start<end range would leave a chef
// getting payout pushes all night and muted all afternoon.
func TestQuietHoursAcrossMidnight(t *testing.T) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	p := prefsAt("Asia/Kolkata", "22:00", "07:00", true)

	cases := []struct {
		hour, min int
		want      bool
	}{
		{23, 30, true},  // after start, before midnight
		{2, 0, true},    // after midnight, before end
		{6, 59, true},   // last minute inside
		{7, 0, false},   // end is exclusive
		{12, 0, false},  // mid-afternoon
		{21, 59, false}, // one minute before start
		{22, 0, true},   // start is inclusive
	}
	for _, c := range cases {
		now := time.Date(2026, 8, 2, c.hour, c.min, 0, 0, ist)
		if got := insideQuietHours(p, now); got != c.want {
			t.Errorf("at %02d:%02d IST: insideQuietHours=%v, want %v", c.hour, c.min, got, c.want)
		}
	}
}

// A same-day window must not be read as a midnight-crossing one.
func TestQuietHoursSameDayWindow(t *testing.T) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	p := prefsAt("Asia/Kolkata", "13:00", "15:00", true)
	if !insideQuietHours(p, time.Date(2026, 8, 2, 14, 0, 0, 0, ist)) {
		t.Error("14:00 should be inside a 13:00-15:00 window")
	}
	if insideQuietHours(p, time.Date(2026, 8, 2, 23, 0, 0, 0, ist)) {
		t.Error("23:00 must be outside a 13:00-15:00 window")
	}
}

// A broken preference must degrade to DELIVERING. Silence is the dangerous
// failure here: a chef missing a payout notification has no way to notice.
func TestQuietHoursFailsOpen(t *testing.T) {
	now := time.Now()
	if insideQuietHours(prefsAt("Asia/Kolkata", "22:00", "07:00", false), now) {
		t.Error("quiet hours disabled must never suppress")
	}
	if insideQuietHours(prefsAt("Not/AZone", "22:00", "07:00", true), now) {
		t.Error("an unparseable timezone must fail open")
	}
	if insideQuietHours(prefsAt("Asia/Kolkata", "boom", "07:00", true), now) {
		t.Error("an unparseable start must fail open")
	}
	if insideQuietHours(prefsAt("Asia/Kolkata", "09:00", "09:00", true), now) {
		t.Error("a zero-length window must fail open")
	}
}

// The topic map decides which toggle governs a push. An unmapped type must be
// chefTopicNone so a newly added chef notification is delivered by default
// rather than silently dropped by a gate that never heard of it.
func TestChefTopicForType(t *testing.T) {
	for notifType, want := range map[string]chefPushTopic{
		"chef_new_order":               chefTopicNewOrders,
		"accept_reminder":              chefTopicNewOrders,
		"payout_hold_released":         chefTopicPayouts,
		"payout_hold_disputed":         chefTopicPayouts,
		"payout_hold_release_eligible": chefTopicPayouts,
		"earnings_threshold_met":       chefTopicPayouts,
		"tip_received":                 chefTopicPayouts,
		"customer_message":             chefTopicCustomerMessages,
		"promo":                        chefTopicPromo,
		"order_status":                 chefTopicNone,
		"something_new":                chefTopicNone,
	} {
		if got := chefTopicForType(notifType); got != want {
			t.Errorf("chefTopicForType(%q)=%v, want %v", notifType, got, want)
		}
	}
}
