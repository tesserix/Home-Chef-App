package services

// docs_deadline_cron_test.go — the 30-day document window state machine:
// nothing before day 25, exactly one warning in the last 5 days, withdrawal
// at day 30, and complete-docs chefs untouched throughout.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

var indiaDocs = []string{string(models.DocIDProof), string(models.DocAddressProof), string(models.DocFSSAILicense)}

func setupDocsDeadlineDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id text PRIMARY KEY, user_id text, business_name text DEFAULT '',
		is_verified integer DEFAULT 0, is_active integer DEFAULT 1,
		accepting_orders integer DEFAULT 0, payout_method text DEFAULT '', payout_country text DEFAULT 'IN',
		onboarded_at datetime, docs_warning_sent_at datetime, payout_reminder_sent_at datetime,
		created_at datetime, updated_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_documents (
		id text PRIMARY KEY, chef_id text, type text, status text DEFAULT 'pending',
		expiry_date datetime, created_at datetime, updated_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE approval_requests (
		id text PRIMARY KEY, type text, status text DEFAULT 'pending',
		chef_id text, submitted_by_id text, admin_notes text DEFAULT '',
		created_at datetime, updated_at datetime
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT,
		payload TEXT, status TEXT, attempts INT, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`).Error)
	return db
}

func seedDeadlineChef(t *testing.T, db *gorm.DB, onboardedAgo time.Duration, docTypes []string) uuid.UUID {
	t.Helper()
	chefID, userID := uuid.New(), uuid.New()
	onboardedAt := time.Now().Add(-onboardedAgo)
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, onboarded_at, created_at, updated_at)
		 VALUES (?, ?, 'Window Kitchen', ?, ?, ?)`,
		chefID.String(), userID.String(), onboardedAt, time.Now(), time.Now()).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO approval_requests (id, type, status, chef_id, submitted_by_id, created_at, updated_at)
		 VALUES (?, ?, 'pending', ?, ?, ?, ?)`,
		uuid.New().String(), string(models.ApprovalKitchenOnboarding), chefID.String(), userID.String(), time.Now(), time.Now()).Error)
	for _, dt := range docTypes {
		require.NoError(t, db.Exec(
			`INSERT INTO chef_documents (id, chef_id, type, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			uuid.New().String(), chefID.String(), dt, time.Now(), time.Now()).Error)
	}
	return chefID
}

func TestDocsDeadline_QuietBeforeWarningWindow(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	seedDeadlineChef(t, db, 10*24*time.Hour, nil) // day 10 — nothing due

	warned, withdrawn := sweepDocsDeadlines(db, time.Now())
	require.Zero(t, warned)
	require.Zero(t, withdrawn)
}

func TestDocsDeadline_WarnsOnceInFinalFiveDays(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	chefID := seedDeadlineChef(t, db, 26*24*time.Hour, nil) // day 26 — warning due

	warned, withdrawn := sweepDocsDeadlines(db, time.Now())
	require.Equal(t, 1, warned)
	require.Zero(t, withdrawn)

	var stamped int
	db.Raw(`SELECT COUNT(*) FROM chef_profiles WHERE id = ? AND docs_warning_sent_at IS NOT NULL`, chefID.String()).Scan(&stamped)
	require.Equal(t, 1, stamped)
	var events int
	db.Raw(`SELECT COUNT(*) FROM outbox_events WHERE subject = ?`, SubjectChefDocsDeadlineWarning).Scan(&events)
	require.Equal(t, 1, events)

	// Second sweep: the stamp holds, no duplicate warning.
	warned, withdrawn = sweepDocsDeadlines(db, time.Now())
	require.Zero(t, warned)
	require.Zero(t, withdrawn)
}

func TestDocsDeadline_WithdrawsAfterThirtyDays(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	chefID := seedDeadlineChef(t, db, 31*24*time.Hour, nil) // day 31 — expired

	warned, withdrawn := sweepDocsDeadlines(db, time.Now())
	require.Zero(t, warned)
	require.Equal(t, 1, withdrawn)

	var status string
	db.Raw(`SELECT status FROM approval_requests WHERE chef_id = ?`, chefID.String()).Scan(&status)
	require.Equal(t, string(models.ApprovalRejected), status)
	var cleared int
	db.Raw(`SELECT COUNT(*) FROM chef_profiles WHERE id = ? AND onboarded_at IS NULL`, chefID.String()).Scan(&cleared)
	require.Equal(t, 1, cleared)
	var events int
	db.Raw(`SELECT COUNT(*) FROM outbox_events WHERE subject = ?`, SubjectChefDocsDeadlineExpired).Scan(&events)
	require.Equal(t, 1, events)

	// Second sweep: the withdrawn chef is out of scope for good.
	warned, withdrawn = sweepDocsDeadlines(db, time.Now())
	require.Zero(t, warned)
	require.Zero(t, withdrawn)
}

func TestPayoutReminder_NudgesOnceAfterDay25(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	seedDeadlineChef(t, db, 26*24*time.Hour, indiaDocs) // docs done, payout not

	nudged := sweepPayoutReminders(db, time.Now())
	require.Equal(t, 1, nudged)
	var events int
	db.Raw(`SELECT COUNT(*) FROM outbox_events WHERE subject = ?`, SubjectChefPayoutReminder).Scan(&events)
	require.Equal(t, 1, events)

	// Stamp holds — never a second nudge.
	require.Zero(t, sweepPayoutReminders(db, time.Now()))
}

func TestPayoutReminder_SkipsChefsWithPayoutMethod(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	chefID := seedDeadlineChef(t, db, 26*24*time.Hour, indiaDocs)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET payout_method = 'bank_transfer' WHERE id = ?`, chefID.String()).Error)

	require.Zero(t, sweepPayoutReminders(db, time.Now()))
}

func TestDocsDeadline_CompleteDocsAreLeftAlone(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	chefID := seedDeadlineChef(t, db, 31*24*time.Hour, indiaDocs) // docs in, even past day 30

	warned, withdrawn := sweepDocsDeadlines(db, time.Now())
	require.Zero(t, warned)
	require.Zero(t, withdrawn)

	var status string
	db.Raw(`SELECT status FROM approval_requests WHERE chef_id = ?`, chefID.String()).Scan(&status)
	require.Equal(t, string(models.ApprovalPending), status, "docs-complete application stays in the admin queue")
}

func TestChefDocsComplete_IgnoresRejectedAndExpiredDocuments(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	chefID := seedDeadlineChef(t, db, time.Hour, indiaDocs)
	require.True(t, ChefDocsComplete(db, chefID))

	require.NoError(t, db.Exec(`UPDATE chef_documents SET status = 'rejected' WHERE chef_id = ? AND type = ?`,
		chefID.String(), string(models.DocFSSAILicense)).Error)
	require.False(t, ChefDocsComplete(db, chefID), "a rejected licence must be re-uploaded")

	require.NoError(t, db.Exec(`UPDATE chef_documents SET status = 'verified', expiry_date = ? WHERE chef_id = ? AND type = ?`,
		time.Now().AddDate(0, 0, -2), chefID.String(), string(models.DocFSSAILicense)).Error)
	require.False(t, ChefDocsComplete(db, chefID), "an expired licence does not count")
}

func TestChefDocsComplete_UnsupportedCountryFailsClosed(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	chefID := seedDeadlineChef(t, db, time.Hour, append(indiaDocs, string(models.DocFoodSafetyCert)))
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET payout_country = 'US' WHERE id = ?`, chefID.String()).Error)
	require.False(t, ChefDocsComplete(db, chefID))
	require.False(t, ChefDocsComplete(db, uuid.New()), "a missing kitchen is never complete")
}

func TestChefDocsComplete_FollowsKitchenCountry(t *testing.T) {
	db := setupDocsDeadlineDB(t)
	au := seedDeadlineChef(t, db, time.Hour, []string{
		string(models.DocIDProof), string(models.DocAddressProof), string(models.DocFoodSafetyCert)})
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET payout_country = 'AU' WHERE id = ?`, au.String()).Error)
	require.True(t, ChefDocsComplete(db, au), "council certificate stands in for FSSAI in AU")

	auWithFSSAI := seedDeadlineChef(t, db, time.Hour, []string{
		string(models.DocIDProof), string(models.DocAddressProof), string(models.DocFSSAILicense)})
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET payout_country = 'AU' WHERE id = ?`, auWithFSSAI.String()).Error)
	require.False(t, ChefDocsComplete(db, auWithFSSAI), "an FSSAI licence is not an Australian registration")

	in := seedDeadlineChef(t, db, time.Hour, []string{
		string(models.DocIDProof), string(models.DocAddressProof), string(models.DocFoodSafetyCert)})
	require.False(t, ChefDocsComplete(db, in), "Indian kitchens still need FSSAI")
}
