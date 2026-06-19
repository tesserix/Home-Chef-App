package services

// campaign_segment_test.go — the recipient segment engine for marketing
// campaigns (#56). Pins the four segment axes (role/status, zone, order recency,
// subscription) and the reachability filter (marketing consent + channel contact
// + explicit opt-out). These are the queries that decide who a campaign reaches,
// so the correctness of each filter — and especially that opt-out/consent is
// honored — is what's worth locking down.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupCampaignDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE users (
			id text PRIMARY KEY, email text, first_name text, last_name text, phone text,
			role text, is_active integer DEFAULT 1, fcm_token text DEFAULT '',
			marketing_consent integer DEFAULT 0, marketing_consent_at datetime,
			created_at datetime, updated_at datetime, deleted_at datetime
		)`,
		`CREATE TABLE addresses (
			id text PRIMARY KEY, user_id text, city text, state text,
			latitude real, longitude real, is_default integer, created_at datetime
		)`,
		`CREATE TABLE orders (
			id text PRIMARY KEY, customer_id text, chef_id text, status text,
			created_at datetime, updated_at datetime, deleted_at datetime
		)`,
		`CREATE TABLE meal_subscriptions (
			id text PRIMARY KEY, customer_id text, chef_id text, status text,
			created_at datetime, updated_at datetime, deleted_at datetime
		)`,
		`CREATE TABLE notification_preferences (
			id text PRIMARY KEY, user_id text, category text,
			email_enabled integer, push_enabled integer, sms_enabled integer,
			created_at datetime, updated_at datetime
		)`,
		`CREATE TABLE campaigns (
			id text PRIMARY KEY, name text, status text, send_push integer, send_email integer,
			push_title text, push_body text, email_subject text, email_html text, segment text,
			scheduled_at datetime, sent_at datetime, created_by text, recipients integer,
			created_at datetime, updated_at datetime, deleted_at datetime
		)`,
		`CREATE TABLE campaign_deliveries (
			id text PRIMARY KEY, campaign_id text, user_id text, channel text, status text,
			failure_reason text, sent_at datetime, opened_at datetime, created_at datetime, updated_at datetime
		)`,
		`CREATE UNIQUE INDEX idx_campaign_delivery_cell ON campaign_deliveries(campaign_id, user_id, channel)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT,
			payload TEXT, status TEXT, attempts INT, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

type seedUser struct {
	role     models.UserRole
	consent  bool
	fcm      string
	email    string
	city     string
	lastOrd  int  // days ago of last delivered order; -1 = never ordered
	sub      string
	ageDays  int  // account age in days
	optOut   string // "", "push", "email"
}

func seedCampaignUser(t *testing.T, db *gorm.DB, u seedUser) uuid.UUID {
	t.Helper()
	id := uuid.New()
	email := u.email
	if email == "" {
		email = id.String()[:8] + "@ex.com"
	}
	created := time.Now().AddDate(0, 0, -u.ageDays)
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, role, is_active, fcm_token, marketing_consent, created_at) VALUES (?,?,?,?,?,?,?)`,
		id.String(), email, string(u.role), 1, u.fcm, boolInt(u.consent), created,
	).Error)
	if u.city != "" {
		require.NoError(t, db.Exec(`INSERT INTO addresses (id, user_id, city) VALUES (?,?,?)`, uuid.New().String(), id.String(), u.city).Error)
	}
	if u.lastOrd >= 0 {
		require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, created_at) VALUES (?,?,?,?)`,
			uuid.New().String(), id.String(), string(models.OrderStatusDelivered), time.Now().AddDate(0, 0, -u.lastOrd)).Error)
	}
	if u.sub != "" {
		require.NoError(t, db.Exec(`INSERT INTO meal_subscriptions (id, customer_id, status, created_at) VALUES (?,?,?,?)`,
			uuid.New().String(), id.String(), u.sub, time.Now()).Error)
	}
	if u.optOut != "" {
		push, email := 1, 1
		if u.optOut == "push" {
			push = 0
		} else if u.optOut == "email" {
			email = 0
		}
		require.NoError(t, db.Exec(`INSERT INTO notification_preferences (id, user_id, category, email_enabled, push_enabled, sms_enabled) VALUES (?,?,?,?,?,?)`,
			uuid.New().String(), id.String(), string(models.NotifCategoryMarketing), email, push, 0).Error)
	}
	return id
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestResolveSegment_RoleAndStatus(t *testing.T) {
	db := setupCampaignDB(t)
	cust := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, lastOrd: -1, ageDays: 5})
	seedCampaignUser(t, db, seedUser{role: models.RoleChef, lastOrd: -1, ageDays: 5})

	ids, err := ResolveSegment(db, SegmentCriteria{}) // defaults to customers
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{cust}, ids)
}

func TestResolveSegment_Zone(t *testing.T) {
	db := setupCampaignDB(t)
	blr := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, city: "Bengaluru", lastOrd: -1})
	seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, city: "Mumbai", lastOrd: -1})

	ids, err := ResolveSegment(db, SegmentCriteria{Cities: []string{"Bengaluru"}})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{blr}, ids)
}

func TestResolveSegment_Recency(t *testing.T) {
	db := setupCampaignDB(t)
	active := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, lastOrd: 3})  // ordered 3 days ago
	lapsed := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, lastOrd: 90}) // 90 days ago
	seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, lastOrd: -1})           // never ordered

	got, err := ResolveSegment(db, SegmentCriteria{Recency: "active", RecencyDays: 30})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{active}, got)

	got, err = ResolveSegment(db, SegmentCriteria{Recency: "lapsed", RecencyDays: 30})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{lapsed}, got)
}

func TestResolveSegment_Subscription(t *testing.T) {
	db := setupCampaignDB(t)
	activeSub := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, lastOrd: -1, sub: models.MealSubStatusActive})
	noSub := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, lastOrd: -1})

	got, err := ResolveSegment(db, SegmentCriteria{Subscription: "active"})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{activeSub}, got)

	got, err = ResolveSegment(db, SegmentCriteria{Subscription: "none"})
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{noSub}, got)
}

func TestSegmentReachable_HonorsConsentContactAndOptOut(t *testing.T) {
	db := setupCampaignDB(t)
	// reachable on both channels
	ok := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, fcm: "tok", lastOrd: -1})
	// consented but no FCM token → not reachable on push, reachable on email
	noTok := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, fcm: "", lastOrd: -1})
	// no consent → reachable on neither
	seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: false, fcm: "tok", lastOrd: -1})
	// consented but opted out of marketing email
	optEmail := seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, fcm: "tok", lastOrd: -1, optOut: "email"})

	push, err := SegmentReachable(db, SegmentCriteria{}, models.CampaignChannelPush)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{ok, optEmail}, push) // optEmail still reachable on push

	email, err := SegmentReachable(db, SegmentCriteria{}, models.CampaignChannelEmail)
	require.NoError(t, err)
	require.ElementsMatch(t, []uuid.UUID{ok, noTok}, email) // optEmail excluded, noConsent excluded
}

func TestSegmentPreview_MatchedVsReachable(t *testing.T) {
	db := setupCampaignDB(t)
	seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: true, fcm: "tok", lastOrd: -1})
	seedCampaignUser(t, db, seedUser{role: models.RoleCustomer, consent: false, fcm: "tok", lastOrd: -1})

	p, err := SegmentPreview(db, SegmentCriteria{}, true, true)
	require.NoError(t, err)
	require.Equal(t, 2, p.Matched)
	require.Equal(t, 1, p.ReachablePush)  // only the consented user
	require.Equal(t, 1, p.ReachableEmail)
}
