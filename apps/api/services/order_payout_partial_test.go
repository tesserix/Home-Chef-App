package services

// order_payout_partial_test.go — #549. A PARTIAL goodwill refund must claw back
// ONLY the refunded portion from the CHEF's held Route transfer (never the
// rider's, never more than the chef was paid) and must LEAVE the payout hold
// releasable so the chef is still paid the remainder. Driven end-to-end against an
// httptest gateway via the injectable RazorpayClient.baseURL (#559).

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedPartialOrder inserts a delivered gateway-charged order plus its chef_profile
// carrying the given Route account. Returns the order id.
func seedPartialOrder(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, chefAccount, rzOrderID string) uuid.UUID {
	t.Helper()
	orderID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, razorpay_account_id) VALUES (?, ?)`,
		chefID.String(), chefAccount).Error)
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, chef_id, status, razorpay_order_id, total, payout_hold_status, delivered_at)
		VALUES (?,?,?,?,?,?,?)`,
		orderID.String(), chefID.String(), "delivered", rzOrderID, 250.0, string(hold),
		time.Now().Add(-5*time.Hour)).Error)
	return orderID
}

// reversalCapture records the transfer id + amount of the single POST /reversals the
// seam issues, so a test can assert exactly which transfer moved and by how much.
type reversalCapture struct {
	id     string
	amount int
	hits   int
}

// partialTransfersServer canned-answers FetchOrderTransfers with `items` and captures
// the reversal. transfers is the raw JSON items list.
func partialTransfersServer(t *testing.T, cap *reversalCapture, items []map[string]any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transfers"):
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/reversals"):
			cap.hits++
			cap.id = strings.Split(strings.TrimPrefix(r.URL.Path, "/transfers/"), "/")[0]
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if a, ok := body["amount"].(float64); ok {
				cap.amount = int(a)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "rvsl_1"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

// The refunded portion is clawed back from the CHEF transfer only — the rider, who
// still delivered, is untouched.
func TestReverseOrderChefTransferPartial_ClawsBackChefPortionOnly(t *testing.T) {
	orderPayoutFlagOn(t)
	db := setupCrossguardDB(t)
	orderID := seedPartialOrder(t, db, models.PayoutHoldReleaseEligible, "acc_chef", "order_rzp_p1")

	cap := &reversalCapture{}
	withRazorpayTestServer(t, partialTransfersServer(t, cap, []map[string]any{
		{"id": "trf_chef", "recipient": "acc_chef", "amount": 90000},
		{"id": "trf_rider", "recipient": "acc_rider", "amount": 4000},
	}))

	require.NoError(t, ReverseOrderChefTransferPartial(db, orderID, 30000, "goodwill"))
	require.Equal(t, 1, cap.hits, "exactly one transfer reversed")
	require.Equal(t, "trf_chef", cap.id, "only the chef transfer is reversed, never the rider's")
	require.Equal(t, 30000, cap.amount, "reverses exactly the refunded portion")

	rows := auditRows(t, db, auditTransferReverse)
	require.Len(t, rows, 1)
	require.Equal(t, "trf_chef", rows[0]["transfer_id"])
	require.Equal(t, float64(30000), rows[0]["amount_paise"])
}

// The claw-back is capped at what the chef was actually paid — a refund larger than
// the chef's net transfer reverses the whole transfer (chef floors at zero, the
// platform absorbs the excess), it never goes negative.
func TestReverseOrderChefTransferPartial_CapsAtChefTransferAmount(t *testing.T) {
	orderPayoutFlagOn(t)
	db := setupCrossguardDB(t)
	orderID := seedPartialOrder(t, db, models.PayoutHoldReleaseEligible, "acc_chef", "order_rzp_p2")

	cap := &reversalCapture{}
	withRazorpayTestServer(t, partialTransfersServer(t, cap, []map[string]any{
		{"id": "trf_chef", "recipient": "acc_chef", "amount": 90000},
	}))

	require.NoError(t, ReverseOrderChefTransferPartial(db, orderID, 200000, "big goodwill"))
	require.Equal(t, 90000, cap.amount, "never claw back more than the chef was paid")
}

// A repeated partial refund reverses only what remains un-reversed — the seam caps
// by (Amount − AmountReversed), so it can't over-claw a transfer already partly
// reversed by an earlier goodwill refund.
func TestReverseOrderChefTransferPartial_CapsByAlreadyReversed(t *testing.T) {
	orderPayoutFlagOn(t)
	db := setupCrossguardDB(t)
	orderID := seedPartialOrder(t, db, models.PayoutHoldReleaseEligible, "acc_chef", "order_rzp_p3")

	cap := &reversalCapture{}
	withRazorpayTestServer(t, partialTransfersServer(t, cap, []map[string]any{
		{"id": "trf_chef", "recipient": "acc_chef", "amount": 90000, "amount_reversed": 60000},
	}))

	require.NoError(t, ReverseOrderChefTransferPartial(db, orderID, 50000, "second goodwill"))
	require.Equal(t, 30000, cap.amount, "caps at the 30000 still un-reversed on the transfer")
}

// Flag OFF → no gateway movement at all (state-only), so nothing is reversed and no
// movement audit is written. The DB state (hold releasable) is the only effect.
func TestReverseOrderChefTransferPartial_NoopWhenFlagOff(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID := seedPartialOrder(t, db, models.PayoutHoldReleaseEligible, "acc_chef", "order_rzp_p4")

	cap := &reversalCapture{}
	withRazorpayTestServer(t, partialTransfersServer(t, cap, []map[string]any{
		{"id": "trf_chef", "recipient": "acc_chef", "amount": 90000},
	}))

	require.NoError(t, ReverseOrderChefTransferPartial(db, orderID, 30000, "goodwill"))
	require.Equal(t, 0, cap.hits, "flag OFF → no gateway reversal")
	require.Empty(t, auditRows(t, db, auditTransferReverse))
}

// The partial cross-guard must NOT drive the hold to withheld/reversed and must NOT
// stamp refunded_at — either would re-block the WHOLE chef payout (the #549 bug),
// because the release-side backstop keys on refunded_at IS NOT NULL. The hold stays
// exactly releasable for the remainder.
func TestWithholdOrReverseOrderHoldForPartialRefund_LeavesHoldReleasable(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID := seedPartialOrder(t, db, models.PayoutHoldReleaseEligible, "acc_chef", "order_rzp_p5")

	require.NoError(t, WithholdOrReverseOrderHoldForPartialRefund(db, orderID, 30000, "goodwill"))

	require.Equal(t, models.PayoutHoldReleaseEligible, loadOrderHold(t, db, orderID),
		"a partial refund must NOT withhold the whole hold (#549)")
	var refundedAt sql.NullTime
	require.NoError(t, db.Raw(`SELECT refunded_at FROM orders WHERE id = ?`, orderID.String()).Scan(&refundedAt).Error)
	require.False(t, refundedAt.Valid, "partial guard must not stamp the terminal refunded_at marker")
}
