package handlers

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
	"github.com/homechef/api/services"
)

// Well-formed sample tokens: real FCM tokens are long URL-safe strings, so the
// handler enforces a 100–4096 charset-bounded shape. These sit inside that range.
var (
	validFCMToken  = strings.Repeat("a", 140)
	validFCMToken2 = strings.Repeat("b", 150)
)

// device_token_test.go — coverage for the push-token registration endpoint
// (handlers/device_token.go) that the mobile apps call after login. Without a
// persisted token, services/push.go can never deliver a notification — so this
// pins the contract: a token is stored, an explicit empty string clears it (used
// on logout / permission revoke), and an omitted field is a 400. This backstops
// the #239 follower-push path, which depends on tokens actually being saved.

func callDeviceToken(t *testing.T, userID uuid.UUID, body any) *httptest.ResponseRecorder {
	return callDeviceTokenFrom(t, userID, "", body)
}

// callDeviceTokenFrom registers a token as a named device. An empty deviceID
// omits the header, which is what a client older than #1164 sends.
func callDeviceTokenFrom(t *testing.T, userID uuid.UUID, deviceID string, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewDeviceTokenHandler()
	r.PUT("/profile/device-token", func(c *gin.Context) {
		c.Set("userID", userID)
		h.UpdateDeviceToken(c)
	})
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/profile/device-token", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if deviceID != "" {
		req.Header.Set("X-Device-Id", deviceID)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// createUserDevicesTable mirrors the user_devices DDL in tesserix-k8s.
func createUserDevicesTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE user_devices (
		id TEXT PRIMARY KEY, user_id TEXT NOT NULL, device_id TEXT NOT NULL,
		app TEXT NOT NULL DEFAULT '', platform TEXT, label TEXT, fcm_token TEXT,
		app_version TEXT, last_ip TEXT, last_country TEXT, last_city TEXT,
		first_seen_at DATETIME, last_seen_at DATETIME, revoked_at DATETIME,
		UNIQUE (user_id, device_id))`).Error)
}

// Two devices on one account must both stay deliverable — the bug in #1164 was
// that registering on the second silently cost the first every notification.
func TestUpdateDeviceToken_KeepsBothDevicesDeliverable(t *testing.T) {
	db := setupDB(t)
	createUserDevicesTable(t, db)
	database.DB = db
	uid := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		uid.String(), "two@example.com", time.Now(), time.Now(),
	).Error)

	require.Equal(t, http.StatusOK,
		callDeviceTokenFrom(t, uid, "phone", map[string]string{"token": validFCMToken}).Code)
	require.Equal(t, http.StatusOK,
		callDeviceTokenFrom(t, uid, "tablet", map[string]string{"token": validFCMToken2}).Code)

	tokens, err := services.ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{validFCMToken, validFCMToken2}, tokens)
}

// Signing out on one device must leave the other receiving push.
func TestUpdateDeviceToken_LogoutOnOneDeviceSparesTheOther(t *testing.T) {
	db := setupDB(t)
	createUserDevicesTable(t, db)
	database.DB = db
	uid := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		uid.String(), "out@example.com", time.Now(), time.Now(),
	).Error)
	require.Equal(t, http.StatusOK,
		callDeviceTokenFrom(t, uid, "phone", map[string]string{"token": validFCMToken}).Code)
	require.Equal(t, http.StatusOK,
		callDeviceTokenFrom(t, uid, "tablet", map[string]string{"token": validFCMToken2}).Code)

	require.Equal(t, http.StatusOK,
		callDeviceTokenFrom(t, uid, "tablet", map[string]string{"token": ""}).Code)

	tokens, err := services.ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	assert.Equal(t, []string{validFCMToken}, tokens)
}

// A client that predates the header still registers successfully, landing on
// the same row the schema backfill created for it.
func TestUpdateDeviceToken_ClientWithoutDeviceHeaderStillWorks(t *testing.T) {
	db := setupDB(t)
	createUserDevicesTable(t, db)
	database.DB = db
	uid := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		uid.String(), "old@example.com", time.Now(), time.Now(),
	).Error)

	require.Equal(t, http.StatusOK, callDeviceToken(t, uid, map[string]string{"token": validFCMToken}).Code)

	tokens, err := services.ActiveDeviceTokens(db, uid)
	require.NoError(t, err)
	assert.Equal(t, []string{validFCMToken}, tokens)

	var deviceID string
	require.NoError(t, db.Raw(`SELECT device_id FROM user_devices WHERE user_id = ?`, uid.String()).Scan(&deviceID).Error)
	assert.Equal(t, "legacy-"+uid.String(), deviceID)
}

func TestUpdateDeviceToken(t *testing.T) {
	db := setupDB(t)
	createUserDevicesTable(t, db)
	database.DB = db
	uid := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		uid.String(), "c@example.com", time.Now(), time.Now(),
	).Error)

	// Reads through the device registry rather than users.fcm_token: #1164
	// moved the token there so one account can hold several deliverable devices.
	read := func() string {
		tokens, err := services.ActiveDeviceTokens(db, uid)
		require.NoError(t, err)
		if len(tokens) == 0 {
			return ""
		}
		return tokens[0]
	}

	t.Run("persists a token", func(t *testing.T) {
		w := callDeviceToken(t, uid, map[string]string{"token": validFCMToken})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, validFCMToken, read())
	})

	t.Run("empty token clears it", func(t *testing.T) {
		w := callDeviceToken(t, uid, map[string]string{"token": ""})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "", read())
	})

	t.Run("omitted token is a 400", func(t *testing.T) {
		w := callDeviceToken(t, uid, map[string]string{})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("malformed token is a 400", func(t *testing.T) {
		// Too short + contains a disallowed character.
		w := callDeviceToken(t, uid, map[string]string{"token": "fcm-abc-123!"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("idempotent re-register", func(t *testing.T) {
		require.Equal(t, http.StatusOK, callDeviceToken(t, uid, map[string]string{"token": validFCMToken2}).Code)
		w := callDeviceToken(t, uid, map[string]string{"token": validFCMToken2})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, validFCMToken2, read())
	})
}
