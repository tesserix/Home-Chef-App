package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupCustomerDPDPDB provisions the users + audit_logs tables (shared helpers)
// plus the customer-owned tables the export/delete paths touch. Missing tables
// are tolerated by the handler (empty arrays), so we only declare the ones we
// assert on.
func setupCustomerDPDPDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupDPDPDB(t) // users + chef_profiles + menu_items + audit_logs, points database.DB
	require.NoError(t, db.Exec(`
		CREATE TABLE addresses (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			label       TEXT NOT NULL DEFAULT '',
			line1       TEXT NOT NULL DEFAULT '',
			city        TEXT NOT NULL DEFAULT '',
			state       TEXT NOT NULL DEFAULT '',
			postal_code TEXT NOT NULL DEFAULT '',
			country     TEXT NOT NULL DEFAULT 'IN',
			created_at  DATETIME,
			updated_at  DATETIME,
			deleted_at  DATETIME
		)
	`).Error)
	return db
}

func seedAddress(t *testing.T, db *gorm.DB, userID uuid.UUID, line1 string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO addresses (id, user_id, label, line1, city, state, postal_code, country, created_at, updated_at)
		 VALUES (?, ?, 'Home', ?, 'Bengaluru', 'KA', '560001', 'IN', ?, ?)`,
		uuid.New().String(), userID.String(), line1, time.Now(), time.Now(),
	).Error)
}

// callSelf routes a method/path to a handler with userID injected on the context.
func callSelf(t *testing.T, userID uuid.UUID, method, path string, register func(*gin.Engine), body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	register(r)

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

// ── Customer ─────────────────────────────────────────────────────────────────

func TestCustomerExport_ReturnsAttachmentWithUserAndAddresses(t *testing.T) {
	db := setupCustomerDPDPDB(t)
	uid := seedUser(t, db, "cust@example.com", "customer")
	seedAddress(t, db, uid, "42 Residency Road")

	w := callSelf(t, uid, http.MethodGet, "/customer/me/export", func(r *gin.Engine) {
		r.GET("/customer/me/export", NewCustomerDPDPHandler().ExportMyData)
	}, nil)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Header().Get("Content-Disposition"), "homechef-data-export-")
	body := w.Body.String()
	assert.Contains(t, body, "cust@example.com")
	assert.Contains(t, body, "42 Residency Road")
	assert.Contains(t, body, "DPDP Act 2023")
}

func TestCustomerExport_OmitsPasswordAndInternalFields(t *testing.T) {
	db := setupCustomerDPDPDB(t)
	uid := seedUser(t, db, "cust2@example.com", "customer")

	w := callSelf(t, uid, http.MethodGet, "/customer/me/export", func(r *gin.Engine) {
		r.GET("/customer/me/export", NewCustomerDPDPHandler().ExportMyData)
	}, nil)

	require.Equal(t, http.StatusOK, w.Code)
	var dump map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dump))
	user := dump["user"].(map[string]any)
	_, hasPass := user["password"]
	_, hasGip := user["gipUid"]
	assert.False(t, hasPass, "export must not include password fields")
	assert.False(t, hasGip, "export must not include internal GIP identifiers")
}

func TestCustomerDelete_WrongConfirmEmail_400(t *testing.T) {
	db := setupCustomerDPDPDB(t)
	uid := seedUser(t, db, "cust3@example.com", "customer")

	w := callSelf(t, uid, http.MethodPost, "/customer/me/delete", func(r *gin.Engine) {
		r.POST("/customer/me/delete", NewCustomerDPDPHandler().DeleteMyAccount)
	}, map[string]string{"confirmEmail": "wrong@example.com"})

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "must match")
}

// The confirm-email match is case-insensitive + trimmed, matching the mobile
// gate — a different-cased email still deletes.
func TestCustomerDelete_ConfirmEmailCaseInsensitive(t *testing.T) {
	db := setupCustomerDPDPDB(t)
	uid := seedUser(t, db, "MixedCase@Example.com", "customer")

	w := callSelf(t, uid, http.MethodPost, "/customer/me/delete", func(r *gin.Engine) {
		r.POST("/customer/me/delete", NewCustomerDPDPHandler().DeleteMyAccount)
	}, map[string]string{"confirmEmail": "  mixedcase@example.com  "})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "deleted")
}

func TestCustomerDelete_SoftDeletesWithRetention(t *testing.T) {
	db := setupCustomerDPDPDB(t)
	uid := seedUser(t, db, "cust4@example.com", "customer")

	w := callSelf(t, uid, http.MethodPost, "/customer/me/delete", func(r *gin.Engine) {
		r.POST("/customer/me/delete", NewCustomerDPDPHandler().DeleteMyAccount)
	}, map[string]string{"confirmEmail": "cust4@example.com"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "deleted")

	// The row is soft-deleted (invisible to the default scope, present unscoped).
	var visible int64
	db.Table("users").Where("id = ? AND deleted_at IS NULL", uid.String()).Count(&visible)
	assert.Equal(t, int64(0), visible)
	var total int64
	db.Table("users").Where("id = ?", uid.String()).Count(&total)
	assert.Equal(t, int64(1), total)

	// An audit row was written for the erasure.
	var audits int64
	db.Table("audit_logs").Where("action = ?", "customer.account.delete").Count(&audits)
	assert.Equal(t, int64(1), audits)
}

func TestCustomerDelete_Idempotent_AlreadyDeleted(t *testing.T) {
	db := setupCustomerDPDPDB(t)
	uid := seedUser(t, db, "cust5@example.com", "customer")
	reg := func(r *gin.Engine) {
		r.POST("/customer/me/delete", NewCustomerDPDPHandler().DeleteMyAccount)
	}
	body := map[string]string{"confirmEmail": "cust5@example.com"}

	require.Equal(t, http.StatusOK, callSelf(t, uid, http.MethodPost, "/customer/me/delete", reg, body).Code)
	w2 := callSelf(t, uid, http.MethodPost, "/customer/me/delete", reg, body)
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "already_deleted")
}

// ── Driver ───────────────────────────────────────────────────────────────────

func setupDriverDPDPDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupDPDPDB(t)
	require.NoError(t, db.Exec(`
		CREATE TABLE delivery_partners (
			id          TEXT PRIMARY KEY,
			user_id     TEXT NOT NULL,
			is_online   INTEGER NOT NULL DEFAULT 1,
			is_active   INTEGER NOT NULL DEFAULT 1,
			created_at  DATETIME,
			updated_at  DATETIME,
			deleted_at  DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE deliveries (
			id                  TEXT PRIMARY KEY,
			delivery_partner_id TEXT,
			created_at          DATETIME,
			updated_at          DATETIME,
			deleted_at          DATETIME
		)
	`).Error)
	return db
}

func seedPartner(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO delivery_partners (id, user_id, is_online, is_active, created_at, updated_at)
		 VALUES (?, ?, 1, 1, ?, ?)`,
		id.String(), userID.String(), time.Now(), time.Now(),
	).Error)
	return id
}

func TestDriverExport_IncludesPartnerAndDeliveries(t *testing.T) {
	db := setupDriverDPDPDB(t)
	uid := seedUser(t, db, "driver@example.com", "delivery")
	pid := seedPartner(t, db, uid)
	require.NoError(t, db.Exec(
		`INSERT INTO deliveries (id, delivery_partner_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		uuid.New().String(), pid.String(), time.Now(), time.Now()).Error)

	w := callSelf(t, uid, http.MethodGet, "/driver/me/export", func(r *gin.Engine) {
		r.GET("/driver/me/export", NewDriverDPDPHandler().ExportMyData)
	}, nil)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var dump map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dump))
	assert.Contains(t, dump, "deliveryPartner")
	deliveries, ok := dump["deliveries"].([]any)
	require.True(t, ok)
	assert.Len(t, deliveries, 1)
}

func TestDriverDelete_SoftDeletesAndTakesOffline(t *testing.T) {
	db := setupDriverDPDPDB(t)
	uid := seedUser(t, db, "driver2@example.com", "delivery")
	pid := seedPartner(t, db, uid)

	w := callSelf(t, uid, http.MethodPost, "/driver/me/delete", func(r *gin.Engine) {
		r.POST("/driver/me/delete", NewDriverDPDPHandler().DeleteMyAccount)
	}, map[string]string{"confirmEmail": "driver2@example.com"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// The partner profile is removed so the assigner can never pick this driver.
	var partners int64
	db.Table("delivery_partners").Where("id = ?", pid.String()).Count(&partners)
	assert.Equal(t, int64(0), partners)

	// The user row is soft-deleted (retained), not hard-deleted.
	var userTotal int64
	db.Table("users").Where("id = ?", uid.String()).Count(&userTotal)
	assert.Equal(t, int64(1), userTotal)

	var audits int64
	db.Table("audit_logs").Where("action = ?", "driver.account.delete").Count(&audits)
	assert.Equal(t, int64(1), audits)
}
