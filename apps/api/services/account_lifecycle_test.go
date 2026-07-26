package services

// account_lifecycle_test.go — the account state machine and its role cascades.
//
// The tests that matter most here are the guardrails, not the happy paths: a
// restored chef must come back UNAPPROVED with their identity documents gone,
// and a deactivated kitchen must not be reopened by the schedule cron. Both are
// silent-failure modes in production — nothing errors, the account is just
// wrongly live.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupAccountDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	// email_enc/…_bidx are the #710 PII-encryption companions on users; GORM
	// writes them on every insert, so the fixture must carry them.
	require.NoError(t, db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT, first_name TEXT,
		last_name TEXT, phone TEXT, avatar TEXT, role TEXT,
		email_enc TEXT DEFAULT '', email_bidx TEXT DEFAULT '', first_name_enc TEXT DEFAULT '',
		last_name_enc TEXT DEFAULT '', phone_enc TEXT DEFAULT '', phone_bidx TEXT DEFAULT '',
		gip_uid TEXT, gip_tenant_id TEXT,
		gip_provider TEXT, auth_pool TEXT, apple_refresh_token_enc TEXT,
		is_active BOOLEAN DEFAULT 1, phone_verified BOOLEAN DEFAULT 0,
		fcm_token TEXT DEFAULT '', marketing_consent BOOLEAN DEFAULT 0, marketing_consent_at DATETIME,
		deactivated_at DATETIME, purge_after DATETIME, deletion_reason TEXT DEFAULT '',
		restored_at DATETIME, last_login_at DATETIME, created_at DATETIME, updated_at DATETIME,
		deleted_at DATETIME)`).Error)
	// NOTE: chef_profiles, addresses, chef_documents and delivery_partners have
	// NO deleted_at, exactly as in production. Adding one here would let a
	// cascade that hard-deletes them appear to pass while destroying data the
	// restore window promises to keep.
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (mode text DEFAULT 'live', first_live_at datetime, active_test_session_id text, id TEXT PRIMARY KEY, user_id TEXT,
		is_verified BOOLEAN DEFAULT 0, verified_at DATETIME, is_active BOOLEAN DEFAULT 1,
		accepting_orders BOOLEAN DEFAULT 1, auto_schedule_enabled BOOLEAN DEFAULT 0,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE menu_items (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id TEXT PRIMARY KEY, chef_id TEXT,
		is_available BOOLEAN DEFAULT 1, is_approved BOOLEAN DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_documents (id TEXT PRIMARY KEY, chef_id TEXT,
		type TEXT, file_name TEXT, file_path TEXT, bucket TEXT, status TEXT,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_schedules (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id TEXT PRIMARY KEY, chef_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_settings (id TEXT PRIMARY KEY, chef_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE addresses (id TEXT PRIMARY KEY, user_id TEXT,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_partners (id TEXT PRIMARY KEY, user_id TEXT,
		is_verified BOOLEAN DEFAULT 0, is_active BOOLEAN DEFAULT 1, is_online BOOLEAN DEFAULT 0,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_partner_documents (id TEXT PRIMARY KEY,
		partner_id TEXT, type TEXT, file_name TEXT, file_path TEXT, bucket TEXT, status TEXT,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE driver_referrals (id TEXT PRIMARY KEY, referrer_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE approval_requests (approved_mode text DEFAULT 'live', id TEXT PRIMARY KEY, type TEXT,
		status TEXT, priority TEXT, chef_id TEXT, partner_id TEXT, submitted_by_id TEXT,
		reviewed_by_id TEXT, entity_type TEXT, entity_id TEXT, title TEXT, description TEXT,
		submitted_data TEXT, admin_notes TEXT, reviewed_at DATETIME, expires_at DATETIME,
		reminder_count INT DEFAULT 0, last_reminded_at DATETIME, escalated_at DATETIME,
		created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE orders (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id TEXT PRIMARY KEY, customer_id TEXT,
		chef_id TEXT, status TEXT, payout_hold_status TEXT DEFAULT '',
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plans (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, id TEXT PRIMARY KEY, customer_id TEXT,
		chef_id TEXT, status TEXT, total REAL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT,
		balance REAL DEFAULT 0, created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE deliveries (id TEXT PRIMARY KEY,
		delivery_partner_id TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedAccountUser(t *testing.T, db *gorm.DB, role models.UserRole) models.User {
	t.Helper()
	u := models.User{
		ID:       uuid.New(),
		Email:    "user-" + uuid.NewString()[:8] + "@example.com",
		Role:     role,
		IsActive: true,
		AuthPool: models.PoolCustomer,
		GIPUid:   "gip-" + uuid.NewString()[:8],
	}
	require.NoError(t, db.Create(&u).Error)
	return u
}

// seedChef creates an approved, open kitchen with one approved menu item and a
// full set of identity documents — the state a live chef is actually in.
func seedAccountChef(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	chefID := uuid.New()
	now := time.Now().UTC()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, is_verified, verified_at,
		is_active, accepting_orders, auto_schedule_enabled) VALUES (?,?,?,?,?,?,?)`,
		chefID.String(), userID.String(), true, now, true, true, true).Error)
	require.NoError(t, db.Exec(`INSERT INTO menu_items (id, chef_id, is_available, is_approved)
		VALUES (?,?,?,?)`, uuid.NewString(), chefID.String(), true, true).Error)
	for _, dt := range []models.DocumentType{
		models.DocPanCard, models.DocFSSAILicense, models.DocKitchenPhoto1,
	} {
		require.NoError(t, db.Exec(`INSERT INTO chef_documents (id, chef_id, type, file_name,
			file_path, bucket, status) VALUES (?,?,?,?,?,?,?)`,
			uuid.NewString(), chefID.String(), string(dt), "f.jpg", "", "", "approved").Error)
	}
	return chefID
}

func TestDeactivate_PausesAccountAndClosesKitchen(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleChef)
	chefID := seedAccountChef(t, db, user.ID)

	require.NoError(t, Deactivate(db, &user, "taking a break"))

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, "id = ?", user.ID).Error)
	require.False(t, reloaded.IsActive, "user should be paused")
	require.NotNil(t, reloaded.DeactivatedAt)
	require.False(t, reloaded.DeletedAt.Valid, "deactivate must not soft-delete")

	var accepting, autoSchedule bool
	require.NoError(t, db.Raw(`SELECT accepting_orders, auto_schedule_enabled
		FROM chef_profiles WHERE id = ?`, chefID.String()).Row().Scan(&accepting, &autoSchedule))
	require.False(t, accepting, "kitchen must stop accepting orders")
	// The load-bearing assertion: the kitchen-schedule cron flips
	// accepting_orders back on from operating hours, so leaving
	// auto_schedule_enabled set would silently reopen a paused kitchen.
	require.False(t, autoSchedule, "auto-schedule must be off or the cron reopens the kitchen")
}

func TestDeactivate_IsIdempotent(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)

	require.NoError(t, Deactivate(db, &user, "first"))
	first := user.DeactivatedAt
	require.NoError(t, Deactivate(db, &user, "second"))
	require.Equal(t, first, user.DeactivatedAt, "second deactivate must not restamp")
}

func TestReactivate_RestoresAccessWithoutResettingApproval(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleChef)
	chefID := seedAccountChef(t, db, user.ID)

	require.NoError(t, Deactivate(db, &user, "break"))
	require.NoError(t, Reactivate(db, &user))

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, "id = ?", user.ID).Error)
	require.True(t, reloaded.IsActive)
	require.Nil(t, reloaded.DeactivatedAt)

	// A pause is not a deletion: the chef never left the trust boundary, so
	// their verification must survive it.
	var verified bool
	require.NoError(t, db.Raw(`SELECT is_verified FROM chef_profiles WHERE id = ?`,
		chefID.String()).Row().Scan(&verified))
	require.True(t, verified, "reactivate must not reset approval")
}

func TestRequestDeletion_BlockedByActiveOrder(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, chef_id, status)
		VALUES (?,?,?,?)`, uuid.NewString(), user.ID.String(), uuid.NewString(),
		string(models.OrderStatusPreparing)).Error)

	blockers, err := RequestDeletion(db, &user, "")
	require.ErrorIs(t, err, ErrBlocked)
	require.Len(t, blockers, 1)
	require.Equal(t, BlockerActiveOrders, blockers[0].Code)

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, "id = ?", user.ID).Error)
	require.False(t, reloaded.DeletedAt.Valid, "blocked deletion must not soft-delete")
}

func TestRequestDeletion_BlockedByWalletBalance(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance) VALUES (?,?,?)`,
		uuid.NewString(), user.ID.String(), 250.0).Error)

	blockers, err := RequestDeletion(db, &user, "")
	require.ErrorIs(t, err, ErrBlocked)
	require.Equal(t, BlockerWalletBalance, blockers[0].Code)
	require.Equal(t, 250.0, blockers[0].Amount)
}

func TestRequestDeletion_CleanAccountSoftDeletesAndStampsWindow(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)
	require.NoError(t, db.Exec(`INSERT INTO addresses (id, user_id) VALUES (?,?)`,
		uuid.NewString(), user.ID.String()).Error)

	blockers, err := RequestDeletion(db, &user, "no longer needed")
	require.NoError(t, err)
	require.Empty(t, blockers)

	// Gone from ordinary queries...
	var visible int64
	db.Model(&models.User{}).Where("id = ?", user.ID).Count(&visible)
	require.Zero(t, visible, "deleted user must be invisible to scoped queries")

	// ...but still there, with a restore window ~180 days out.
	var stored models.User
	require.NoError(t, db.Unscoped().First(&stored, "id = ?", user.ID).Error)
	require.True(t, stored.DeletedAt.Valid)
	require.NotNil(t, stored.PurgeAfter)
	require.WithinDuration(t, time.Now().UTC().Add(RestoreWindow), *stored.PurgeAfter, time.Minute)

	// Addresses survive the window. models.Address has no DeletedAt, so
	// removing it here would be a hard delete and the restore promise would be
	// unkeepable. They are unreachable anyway: the account is soft-deleted and
	// inactive, and addresses are only readable through the owner's session.
	var addrKept int64
	db.Model(&models.Address{}).Where("user_id = ?", user.ID).Count(&addrKept)
	require.Equal(t, int64(1), addrKept, "addresses must survive for the restore window")
}

func TestRequestDeletion_IsIdempotent(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)

	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)
	first := *user.PurgeAfter

	_, err = RequestDeletion(db, &user, "")
	require.NoError(t, err, "re-deleting must be a no-op success")
	require.Equal(t, first, *user.PurgeAfter, "window must not be extended by a retry")
}

// The central guardrail: restoring returns the chef's data but NOT their
// standing. They re-enter the approval queue like a new applicant.
func TestRestore_ReturnsDataButResetsApproval(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleChef)
	chefID := seedAccountChef(t, db, user.ID)

	_, err := RequestDeletion(db, &user, "closing up")
	require.NoError(t, err)

	restored, err := Restore(db, user.ID, "new-gip-uid", "tenant-x", "google.com")
	require.NoError(t, err)
	require.True(t, restored.IsActive)
	require.NotNil(t, restored.RestoredAt)
	require.Equal(t, "new-gip-uid", restored.GIPUid, "must rebind to the new identity")
	require.Nil(t, restored.PurgeAfter, "restore must clear the purge window")

	var verified, accepting, autoSchedule bool
	require.NoError(t, db.Raw(`SELECT is_verified, accepting_orders, auto_schedule_enabled
		FROM chef_profiles WHERE id = ?`, chefID.String()).Row().
		Scan(&verified, &accepting, &autoSchedule))
	require.False(t, verified, "restored chef must NOT be approved")
	require.False(t, accepting, "restored kitchen must not accept orders")
	require.False(t, autoSchedule)

	var approvedItems int64
	db.Raw(`SELECT COUNT(*) FROM menu_items WHERE chef_id = ? AND is_approved = 1`,
		chefID.String()).Scan(&approvedItems)
	require.Zero(t, approvedItems, "every menu item must return unapproved for re-review")

	// Identity documents are dropped so nothing expired or revoked slips back
	// in; the kitchen photo is the user's own content and survives.
	var idDocs, photos int64
	db.Raw(`SELECT COUNT(*) FROM chef_documents WHERE chef_id = ? AND type IN ('pan_card','fssai_license')`,
		chefID.String()).Scan(&idDocs)
	db.Raw(`SELECT COUNT(*) FROM chef_documents WHERE chef_id = ? AND type = 'kitchen_photo_1'`,
		chefID.String()).Scan(&photos)
	require.Zero(t, idDocs, "identity documents must be removed and re-uploaded")
	require.Equal(t, int64(1), photos, "kitchen photos are the user's own content and survive")

	var pending int64
	db.Raw(`SELECT COUNT(*) FROM approval_requests WHERE entity_id = ? AND status = 'pending'`,
		chefID.String()).Scan(&pending)
	require.Equal(t, int64(1), pending, "a re-approval request must be queued")
}

func TestRestore_RefusedAfterWindowExpires(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)
	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)

	// Wind the window back into the past.
	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, db.Exec(`UPDATE users SET purge_after = ? WHERE id = ?`,
		past, user.ID.String()).Error)

	_, err = Restore(db, user.ID, "uid", "t", "p")
	require.Error(t, err, "an expired account must not be restorable")
}

func TestRestore_CustomerRegainsAddresses(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleCustomer)
	require.NoError(t, db.Exec(`INSERT INTO addresses (id, user_id) VALUES (?,?)`,
		uuid.NewString(), user.ID.String()).Error)

	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)
	_, err = Restore(db, user.ID, "uid", "t", "p")
	require.NoError(t, err)

	var count int64
	db.Model(&models.Address{}).Where("user_id = ?", user.ID).Count(&count)
	require.Equal(t, int64(1), count, "addresses must come back on restore")
}

func TestPurgeUser_RemovesEverythingAndIsIdempotent(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleChef)
	chefID := seedAccountChef(t, db, user.ID)

	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)
	require.NoError(t, PurgeUser(db, user.ID, models.RoleChef))

	for _, q := range []struct {
		table string
		sql   string
		arg   string
	}{
		{"users", `SELECT COUNT(*) FROM users WHERE id = ?`, user.ID.String()},
		{"chef_profiles", `SELECT COUNT(*) FROM chef_profiles WHERE id = ?`, chefID.String()},
		{"menu_items", `SELECT COUNT(*) FROM menu_items WHERE chef_id = ?`, chefID.String()},
		{"chef_documents", `SELECT COUNT(*) FROM chef_documents WHERE chef_id = ?`, chefID.String()},
	} {
		var n int64
		db.Raw(q.sql, q.arg).Scan(&n)
		require.Zero(t, n, "%s should be empty after purge", q.table)
	}

	// The sweeper retries on failure, so a second pass must converge rather
	// than error.
	require.NoError(t, PurgeUser(db, user.ID, models.RoleChef),
		"purge must be idempotent so the sweeper can retry")
}

func TestDriverRestore_ResetsVerificationAndQueuesReapproval(t *testing.T) {
	db := setupAccountDB(t)
	user := seedAccountUser(t, db, models.RoleDelivery)
	partnerID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO delivery_partners (id, user_id, is_verified, is_active, is_online)
		VALUES (?,?,?,?,?)`, partnerID.String(), user.ID.String(), true, true, true).Error)
	require.NoError(t, db.Exec(`INSERT INTO delivery_partner_documents (id, partner_id, type,
		file_name, file_path, bucket, status) VALUES (?,?,?,?,?,?,?)`,
		uuid.NewString(), partnerID.String(), "driving_licence", "dl.jpg", "", "", "approved").Error)

	_, err := RequestDeletion(db, &user, "")
	require.NoError(t, err)
	_, err = Restore(db, user.ID, "uid", "t", "p")
	require.NoError(t, err)

	var verified, online, active bool
	require.NoError(t, db.Raw(`SELECT is_verified, is_online, is_active FROM delivery_partners WHERE id = ?`,
		partnerID.String()).Row().Scan(&verified, &online, &active))
	require.False(t, verified, "restored driver must be re-verified")
	require.False(t, online)
	require.False(t, active, "restored driver must stay out of dispatch until approved")

	var docs int64
	db.Raw(`SELECT COUNT(*) FROM delivery_partner_documents WHERE partner_id = ?`,
		partnerID.String()).Scan(&docs)
	require.Zero(t, docs, "driver documents must be re-uploaded")

	var pending int64
	db.Raw(`SELECT COUNT(*) FROM approval_requests WHERE entity_id = ? AND status = 'pending'`,
		partnerID.String()).Scan(&pending)
	require.Equal(t, int64(1), pending)
}
