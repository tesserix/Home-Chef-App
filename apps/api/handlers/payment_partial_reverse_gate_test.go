package handlers

// payment_partial_reverse_gate_test.go — #568. A PARTIAL refund's chef-transfer
// claw-back (#549) is NOT re-fire idempotent (it only caps at the transfer total,
// not per refund instance). So it must run ONLY when the refund actually persisted:
// if the persist tx fails (best-effort, swallowed) and the client retries, an
// unconditional claw would reverse the chef twice for one customer refund. The FULL
// path stays unconditional (its cross-guard is idempotent AND the safety net that
// blocks the hold even when the persist that stamps refunded_at failed).

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"encoding/json"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
	"github.com/homechef/api/services"
)

// withPartialReverseGateway points GetRazorpay at an httptest server answering
// FetchOrderTransfers (one chef transfer) + reversals, turns the order-payout flag
// ON, and returns a pointer to the reversal-POST counter.
func withPartialReverseGateway(t *testing.T) *int {
	t.Helper()
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transfers"):
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{
				{"id": "trf_chef", "recipient": "acc_chef", "amount": 90000},
			}})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/reversals"):
			count++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "rvsl_1"})
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "rzp_test_key", "rzp_test_secret", ""))
	prev := config.AppConfig
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: true}
	t.Cleanup(func() {
		config.AppConfig = prev
		services.SetRazorpayClient(nil)
	})
	return &count
}

// A partial refund whose persist tx FAILS must NOT claw the chef transfer — else a
// retry (the persist failure is swallowed + returns 200) double-claws for one refund.
func TestInitiateRefund_PartialClaw_SkippedWhenPersistFails(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	reversals := withPartialReverseGateway(t)

	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET razorpay_account_id = ? WHERE id = ?`, "acc_chef", chef.String()).Error)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "order_rev1", "pay_x")
	// Force the persist transaction to fail: EnqueueEvent needs outbox_events.
	require.NoError(t, db.Exec(`DROP TABLE outbox_events`).Error)

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial goodwill", "amount": 100.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String()) // credit committed; persist failure is best-effort

	require.Equal(t, 0, *reversals, "no chef claw when the refund persist failed — a retry would otherwise double-claw (#568)")
}

// The counterpart: a partial refund that DOES persist claws the chef exactly once.
func TestInitiateRefund_PartialClaw_FiresOnceOnPersistSuccess(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	reversals := withPartialReverseGateway(t)

	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET razorpay_account_id = ? WHERE id = ?`, "acc_chef", chef.String()).Error)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "order_rev2", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial goodwill", "amount": 100.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 1, *reversals, "a persisted partial refund claws the chef once")
}
