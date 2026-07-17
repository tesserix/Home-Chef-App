package services

// orderrefund_gateway_test.go — #690. The adapter that routes a coordinator refund to the
// right provider and splits the wallet-funded slice off the captured one.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
)

// razorpayRefundSpy captures what actually reached Razorpay.
type razorpayRefundSpy struct {
	amountPaise int
	idemKey     string
	calls       int
}

func newRazorpaySpy(t *testing.T) *razorpayRefundSpy {
	t.Helper()
	spy := &razorpayRefundSpy{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Amount int `json:"amount"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		spy.calls++
		spy.amountPaise = body.Amount
		spy.idemKey = r.Header.Get("X-Refund-Idempotency")
		_, _ = w.Write([]byte(`{"id":"rfnd_spy","status":"processed"}`))
	}))
	t.Cleanup(srv.Close)
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })
	return spy
}

// The coordinator gives every logical refund its OWN key (refund:<order>:<scope>) — that is
// how two different refunds on one order stay distinct at the gateway. The adapter must pass
// it through.
//
// It did not, at first. runCancellationGatewayRefund HARDCODED RefundFullIdempotencyKey,
// correct for its own caller (one full cancellation refund per order) and silently wrong the
// moment the coordinator drives it: an issue refund and a cancellation refund would arrive
// under the SAME key, Razorpay would dedup the second as a retry of the first, and the
// customer would simply never receive it. No error, no trace — just missing money.
func TestOrderRefundGateway_PassesTheCoordinatorsPerScopeKey_NotTheFullRefundKey(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 0)
	spy := newRazorpaySpy(t)

	key := "refund:" + o.ID.String() + ":issue:abc"
	_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
		OrderID: o.ID, Amount: 100, IdempotencyKey: key,
		Reason: "damaged", Actor: "admin", ScopeID: "issue:abc",
	})
	require.NoError(t, err)

	require.Equal(t, 1, spy.calls)
	require.NotEqual(t, normalizeIdempotencyKey(RefundFullIdempotencyKey(o.ID)), spy.idemKey,
		"the adapter must NOT collapse every scope onto the full-refund key — that silently "+
			"dedups a legitimate second refund into nothing")
	require.Equal(t, normalizeIdempotencyKey(key), spy.idemKey,
		"the gateway sees the coordinator's per-scope key (hashed by the client, #574)")
}

// #141: the provider can only refund what it CAPTURED. The rest is store credit. The
// coordinator hands over the full amount owed and must stay ignorant of this split.
func TestOrderRefundGateway_WalletFundedOrder_SplitsStoreCreditFromTheCapturedSlice(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 120) // ₹120 of ₹300 came from the wallet
	spy := newRazorpaySpy(t)

	_, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
		OrderID: o.ID, Amount: 300, IdempotencyKey: "refund:" + o.ID.String() + ":cancel",
		Reason: "cancel", Actor: "customer", ScopeID: "cancel",
	})
	require.NoError(t, err)

	require.Equal(t, 18000, spy.amountPaise, "only the captured ₹180 can go back to the card")
	var bal float64
	db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal)
	require.Equal(t, 120.0, bal, "the wallet-funded ₹120 returns as store credit — ₹300 conserved")
}

// A wallet-PAID order has no provider payment at all. Routing it to Razorpay is the shape of
// the #691 bug in reverse, and the adapter exists to make it impossible.
func TestOrderRefundGateway_WalletPaidOrder_NeverTouchesTheProvider(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 200, 0)
	spy := newRazorpaySpy(t)

	ref, err := OrderRefundGateway{}.RefundPayment(context.Background(), orderrefund.GatewayRequest{
		OrderID: o.ID, Amount: 200, IdempotencyKey: "refund:" + o.ID.String() + ":cancel",
		Reason: "cancel", Actor: "customer", ScopeID: "cancel",
	})
	require.NoError(t, err)

	require.Zero(t, spy.calls, "a wallet-paid order has no card to refund")
	require.Contains(t, ref, "wallet:", "the reference records where the money actually went")
	var bal float64
	db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal)
	require.Equal(t, 200.0, bal)
}
