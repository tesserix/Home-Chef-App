package handlers

// chef_dpdp_test.go — automated verification for the DPDP Act 2023
// data-subject endpoints (Issue #12):
//
//	GET  /chef/me/export  → ChefDPDPHandler.ExportMyData
//	POST /chef/me/delete  → ChefDPDPHandler.DeleteMyAccount
//
// The handler reads the global database.DB and writes audit rows via
// services.LogAudit (also on the global DB). We point database.DB at a
// throwaway in-memory SQLite DB for the duration of each test and restore it
// afterwards (see setupDPDPDB). Tests must NOT run in parallel because they
// share that global.
//
// Schema note: like internal_users_test.go, we hand-roll the tables. The
// models carry `default:gen_random_uuid()` (a Postgres function) which makes
// gorm.AutoMigrate emit invalid SQLite DDL, so we declare only the columns the
// handler touches. Columns absent from the table are simply left zero on the
// struct after SELECT *, which is exactly what we want.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// setupDPDPDB builds an in-memory SQLite DB with the tables the DPDP handler
// touches, points the global database.DB at it, and restores the previous DB
// on cleanup. Returns the *gorm.DB so tests can seed/assert directly.
func setupDPDPDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupDB(t) // users table (shared helper from internal_users_test.go)

	require.NoError(t, db.Exec(`
		CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', 
			id                TEXT PRIMARY KEY,
			user_id           TEXT NOT NULL,
			business_name     TEXT NOT NULL DEFAULT '',
			description       TEXT NOT NULL DEFAULT '',
			accepting_orders  INTEGER NOT NULL DEFAULT 1,
			created_at        DATETIME,
			updated_at        DATETIME
		)
	`).Error)

	require.NoError(t, db.Exec(`
		CREATE TABLE menu_items (
			id            TEXT PRIMARY KEY,
			chef_id       TEXT NOT NULL,
			name          TEXT NOT NULL DEFAULT '',
			is_available  INTEGER NOT NULL DEFAULT 1,
			created_at    DATETIME,
			updated_at    DATETIME,
			deleted_at    DATETIME
		)
	`).Error)

	// audit_logs: id has no NOT NULL / autoincrement — GORM omits it on insert
	// (it carries a `default:` tag) so SQLite stores NULL, which is fine; we
	// assert on the row's action/entity, not its id.
	require.NoError(t, db.Exec(`
		CREATE TABLE audit_logs (
			id              TEXT,
			user_id         TEXT,
			action          TEXT NOT NULL,
			entity_type     TEXT NOT NULL DEFAULT '',
			entity_id       TEXT NOT NULL DEFAULT '',
			old_value       TEXT NOT NULL DEFAULT '',
			new_value       TEXT NOT NULL DEFAULT '',
			ip_address      TEXT NOT NULL DEFAULT '',
			user_agent      TEXT NOT NULL DEFAULT '',
			correlation_id  TEXT NOT NULL DEFAULT '',
			created_at      DATETIME
		)
	`).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedUser inserts a user row with an explicit UUID (the model's DB-side
// default is Postgres-only) and returns its id.
func seedUser(t *testing.T, db *gorm.DB, email string, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, first_name, last_name, phone, role, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)`,
		id.String(), email, "Test", "Chef", "+910000000000", role, time.Now(), time.Now(),
	).Error)
	return id
}

func seedChef(t *testing.T, db *gorm.DB, userID uuid.UUID, business string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, description, accepting_orders, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?)`,
		id.String(), userID.String(), business, "Best home food in town", time.Now(), time.Now(),
	).Error)
	return id
}

func seedMenuItem(t *testing.T, db *gorm.DB, chefID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO menu_items (id, chef_id, name, is_available, created_at, updated_at)
		 VALUES (?, ?, ?, 1, ?, ?)`,
		id.String(), chefID.String(), name, time.Now(), time.Now(),
	).Error)
	return id
}

// callDPDP wires a one-off router that injects userID onto the context (the
// auth middleware's job in production) and routes the given method/path to the
// handler. body is JSON-encoded when non-nil.
func callDPDP(t *testing.T, userID uuid.UUID, method, path string, register func(*gin.Engine, *ChefDPDPHandler), body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})
	register(r, NewChefDPDPHandler())

	var reader *bytes.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		reader = bytes.NewReader(buf)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doExport(t *testing.T, userID uuid.UUID) *httptest.ResponseRecorder {
	return callDPDP(t, userID, http.MethodGet, "/chef/me/export",
		func(r *gin.Engine, h *ChefDPDPHandler) { r.GET("/chef/me/export", h.ExportMyData) }, nil)
}

func doDelete(t *testing.T, userID uuid.UUID, body any) *httptest.ResponseRecorder {
	return callDPDP(t, userID, http.MethodPost, "/chef/me/delete",
		func(r *gin.Engine, h *ChefDPDPHandler) { r.POST("/chef/me/delete", h.DeleteMyAccount) }, body)
}

// ── Export ──────────────────────────────────────────────────────────────────

func TestExportMyData_ReturnsAttachmentWithUserData(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")

	w := doExport(t, uid)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	// Downloadable attachment with a dated filename.
	cd := w.Header().Get("Content-Disposition")
	assert.Contains(t, cd, "attachment;", "export should download as a file")
	assert.Contains(t, cd, "homechef-data-export-", "filename should be branded")
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var dump map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dump))

	// DPDP §11 notice + provenance block.
	assert.Contains(t, dump, "notice")
	assert.Contains(t, dump, "exportedAt")
	reqBy, ok := dump["requestedBy"].(map[string]any)
	require.True(t, ok, "requestedBy block present")
	assert.Equal(t, "chef@example.com", reqBy["email"])

	user, ok := dump["user"].(map[string]any)
	require.True(t, ok, "user block present")
	assert.Equal(t, "chef@example.com", user["email"])
}

func TestExportMyData_OmitsSensitiveUserFields(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")

	w := doExport(t, uid)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	// The raw export must never carry auth-internal fields. sanitizeUserForExport
	// builds an allow-list, so these keys must be absent regardless of model drift.
	var dump map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dump))
	user := dump["user"].(map[string]any)
	for _, banned := range []string{"password", "passwordHash", "gipUid", "gip_uid", "isActive"} {
		_, present := user[banned]
		assert.Falsef(t, present, "export user block must not expose %q", banned)
	}
}

func TestExportMyData_IncludesChefProfileAndMenu(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")
	cid := seedChef(t, db, uid, "Auntie's Kitchen")
	seedMenuItem(t, db, cid, "Butter Chicken")

	w := doExport(t, uid)
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var dump map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dump))

	chef, ok := dump["chefProfile"].(map[string]any)
	require.True(t, ok, "chefProfile block present when a profile exists")
	assert.Equal(t, "Auntie's Kitchen", chef["businessName"])

	items, ok := dump["menuItems"].([]any)
	require.True(t, ok, "menuItems present")
	require.Len(t, items, 1)
	assert.Equal(t, "Butter Chicken", items[0].(map[string]any)["name"])
}

func TestExportMyData_UserNotFound_404(t *testing.T) {
	setupDPDPDB(t)
	w := doExport(t, uuid.New()) // no such user
	require.Equal(t, http.StatusNotFound, w.Code, "body: %s", w.Body.String())
}

// ── Delete: email confirmation gate ──────────────────────────────────────────

func TestDeleteMyAccount_MissingConfirmEmail_400(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")

	w := doDelete(t, uid, map[string]any{}) // confirmEmail omitted → binding fails
	require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())

	// Row must still be live.
	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", uid).Count(&count).Error)
	assert.Equal(t, int64(1), count, "account must NOT be deleted without confirmation")
}

func TestDeleteMyAccount_WrongConfirmEmail_400(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")

	w := doDelete(t, uid, map[string]any{"confirmEmail": "someoneelse@example.com"})
	require.Equal(t, http.StatusBadRequest, w.Code, "body: %s", w.Body.String())
	assert.Contains(t, w.Body.String(), "confirmEmail", "error should name the field")

	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", uid).Count(&count).Error)
	assert.Equal(t, int64(1), count, "mismatched email must NOT delete the account")
}

// ── Delete: soft-delete + retention ───────────────────────────────────────────

func TestDeleteMyAccount_SoftDeletesUserWith30DayRetention(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")

	w := doDelete(t, uid, map[string]any{"confirmEmail": "chef@example.com"})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var resp struct {
		Status      string    `json:"status"`
		DeletedAt   time.Time `json:"deletedAt"`
		RetainUntil time.Time `json:"retainUntil"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "deleted", resp.Status)

	// Retention window ≈ 30 days after deletion.
	gap := resp.RetainUntil.Sub(resp.DeletedAt)
	assert.InDelta(t, (30 * 24 * time.Hour).Seconds(), gap.Seconds(), 60,
		"retainUntil should be ~30 days after deletedAt")

	// Default-scoped lookup hides the row (soft delete in effect)...
	var live int64
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", uid).Count(&live).Error)
	assert.Equal(t, int64(0), live, "soft-deleted user is hidden from normal queries")

	// ...but the row still exists with deleted_at set (retention, not erasure).
	var deletedAt *time.Time
	require.NoError(t, db.Raw(`SELECT deleted_at FROM users WHERE id = ?`, uid.String()).Scan(&deletedAt).Error)
	require.NotNil(t, deletedAt, "row must be retained with a deleted_at timestamp")
}

func TestDeleteMyAccount_CascadesChefProfileAndMenu(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")
	cid := seedChef(t, db, uid, "Auntie's Kitchen")
	mid := seedMenuItem(t, db, cid, "Butter Chicken")

	w := doDelete(t, uid, map[string]any{"confirmEmail": "chef@example.com"})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	// Chef profile carries no gorm.DeletedAt → the handler HARD-deletes it so
	// the kitchen disappears from the marketplace immediately.
	var chefRows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM chef_profiles WHERE id = ?`, cid.String()).Scan(&chefRows).Error)
	assert.Equal(t, int64(0), chefRows, "chef profile is hard-deleted on erasure")

	// Menu items are taken offline (is_available=false) so no phantom kitchen.
	var available int
	require.NoError(t, db.Raw(`SELECT is_available FROM menu_items WHERE id = ?`, mid.String()).Scan(&available).Error)
	assert.Equal(t, 0, available, "menu items must go offline on account deletion")
}

// ── Delete: PII-safe audit trail ──────────────────────────────────────────────

func TestDeleteMyAccount_WritesPIISafeAuditLog(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")

	w := doDelete(t, uid, map[string]any{"confirmEmail": "chef@example.com"})
	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())

	var row struct {
		Action     string
		EntityType string
		EntityID   string
		OldValue   string
		NewValue   string
	}
	require.NoError(t, db.Raw(
		`SELECT action, entity_type, entity_id, old_value, new_value FROM audit_logs WHERE action = ?`,
		"chef.account.delete",
	).Scan(&row).Error)

	assert.Equal(t, "chef.account.delete", row.Action)
	assert.Equal(t, "user", row.EntityType)
	assert.Equal(t, uid.String(), row.EntityID, "audit subject is the user id")

	// PII-safe: the audit payload records timestamps only — never the email,
	// phone, or name of the data subject.
	blob := row.OldValue + row.NewValue
	assert.NotContains(t, strings.ToLower(blob), "chef@example.com", "audit must not store the email")
	assert.NotContains(t, blob, "+910000000000", "audit must not store the phone")
	assert.Contains(t, row.NewValue, "retainUntil", "audit records the retention deadline")
}

// TestDeleteMyAccount_RetryIsIdempotent verifies that a retried delete is safe
// (DPDP retries from a flaky mobile network must not error). The handler looks
// up the user with .Unscoped() so an already-soft-deleted account is still
// found, returning 200 "already_deleted" with the original retention deadline
// rather than a 404 (#106).
func TestDeleteMyAccount_RetryIsIdempotent(t *testing.T) {
	db := setupDPDPDB(t)
	uid := seedUser(t, db, "chef@example.com", "chef")

	first := doDelete(t, uid, map[string]any{"confirmEmail": "chef@example.com"})
	require.Equal(t, http.StatusOK, first.Code, "body: %s", first.Body.String())
	var firstResp struct {
		Status      string    `json:"status"`
		RetainUntil time.Time `json:"retainUntil"`
	}
	require.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstResp))
	assert.Equal(t, "deleted", firstResp.Status)

	second := doDelete(t, uid, map[string]any{"confirmEmail": "chef@example.com"})
	require.Equal(t, http.StatusOK, second.Code, "retry must be idempotent, not 404")
	var secondResp struct {
		Status      string    `json:"status"`
		RetainUntil time.Time `json:"retainUntil"`
	}
	require.NoError(t, json.Unmarshal(second.Body.Bytes(), &secondResp))
	assert.Equal(t, "already_deleted", secondResp.Status, "repeat delete reports already_deleted")
	assert.WithinDuration(t, firstResp.RetainUntil, secondResp.RetainUntil, time.Second,
		"retention deadline stays anchored to the original deletion")
}

// ── sanitizeUserForExport (pure unit, no DB) ─────────────────────────────────

func TestSanitizeUserForExport_AllowListsSafeFieldsOnly(t *testing.T) {
	u := models.User{
		ID:        uuid.New(),
		Email:     "chef@example.com",
		FirstName: "Asha",
		LastName:  "Rao",
		Phone:     "+919999999999",
		Role:      models.UserRole("chef"),
	}

	out := sanitizeUserForExport(u)

	// Portability fields the data subject is entitled to.
	assert.Equal(t, "chef@example.com", out["email"])
	assert.Equal(t, "Asha", out["firstName"])
	assert.Equal(t, "Rao", out["lastName"])
	assert.Equal(t, "+919999999999", out["phone"])

	// Internal/auth fields must never appear, even if added to the model later.
	for _, banned := range []string{"password", "passwordHash", "gipUid", "isActive", "fcmToken"} {
		_, present := out[banned]
		assert.Falsef(t, present, "sanitized export must not include %q", banned)
	}
}
