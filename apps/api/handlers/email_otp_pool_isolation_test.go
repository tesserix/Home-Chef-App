package handlers

// email_otp_pool_isolation_test.go — the onboarding email conflict check is
// scoped to the caller's auth pool. One human may hold a customer account and a
// business account on the same address (idx_users_email_per_pool); the vendor
// wizard used to reject that with "already registered with another account".

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func setupOTPPoolDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupDB(t)
	origDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = origDB })

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prevRedis := services.SetRedisClientForTest(client)
	t.Cleanup(func() { _ = client.Close(); services.SetRedisClientForTest(prevRedis) })

	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{EmailOTPEnabled: true}
	t.Cleanup(func() { config.AppConfig = prevCfg })

	return db
}

// seedPoolUser inserts a user with an explicit auth_pool. Pass "" for the
// legacy rows created before pools existed, which store NULL.
func seedPoolUser(t *testing.T, db *gorm.DB, email string, pool models.AuthPool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if pool == "" {
		require.NoError(t, db.Exec(
			`INSERT INTO users (id, email, role, auth_pool) VALUES (?, ?, 'customer', NULL)`,
			id.String(), email).Error)
		return id
	}
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, role, auth_pool) VALUES (?, ?, 'customer', ?)`,
		id.String(), email, string(pool)).Error)
	return id
}

func requestOTP(t *testing.T, userID uuid.UUID, email string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	body, err := json.Marshal(map[string]string{"email": email})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userID", userID)
	NewEmailOTPHandler().RequestOTP(c)
	return w
}

func TestRequestOTP_EmailConflictIsScopedToAuthPool(t *testing.T) {
	db := setupOTPPoolDB(t)

	t.Run("same email in another pool does not block", func(t *testing.T) {
		shared := "chef.and.customer@fe3dr.com"
		seedPoolUser(t, db, shared, models.PoolCustomer)
		vendor := seedPoolUser(t, db, "vendor-a@fe3dr.com", models.PoolBusiness)

		w := requestOTP(t, vendor, shared)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})

	t.Run("same email in the same pool still conflicts", func(t *testing.T) {
		taken := "taken@fe3dr.com"
		seedPoolUser(t, db, taken, models.PoolBusiness)
		vendor := seedPoolUser(t, db, "vendor-b@fe3dr.com", models.PoolBusiness)

		w := requestOTP(t, vendor, taken)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Contains(t, w.Body.String(), "already registered")
	})

	t.Run("legacy pool-less row does not block a pooled signup", func(t *testing.T) {
		legacy := "legacy@fe3dr.com"
		seedPoolUser(t, db, legacy, "")
		vendor := seedPoolUser(t, db, "vendor-c@fe3dr.com", models.PoolBusiness)

		w := requestOTP(t, vendor, legacy)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})

	t.Run("the caller's own email is not a conflict", func(t *testing.T) {
		own := "self@fe3dr.com"
		vendor := seedPoolUser(t, db, own, models.PoolBusiness)

		w := requestOTP(t, vendor, own)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	})
}
