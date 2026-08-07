package services

// payout_audit_test.go — #397. The money-movement audit trail: every real gateway
// transfer hold/release/reverse writes one append-only system AuditLog row. The
// injectable RazorpayClient.baseURL (pointed at an httptest.Server) lets the order
// release/reverse seams run end-to-end WITHOUT a live gateway, so we can assert the
// audit row is actually written on a real movement and NOT on an idempotent no-op.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// withRazorpayTestServer points GetRazorpay() at an httptest.Server so the transfer
// seams execute against canned gateway responses. Restores the previous client.
func withRazorpayTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	withRazorpayClient(t, &RazorpayClient{
		keyID: "rzp_test", keySecret: "secret", baseURL: srv.URL, fetchedAt: time.Now(),
	})
}

// orderPayoutFlagOn turns ORDER_PAYOUT_AUTO_RELEASE_ENABLED on for the test.
func orderPayoutFlagOn(t *testing.T) {
	t.Helper()
	saved := config.AppConfig
	t.Cleanup(func() { config.AppConfig = saved })
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}
}

// auditRows returns the new_value JSON of every audit_logs row with the given action,
// asserting each is a SYSTEM row (user_id NULL — no human actor on a transfer seam).
func auditRows(t *testing.T, db *gorm.DB, action string) []map[string]any {
	t.Helper()
	var rows []struct {
		UserID   *string
		NewValue string
	}
	require.NoError(t, db.Raw(`SELECT user_id, new_value FROM audit_logs WHERE action = ? ORDER BY created_at`, action).Scan(&rows).Error)
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		require.Nil(t, r.UserID, "transfer-movement audit row must be system-actor (no user_id)")
		var m map[string]any
		require.NoError(t, json.Unmarshal([]byte(r.NewValue), &m))
		out = append(out, m)
	}
	return out
}

func seedGatewayOrder(t *testing.T, db *gorm.DB, rzOrderID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, status, razorpay_order_id) VALUES (?,?,?)`,
		id.String(), "delivered", rzOrderID).Error)
	return id
}

// The helper writes exactly the compliance shape: a system-actor row keyed by
// aggregate, with transfer id + amount + reason in new_value.
func TestAuditTransferMovement_WritesAppendOnlySystemRow(t *testing.T) {
	db := setupCrossguardDB(t)
	aggID := uuid.New()

	auditTransferMovement(auditTransferReverse, aggTypeOrder, aggID, "trf_x1", 12345, "order refunded")

	var row struct {
		UserID     *string
		Action     string
		EntityType string
		EntityID   string
		NewValue   string
	}
	require.NoError(t, db.Raw(`SELECT user_id, action, entity_type, entity_id, new_value FROM audit_logs`).Scan(&row).Error)
	require.Nil(t, row.UserID, "system actor — no user_id")
	require.Equal(t, "payout.transfer.reverse", row.Action)
	require.Equal(t, aggTypeOrder, row.EntityType)
	require.Equal(t, aggID.String(), row.EntityID)

	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(row.NewValue), &payload))
	require.Equal(t, "trf_x1", payload["transfer_id"])
	require.Equal(t, float64(12345), payload["amount_paise"])
	require.Equal(t, "order refunded", payload["reason"])
}

func TestSettleReverse_Day_AuditsSettleMachineryReversal(t *testing.T) {
	escrowOn(t) // MealPlanEscrowActive() gates reverseMoney's day branch
	db := setupCrossguardDB(t)
	// A day at reversed+unsettled with a live transfer = the exact drift the reconcile re-drives.
	dayID := seedCrossDay(t, db, models.PayoutHoldReversed, nil)

	withRazorpayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/reversals") {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "rvsl_day"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	require.NoError(t, settleReverse(db, aggTypeMealPlanDay, dayID))
	require.True(t, dayIsSettled(t, db, dayID.String()), "seam succeeded → settled stamped")

	rows := auditRows(t, db, auditTransferReverse)
	require.Len(t, rows, 1, "the settle-machinery day claw-back is audited")
	require.Equal(t, "trf_abc123", rows[0]["transfer_id"])
}

// The idempotent already-reversed re-drive settles (tolerated) but writes NO audit row.
func TestSettleReverse_Day_NoAuditWhenAlreadyReversed(t *testing.T) {
	escrowOn(t)
	db := setupCrossguardDB(t)
	dayID := seedCrossDay(t, db, models.PayoutHoldReversed, nil)

	withRazorpayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/reversals") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{
				"description": "The transfer has been fully reversed",
			}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	require.NoError(t, settleReverse(db, aggTypeMealPlanDay, dayID), "already-reversed is tolerated")
	require.True(t, dayIsSettled(t, db, dayID.String()), "idempotent re-drive still settles the row")
	require.Empty(t, auditRows(t, db, auditTransferReverse), "no new money moved → no audit row")
}
