package handlers

// payment_webhook_dedup_test.go — #462 leftover: event-level replay dedup on the
// Razorpay webhook. A replayed (provider-retried or maliciously re-POSTed) event
// that verified once must NOT re-run the handler a second time; the endpoint
// should ACK the duplicate with 200 {"status":"duplicate"} and leave a single
// processed_events ledger row. Uses payment.failed (the leanest handler — one
// conditional UPDATE on orders, no referral/notify/tips side-effects) so the
// test stays focused on the dedup seam.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/services"
)

func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// decodeStatus extracts the "status" field from a webhook JSON response.
func decodeStatus(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Status
}

// withRazorpayWebhookSecret injects a webhook secret via the config fallback path
// (Secret Manager is unconfigured in tests → GetRazorpay builds from config), so
// VerifyWebhookSignature accepts test-signed payloads. Restores prior state.
func withRazorpayWebhookSecret(t *testing.T, secret string) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{
		RazorpayKeyID:         "rzp_test_key",
		RazorpayKeySecret:     "rzp_test_secret",
		RazorpayWebhookSecret: secret,
	}
	services.InvalidateRazorpay()
	services.GetRazorpay() // populate the package client from the config fallback
	t.Cleanup(func() {
		config.AppConfig = prev
		services.InvalidateRazorpay()
	})
}

// addProcessedEventsTable adds the dedup ledger to a setupPayDB() database.
func addProcessedEventsTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS processed_events (
		consumer TEXT NOT NULL, msg_id TEXT NOT NULL, subject TEXT DEFAULT '',
		processed_at DATETIME, PRIMARY KEY (consumer, msg_id)
	)`).Error)
}

func razorpaySign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// postRazorpayWebhook signs the body and posts it with the given event-id header.
func postRazorpayWebhook(t *testing.T, secret, eventID string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhooks/razorpay", NewPaymentHandler().RazorpayWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", bytesReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Razorpay-Signature", razorpaySign(body, secret))
	if eventID != "" {
		req.Header.Set("X-Razorpay-Event-Id", eventID)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRazorpayWebhook_ReplayReturnsDuplicate(t *testing.T) {
	db := setupPayDB(t)
	addProcessedEventsTable(t, db)
	secret := "whsec_dedup_test"
	withRazorpayWebhookSecret(t, secret)

	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	// Order sits pending with razorpay_order_id order_dedup.
	payOrder(t, db, cust, chef, "pending", 500, "order_dedup", "")

	body, err := json.Marshal(map[string]any{
		"event": "payment.failed",
		"payload": map[string]any{
			"payment": map[string]any{
				"entity": map[string]any{"id": "pay_dedup", "order_id": "order_dedup"},
			},
		},
	})
	require.NoError(t, err)

	// First delivery: processes, order → failed, plain ok.
	w1 := postRazorpayWebhook(t, secret, "evt_rp_dedup", body)
	require.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "ok", decodeStatus(t, w1))

	var statusAfter1 string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE razorpay_order_id = ?`, "order_dedup").Scan(&statusAfter1).Error)
	require.Equal(t, "failed", statusAfter1, "first delivery must process the event")

	// Simulate a later legitimate state change that a replay must NOT clobber.
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded' WHERE razorpay_order_id = ?`, "order_dedup").Error)

	// Replay: same event id → deduped. Response is duplicate; handler is NOT
	// re-run, so the order stays 'refunded' (a re-run would be a no-op here since
	// refunded is terminal, but the response + single ledger row prove the skip).
	w2 := postRazorpayWebhook(t, secret, "evt_rp_dedup", body)
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "duplicate", decodeStatus(t, w2), "replay must be acked as duplicate")

	var statusAfter2 string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE razorpay_order_id = ?`, "order_dedup").Scan(&statusAfter2).Error)
	assert.Equal(t, "refunded", statusAfter2, "replay must not re-run the handler")

	var rows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM processed_events WHERE consumer = ? AND msg_id = ?`, "webhook:razorpay", "evt_rp_dedup").Scan(&rows).Error)
	assert.Equal(t, int64(1), rows, "exactly one dedup ledger row")
}

func TestRazorpayWebhook_BadSignatureWritesNoLedgerRow(t *testing.T) {
	db := setupPayDB(t)
	addProcessedEventsTable(t, db)
	withRazorpayWebhookSecret(t, "whsec_dedup_test")

	body := []byte(`{"event":"payment.failed","payload":{}}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/razorpay", bytesReader(body))
	req.Header.Set("X-Razorpay-Signature", "deadbeef") // invalid
	req.Header.Set("X-Razorpay-Event-Id", "evt_forged")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhooks/razorpay", NewPaymentHandler().RazorpayWebhook)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	var rows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM processed_events WHERE msg_id = ?`, "evt_forged").Scan(&rows).Error)
	assert.Equal(t, int64(0), rows, "a forged/unauthenticated request must never claim a dedup row")
}
