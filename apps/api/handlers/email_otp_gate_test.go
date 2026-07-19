package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

func otpGateCtx(userID uuid.UUID) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("userID", userID)
	return c, w
}

func TestEnsureLoginEmailVerified(t *testing.T) {
	db := setupDB(t)
	origDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = origDB })

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prevRedis := services.SetRedisClientForTest(client)
	t.Cleanup(func() { _ = client.Close(); services.SetRedisClientForTest(prevRedis) })

	prevCfg := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prevCfg })

	userID := uuid.New()
	email := "cust@fe3dr.com"
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, role) VALUES (?, ?, 'customer')`,
		userID.String(), email).Error)

	// Flag off → no-op pass, nothing written.
	config.AppConfig = &config.Config{EmailOTPEnabled: false}
	c, w := otpGateCtx(userID)
	require.True(t, EnsureLoginEmailVerified(c, userID))
	require.Equal(t, http.StatusOK, w.Code)

	// Flag on, unverified → blocked with 428.
	config.AppConfig = &config.Config{EmailOTPEnabled: true}
	c, w = otpGateCtx(userID)
	require.False(t, EnsureLoginEmailVerified(c, userID))
	require.Equal(t, http.StatusPreconditionRequired, w.Code)

	// Flag on, verified marker present → pass. Key mirrors services.otpVerifiedKey.
	mr.Set(fmt.Sprintf("email_otp:ok:%s:%s", userID.String(), services.NormalizeEmail(email)), "1")
	c, w = otpGateCtx(userID)
	require.True(t, EnsureLoginEmailVerified(c, userID))
	require.Equal(t, http.StatusOK, w.Code)
}
