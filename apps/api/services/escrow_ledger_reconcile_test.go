package services

// escrow_ledger_reconcile_test.go — #398 item 1. The escrow conservation ledger detects a
// gateway-vs-platform payout mismatch — above all a REFUNDED order whose chef transfer is still
// settled at the gateway (chef paid AND customer refunded) — and records it as a PaymentDrift
// row, resolving it when a later scan finds it healed. Pure detector unit tests + a DB/httptest
// runner test using the shared razorpay seam.

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupLedgerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, order_number TEXT DEFAULT '',
		status TEXT, payout_hold_status TEXT DEFAULT '', razorpay_order_id TEXT DEFAULT '',
		refunded_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	// Hand-DDL payment_drifts (AutoMigrate emits the Postgres gen_random_uuid() default, which
	// sqlite rejects). Production uses AutoMigrate on Postgres where the default is valid.
	require.NoError(t, db.Exec(`CREATE TABLE payment_drifts (id TEXT PRIMARY KEY, agg_type TEXT, agg_id TEXT,
		kind TEXT, detail TEXT, expected_paise INTEGER DEFAULT 0, gateway_paise INTEGER DEFAULT 0,
		detected_at DATETIME, resolved_at DATETIME, updated_at DATETIME)`).Error)
	// meal_plan_days / group_orders carry DIRECT chef payout transfers (PayoutTransferID),
	// reconciled via FetchTransfer(id) — the #398 day/group slice. Only the columns the scans
	// Select are present (neither model has a soft-delete column).
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, status TEXT DEFAULT '',
		payout_hold_status TEXT DEFAULT '', payout_transfer_id TEXT DEFAULT '', refund_txn_id TEXT,
		updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, status TEXT DEFAULT '',
		payout_hold_status TEXT DEFAULT '', payout_transfer_id TEXT DEFAULT '', updated_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedHeldOrder(t *testing.T, db *gorm.DB, hold models.PayoutHoldStatus, status models.OrderStatus, refunded bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	refClause := "NULL"
	if refunded {
		refClause = "CURRENT_TIMESTAMP"
	}
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, status, payout_hold_status, razorpay_order_id, refunded_at, updated_at)
		 VALUES (?,?,?,?,?,`+refClause+`,CURRENT_TIMESTAMP)`,
		id.String(), "ORD-L", string(status), string(hold), "order_rzp_"+id.String()[:8]).Error)
	return id
}

func openDrifts(t *testing.T, db *gorm.DB, aggID uuid.UUID) []models.PaymentDrift {
	t.Helper()
	var rows []models.PaymentDrift
	require.NoError(t, db.Where("agg_id = ? AND resolved_at IS NULL", aggID).Find(&rows).Error)
	return rows
}

// transfersHandler serves canned FetchOrderTransfers for any order id.
func transfersHandler(items []map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transfers") {
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

// --- pure detector ---

func TestSummariseTransfers_Buckets(t *testing.T) {
	l := summariseTransfers([]TransferResponse{
		{ID: "t1", Amount: 10000, AmountReversed: 0, OnHold: true},    // held 10000
		{ID: "t2", Amount: 8000, AmountReversed: 0, OnHold: false},    // settled 8000
		{ID: "t3", Amount: 5000, AmountReversed: 5000, OnHold: false}, // fully reversed
		{ID: "t4", Amount: 4000, AmountReversed: 1000, OnHold: false}, // settled 3000 + reversed 1000
		{ID: "", Amount: 999}, // no id → ignored
	})
	require.Equal(t, int64(6000), l.reversedPaise)
	require.Equal(t, int64(11000), l.settledPaise) // 8000 + 3000
	require.Equal(t, int64(10000), l.heldPaise)
}

func TestDetectTransferDrift_ChefPaidOnRefund(t *testing.T) {
	d := detectTransferDrift(aggLedgerState{mustNotPay: true}, transferLedger{settledPaise: 8000})
	require.Len(t, d, 1)
	require.Equal(t, models.DriftChefPaidOnRefund, d[0].kind)
	require.Equal(t, int64(8000), d[0].gatewayPaise)
}

// Mixed chef-settled + rider-still-held on a refunded order reports the TOTAL not-yet-reversed
// exposure (settled + held), not just the settled slice.
func TestDetectTransferDrift_ChefPaidOnRefund_MixedReportsTotalAtRisk(t *testing.T) {
	d := detectTransferDrift(aggLedgerState{mustNotPay: true}, transferLedger{settledPaise: 8000, heldPaise: 3000})
	require.Len(t, d, 1)
	require.Equal(t, models.DriftChefPaidOnRefund, d[0].kind)
	require.Equal(t, int64(11000), d[0].gatewayPaise, "settled 8000 + still-held 3000 = total at risk")
}

func TestDetectTransferDrift_RefundedStillHeld(t *testing.T) {
	d := detectTransferDrift(aggLedgerState{mustNotPay: true}, transferLedger{heldPaise: 8000})
	require.Len(t, d, 1)
	require.Equal(t, models.DriftReversedButNotAtGateway, d[0].kind)
}

func TestDetectTransferDrift_HeldButReleased(t *testing.T) {
	d := detectTransferDrift(aggLedgerState{holdStatus: models.PayoutHoldAwaitingConfirmation}, transferLedger{settledPaise: 8000})
	require.Len(t, d, 1)
	require.Equal(t, models.DriftHeldButReleasedAtGateway, d[0].kind)
}

func TestDetectTransferDrift_ReleasedButHeld(t *testing.T) {
	d := detectTransferDrift(aggLedgerState{holdStatus: models.PayoutHoldReleased}, transferLedger{heldPaise: 8000})
	require.Len(t, d, 1)
	require.Equal(t, models.DriftReleasedButHeldAtGateway, d[0].kind)
}

func TestDetectTransferDrift_ConsistentStates_NoDrift(t *testing.T) {
	// released + all settled → consistent.
	require.Empty(t, detectTransferDrift(aggLedgerState{holdStatus: models.PayoutHoldReleased}, transferLedger{settledPaise: 8000}))
	// awaiting + all held → consistent.
	require.Empty(t, detectTransferDrift(aggLedgerState{holdStatus: models.PayoutHoldAwaitingConfirmation}, transferLedger{heldPaise: 8000}))
	// refunded + fully reversed → consistent.
	require.Empty(t, detectTransferDrift(aggLedgerState{mustNotPay: true}, transferLedger{reversedPaise: 8000}))
}

// --- runner ---

// A refunded order whose chef transfer is still settled at the gateway opens the alarm row;
// once the gateway shows it reversed, a re-scan RESOLVES it.
func TestRunEscrowLedgerReconcile_OpensThenResolves(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldOrder(t, db, models.PayoutHoldDisputed, models.OrderStatusDelivered, true) // refunded=true

	// First scan: gateway shows a settled (paid) transfer → chef paid on refund.
	withRazorpayTestServer(t, transfersHandler([]map[string]any{{"id": "trf1", "amount": 8000, "on_hold": false}}))
	require.Equal(t, 1, RunEscrowLedgerReconcile())
	open := openDrifts(t, db, id)
	require.Len(t, open, 1)
	require.Equal(t, models.DriftChefPaidOnRefund, open[0].Kind)

	// Second scan: gateway now shows it fully reversed → healed → the open row resolves.
	withRazorpayTestServer(t, transfersHandler([]map[string]any{{"id": "trf1", "amount": 8000, "amount_reversed": 8000, "on_hold": false}}))
	require.Equal(t, 0, RunEscrowLedgerReconcile())
	require.Empty(t, openDrifts(t, db, id), "healed drift is resolved, not left open")
}

// Re-scanning a still-drifting order does not duplicate the open row.
func TestRunEscrowLedgerReconcile_Idempotent(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldOrder(t, db, models.PayoutHoldWithheld, models.OrderStatusDelivered, false) // withheld → mustNotPay
	withRazorpayTestServer(t, transfersHandler([]map[string]any{{"id": "trf1", "amount": 8000, "on_hold": false}}))

	require.Equal(t, 1, RunEscrowLedgerReconcile())
	require.Equal(t, 1, RunEscrowLedgerReconcile())
	require.Len(t, openDrifts(t, db, id), 1, "a re-scan updates the existing open row, never duplicates")
}

// An order with no escrow hold is never scanned (flags-OFF safety: the scan set is empty).
func TestRunEscrowLedgerReconcile_SkipsUnheldOrders(t *testing.T) {
	db := setupLedgerDB(t)
	id := seedHeldOrder(t, db, models.PayoutHoldNone, models.OrderStatusRefunded, true) // hold '' → not scanned
	withRazorpayTestServer(t, transfersHandler([]map[string]any{{"id": "trf1", "amount": 8000, "on_hold": false}}))

	require.Equal(t, 0, RunEscrowLedgerReconcile(), "no escrow hold → not in the scan set")
	require.Empty(t, openDrifts(t, db, id))
}
