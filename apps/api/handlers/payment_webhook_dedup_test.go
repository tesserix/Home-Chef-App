package handlers

// payment_webhook_dedup_test.go — #462 leftover: event-level replay dedup on the
// payment webhook. A replayed (provider-retried or maliciously re-POSTed) event
// that verified once must NOT re-run the handler a second time; the endpoint
// should ACK the duplicate with 200 {"status":"duplicate"} and leave a single
// processed_events ledger row. Uses the failure event (the leanest handler — one
// conditional UPDATE on orders, no referral/notify/tips side-effects) so the
// test stays focused on the dedup seam.
//
// Driven on the Cashfree webhook since #1086 — the Razorpay one is gone.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// decodeStatus extracts the "status" field from a webhook JSON response.
func decodeStatus(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var resp struct {
		Status string `json:"status"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.Status
}

// addProcessedEventsTable adds the dedup ledger to a setupPayDB() database.
func addProcessedEventsTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS processed_events (
		consumer TEXT NOT NULL, msg_id TEXT NOT NULL, subject TEXT DEFAULT '',
		processed_at DATETIME, PRIMARY KEY (consumer, msg_id)
	)`).Error)
}

func failedWebhookBody(t *testing.T, cfOrderID string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{
		"type": "PAYMENT_FAILED_WEBHOOK",
		"data": map[string]any{
			"order":   map[string]any{"order_id": cfOrderID},
			"payment": map[string]any{"cf_payment_id": 7001, "payment_status": "FAILED", "payment_amount": 500.00},
		},
	})
	require.NoError(t, err)
	return b
}

func TestPaymentWebhook_ReplayReturnsDuplicate(t *testing.T) {
	db := setupPayDB(t)
	addProcessedEventsTable(t, db)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "order_dedup")

	withCashfreeGateway(t, cfGateway("order_dedup", "7001", 500, "ACTIVE"))
	body := failedWebhookBody(t, "order_dedup")
	ts := nowUnix()

	// First delivery: processes, order → failed, plain ok.
	w1 := cfWebhookRequest(t, NewPaymentHandler(), body, true, ts)
	require.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "ok", decodeStatus(t, w1))
	require.Equal(t, string(models.PaymentFailed), paymentStatusOf(t, db, orderID), "first delivery must process the event")

	// Simulate a later legitimate state change that a replay must NOT clobber.
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded' WHERE id = ?`, orderID.String()).Error)

	// Replay: byte-identical body → same event id → deduped. The handler is NOT
	// re-run, so the order stays 'refunded' (a re-run would be a no-op here since
	// refunded is terminal, but the response + single ledger row prove the skip).
	w2 := cfWebhookRequest(t, NewPaymentHandler(), body, true, ts)
	require.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "duplicate", decodeStatus(t, w2), "replay must be acked as duplicate")
	assert.Equal(t, string(models.PaymentRefunded), paymentStatusOf(t, db, orderID), "replay must not re-run the handler")

	var rows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM processed_events`).Scan(&rows).Error)
	assert.Equal(t, int64(1), rows, "exactly one dedup ledger row")
}

func TestPaymentWebhook_BadSignatureWritesNoLedgerRow(t *testing.T) {
	db := setupPayDB(t)
	addProcessedEventsTable(t, db)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "order_forged")

	withCashfreeGateway(t, cfGateway("order_forged", "7001", 500, "ACTIVE"))

	w := cfWebhookRequest(t, NewPaymentHandler(), failedWebhookBody(t, "order_forged"), false, nowUnix())

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, string(models.PaymentPending), paymentStatusOf(t, db, orderID))
	var rows int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM processed_events`).Scan(&rows).Error)
	assert.Equal(t, int64(0), rows, "a forged/unauthenticated request must never claim a dedup row")
}
