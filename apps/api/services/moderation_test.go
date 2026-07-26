package services

// moderation_test.go — the report/block primitives behind App Review 1.2.
//
// Schema is hand-rolled rather than AutoMigrate'd: the models carry
// `default:gen_random_uuid()`, a Postgres function that makes gorm emit invalid
// SQLite DDL. Only the columns these functions touch are declared.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupModerationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_pragma=foreign_keys(0)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Each test gets its own tables — the shared-cache DSN above otherwise
	// leaks rows between tests and turns duplicate-report assertions into
	// order-dependent flakes.
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS content_reports`,
		`DROP TABLE IF EXISTS user_blocks`,
		`DROP TABLE IF EXISTS reviews`,
		`DROP TABLE IF EXISTS posts`,
		`DROP TABLE IF EXISTS post_comments`,
		`DROP TABLE IF EXISTS chef_profiles`,
		`CREATE TABLE content_reports (
			id text PRIMARY KEY, reporter_id text, target_type text, target_id text,
			target_owner_id text, reason text, details text, status text,
			reviewed_by text, reviewed_at datetime, resolution_note text,
			created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE user_blocks (
			id text PRIMARY KEY, blocker_id text, blocked_id text, reason text,
			created_at datetime)`,
		// deleted_at and the mode partition columns are declared because the
		// models carry gorm.DeletedAt / ModePartition — gorm appends
		// `deleted_at IS NULL` to every query, and omitting the column fails
		// the whole statement rather than being ignored.
		`CREATE TABLE reviews (
			id text PRIMARY KEY, customer_id text, chef_id text,
			is_hidden numeric DEFAULT 0, hidden_reason text,
			mode text DEFAULT 'live', test_session_id text, cloned_from_id text,
			created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE posts (
			id text PRIMARY KEY, chef_id text,
			is_moderated numeric DEFAULT 0, moderator_note text,
			created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE post_comments (
			id text PRIMARY KEY, post_id text, user_id text, is_hidden numeric DEFAULT 0,
			created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE chef_profiles (id text PRIMARY KEY, user_id text)`,
	} {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

// newID keeps the tests readable — gorm's uuid default does not fire on SQLite,
// so ids are supplied explicitly.
func newID() uuid.UUID { return uuid.New() }

func seedReview(t *testing.T, db *gorm.DB, customerID uuid.UUID) uuid.UUID {
	t.Helper()
	id := newID()
	require.NoError(t, db.Exec(
		`INSERT INTO reviews (id, customer_id, chef_id) VALUES (?,?,?)`,
		id.String(), customerID.String(), newID().String()).Error)
	return id
}

func TestResolveTargetOwnerReview(t *testing.T) {
	db := setupModerationDB(t)
	author := newID()
	reviewID := seedReview(t, db, author)

	got, err := ResolveTargetOwner(db, models.ReportableReview, reviewID)
	require.NoError(t, err)
	require.Equal(t, author, got)
}

// A post is owned by a chef PROFILE, but reports are about people — the
// resolver must walk through to the underlying user, or the triage queue counts
// reports against an id that is not a user.
func TestResolveTargetOwnerSocialPostWalksToUser(t *testing.T) {
	db := setupModerationDB(t)
	chefUser, chefProfile, postID := newID(), newID(), newID()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		chefProfile.String(), chefUser.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO posts (id, chef_id) VALUES (?,?)`,
		postID.String(), chefProfile.String()).Error)

	got, err := ResolveTargetOwner(db, models.ReportableSocialPost, postID)
	require.NoError(t, err)
	require.Equal(t, chefUser, got, "should resolve to the chef's user id, not the profile id")
}

func TestResolveTargetOwnerMissingContent(t *testing.T) {
	db := setupModerationDB(t)
	_, err := ResolveTargetOwner(db, models.ReportableReview, newID())
	require.ErrorIs(t, err, ErrTargetNotFound)
}

func TestCreateReportSnapshotsOwner(t *testing.T) {
	db := setupModerationDB(t)
	author, reporter := newID(), newID()
	reviewID := seedReview(t, db, author)

	report, err := CreateReport(db, reporter, models.ReportableReview, reviewID,
		models.ReasonHarassment, "abusive language")
	require.NoError(t, err)
	require.Equal(t, models.ReportPending, report.Status)
	require.NotNil(t, report.TargetOwnerID)
	require.Equal(t, author, *report.TargetOwnerID)
}

// Tapping Report twice must not create two queue entries.
func TestCreateReportIsIdempotentPerReporter(t *testing.T) {
	db := setupModerationDB(t)
	reporter := newID()
	reviewID := seedReview(t, db, newID())

	first, err := CreateReport(db, reporter, models.ReportableReview, reviewID, models.ReasonSpam, "")
	require.NoError(t, err)

	second, err := CreateReport(db, reporter, models.ReportableReview, reviewID, models.ReasonSpam, "")
	require.ErrorIs(t, err, ErrDuplicateReport)
	require.Equal(t, first.ID, second.ID, "the existing report should come back, not a new one")

	var count int64
	require.NoError(t, db.Model(&models.ContentReport{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

// Reporting content that does not exist must not create a queue entry — it is
// either a stale client or someone probing ids.
func TestCreateReportRejectsMissingTarget(t *testing.T) {
	db := setupModerationDB(t)
	_, err := CreateReport(db, newID(), models.ReportableReview, newID(), models.ReasonSpam, "")
	require.ErrorIs(t, err, ErrTargetNotFound)

	var count int64
	require.NoError(t, db.Model(&models.ContentReport{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestBlockUserIsIdempotent(t *testing.T) {
	db := setupModerationDB(t)
	a, b := newID(), newID()

	require.NoError(t, BlockUser(db, a, b, models.ReasonHarassment))
	require.NoError(t, BlockUser(db, a, b, models.ReasonHarassment))

	var count int64
	require.NoError(t, db.Model(&models.UserBlock{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestBlockUserRejectsSelfBlock(t *testing.T) {
	db := setupModerationDB(t)
	a := newID()
	require.ErrorIs(t, BlockUser(db, a, a, models.ReasonOther), ErrSelfBlock)
}

func TestUnblockUser(t *testing.T) {
	db := setupModerationDB(t)
	a, b := newID(), newID()
	require.NoError(t, BlockUser(db, a, b, models.ReasonSpam))
	require.NoError(t, UnblockUser(db, a, b))

	ids, err := BlockedUserIDs(db, a)
	require.NoError(t, err)
	require.Empty(t, ids)
}

// The feed builds a NOT IN clause from this. A nil slice would produce
// `NOT IN (NULL)`, which matches nothing and silently empties the feed — so an
// empty, non-nil slice is part of the contract.
func TestBlockedUserIDsReturnsEmptyNotNil(t *testing.T) {
	db := setupModerationDB(t)

	ids, err := BlockedUserIDs(db, newID())
	require.NoError(t, err)
	require.NotNil(t, ids)
	require.Empty(t, ids)

	// Anonymous browsing: no user, nobody blocked, no query.
	ids, err = BlockedUserIDs(db, uuid.Nil)
	require.NoError(t, err)
	require.NotNil(t, ids)
	require.Empty(t, ids)
}

// A one-way check leaves an obvious harassment path open, so messaging asks
// both directions.
func TestIsBlockedEitherWay(t *testing.T) {
	db := setupModerationDB(t)
	customer, chef := newID(), newID()

	blocked, err := IsBlockedEitherWay(db, customer, chef)
	require.NoError(t, err)
	require.False(t, blocked)

	require.NoError(t, BlockUser(db, chef, customer, models.ReasonHarassment))

	blocked, err = IsBlockedEitherWay(db, customer, chef)
	require.NoError(t, err)
	require.True(t, blocked, "a block by either party must stop the conversation")
}

func TestPendingReportCountCountsDistinctReporters(t *testing.T) {
	db := setupModerationDB(t)
	reviewID := seedReview(t, db, newID())

	for i := 0; i < 2; i++ {
		_, err := CreateReport(db, newID(), models.ReportableReview, reviewID, models.ReasonSpam, "")
		require.NoError(t, err)
	}

	count, err := PendingReportCount(db, models.ReportableReview, reviewID)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
}

// Below the threshold nothing is hidden — one angry customer must not be able
// to suppress a chef.
func TestApplyAutoHideBelowThresholdLeavesContentVisible(t *testing.T) {
	db := setupModerationDB(t)
	reviewID := seedReview(t, db, newID())

	_, err := CreateReport(db, newID(), models.ReportableReview, reviewID, models.ReasonSpam, "")
	require.NoError(t, err)

	hidden, err := ApplyAutoHide(db, models.ReportableReview, reviewID)
	require.NoError(t, err)
	require.False(t, hidden)

	var isHidden bool
	require.NoError(t, db.Raw(`SELECT is_hidden FROM reviews WHERE id = ?`, reviewID.String()).
		Scan(&isHidden).Error)
	require.False(t, isHidden)
}

// At the threshold the content is hidden pending triage. This is the "filtering
// objectionable content" half of guideline 1.2 — a queue nobody reads at 3am is
// not a filter.
func TestApplyAutoHideAtThresholdHidesReview(t *testing.T) {
	db := setupModerationDB(t)
	reviewID := seedReview(t, db, newID())

	for i := 0; i < AutoHideThreshold; i++ {
		_, err := CreateReport(db, newID(), models.ReportableReview, reviewID, models.ReasonHateSpeech, "")
		require.NoError(t, err)
	}

	hidden, err := ApplyAutoHide(db, models.ReportableReview, reviewID)
	require.NoError(t, err)
	require.True(t, hidden)

	var isHidden bool
	require.NoError(t, db.Raw(`SELECT is_hidden FROM reviews WHERE id = ?`, reviewID.String()).
		Scan(&isHidden).Error)
	require.True(t, isHidden)
}

func TestApplyAutoHideHidesSocialPost(t *testing.T) {
	db := setupModerationDB(t)
	chefProfile, postID := newID(), newID()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		chefProfile.String(), newID().String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO posts (id, chef_id) VALUES (?,?)`,
		postID.String(), chefProfile.String()).Error)

	for i := 0; i < AutoHideThreshold; i++ {
		_, err := CreateReport(db, newID(), models.ReportableSocialPost, postID, models.ReasonSexualContent, "")
		require.NoError(t, err)
	}

	hidden, err := ApplyAutoHide(db, models.ReportableSocialPost, postID)
	require.NoError(t, err)
	require.True(t, hidden)

	var isModerated bool
	require.NoError(t, db.Raw(`SELECT is_moderated FROM posts WHERE id = ?`, postID.String()).
		Scan(&isModerated).Error)
	require.True(t, isModerated)
}

// Suppressing a whole account or a delivery conversation on report volume alone
// is a denial of service against the reported party, so those go to a human.
func TestApplyAutoHideSkipsUserAndMessageTargets(t *testing.T) {
	db := setupModerationDB(t)
	targetID := newID()

	for _, tt := range []models.ReportableType{models.ReportableUser, models.ReportableMessage} {
		hidden, err := ApplyAutoHide(db, tt, targetID)
		require.NoError(t, err)
		require.False(t, hidden, "%s must not be auto-hidden", tt)
	}
}

func TestValidReportableTypeAndReason(t *testing.T) {
	require.True(t, models.ValidReportableType(models.ReportableReview))
	require.False(t, models.ValidReportableType(models.ReportableType("order")))
	require.True(t, models.ValidReportReason(models.ReasonFoodSafety))
	require.False(t, models.ValidReportReason(models.ReportReason("because")))
}
