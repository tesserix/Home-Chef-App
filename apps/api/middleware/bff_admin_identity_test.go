package middleware

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// bff_admin_identity_test.go — the tesserix-home admin gateway signs with the
// OIDC `sub`, not a HomeChef UUID, so uuid.Parse failed and every admin action
// was attributed to uuid.Nil (#968).

func setupAdminIdentityDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	// The model carries a Postgres-only `default:gen_random_uuid()`, so declare
	// the table by hand (mirrors handlers/internal_users_test.go).
	require.NoError(t, db.Exec(`
		CREATE TABLE users (
			email_enc TEXT DEFAULT '', email_bidx TEXT DEFAULT '',
			first_name_enc TEXT DEFAULT '', last_name_enc TEXT DEFAULT '',
			phone_enc TEXT DEFAULT '', phone_bidx TEXT DEFAULT '',
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL,
			first_name TEXT NOT NULL DEFAULT '',
			last_name TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			avatar TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT 'customer',
			gip_uid TEXT, gip_tenant_id TEXT, gip_provider TEXT,
			apple_refresh_token_enc TEXT,
			auth_pool TEXT,
			is_active INTEGER NOT NULL DEFAULT 1,
			phone_verified INTEGER NOT NULL DEFAULT 0,
			fcm_token TEXT NOT NULL DEFAULT '',
			marketing_consent INTEGER NOT NULL DEFAULT 0,
			marketing_consent_at DATETIME,
			deactivated_at DATETIME, purge_after DATETIME,
			deletion_reason TEXT NOT NULL DEFAULT '',
			restored_at DATETIME, last_login_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE user_identities (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			subject TEXT NOT NULL,
			email TEXT DEFAULT '',
			created_at DATETIME, last_login_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_user_identities_provider_subject ON user_identities(provider, subject)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func TestResolveLocalUserID_MatchesExistingGIPUid(t *testing.T) {
	db := setupAdminIdentityDB(t)
	want := uuid.New()
	require.NoError(t, db.Create(&models.User{
		ID: want, Email: "ops@tesserix.com", GIPUid: "gip-sub-123",
		AuthPool: models.PoolInternal, Role: models.RoleAdmin, IsActive: true,
	}).Error)

	got, ok := resolveLocalUserID(&BFFIdentity{
		UserID: "gip-sub-123", Email: "ops@tesserix.com",
		Role: string(models.RoleAdmin), Pool: string(models.PoolInternal),
	})
	require.True(t, ok)
	require.Equal(t, want, got)
}

func TestResolveLocalUserID_MaterialisesAdminOnFirstUse(t *testing.T) {
	db := setupAdminIdentityDB(t)
	id := &BFFIdentity{
		UserID: "gip-sub-new", Email: "New.Admin@Tesserix.com",
		Role: string(models.RoleAdmin), Pool: string(models.PoolInternal),
	}

	got, ok := resolveLocalUserID(id)
	require.True(t, ok, "a signed internal admin must resolve to a real row")
	require.NotEqual(t, uuid.Nil, got)

	var u models.User
	require.NoError(t, db.First(&u, "id = ?", got).Error)
	require.Equal(t, "new.admin@tesserix.com", u.Email, "email is normalised")
	require.Equal(t, models.RoleAdmin, u.Role)
	require.Equal(t, models.PoolInternal, u.AuthPool)

	// Second call reuses the row rather than minting another.
	again, ok := resolveLocalUserID(id)
	require.True(t, ok)
	require.Equal(t, got, again)
	var n int64
	require.NoError(t, db.Model(&models.User{}).Count(&n).Error)
	require.EqualValues(t, 1, n)
}

func TestResolveLocalUserID_RefusesNonAdmin(t *testing.T) {
	setupAdminIdentityDB(t)
	// role/pool are inside the signed payload, so a caller cannot claim admin —
	// but anything that is not an internal admin must never mint a row.
	_, ok := resolveLocalUserID(&BFFIdentity{
		UserID: "gip-sub-cust", Email: "someone@example.com",
		Role: string(models.RoleCustomer), Pool: "customer",
	})
	require.False(t, ok)
}
