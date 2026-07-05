package handlers

// delivery_webhook_dedup_test.go — #462 leftover: event-level replay dedup on the
// 3PL delivery webhook (the most sensitive replay surface — a "delivered" event
// releases held escrow). Verifies:
//   - an exact replay is skipped (does NOT re-run HandleProviderWebhook, so it
//     can't clobber a later status) and is acked 200 {"status":"duplicate"};
//   - a TRANSIENT dispatch error (delivery-not-found booking race) releases the
//     claim so the provider retry re-processes;
//   - a PERMANENT dispatch error (unmapped status) is acked 200 {"status":"ignored"}
//     with the claim KEPT — no 500 retry-storm (finding-2 guard).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

// setupDeliveryWebhookDB uses raw CREATE TABLE (only the columns the webhook path
// touches) — gorm.AutoMigrate emits invalid SQLite DDL for these uuid-defaulted
// models (see chef_dpdp_test.go). The best-effort outbox insert is left without a
// table on purpose: EnqueueEvent logs+swallows the error, so it doesn't affect the
// dedup assertions. Loads use SELECT * so a column subset is fine.
func setupDeliveryWebhookDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	require.NoError(t, db.Exec(`CREATE TABLE delivery_providers (
		id TEXT PRIMARY KEY, name TEXT DEFAULT '', code TEXT, webhook_secret TEXT DEFAULT '',
		status_mapping TEXT DEFAULT '{}', is_enabled INTEGER DEFAULT 0, is_active INTEGER DEFAULT 1,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE deliveries (
		id TEXT PRIMARY KEY, order_id TEXT, provider_id TEXT, external_delivery_id TEXT DEFAULT '',
		status TEXT DEFAULT 'pending', assignment_type TEXT DEFAULT 'manual',
		picked_up_at DATETIME, delivered_at DATETIME, cancelled_at DATETIME,
		cancel_reason TEXT DEFAULT '', failure_reason TEXT DEFAULT '',
		external_tracking_url TEXT DEFAULT '', provider_status TEXT DEFAULT '',
		rider_name TEXT DEFAULT '', rider_phone TEXT DEFAULT '',
		rider_latitude REAL DEFAULT 0, rider_longitude REAL DEFAULT 0,
		assigned_at DATETIME, created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE processed_events (
		consumer TEXT NOT NULL, msg_id TEXT NOT NULL, subject TEXT DEFAULT '',
		processed_at DATETIME, PRIMARY KEY (consumer, msg_id)
	)`).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedBorzoProvider(t *testing.T, db *gorm.DB, secret, statusMapping string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO delivery_providers (id, name, code, webhook_secret, status_mapping, is_enabled, is_active)
		 VALUES (?, 'Borzo', 'borzo', ?, ?, 1, 1)`,
		id.String(), secret, statusMapping).Error)
	return id
}

func seedDelivery(t *testing.T, db *gorm.DB, providerID uuid.UUID, externalID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO deliveries (id, order_id, provider_id, external_delivery_id, status, assignment_type, assigned_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'assigned', 'third_party', ?, ?, ?)`,
		id.String(), uuid.New().String(), providerID.String(), externalID, time.Now(), time.Now(), time.Now()).Error)
	return id
}

func postDeliveryWebhook(t *testing.T, providerCode, secret string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhooks/delivery/:provider", NewDeliveryProviderHandler().HandleWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/delivery/"+providerCode, bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signDelivery(body, secret))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDeliveryWebhook_ReplaySkipsReprocess(t *testing.T) {
	db := setupDeliveryWebhookDB(t)
	secret := "borzo_wh_secret"
	prov := seedBorzoProvider(t, db, secret, `{"PICKED_UP":"picked_up"}`)
	del := seedDelivery(t, db, prov, "EXT-1")

	body, err := json.Marshal(map[string]any{"delivery_id": "EXT-1", "status": "PICKED_UP"})
	require.NoError(t, err)

	// First delivery processes → status becomes picked_up.
	w1 := postDeliveryWebhook(t, "borzo", secret, body)
	require.Equal(t, http.StatusOK, w1.Code)
	var s1 string
	require.NoError(t, db.Raw(`SELECT status FROM deliveries WHERE id = ?`, del).Scan(&s1).Error)
	require.Equal(t, "picked_up", s1)

	// A later legitimate status change the replay must NOT clobber.
	require.NoError(t, db.Exec(`UPDATE deliveries SET status = 'in_transit' WHERE id = ?`, del).Error)

	// Exact replay → deduped, handler NOT re-run → status stays in_transit.
	w2 := postDeliveryWebhook(t, "borzo", secret, body)
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "duplicate", decodeStatus(t, w2))

	var s2 string
	require.NoError(t, db.Raw(`SELECT status FROM deliveries WHERE id = ?`, del).Scan(&s2).Error)
	assert.Equal(t, "in_transit", s2, "replay must not re-run the webhook and clobber the newer status")
}

func TestDeliveryWebhook_TransientErrorReleasesClaim(t *testing.T) {
	db := setupDeliveryWebhookDB(t)
	secret := "borzo_wh_secret"
	prov := seedBorzoProvider(t, db, secret, `{"PICKED_UP":"picked_up"}`)

	// No delivery row for EXT-2 yet → HandleProviderWebhook returns a (transient)
	// not-found error → handler releases the claim and 500s so the provider retries.
	body, err := json.Marshal(map[string]any{"delivery_id": "EXT-2", "status": "PICKED_UP"})
	require.NoError(t, err)

	w1 := postDeliveryWebhook(t, "borzo", secret, body)
	require.Equal(t, http.StatusInternalServerError, w1.Code)

	var rows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM processed_events WHERE consumer = ?`, "webhook:delivery:borzo").Scan(&rows).Error)
	require.Equal(t, int64(0), rows, "a transient failure must release the claim (no ledger row)")

	// The booking row now exists (the race resolved); the provider retry (same
	// body) must re-process rather than being silently deduped.
	del := seedDelivery(t, db, prov, "EXT-2")
	w2 := postDeliveryWebhook(t, "borzo", secret, body)
	require.Equal(t, http.StatusOK, w2.Code)
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM deliveries WHERE id = ?`, del).Scan(&s).Error)
	assert.Equal(t, "picked_up", s, "the retry after a released claim must process")
}

func TestDeliveryWebhook_PermanentErrorAcksAndKeepsClaim(t *testing.T) {
	db := setupDeliveryWebhookDB(t)
	secret := "borzo_wh_secret"
	prov := seedBorzoProvider(t, db, secret, `{"PICKED_UP":"picked_up"}`) // WEIRD not mapped
	seedDelivery(t, db, prov, "EXT-3")

	body, err := json.Marshal(map[string]any{"delivery_id": "EXT-3", "status": "WEIRD_STATUS"})
	require.NoError(t, err)

	// Unmapped status is un-processable (permanent) → ACK 200 "ignored", claim kept,
	// so the provider stops retrying instead of hammering a 500 forever.
	w1 := postDeliveryWebhook(t, "borzo", secret, body)
	require.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "ignored", decodeStatus(t, w1))

	var rows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM processed_events WHERE consumer = ?`, "webhook:delivery:borzo").Scan(&rows).Error)
	assert.Equal(t, int64(1), rows, "a permanent error keeps the claim (acked, no retry storm)")

	// A replay of the same un-processable event is deduped.
	w2 := postDeliveryWebhook(t, "borzo", secret, body)
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "duplicate", decodeStatus(t, w2))
}
