package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// setupDB builds a throwaway in-memory sqlite DB with the users table.
//
// We can't AutoMigrate(&models.User{}) directly because the model carries
// `default:gen_random_uuid()` on the ID column — that's a Postgres function
// SQLite doesn't understand. The handler always sets ID via uuid.New()
// before INSERT, so we don't need the DB-side default; we just declare the
// table by hand with a column list that matches the model.
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE users (
			id                    TEXT PRIMARY KEY,
			email                 TEXT NOT NULL,
			first_name            TEXT NOT NULL DEFAULT '',
			last_name             TEXT NOT NULL DEFAULT '',
			phone                 TEXT NOT NULL DEFAULT '',
			avatar                TEXT NOT NULL DEFAULT '',
			role                  TEXT NOT NULL DEFAULT 'customer',
			gip_uid               TEXT,
			gip_tenant_id         TEXT,
			gip_provider          TEXT,
			auth_pool             TEXT,
			is_active             INTEGER NOT NULL DEFAULT 1,
			phone_verified        INTEGER NOT NULL DEFAULT 0,
			fcm_token             TEXT NOT NULL DEFAULT '',
			marketing_consent     INTEGER NOT NULL DEFAULT 0,
			marketing_consent_at  DATETIME,
			last_login_at         DATETIME,
			created_at            DATETIME,
			updated_at            DATETIME,
			deleted_at            DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_users_gip_uid ON users(gip_uid)`).Error)
	require.NoError(t, db.Exec(`CREATE INDEX idx_users_deleted_at ON users(deleted_at)`).Error)
	return db
}

// postUpsert serializes body to JSON and POSTs it at /internal/users/upsert.
// Returns the recorder so each test can assert on status + body.
func postUpsert(t *testing.T, h *InternalUsersHandler, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/internal/users/upsert", h.Upsert)

	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/users/upsert", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpsert_NewUser_Inserts(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)
	w := postUpsert(t, h, UpsertUserRequest{
		GIPUid:      "gip-1",
		GIPTenantID: "HomeChef-Customer-rqg8a",
		GIPProvider: "google.com",
		AuthPool:    "customer",
		Email:       "Alice@Example.com",
		Name:        "Alice Anderson",
		Role:        "customer",
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp UpsertUserResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	_, err := uuid.Parse(resp.UserID)
	require.NoError(t, err, "response should be a UUID")

	var got models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-1").First(&got).Error)
	assert.Equal(t, "alice@example.com", got.Email, "email should be lowercased")
	assert.Equal(t, "Alice", got.FirstName)
	assert.Equal(t, "Anderson", got.LastName)
	assert.Equal(t, "HomeChef-Customer-rqg8a", got.GIPTenantID)
	assert.Equal(t, models.AuthPool("customer"), got.AuthPool)
	assert.Equal(t, models.UserRole("customer"), got.Role)
	assert.True(t, got.IsActive, "new users should be active by default")
	require.NotNil(t, got.LastLoginAt)
}

func TestUpsert_ExistingUser_UpdatesLastLogin(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	// Pre-seed via first upsert.
	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-1", GIPTenantID: "T", GIPProvider: "password",
		AuthPool: "business", Email: "b@b.com", Name: "Bob Bly", Role: "chef",
	})
	require.Equal(t, http.StatusOK, w1.Code, "body: %s", w1.Body.String())
	var first models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-1").First(&first).Error)
	require.NotNil(t, first.LastLoginAt)
	firstLoginAt := *first.LastLoginAt

	// Second call — should be idempotent and only refresh last_login_at.
	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-1", GIPTenantID: "T", GIPProvider: "password",
		AuthPool: "business", Email: "b@b.com", Name: "Bob Bly", Role: "chef",
	})
	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())

	var second models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-1").First(&second).Error)
	assert.Equal(t, first.ID, second.ID, "user_id stable across upserts")
	require.NotNil(t, second.LastLoginAt)
	assert.True(t, second.LastLoginAt.After(firstLoginAt) || second.LastLoginAt.Equal(firstLoginAt))

	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("gip_uid = ?", "gip-1").Count(&count).Error)
	assert.Equal(t, int64(1), count, "no duplicate row created")
}

func TestUpsert_BadEmail_400(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)
	w := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "x", GIPTenantID: "T", GIPProvider: "password",
		AuthPool: "customer", Email: "not-an-email", Role: "customer",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
}

func TestUpsert_MissingRequired_400(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)
	w := postUpsert(t, h, map[string]string{"email": "x@y.com"})
	require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
}

func TestUpsert_NewUser_PersistsNameAndAvatar(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)
	w := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-av", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "ava@example.com", Name: "Ava Vega",
		Avatar: "https://example.com/ava.png", EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var got models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-av").First(&got).Error)
	assert.Equal(t, "Ava", got.FirstName)
	assert.Equal(t, "Vega", got.LastName)
	assert.Equal(t, "https://example.com/ava.png", got.Avatar)
}

func TestUpsert_RepeatLogin_DoesNotOverwriteAvatarOrName(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	// First sign-in seeds name + avatar.
	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-keep", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "keep@example.com", Name: "Original Name",
		Avatar: "https://example.com/original.png", EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code, "body: %s", w1.Body.String())

	// User then edits their profile out-of-band.
	require.NoError(t, db.Model(&models.User{}).Where("gip_uid = ?", "gip-keep").
		Updates(map[string]any{
			"first_name": "Edited",
			"last_name":  "Profile",
			"avatar":     "https://example.com/edited.png",
		}).Error)

	// Repeat login with a different name/avatar from the provider — must NOT clobber.
	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-keep", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "keep@example.com", Name: "Provider Name",
		Avatar: "https://example.com/provider.png", EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())

	var got models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-keep").First(&got).Error)
	assert.Equal(t, "Edited", got.FirstName, "edited name preserved")
	assert.Equal(t, "Profile", got.LastName, "edited name preserved")
	assert.Equal(t, "https://example.com/edited.png", got.Avatar, "edited avatar preserved")
}

func TestUpsert_UnverifiedEmail_DoesNotRebind_CreatesNewRow(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	// Existing verified social account.
	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-social", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "shared@example.com", Name: "Social User",
		Avatar: "https://example.com/social.png", EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code, "body: %s", w1.Body.String())
	var social models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-social").First(&social).Error)

	// Unverified password signup with the SAME email + pool, NEW gip_uid.
	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-password", GIPTenantID: "T", GIPProvider: "password",
		AuthPool: "customer", Email: "shared@example.com", Name: "Imposter",
		EmailVerified: false, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())

	// The original verified row must NOT have been re-bound.
	var stillSocial models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-social").First(&stillSocial).Error)
	assert.Equal(t, social.ID, stillSocial.ID)
	assert.Equal(t, "gip-social", stillSocial.GIPUid, "verified row keeps its gip_uid")

	// A NEW separate row exists for the unverified signup.
	var imposter models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-password").First(&imposter).Error)
	assert.NotEqual(t, social.ID, imposter.ID, "unverified signup gets its own row")

	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("email = ?", "shared@example.com").Count(&count).Error)
	assert.Equal(t, int64(2), count, "two distinct rows for the shared email")
}

func TestUpsert_VerifiedEmail_DoesRebind(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)

	// Existing row (e.g., created via password originally).
	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-old", GIPTenantID: "T", GIPProvider: "password",
		AuthPool: "customer", Email: "rebind@example.com", Name: "Re Bind",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code, "body: %s", w1.Body.String())
	var old models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-old").First(&old).Error)

	// New verified GIP identity (new gip_uid) for the same email + pool.
	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-new", GIPTenantID: "T", GIPProvider: "google.com",
		AuthPool: "customer", Email: "rebind@example.com", Name: "Re Bind",
		EmailVerified: true, Role: "customer",
	})
	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())

	// Same row re-bound to the new gip_uid — no second row.
	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("email = ?", "rebind@example.com").Count(&count).Error)
	assert.Equal(t, int64(1), count, "verified re-bind reuses the existing row")

	var rebound models.User
	require.NoError(t, db.Where("gip_uid = ?", "gip-new").First(&rebound).Error)
	assert.Equal(t, old.ID, rebound.ID, "row id stable across verified re-bind")
}

func TestUpsert_SameEmailDifferentPool_AllowsTwoRows(t *testing.T) {
	db := setupDB(t)
	h := NewInternalUsersHandler(db)
	w1 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-c", GIPTenantID: "T1", GIPProvider: "google.com",
		AuthPool: "customer", Email: "person@example.com", Role: "customer",
	})
	require.Equal(t, http.StatusOK, w1.Code, "body: %s", w1.Body.String())

	w2 := postUpsert(t, h, UpsertUserRequest{
		GIPUid: "gip-b", GIPTenantID: "T2", GIPProvider: "google.com",
		AuthPool: "business", Email: "person@example.com", Role: "chef",
	})
	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())

	// NOTE: SQLite has limited support for partial-index unique constraints,
	// so it won't enforce the per-pool email uniqueness the way Postgres does
	// in production. This test only verifies the handler doesn't reject the
	// second insert — DB-level enforcement is covered by integration tests.
	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("email = ?", "person@example.com").Count(&count).Error)
	assert.Equal(t, int64(2), count)
}
