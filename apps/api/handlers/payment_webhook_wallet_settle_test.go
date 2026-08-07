package handlers

// payment_webhook_wallet_settle_test.go — #395·3. An order completed ONLY via the
// payment-success webhook (client dropped before calling verify) must still debit
// the store credit applied at checkout. Before the fix that settlement ran only in
// the verify path, so a webhook-only completion left the wallet un-debited and the
// customer kept credit they had spent.
//
// The chef/driver top-up transfers this file also used to assert went with the Route
// rail in #1086: the chef is paid the whole delivered order on the statement path.
// Driven on the Cashfree webhook since #1086 — the Razorpay webhook is gone and no
// order can be captured on it.

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedWalletOrder inserts a pending, wallet-at-checkout Cashfree order plus a
// customer wallet pre-funded with the applied credit.
func seedWalletOrder(t *testing.T, db *gorm.DB, cfOrderID string, total, walletApplied float64) (order, cust uuid.UUID) {
	t.Helper()
	cust = payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	order = cfPayOrder(t, db, cust, chef, "pending", total, cfOrderID)
	require.NoError(t, db.Exec(`UPDATE orders SET wallet_applied = ? WHERE id = ?`, walletApplied, order.String()).Error)
	// Pre-fund the customer wallet with the applied credit so the debit succeeds.
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance, currency, created_at, updated_at)
		VALUES (?,?,?,?,datetime('now'),datetime('now'))`, uuid.NewString(), cust.String(), walletApplied, "INR").Error)
	return order, cust
}

func capturedPayload(cfOrderID, cfPaymentID string, rupees float64) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"order": map[string]any{"order_id": cfOrderID},
		"payment": map[string]any{
			"cf_payment_id": cfPaymentID, "payment_status": "SUCCESS",
			"payment_amount": rupees, "payment_group": "upi",
		},
	})
	return b
}

// A duplicate delivery matches no order and falls through to the FSSAI fallback,
// which is a real query in production — without the table the fallback errors and
// the test would be asserting a missing fixture rather than the settlement.
func addFssaiRequestsTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE fssai_requests (id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT,
		status TEXT DEFAULT 'awaiting_payment', gateway_order TEXT DEFAULT '', mode TEXT DEFAULT 'live',
		created_at DATETIME, updated_at DATETIME)`).Error)
}

func walletDebitCount(t *testing.T, db *gorm.DB, orderID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM wallet_txns WHERE idempotency_key = ?`,
		"wallet-debit:"+orderID.String()).Scan(&n).Error)
	return n
}

// A webhook-only completion settles the wallet: order completed, credit debited.
func TestPaymentSuccessWebhook_SettlesWalletForWebhookOnlyCompletion(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	addProcessedEventsTable(t, db)

	orderID, cust := seedWalletOrder(t, db, "order_wh1", 500, 100)

	require.NoError(t, NewPaymentHandler().handleCashfreePaymentSuccess(capturedPayload("order_wh1", "9911", 400), models.ChefModeLive))

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, "completed", status, "webhook completes the order")

	require.Equal(t, int64(1), walletDebitCount(t, db, orderID), "the applied store credit is debited (was skipped pre-fix)")
	require.Equal(t, 0.0, walletBalance(t, db, cust), "balance 100 − 100 applied = 0")
}

// Settling twice (webhook wins, then a retry / a later verify also settles) must NOT
// double-debit — the idempotency that makes verify+webhook coexist.
func TestPaymentSuccessWebhook_WalletSettlementIdempotent(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	addProcessedEventsTable(t, db)
	addFssaiRequestsTable(t, db)

	orderID, cust := seedWalletOrder(t, db, "order_wh2", 500, 100)
	payload := capturedPayload("order_wh2", "9912", 400)

	require.NoError(t, NewPaymentHandler().handleCashfreePaymentSuccess(payload, models.ChefModeLive))
	require.NoError(t, NewPaymentHandler().handleCashfreePaymentSuccess(payload, models.ChefModeLive)) // retry / duplicate delivery

	require.Equal(t, int64(1), walletDebitCount(t, db, orderID), "debited exactly once across two settlements")
	require.Equal(t, 0.0, walletBalance(t, db, cust), "no double debit (balance not negative)")
}

// If the store-credit debit genuinely FAILS (balance drained between checkout and this
// delayed settlement), the order still completes and the wallet slice is left for
// reconcile — nothing is written off a credit the platform never collected.
func TestPaymentSuccessWebhook_DebitFailure_LeavesWalletUntouched(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	addProcessedEventsTable(t, db)

	// Applied 100 but the wallet only holds 40 now → DebitWallet fails (insufficient).
	orderID, cust := seedWalletOrder(t, db, "order_wh3", 500, 100)
	require.NoError(t, db.Exec(`UPDATE wallets SET balance = 40 WHERE user_id = ?`, cust.String()).Error)

	require.NoError(t, NewPaymentHandler().handleCashfreePaymentSuccess(capturedPayload("order_wh3", "9913", 400), models.ChefModeLive))

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, "completed", status, "the gateway capture still completes the order")
	require.Equal(t, int64(0), walletDebitCount(t, db, orderID), "no debit landed (insufficient balance)")
	require.Equal(t, 40.0, walletBalance(t, db, cust), "balance untouched by the failed debit")
}
