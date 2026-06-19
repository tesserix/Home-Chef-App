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

// device_token_test.go — coverage for the push-token registration endpoint
// (handlers/device_token.go) that the mobile apps call after login. Without a
// persisted token, services/push.go can never deliver a notification — so this
// pins the contract: a token is stored, an explicit empty string clears it (used
// on logout / permission revoke), and an omitted field is a 400. This backstops
// the #239 follower-push path, which depends on tokens actually being saved.

func callDeviceToken(t *testing.T, userID uuid.UUID, body any) *httptest.ResponseRecorder {
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

	t.Run("persists a token", func(t *testing.T) {
		w := callDeviceToken(t, uid, map[string]string{"token": "fcm-abc-123"})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "fcm-abc-123", read())
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

	t.Run("idempotent re-register", func(t *testing.T) {
		require.Equal(t, http.StatusOK, callDeviceToken(t, uid, map[string]string{"token": "tok-1"}).Code)
		w := callDeviceToken(t, uid, map[string]string{"token": "tok-1"})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "tok-1", read())
	})
}
