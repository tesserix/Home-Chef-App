package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

// bff_auth_purged_test.go — a signed session whose SUBJECT ROW no longer exists
// must be rejected with 401, not fall through as an authenticated request.
//
// Found live: after an account was hard-purged, its still-valid token kept
// working — the hydration block only acted on err == nil, so a missing users
// row skipped every deleted/deactivated check and the app kept rendering
// signed-in chrome (profile shell, ₹0 wallet) with no call ever telling it to
// sign out. 401 is what the mobile clients turn into logout → guest landing.

func setupBFFUserDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT,
		email_enc TEXT DEFAULT '', email_bidx TEXT DEFAULT '',
		is_active BOOLEAN DEFAULT 1, deleted_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func bffEngine(key []byte) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/x", BFFAuth(BFFAuthConfig{HMACKey: key, Window: time.Minute}), func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	return r
}

func TestBFFAuth_PurgedSubject_401(t *testing.T) {
	setupBFFUserDB(t) // users table exists; the subject row does NOT
	key := []byte("test-key-32-bytes-padding-padding!")
	r := bffEngine(key)

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	attachSigned(req, body, key, BFFIdentity{
		UserID: uuid.NewString(), Email: "gone@example.com", Role: "customer", Pool: "customer",
	}, time.Now().Unix())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code,
		"a session for a purged account must 401, not act authenticated: %s", w.Body.String())
}

func TestBFFAuth_LiveSubject_StillPasses(t *testing.T) {
	db := setupBFFUserDB(t)
	key := []byte("test-key-32-bytes-padding-padding!")
	r := bffEngine(key)

	uid := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, is_active) VALUES (?,?,1)`,
		uid, "alive@example.com").Error)

	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(body))
	attachSigned(req, body, key, BFFIdentity{
		UserID: uid, Email: "alive@example.com", Role: "customer", Pool: "customer",
	}, time.Now().Unix())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, "a live account must keep working: %s", w.Body.String())
}
