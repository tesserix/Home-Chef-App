package services

// gateway_idempotency_test.go — #574. The Razorpay money-moving POSTs (CreateRefund,
// CreateTransfer) must carry a gateway idempotency header so a timeout-AFTER-success
// retry is deduped by Razorpay instead of issuing a SECOND real refund/transfer.
//
// Razorpay uses endpoint-specific headers with charset/length limits (verified against
// razorpay.com/docs): refunds → X-Refund-Idempotency (>=10 chars, [A-Za-z0-9-_], both
// Normal and Instant), direct transfers → X-Transfer-Idempotency (4-36 chars,
// [A-Za-z0-9-_ ]). Colons are illegal and a UUID alone is already 36 chars, so logical
// keys are normalized to a 32-char hex digest valid for BOTH endpoints.

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// headerRefundIdempotency / headerTransferIdempotency are defined in razorpay.go.

// razorpayCharset is the intersection of both endpoints' allowed characters that our
// normalized keys must satisfy (hex is a strict subset).
var razorpayKeyCharset = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func testClient(baseURL string) *RazorpayClient {
	return &RazorpayClient{keyID: "rzp_test", keySecret: "secret", baseURL: baseURL, fetchedAt: time.Now()}
}

// captureHeader spins up a server that records one request header and returns canned JSON.
func captureHeader(t *testing.T, headerName, respJSON string) (*RazorpayClient, *string, *int) {
	t.Helper()
	var seen string
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		seen = r.Header.Get(headerName)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respJSON))
	}))
	t.Cleanup(srv.Close)
	return testClient(srv.URL), &seen, &calls
}

func TestCreateRefund_SetsRefundIdempotencyHeader(t *testing.T) {
	c, seen, _ := captureHeader(t, headerRefundIdempotency, `{"id":"rfnd_1","status":"processed"}`)
	logical := RefundFullIdempotencyKey(uuid.New())
	_, err := c.CreateRefund("pay_123", &RefundRequest{Amount: 5000, Speed: "normal", IdempotencyKey: logical})
	require.NoError(t, err)
	require.NotEmpty(t, *seen, "X-Refund-Idempotency header must be sent when a key is provided")
	require.Equal(t, normalizeIdempotencyKey(logical), *seen, "header must be the normalized logical key")
	require.GreaterOrEqual(t, len(*seen), 10, "refund idempotency key must be >=10 chars")
	require.Regexp(t, razorpayKeyCharset, *seen, "key must be in Razorpay's allowed charset (no colons)")
}

func TestCreateRefund_NoHeaderWhenKeyEmpty(t *testing.T) {
	c, seen, _ := captureHeader(t, headerRefundIdempotency, `{"id":"rfnd_1","status":"processed"}`)
	_, err := c.CreateRefund("pay_123", &RefundRequest{Amount: 5000, Speed: "normal"})
	require.NoError(t, err)
	require.Empty(t, *seen, "no idempotency header when key is unset (backward compatible)")
}

func TestCreateTransfer_SetsTransferIdempotencyHeader(t *testing.T) {
	c, seen, _ := captureHeader(t, headerTransferIdempotency, `{"id":"trf_1","status":"created"}`)
	logical := TopupIdempotencyKey(uuid.New(), 0, "acc_abc123")
	_, err := c.CreateTransfer(&DirectTransferRequest{Account: "acc_abc123", Amount: 5000, Currency: "INR", IdempotencyKey: logical})
	require.NoError(t, err)
	require.NotEmpty(t, *seen, "X-Transfer-Idempotency header must be sent when a key is provided")
	require.Equal(t, normalizeIdempotencyKey(logical), *seen)
	require.GreaterOrEqual(t, len(*seen), 4, "transfer idempotency key must be >=4 chars")
	require.LessOrEqual(t, len(*seen), 36, "transfer idempotency key must be <=36 chars (Razorpay hard limit)")
	require.Regexp(t, razorpayKeyCharset, *seen)
}

func TestCreateTransfer_NoHeaderWhenKeyEmpty(t *testing.T) {
	c, seen, _ := captureHeader(t, headerTransferIdempotency, `{"id":"trf_1","status":"created"}`)
	_, err := c.CreateTransfer(&DirectTransferRequest{Account: "acc_abc", Amount: 5000, Currency: "INR"})
	require.NoError(t, err)
	require.Empty(t, *seen)
}

// A timeout-after-success retry must present the SAME key so Razorpay dedups it — the
// whole point of the change. Same logical operation → identical header on both calls.
func TestCreateRefund_RetrySendsSameKey(t *testing.T) {
	c, seen, calls := captureHeader(t, headerRefundIdempotency, `{"id":"rfnd_1","status":"processed"}`)
	orderID := uuid.New()
	first := RefundPartialIdempotencyKey(orderID, 2500)
	_, err := c.CreateRefund("pay_1", &RefundRequest{Amount: 2500, Speed: "normal", IdempotencyKey: first})
	require.NoError(t, err)
	firstSeen := *seen
	// Retry of the SAME logical refund instance (prior refunded unchanged).
	_, err = c.CreateRefund("pay_1", &RefundRequest{Amount: 2500, Speed: "normal", IdempotencyKey: RefundPartialIdempotencyKey(orderID, 2500)})
	require.NoError(t, err)
	require.Equal(t, 2, *calls)
	require.Equal(t, firstSeen, *seen, "retry of the same operation must send an identical idempotency key")
}

func TestNormalizeIdempotencyKey_ValidForBothEndpoints(t *testing.T) {
	// Even a long, colon-laden logical key normalizes to a charset+length-valid token.
	logical := "refund:" + uuid.New().String() + ":line:" + uuid.New().String()
	got := normalizeIdempotencyKey(logical)
	require.Regexp(t, razorpayKeyCharset, got)
	require.GreaterOrEqual(t, len(got), 10)
	require.LessOrEqual(t, len(got), 36)
	// Deterministic: same logical input → same normalized key (required for retry dedup).
	require.Equal(t, got, normalizeIdempotencyKey(logical))
	// Distinct logical inputs → distinct normalized keys.
	require.NotEqual(t, got, normalizeIdempotencyKey(logical+"x"))
}

func TestGatewayIdempotencyKeys_StableAndDistinct(t *testing.T) {
	o1, o2 := uuid.New(), uuid.New()
	line1, line2 := uuid.New(), uuid.New()

	// Deterministic: same inputs → same logical key.
	require.Equal(t, RefundFullIdempotencyKey(o1), RefundFullIdempotencyKey(o1))
	require.Equal(t, RefundLineIdempotencyKey(o1, line1), RefundLineIdempotencyKey(o1, line1))
	require.Equal(t, RefundPartialIdempotencyKey(o1, 100), RefundPartialIdempotencyKey(o1, 100))
	require.Equal(t, TopupIdempotencyKey(o1, 0, ")acc_x"), TopupIdempotencyKey(o1, 0, ")acc_x"))
	require.Equal(t, HoldPayoutIdempotencyKey("group", o1), HoldPayoutIdempotencyKey("group", o1))

	// Distinct across genuinely-different operations (the anti-#549 property: a
	// too-stable key would silently drop the second refund/transfer).
	require.NotEqual(t, RefundFullIdempotencyKey(o1), RefundFullIdempotencyKey(o2))
	require.NotEqual(t, RefundLineIdempotencyKey(o1, line1), RefundLineIdempotencyKey(o1, line2), "different lines of the same order must not collide")
	require.NotEqual(t, RefundPartialIdempotencyKey(o1, 100), RefundPartialIdempotencyKey(o1, 200), "sequential partials must not collide")
	require.NotEqual(t, TopupIdempotencyKey(o1, 0, ")acc_chef"), TopupIdempotencyKey(o1, 0, ")acc_driver"), "chef vs driver top-up must not collide")
	require.NotEqual(t, HoldPayoutIdempotencyKey("group", o1), HoldPayoutIdempotencyKey("mealplanday", o1))
	// A full-order refund and a line refund of the same order are distinct operations.
	require.NotEqual(t, RefundFullIdempotencyKey(o1), RefundLineIdempotencyKey(o1, line1))

	// And every normalized form is gateway-valid.
	for _, k := range []string{
		RefundFullIdempotencyKey(o1), RefundLineIdempotencyKey(o1, line1),
		RefundPartialIdempotencyKey(o1, 100), TopupIdempotencyKey(o1, 0, ")acc_x"),
		HoldPayoutIdempotencyKey("group", o1),
	} {
		n := normalizeIdempotencyKey(k)
		require.Regexp(t, razorpayKeyCharset, n)
		require.GreaterOrEqual(t, len(n), 10)
		require.LessOrEqual(t, len(n), 36)
	}
}
