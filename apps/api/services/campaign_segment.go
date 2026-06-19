package services

// campaign_segment.go — the recipient segment engine for marketing campaigns
// (#56). Translates an admin-built SegmentCriteria into a user query across four
// axes (role/status, delivery zone, order recency, subscription) and layers the
// reachability filter on top (marketing consent + channel contact + explicit
// opt-out). All queries are composed with GORM subqueries so they run unchanged
// on Postgres (prod) and the sqlite test driver, and so the segment is never
// materialized into a giant IN-list. Recency reuses the same delivered-order
// shape as winback's FindLapsedCustomers.

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// SegmentCriteria is the admin-defined recipient filter, stored as JSON on the
// Campaign. Zero values mean "no filter on that axis" (default audience = all
// active customers).
type SegmentCriteria struct {
	Roles         []string `json:"roles"`         // default ["customer"]
	Recency       string   `json:"recency"`       // "" (any) | "active" | "lapsed"
	RecencyDays   int      `json:"recencyDays"`   // window for active/lapsed (default 30)
	Cities        []string `json:"cities"`        // delivery-zone filter (address city); empty = all
	Subscription  string   `json:"subscription"`  // "" (any) | "active" | "paused" | "none"
	NewWithinDays int      `json:"newWithinDays"` // account age ≤ N days; 0 = ignore
}

// defaultRecencyDays is the active/lapsed window when the admin doesn't set one.
const defaultRecencyDays = 30

// buildSegmentQuery returns a fresh *gorm.DB scoped to the users matching the
// segment (no consent/contact filtering). Soft-deleted users/orders/subs are
// excluded automatically by GORM's deleted_at handling.
func buildSegmentQuery(db *gorm.DB, c SegmentCriteria) *gorm.DB {
	roles := c.Roles
	if len(roles) == 0 {
		roles = []string{string(models.RoleCustomer)}
	}
	q := db.Model(&models.User{}).Where("role IN ? AND is_active = ?", roles, true)

	if len(c.Cities) > 0 {
		q = q.Where("id IN (?)",
			db.Model(&models.Address{}).Select("user_id").Where("city IN ?", c.Cities))
	}

	switch c.Subscription {
	case "active":
		q = q.Where("id IN (?)",
			db.Model(&models.MealSubscription{}).Select("customer_id").Where("status = ?", models.MealSubStatusActive))
	case "paused":
		q = q.Where("id IN (?)",
			db.Model(&models.MealSubscription{}).Select("customer_id").Where("status = ?", models.MealSubStatusPaused))
	case "none":
		q = q.Where("id NOT IN (?)",
			db.Model(&models.MealSubscription{}).Select("customer_id").
				Where("status IN ?", []string{models.MealSubStatusActive, models.MealSubStatusTrialing, models.MealSubStatusPaused}))
	}

	recencyDays := c.RecencyDays
	if recencyDays <= 0 {
		recencyDays = defaultRecencyDays
	}
	cutoff := time.Now().AddDate(0, 0, -recencyDays)
	switch c.Recency {
	case "active": // a delivered order within the window
		q = q.Where("id IN (?)",
			db.Model(&models.Order{}).Select("customer_id").
				Where("status = ? AND created_at >= ?", models.OrderStatusDelivered, cutoff))
	case "lapsed": // ordered before, but the most recent delivered order predates the window
		q = q.Where("id IN (?)",
			db.Model(&models.Order{}).Select("customer_id").
				Where("status = ?", models.OrderStatusDelivered).
				Group("customer_id").Having("MAX(created_at) < ?", cutoff))
	}

	if c.NewWithinDays > 0 {
		q = q.Where("created_at >= ?", time.Now().AddDate(0, 0, -c.NewWithinDays))
	}
	return q
}

// ResolveSegment returns every user id matching the segment (before reachability).
func ResolveSegment(db *gorm.DB, c SegmentCriteria) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if err := buildSegmentQuery(db, c).Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// SegmentReachable returns the subset of the segment actually reachable on a
// channel: marketing consent granted (DPDP §6), the channel's contact present
// (FCM token / email), and not explicitly opted out of marketing for that
// channel. This is the single source of truth for both the preview count and the
// real dispatch recipient list, so what the admin sees is exactly what gets sent.
func SegmentReachable(db *gorm.DB, c SegmentCriteria, channel string) ([]uuid.UUID, error) {
	q := buildSegmentQuery(db, c).Where("marketing_consent = ?", true)

	col := "email_enabled"
	if channel == models.CampaignChannelPush {
		q = q.Where("fcm_token <> ''")
		col = "push_enabled"
	} else {
		q = q.Where("email <> ''")
	}

	// Explicit per-channel marketing opt-out (a preference row that disables it).
	q = q.Where("id NOT IN (?)",
		db.Model(&models.NotificationPreference{}).Select("user_id").
			Where("category = ? AND "+col+" = ?", models.NotifCategoryMarketing, false))

	var ids []uuid.UUID
	if err := q.Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// SegmentPreviewResult is the audience breakdown shown before sending so the
// admin sees how consent/opt-out narrows the matched audience per channel.
type SegmentPreviewResult struct {
	Matched        int `json:"matched"`
	ReachablePush  int `json:"reachablePush"`
	ReachableEmail int `json:"reachableEmail"`
}

// SegmentPreview counts matched vs reachable for the requested channels.
func SegmentPreview(db *gorm.DB, c SegmentCriteria, sendPush, sendEmail bool) (SegmentPreviewResult, error) {
	var res SegmentPreviewResult
	matched, err := ResolveSegment(db, c)
	if err != nil {
		return res, err
	}
	res.Matched = len(matched)

	if sendPush {
		push, err := SegmentReachable(db, c, models.CampaignChannelPush)
		if err != nil {
			return res, err
		}
		res.ReachablePush = len(push)
	}
	if sendEmail {
		email, err := SegmentReachable(db, c, models.CampaignChannelEmail)
		if err != nil {
			return res, err
		}
		res.ReachableEmail = len(email)
	}
	return res, nil
}
