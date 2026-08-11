package handlers

// chef_onboarding_phone_pool_test.go — the chef wizard's phone conflict check is
// scoped to the caller's auth pool, for the same reason the email one is: a
// customer who becomes a chef signs up again in the business pool with the same
// contact details. Sibling of email_otp_pool_isolation_test.go.
//
// DDL reuse: chefProfilesGuardDDL (chef_fulfillment_guard_test.go), setupDB
// (internal_users_test.go).

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

func setupOnboardingPhoneDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupDB(t)
	require.NoError(t, db.Exec(chefProfilesGuardDDL).Error)

	origDB := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = origDB })

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	prevRedis := services.SetRedisClientForTest(client)
	t.Cleanup(func() { _ = client.Close(); services.SetRedisClientForTest(prevRedis) })

	prevCfg := config.AppConfig
	// On, so a phone that clears the check stops at the unverified-email gate
	// (428) rather than running the whole profile-creation transaction.
	config.AppConfig = &config.Config{EmailOTPEnabled: true}
	t.Cleanup(func() { config.AppConfig = prevCfg })

	return db
}

func seedPhoneUser(t *testing.T, db *gorm.DB, phone string, pool models.AuthPool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, phone, role, auth_pool) VALUES (?, ?, ?, 'customer', ?)`,
		id.String(), id.String()+"@fe3dr.com", phone, string(pool)).Error)
	return id
}

func postChefOnboarding(t *testing.T, userID uuid.UUID, phone string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	body, err := json.Marshal(map[string]any{
		"fullName":     "Chef Tester",
		"phone":        phone,
		"email":        "chef.tester@fe3dr.com",
		"businessName": "Kitchen " + uuid.NewString()[:8],
		"description":  "Home cooked meals",
		"kitchenType":  "home_kitchen",
		"cuisines":     []string{"indian"},
		"kitchenAddress": map[string]any{
			"line1":      "1 Test Street",
			"city":       "Bengaluru",
			"state":      "Karnataka",
			"postalCode": "560001",
		},
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/chef/onboarding", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("userID", userID)
	(&UploadHandler{}).Onboarding(c)
	return w
}

func TestChefOnboarding_PhoneConflictIsScopedToAuthPool(t *testing.T) {
	db := setupOnboardingPhoneDB(t)

	t.Run("same phone in another pool does not block", func(t *testing.T) {
		phone := "9876543210"
		seedPhoneUser(t, db, phone, models.PoolCustomer)
		vendor := seedPhoneUser(t, db, "", models.PoolBusiness)

		w := postChefOnboarding(t, vendor, phone)
		require.Equal(t, http.StatusPreconditionRequired, w.Code, "body: %s", w.Body.String())
	})

	t.Run("same phone in the same pool still conflicts", func(t *testing.T) {
		phone := "9876500001"
		seedPhoneUser(t, db, phone, models.PoolBusiness)
		vendor := seedPhoneUser(t, db, "", models.PoolBusiness)

		w := postChefOnboarding(t, vendor, phone)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Contains(t, w.Body.String(), "phone number is already registered")
	})
}
