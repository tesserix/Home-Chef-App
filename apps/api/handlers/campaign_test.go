package handlers

// campaign_test.go — HTTP-level checks for the marketing campaign surface (#56):
// the admin create + audience-preview path, and the public open-pixel +
// unsubscribe endpoints. The segment/dispatch logic is pinned in the services
// package; here we verify the handler wiring, status codes, and that the public
// tracking endpoints move state and stay leak-free.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupCampaignHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE users (email_enc text DEFAULT '', email_bidx text DEFAULT '', first_name_enc text DEFAULT '', last_name_enc text DEFAULT '', phone_enc text DEFAULT '', phone_bidx text DEFAULT '', id text PRIMARY KEY, email text, role text, is_active integer DEFAULT 1,
			fcm_token text DEFAULT '', marketing_consent integer DEFAULT 0, marketing_consent_at datetime,
			created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE addresses (line1_enc text DEFAULT '', line2_enc text DEFAULT '', id text PRIMARY KEY, user_id text, city text, created_at datetime)`,
		`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id text PRIMARY KEY, customer_id text, status text, created_at datetime, deleted_at datetime)`,
		`CREATE TABLE meal_subscriptions (id text PRIMARY KEY, customer_id text, status text, created_at datetime, deleted_at datetime)`,
		`CREATE TABLE notification_preferences (id text PRIMARY KEY, user_id text, category text,
			email_enabled integer, push_enabled integer, sms_enabled integer, created_at datetime, updated_at datetime)`,
		`CREATE TABLE campaigns (id text PRIMARY KEY, name text, status text, send_push integer, send_email integer,
			push_title text, push_body text, email_subject text, email_html text, segment text,
			scheduled_at datetime, sent_at datetime, created_by text, recipients integer,
			created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE campaign_deliveries (id text PRIMARY KEY, campaign_id text, user_id text, channel text, status text,
			failure_reason text, sent_at datetime, opened_at datetime, created_at datetime, updated_at datetime)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func adminReq(t *testing.T, method, path string, register func(*gin.Engine), body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", uuid.New()); c.Next() })
	register(r)
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAdminCreateAndPreviewCampaign(t *testing.T) {
	db := setupCampaignHandlerDB(t)
	// one reachable customer
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, role, is_active, fcm_token, marketing_consent) VALUES (?,?,?,?,?,?)`,
		uuid.New().String(), "a@ex.com", "customer", 1, "tok", 1).Error)

	h := &AdminHandler{}

	// Create
	w := adminReq(t, http.MethodPost, "/admin/campaigns",
		func(r *gin.Engine) { r.POST("/admin/campaigns", h.CreateCampaign) },
		map[string]any{
			"name": "Launch", "sendPush": true, "pushTitle": "Hi", "pushBody": "Yo",
			"segment": map[string]any{"recency": "all"},
		})
	require.Equal(t, http.StatusCreated, w.Code)
	var created models.Campaign
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, models.CampaignStatusDraft, created.Status)

	// Preview
	w = adminReq(t, http.MethodPost, "/admin/campaigns/preview",
		func(r *gin.Engine) { r.POST("/admin/campaigns/preview", h.PreviewCampaignSegment) },
		map[string]any{"segment": map[string]any{}, "sendPush": true, "sendEmail": true})
	require.Equal(t, http.StatusOK, w.Code)
	var prev map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &prev))
	require.Equal(t, float64(1), prev["matched"])
	require.Equal(t, float64(1), prev["reachablePush"])
}

func TestAdminCreateCampaign_ValidationError(t *testing.T) {
	setupCampaignHandlerDB(t)
	h := &AdminHandler{}
	// push channel with no content → 400
	w := adminReq(t, http.MethodPost, "/admin/campaigns",
		func(r *gin.Engine) { r.POST("/admin/campaigns", h.CreateCampaign) },
		map[string]any{"name": "x", "sendPush": true})
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCampaignPublicTracking(t *testing.T) {
	db := setupCampaignHandlerDB(t)
	uid := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, role, marketing_consent) VALUES (?,?,?,?)`,
		uid.String(), "b@ex.com", "customer", 1).Error)
	deliveryID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO campaign_deliveries (id, campaign_id, user_id, channel, status) VALUES (?,?,?,?,?)`,
		deliveryID.String(), uuid.New().String(), uid.String(), "email", "sent").Error)

	h := NewCampaignHandler()

	// Open pixel → 200 image/gif, sets opened_at.
	w := adminReq(t, http.MethodGet, "/api/v1/campaigns/track/open/"+deliveryID.String(),
		func(r *gin.Engine) { r.GET("/api/v1/campaigns/track/open/:id", h.TrackOpen) }, nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "image/gif", w.Header().Get("Content-Type"))
	var opened int64
	db.Raw(`SELECT count(*) FROM campaign_deliveries WHERE id = ? AND opened_at IS NOT NULL`, deliveryID.String()).Scan(&opened)
	require.Equal(t, int64(1), opened)

	// Unsubscribe → withdraws marketing consent.
	w = adminReq(t, http.MethodGet, "/api/v1/campaigns/unsubscribe/"+deliveryID.String(),
		func(r *gin.Engine) { r.GET("/api/v1/campaigns/unsubscribe/:id", h.Unsubscribe) }, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var consent int
	db.Raw(`SELECT marketing_consent FROM users WHERE id = ?`, uid.String()).Scan(&consent)
	require.Equal(t, 0, consent)
}
