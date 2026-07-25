package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func gateDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	// Pin the pool to one connection. Every ":memory:" connection is its own
	// empty database, so a second pooled conn silently loses the seeded rows —
	// which shows up as the gate passing requests it should refuse, at random.
	raw, err := db.DB()
	require.NoError(t, err)
	raw.SetMaxOpenConns(1)
	for _, s := range []string{
		`CREATE TABLE user_mfa_settings (user_id TEXT PRIMARY KEY, enabled INTEGER DEFAULT 0,
			email_enrolled INTEGER DEFAULT 0, phone_enrolled INTEGER DEFAULT 0,
			phone_e164_enc TEXT DEFAULT '', phone_e164_bidx TEXT DEFAULT '',
			enrolled_at DATETIME, disabled_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE trusted_devices (id TEXT PRIMARY KEY, user_id TEXT, app TEXT, token_hash TEXT,
			label TEXT, platform TEXT, created_at DATETIME, last_seen_at DATETIME, revoked_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func gateRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prev := services.SetRedisClientForTest(client)
	t.Cleanup(func() {
		_ = client.Close()
		services.SetRedisClientForTest(prev)
	})
	return mr
}

func enrol(t *testing.T, db *gorm.DB, userID uuid.UUID) {
	t.Helper()
	now := time.Now()
	require.NoError(t, db.Create(&models.UserMFASettings{
		UserID: userID, Enabled: true, EmailEnrolled: true,
		EnrolledAt: &now, UpdatedAt: now,
	}).Error)
}

// run drives one request through the gate and reports the outcome.
func run(t *testing.T, db *gorm.DB, enabled bool, userID *uuid.UUID, path, app, deviceToken string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if userID != nil {
			// Mirror what BFFAuth actually sets. GetUserID reads "userID"
			// (uuid.UUID), NOT CtxUserID ("user_id") — setting only the latter
			// makes the gate no-op and every assertion here vacuous.
			c.Set("userID", *userID)
			c.Set(CtxUserID, userID.String())
			c.Set(CtxUserEmail, "chef@fe3dr.com")
			c.Set("userEmail", "chef@fe3dr.com")
		}
		c.Next()
	})
	r.Use(MFAGate(db, enabled))
	r.Any("/*any", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if app != "" {
		req.Header.Set(HdrClientApp, app)
	}
	if deviceToken != "" {
		req.Header.Set(HdrDeviceToken, deviceToken)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestMFAGate_FlagOffPassesEverything(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	w := run(t, db, false, &uid, "/api/v1/orders", "vendor", "")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMFAGate_EnrolledUntrustedDeviceIsRefused(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	w := run(t, db, true, &uid, "/api/v1/orders", "vendor", "")
	require.Equal(t, http.StatusForbidden, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "mfa_required", body["error"])
	require.Contains(t, body["channels"], "email")
}

// The hint must identify the address to its owner without disclosing it.
func TestMFAGate_RefusalCarriesMaskedHintOnly(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	w := run(t, db, true, &uid, "/api/v1/orders", "vendor", "")
	require.NotContains(t, w.Body.String(), "chef@fe3dr.com")
	require.Contains(t, w.Body.String(), "@fe3dr.com")
}

func TestMFAGate_NotEnrolledPasses(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)

	w := run(t, db, true, &uid, "/api/v1/orders", "vendor", "")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMFAGate_TrustedDevicePasses(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	token, err := services.IssueTrustedDevice(db, uid, "vendor", "Pixel", "android")
	require.NoError(t, err)

	w := run(t, db, true, &uid, "/api/v1/orders", "vendor", token)
	require.Equal(t, http.StatusOK, w.Code)
}

// Trust must not cross apps even with a genuine token.
func TestMFAGate_TrustDoesNotCrossApps(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	token, err := services.IssueTrustedDevice(db, uid, "vendor", "Pixel", "android")
	require.NoError(t, err)

	w := run(t, db, true, &uid, "/api/v1/orders", "admin", token)
	require.Equal(t, http.StatusForbidden, w.Code)
}

// Too narrow an allowlist and a challenged user cannot reach the endpoints that
// would unblock them — a permanent lockout.
func TestMFAGate_ChallengeEndpointsStayReachable(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	for _, path := range []string{
		"/api/v1/auth/mfa/challenge",
		"/api/v1/auth/mfa/verify",
		"/api/v1/auth/mfa/status",
		"/health",
		"/readyz",
		"/metrics",
		"/version",
	} {
		w := run(t, db, true, &uid, path, "vendor", "")
		require.Equal(t, http.StatusOK, w.Code, "path %s must stay reachable mid-challenge", path)
	}
}

// Too broad an allowlist and the gate leaks. These must all be refused.
func TestMFAGate_ProtectedPathsAreNotExempt(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	for _, path := range []string{
		"/api/v1/orders",
		"/api/v1/account/profile",
		"/api/v1/admin/chefs",
		"/api/v1/auth/mfa", // no trailing slash — must NOT match the prefix
		"/api/v1/wallet",
	} {
		w := run(t, db, true, &uid, path, "vendor", "")
		require.Equal(t, http.StatusForbidden, w.Code, "path %s must be gated", path)
	}
}

func TestMFAGate_UnauthenticatedRequestsAreNotThisGatesBusiness(t *testing.T) {
	db := gateDB(t)
	gateRedis(t)

	w := run(t, db, true, nil, "/api/v1/chefs", "customer", "")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestMFAGate_ElevatedSessionPasses(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	token, err := services.ElevateSession(t.Context(), uid, "vendor")
	require.NoError(t, err)

	w := run(t, db, true, &uid, "/api/v1/orders", "vendor", token)
	require.Equal(t, http.StatusOK, w.Code)
}

// An unknown or absent app header must fall back to a real scope rather than
// accidentally matching every device row.
func TestMFAGate_UnknownAppHeaderFallsBackToCustomer(t *testing.T) {
	db, uid := gateDB(t), uuid.New()
	gateRedis(t)
	enrol(t, db, uid)

	token, err := services.IssueTrustedDevice(db, uid, "customer", "iPhone", "ios")
	require.NoError(t, err)

	// Garbage header → customer scope, so a customer token still works.
	w := run(t, db, true, &uid, "/api/v1/orders", "not-an-app", token)
	require.Equal(t, http.StatusOK, w.Code)

	// ...and a vendor token does not.
	vendorToken, err := services.IssueTrustedDevice(db, uid, "vendor", "Pixel", "android")
	require.NoError(t, err)
	w = run(t, db, true, &uid, "/api/v1/orders", "not-an-app", vendorToken)
	require.Equal(t, http.StatusForbidden, w.Code)
}
