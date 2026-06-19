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

	"github.com/homechef/api/database"
)

// device_token_test.go — #236. The mobile apps register their raw FCM token via
// PUT /api/v1/profile/device-token. Without this, User.FCMToken is always empty
// and services/push.go can never deliver a push. These tests pin the contract:
// a valid raw token is persisted, junk is rejected, and re-registration is a
// no-op success (idempotent on app restart / token rotation).

func callDeviceToken(t *testing.T, userID uuid.UUID, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewCustomerHandler()
	r.PUT("/profile/device-token", func(c *gin.Context) {
		c.Set("userID", userID)
		h.UpdateDeviceToken(c)
	})
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/profile/device-token", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateDeviceToken(t *testing.T) {
	db := setupDB(t)
	database.DB = db
	uid := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		uid.String(), "c@example.com", time.Now(), time.Now(),
	).Error)

	read := func() string {
		var tok string
		require.NoError(t, db.Raw(`SELECT fcm_token FROM users WHERE id = ?`, uid.String()).Scan(&tok).Error)
		return tok
	}

	t.Run("persists a raw token", func(t *testing.T) {
		w := callDeviceToken(t, uid, map[string]string{"token": "fcm-abc-123"})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "fcm-abc-123", read())
	})

	t.Run("rejects an empty token", func(t *testing.T) {
		w := callDeviceToken(t, uid, map[string]string{"token": "  "})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("rejects an Expo push token", func(t *testing.T) {
		w := callDeviceToken(t, uid, map[string]string{"token": "ExponentPushToken[abc]"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("idempotent re-register", func(t *testing.T) {
		require.Equal(t, http.StatusOK, callDeviceToken(t, uid, map[string]string{"token": "tok-1"}).Code)
		w := callDeviceToken(t, uid, map[string]string{"token": "tok-1"})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "tok-1", read())
	})
}
