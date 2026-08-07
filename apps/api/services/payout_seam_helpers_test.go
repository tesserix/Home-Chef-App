package services

// payout_seam_helpers_test.go — shared harness for the payout/advance seams: a
// gateway pointed at an httptest.Server and the order-payout flag.
//
// It was payout_audit_test.go (#397), covering the append-only audit row every real
// Route transfer hold/release/reverse wrote. #1086 removed those transfers, so the
// audit module had no callers and went with them; the money trail for what remains
// is the Cashfree split stamp on the order plus the weekly statement.

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// withRazorpayClient swaps the live-slot client under the cache mutex and restores it.
func withRazorpayClient(t *testing.T, c *RazorpayClient) {
	t.Helper()
	razorpayMu.Lock()
	prev := razorpayClients[models.ChefModeLive]
	razorpayClients[models.ChefModeLive] = c
	razorpayMu.Unlock()
	t.Cleanup(func() {
		razorpayMu.Lock()
		razorpayClients[models.ChefModeLive] = prev
		razorpayMu.Unlock()
	})
}

// withRazorpayTestServer points GetRazorpay() at an httptest.Server so the advance
// capture/refund calls execute against canned responses. Restores the previous client.
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
