package handlers

// payment_webhook_wallet_settle_test.go — #395·3. An order completed ONLY via the
// payment.captured webhook (client dropped before calling verify) must still settle
// its wallet-at-checkout slice: debit the applied store credit and issue the
// platform-funded chef/driver top-ups. Before the fix that settlement ran only in the
// verify path, so a webhook-only completion left the wallet un-debited (customer keeps
// credit they spent) and the chef/driver top-up unpaid.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/services"
)

// withRazorpayTransferServer points GetRazorpay at an httptest server that answers
// CreateTransfer (POST /transfers) and carries webhookSecret so VerifyWebhookSignature
// accepts test-signed payloads. Returns a pointer to the transfer-POST counter.
func withRazorpayTransferServer(t *testing.T, webhookSecret string) *int {
	t.Helper()
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/transfers") {
			count++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "trf_wallet"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "rzp_test_key", "rzp_test_secret", webhookSecret))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })
	return &count
}

// seedWalletOrder inserts a pending, wallet-at-checkout order (wallet_applied) whose
// chef has a Route account, plus a customer wallet pre-funded with the applied credit.
func seedWalletOrder(t *testing.T, db *gorm.DB, rzOrderID string, total, walletApplied float64) (order, cust uuid.UUID) {
	t.Helper()
	cust = payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET razorpay_account_id = ? WHERE id = ?`, "acc_chef", chef.String()).Error)
	order = payOrder(t, db, cust, chef, "pending", total, rzOrderID, "")
	require.NoError(t, db.Exec(`UPDATE orders SET wallet_applied = ? WHERE id = ?`, walletApplied, order.String()).Error)
	// Pre-fund the customer wallet with the applied credit so the debit succeeds.
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance, currency, created_at, updated_at)
		VALUES (?,?,?,?,datetime('now'),datetime('now'))`, uuid.NewString(), cust.String(), walletApplied, "INR").Error)
	return order, cust
}

func capturedPayload(rzOrderID, paymentID string, amountPaise int) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"payment": map[string]any{"entity": map[string]any{
			"id": paymentID, "order_id": rzOrderID, "amount": amountPaise, "method": "card", "status": "captured",
		}},
	})
	return b
}

func walletDebitCount(t *testing.T, db *gorm.DB, orderID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM wallet_txns WHERE idempotency_key = ?`,
		"wallet-debit:"+orderID.String()).Scan(&n).Error)
	return n
}

// A webhook-only completion settles the wallet: order completed, credit debited,
// chef top-up issued.
func TestHandlePaymentCaptured_SettlesWalletForWebhookOnlyCompletion(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	addProcessedEventsTable(t, db)
	transfers := withRazorpayTransferServer(t, "whsec_x")

	orderID, cust := seedWalletOrder(t, db, "order_wh1", 500, 100)

	require.NoError(t, NewPaymentHandler().handlePaymentCaptured(capturedPayload("order_wh1", "pay_wh1", 40000)))

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, "completed", status, "webhook completes the order")

	require.Equal(t, int64(1), walletDebitCount(t, db, orderID), "the applied store credit is debited (was skipped pre-fix)")
	require.Equal(t, 0.0, walletBalance(t, db, cust), "balance 100 − 100 applied = 0")
	require.GreaterOrEqual(t, *transfers, 1, "at least the chef top-up transfer is issued")
}

// Settling twice (webhook wins, then a retry / a later verify also settles) must NOT
// double-debit or double-transfer — the idempotency that makes verify+webhook coexist.
func TestHandlePaymentCaptured_WalletSettlementIdempotent(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	addProcessedEventsTable(t, db)
	transfers := withRazorpayTransferServer(t, "whsec_x")

	orderID, cust := seedWalletOrder(t, db, "order_wh2", 500, 100)
	payload := capturedPayload("order_wh2", "pay_wh2", 40000)

	require.NoError(t, NewPaymentHandler().handlePaymentCaptured(payload))
	first := *transfers
	require.NoError(t, NewPaymentHandler().handlePaymentCaptured(payload)) // retry / duplicate delivery

	require.Equal(t, int64(1), walletDebitCount(t, db, orderID), "debited exactly once across two settlements")
	require.Equal(t, 0.0, walletBalance(t, db, cust), "no double debit (balance not negative)")
	require.Equal(t, first, *transfers, "no additional top-up transfer on the second settlement")
}

// If the store-credit debit genuinely FAILS (balance drained between checkout and this
// delayed settlement), the chef/driver top-up must NOT be funded off credit the platform
// never collected — the order still completes, but the wallet slice is left for reconcile.
func TestHandlePaymentCaptured_DebitFailure_DoesNotFundTopUp(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	addProcessedEventsTable(t, db)
	transfers := withRazorpayTransferServer(t, "whsec_x")

	// Applied 100 but the wallet only holds 40 now → DebitWallet fails (insufficient).
	orderID, cust := seedWalletOrder(t, db, "order_wh3", 500, 100)
	require.NoError(t, db.Exec(`UPDATE wallets SET balance = 40 WHERE user_id = ?`, cust.String()).Error)

	require.NoError(t, NewPaymentHandler().handlePaymentCaptured(capturedPayload("order_wh3", "pay_wh3", 40000)))

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, "completed", status, "the gateway capture still completes the order")
	require.Equal(t, int64(0), walletDebitCount(t, db, orderID), "no debit landed (insufficient balance)")
	require.Equal(t, 40.0, walletBalance(t, db, cust), "balance untouched by the failed debit")
	require.Equal(t, 0, *transfers, "no top-up funded off an uncollected credit")
}
