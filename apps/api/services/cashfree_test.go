package services

// cashfree_test.go — the Cashfree adapter's money-critical edges.
//
// Every test here targets something that would move the wrong amount, refund
// twice, or accept a forged webhook if it regressed. The pure-arithmetic ones
// (amounts) matter most: they are the single conversion between the platform's
// integer paise and Cashfree's rupee decimals, and a rounding slip there is
// invisible until a settlement is short.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// withCashfreeClient installs c in a mode slot for the test and restores the
// previous occupant, so slot state never leaks between tests.
func withCashfreeClient(t *testing.T, mode string, c *CashfreeClient) {
	t.Helper()
	prev := snapshotCashfreeClient(mode)
	t.Cleanup(func() { SetCashfreeClientFor(mode, prev) })
	SetCashfreeClientFor(mode, c)
}

// withCashfreeServer points a mode slot at an httptest.Server.
func withCashfreeServer(t *testing.T, mode string, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	withCashfreeClient(t, mode, NewCashfreeTestClient(srv.URL, "app_test", "secret_test", "whsec_test", mode))
	return srv
}

// withCashfreeOrderPayments points the live-mode client at a stub serving Cashfree's
// order-scoped payments list — the one seam SuccessfulPayment reads. status "" means the
// order carries no payment at all, which is a different gateway shape from a payment that
// exists but never succeeded.
func withCashfreeOrderPayments(t *testing.T, orderID string, amountPaise int, status string) {
	t.Helper()
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/payments") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if status == "" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"cf_payment_id": 4242, "order_id": orderID,
			"payment_status": status, "payment_amount": float64(amountPaise) / 100,
		}})
	})
}

// cfRefundSpy records what reached Cashfree's order-scoped refund POST.
type cfRefundSpy struct {
	calls       int
	amountPaise int
	idemKey     string // the refund_id Cashfree is asked to mint
	failing     bool   // flip mid-test to let a retry-after-failure succeed
}

// withCashfreeRefundSpy points the live-mode client at a stub answering the refund
// POST with status — anything but 200 is a hard gateway failure, the shape the
// deferral and retry paths need.
func withCashfreeRefundSpy(t *testing.T, status int) *cfRefundSpy {
	t.Helper()
	spy := &cfRefundSpy{failing: status != http.StatusOK}
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/refunds") {
			w.WriteHeader(http.StatusOK)
			return
		}
		if spy.failing {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"gateway unavailable"}`))
			return
		}
		var body struct {
			RefundID string  `json:"refund_id"`
			Amount   float64 `json:"refund_amount"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		spy.calls++
		spy.amountPaise = int(math.Round(body.Amount * 100))
		spy.idemKey = body.RefundID
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"cf_refund_id":1,"refund_id":%q,"order_id":"o","refund_status":"SUCCESS","refund_amount":%.2f}`,
			body.RefundID, body.Amount)))
	})
	return spy
}

// --- Amounts: the paise ↔ rupee-decimal boundary ---

// The wire format must be an exact two-decimal rupee string built from integer
// paise. Formatting a float64 is how 140.25 becomes 140.25000000000001 and how a
// gateway rejects an order for an amount mismatch.
func TestCashfreeAmount_MarshalsExactRupeeDecimal(t *testing.T) {
	for _, tc := range []struct {
		paise int
		want  string
	}{
		{0, "0.00"},
		{1, "0.01"},
		{9, "0.09"},
		{10, "0.10"},
		{99, "0.99"},
		{100, "1.00"},
		{14025, "140.25"},
		{2900, "29.00"},
		{29, "0.29"}, // the IEEE-754 trap value from ToPaise's doc comment
		{123456789, "1234567.89"},
	} {
		got, err := json.Marshal(cashfreeAmount(tc.paise))
		require.NoError(t, err)
		require.Equal(t, tc.want, string(got), "%d paise", tc.paise)
	}
}

// A negative amount is a caller bug, but it must render as a readable negative
// rather than the "-1.-5" that naive integer division would produce — a
// malformed body is harder to diagnose than a wrong-but-legible one.
func TestCashfreeAmount_MarshalsNegativeLegibly(t *testing.T) {
	got, err := json.Marshal(cashfreeAmount(-105))
	require.NoError(t, err)
	require.Equal(t, "-1.05", string(got))
}

// Reading an amount back must land on the SAME paise value we sent, or a verify
// would reject its own order for an amount mismatch.
func TestCashfreeAmount_RoundTripsThroughTheWire(t *testing.T) {
	for _, paise := range []int{0, 1, 29, 99, 100, 14025, 999999} {
		wire, err := json.Marshal(cashfreeAmount(paise))
		require.NoError(t, err)
		var back cashfreeAmount
		require.NoError(t, json.Unmarshal(wire, &back))
		require.Equal(t, paise, back.Paise(), "round trip of %d paise", paise)
	}
}

// Cashfree sends some amounts as JSON strings rather than numbers, and rounding
// must match ToPaise so both gateway paths agree on the minor-unit value.
func TestCashfreeAmount_UnmarshalsStringsAndRounds(t *testing.T) {
	var a cashfreeAmount
	require.NoError(t, json.Unmarshal([]byte(`"140.25"`), &a))
	require.Equal(t, 14025, a.Paise())

	require.NoError(t, json.Unmarshal([]byte(`null`), &a))
	require.Equal(t, 0, a.Paise())

	// Third-decimal noise rounds the same way ToPaise does rather than truncating.
	require.NoError(t, json.Unmarshal([]byte(`0.289999999`), &a))
	require.Equal(t, ToPaise(0.29), a.Paise())
}

// The refund's own reported processing charge (#885, observability only) parses from the
// documented refund_charge field, and a body omitting it entirely still unmarshals cleanly as
// zero — Cashfree does not always charge/report a fee.
func TestCashfreeRefund_ParsesRefundCharge(t *testing.T) {
	var withCharge CashfreeRefund
	require.NoError(t, json.Unmarshal([]byte(`{"refund_amount": 100.00, "refund_charge": 5.90}`), &withCharge))
	require.Equal(t, 590, withCharge.RefundChargePaise.Paise())

	var withoutCharge CashfreeRefund
	require.NoError(t, json.Unmarshal([]byte(`{"refund_amount": 100.00}`), &withoutCharge))
	require.Equal(t, 0, withoutCharge.RefundChargePaise.Paise())
}

// --- Webhook signature ---

func cashfreeSign(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func nowTS() string { return strconv.FormatInt(time.Now().Unix(), 10) }

// A correctly signed webhook verifies and reports the slot that signed it.
func TestVerifyCashfreeWebhook_AcceptsGenuineSignature(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "live_secret", models.ChefModeLive))
	withCashfreeClient(t, models.ChefModeTest, nil)

	body := []byte(`{"type":"PAYMENT_SUCCESS_WEBHOOK"}`)
	ts := nowTS()
	ok, mode := VerifyCashfreeWebhookMode(ts, body, cashfreeSign("live_secret", ts, body))
	require.True(t, ok)
	require.Equal(t, models.ChefModeLive, mode)
}

// The TEST slot's secret must also be tried, and must report test — a
// test-signed event that resolved to live could mutate a live-partition record.
func TestVerifyCashfreeWebhook_ResolvesTestSlot(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "live_secret", models.ChefModeLive))
	withCashfreeClient(t, models.ChefModeTest,
		NewCashfreeTestClient("", "app_t", "sk_t", "test_secret", models.ChefModeTest))

	body := []byte(`{"type":"PAYMENT_SUCCESS_WEBHOOK"}`)
	ts := nowTS()
	ok, mode := VerifyCashfreeWebhookMode(ts, body, cashfreeSign("test_secret", ts, body))
	require.True(t, ok)
	require.Equal(t, models.ChefModeTest, mode)
}

// A forged signature is rejected by BOTH slots — the whole point of the endpoint.
func TestVerifyCashfreeWebhook_RejectsForgedSignature(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "live_secret", models.ChefModeLive))
	withCashfreeClient(t, models.ChefModeTest,
		NewCashfreeTestClient("", "app_t", "sk_t", "test_secret", models.ChefModeTest))

	body := []byte(`{"type":"PAYMENT_SUCCESS_WEBHOOK"}`)
	ts := nowTS()
	ok, _ := VerifyCashfreeWebhookMode(ts, body, cashfreeSign("attacker_secret", ts, body))
	require.False(t, ok)
}

// The signature covers timestamp+body, so a body swapped after signing must fail.
// This is the check that stops an attacker replaying a genuine signature over a
// payload of their own (e.g. a larger refund, or a different order).
func TestVerifyCashfreeWebhook_RejectsTamperedBody(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "live_secret", models.ChefModeLive))
	withCashfreeClient(t, models.ChefModeTest, nil)

	signed := []byte(`{"order_id":"a","amount":1}`)
	ts := nowTS()
	sig := cashfreeSign("live_secret", ts, signed)

	ok, _ := VerifyCashfreeWebhookMode(ts, []byte(`{"order_id":"a","amount":99999}`), sig)
	require.False(t, ok, "a tampered body must not verify against the original signature")
}

// A stale timestamp is rejected even when the signature itself is valid: without
// this a single captured delivery could be replayed verbatim forever.
func TestVerifyCashfreeWebhook_RejectsStaleTimestamp(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "live_secret", models.ChefModeLive))
	withCashfreeClient(t, models.ChefModeTest, nil)

	body := []byte(`{"type":"PAYMENT_SUCCESS_WEBHOOK"}`)
	stale := strconv.FormatInt(time.Now().Add(-2*cashfreeWebhookMaxSkew).Unix(), 10)
	ok, _ := VerifyCashfreeWebhookMode(stale, body, cashfreeSign("live_secret", stale, body))
	require.False(t, ok)
}

// An unparseable timestamp fails CLOSED. A caller cannot tell "verified" from
// "verified but with no replay protection", so the ambiguity must not verify.
func TestVerifyCashfreeWebhook_RejectsUnparseableTimestamp(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "live_secret", models.ChefModeLive))
	withCashfreeClient(t, models.ChefModeTest, nil)

	body := []byte(`{}`)
	ok, _ := VerifyCashfreeWebhookMode("not-a-timestamp", body, cashfreeSign("live_secret", "not-a-timestamp", body))
	require.False(t, ok)
}

// Missing headers are rejected rather than treated as "nothing to check".
func TestVerifyCashfreeWebhook_RejectsMissingHeaders(t *testing.T) {
	withCashfreeClient(t, models.ChefModeLive,
		NewCashfreeTestClient("", "app", "sk", "live_secret", models.ChefModeLive))

	ok, _ := VerifyCashfreeWebhookMode("", []byte(`{}`), "sig")
	require.False(t, ok)
	ok, _ = VerifyCashfreeWebhookMode(nowTS(), []byte(`{}`), "")
	require.False(t, ok)
}

// --- Environment selection ---

// The host follows the CREDENTIALS, not the slot — the same way the retired gateway's key
// prefix decides its environment.
//
// This is what lets the live slot run sandbox keys and actually work, instead of
// producing a valid client that 401s against production and silently sends every
// checkout to the fallback gateway.
func TestCashfreeBaseURL_FollowsCredentials(t *testing.T) {
	const testApp, testSecret = "TEST1234567890", "cfsk_ma_test_abc"
	const liveApp, liveSecret = "1234567890abcd", "cfsk_ma_prod_abc"

	// Live slot + sandbox credentials → sandbox. Harmless, and the normal state
	// while a merchant account is still in review.
	require.Equal(t, cashfreeTestBaseURL,
		cashfreeBaseURLFor(models.ChefModeLive, testApp, testSecret))

	// Live slot + real credentials → production.
	require.Equal(t, cashfreeLiveBaseURL,
		cashfreeBaseURLFor(models.ChefModeLive, liveApp, liveSecret))

	// THE SAFETY DIRECTION. The test slot is pinned to sandbox even when handed
	// production credentials — honouring them would let a sandbox order move real
	// money, which is the one outcome worth hard-coding against.
	require.Equal(t, cashfreeTestBaseURL,
		cashfreeBaseURLFor(models.ChefModeTest, liveApp, liveSecret))

	// An unrecognised mode is coerced to live by NormalizeMode, so real
	// credentials there still reach production.
	require.Equal(t, cashfreeLiveBaseURL,
		cashfreeBaseURLFor("nonsense", liveApp, liveSecret))

	require.Contains(t, cashfreeTestBaseURL, "sandbox.cashfree.com")
}

// Either half of a Cashfree credential pair identifies the sandbox, so a
// rotation that changes one format cannot silently flip an environment.
func TestCashfreeCredentialsAreSandbox_DetectsEitherHalf(t *testing.T) {
	require.True(t, cashfreeCredentialsAreSandbox("TEST1234", "cfsk_ma_prod_x"))
	require.True(t, cashfreeCredentialsAreSandbox("1234", "cfsk_ma_test_x"))
	require.True(t, cashfreeCredentialsAreSandbox("test1234", "")) // case-insensitive
	require.False(t, cashfreeCredentialsAreSandbox("1234abcd", "cfsk_ma_prod_x"))
	require.False(t, cashfreeCredentialsAreSandbox("", ""))
}

// Payouts follows the identical rule, including the pinned test slot.
func TestCashfreePayoutBaseURL_FollowsCredentials(t *testing.T) {
	require.Equal(t, cashfreePayoutTestBaseURL,
		cashfreePayoutBaseURLFor(models.ChefModeLive, "CF123", "cfsk_ma_test_x"))
	require.Equal(t, cashfreePayoutLiveBaseURL,
		cashfreePayoutBaseURLFor(models.ChefModeLive, "CF123", "cfsk_ma_prod_x"))
	require.Equal(t, cashfreePayoutTestBaseURL,
		cashfreePayoutBaseURLFor(models.ChefModeTest, "CF123", "cfsk_ma_prod_x"),
		"a sandbox disbursement must never be able to reach a real bank account")
}

// --- Refunds ---

// The refund_id must satisfy Cashfree's 3–40 alphanumeric window AND be the same
// digest the retired gateway header carries, so one logical refund has one identity on
// either gateway.
func TestCashfreeRefundID_IsDeterministicAndWithinLimits(t *testing.T) {
	logical := "refund:11111111-2222-3333-4444-555555555555:full"
	id := cashfreeRefundID(logical)

	require.Len(t, id, 32)
	require.GreaterOrEqual(t, len(id), 3)
	require.LessOrEqual(t, len(id), 40)
	for _, r := range id {
		require.True(t, (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'), "non-alphanumeric %q in refund_id", r)
	}
	require.Equal(t, id, cashfreeRefundID(logical), "same logical key ⇒ same refund_id")
	require.Equal(t, normalizeIdempotencyKey(logical), id, "must match the platform-wide idempotency digest")
	require.NotEqual(t, id, cashfreeRefundID(logical+":other"), "different operations ⇒ different ids")
}

// A refund with no idempotency key must be REFUSED rather than sent without one:
// letting Cashfree mint the id removes the dedup that stops a retry becoming a
// second real refund.
func TestCashfreeCreateRefund_RequiresAnIdempotencyKey(t *testing.T) {
	c := NewCashfreeTestClient("http://unused", "app", "sk", "wh", models.ChefModeTest)
	_, err := c.CreateRefund("order-1", &CashfreeRefundRequest{AmountPaise: 100})
	require.Error(t, err)
	require.Contains(t, err.Error(), "idempotency key")
}

// A duplicate refund_id comes back as 409. That means the refund already exists,
// so it must be FETCHED and returned as success — reporting an error would send
// the caller round again and risk a second refund.
func TestCashfreeCreateRefund_409ReusesTheExistingRefund(t *testing.T) {
	var posts, gets int
	srv := withCashfreeServer(t, models.ChefModeTest, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			posts++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"refund_already_exists","message":"duplicate refund_id"}`))
		case http.MethodGet:
			gets++
			_, _ = w.Write([]byte(`{"cf_refund_id":8811,"refund_id":"abc","order_id":"order-1",
				"refund_status":"SUCCESS","refund_amount":12.50}`))
		}
	})
	require.NotEmpty(t, srv.URL)

	c := GetCashfreeFor(models.ChefModeTest)
	res, err := c.CreateRefund("order-1", &CashfreeRefundRequest{
		AmountPaise:    1250,
		IdempotencyKey: "refund:x:full",
	})
	require.NoError(t, err, "a duplicate refund_id is idempotent success, not a failure")
	require.Equal(t, 1, posts)
	require.Equal(t, 1, gets)
	require.Equal(t, CashfreeRefundSuccess, res.RefundStatus)
	require.Equal(t, 1250, res.AmountPaise.Paise())
}

// The refund_id we send must be the normalized digest of the logical key, and the
// amount must be the rupee decimal — asserted against the real request body.
func TestCashfreeCreateRefund_SendsNormalizedIDAndRupeeAmount(t *testing.T) {
	var gotBody map[string]any
	var gotPath string
	withCashfreeServer(t, models.ChefModeTest, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_, _ = w.Write([]byte(`{"cf_refund_id":1,"refund_id":"r","order_id":"o","refund_status":"PENDING","refund_amount":140.25}`))
	})

	c := GetCashfreeFor(models.ChefModeTest)
	res, err := c.CreateRefund("order-9", &CashfreeRefundRequest{
		AmountPaise:    14025,
		IdempotencyKey: "refund:abc:full",
	})
	require.NoError(t, err)

	require.Equal(t, "/orders/order-9/refunds", gotPath, "Cashfree refunds are ORDER-scoped")
	require.Equal(t, normalizeIdempotencyKey("refund:abc:full"), gotBody["refund_id"])
	require.Equal(t, 140.25, gotBody["refund_amount"], "amount goes out as a rupee decimal")
	require.Equal(t, "STANDARD", gotBody["refund_speed"], "defaults to STANDARD, the ordinary bank-rail speed")
	require.Equal(t, "pending", PlatformRefundStatus(res.RefundStatus))
}

// ONHOLD must map to pending, NOT failed. Calling a held refund failed would make
// the reconcile cron re-issue a refund that is already in flight.
func TestPlatformRefundStatus_MapsOntoTheSharedVocabulary(t *testing.T) {
	require.Equal(t, "processed", PlatformRefundStatus(CashfreeRefundSuccess))
	require.Equal(t, "pending", PlatformRefundStatus(CashfreeRefundPending))
	require.Equal(t, "pending", PlatformRefundStatus(CashfreeRefundOnHold))
	require.Equal(t, "failed", PlatformRefundStatus(CashfreeRefundFailed))
	require.Equal(t, "failed", PlatformRefundStatus(CashfreeRefundCancelled))
}

// The reconciliation basis must count only SETTLED refunds. Counting PENDING
// would overstate what reached the customer and flag a phantom full refund;
// counting FAILED would count money that never moved.
func TestOrderRefundedPaise_CountsOnlySuccessfulRefunds(t *testing.T) {
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"refund_id":"a","refund_status":"SUCCESS","refund_amount":100.00},
			{"refund_id":"b","refund_status":"PENDING","refund_amount":50.00},
			{"refund_id":"c","refund_status":"FAILED","refund_amount":25.00},
			{"refund_id":"d","refund_status":"ONHOLD","refund_amount":10.00},
			{"refund_id":"e","refund_status":"SUCCESS","refund_amount":0.50}
		]`))
	})

	total, err := GetCashfreeFor(models.ChefModeLive).OrderRefundedPaise("order-1")
	require.NoError(t, err)
	require.Equal(t, 10050, total, "only the two SUCCESS refunds (₹100.00 + ₹0.50)")
}

// --- Payments ---

// SuccessfulPayment must pick the CAPTURED attempt, not the first element. A
// customer who fails once and retries leaves a FAILED attempt ahead of the real
// one, and settling on the first would read the wrong amount and id.
func TestSuccessfulPayment_SkipsFailedAttempts(t *testing.T) {
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"cf_payment_id":111,"order_id":"o1","payment_status":"FAILED","payment_amount":140.25,"payment_group":"card"},
			{"cf_payment_id":222,"order_id":"o1","payment_status":"USER_DROPPED","payment_amount":140.25,"payment_group":"upi"},
			{"cf_payment_id":333,"order_id":"o1","payment_status":"SUCCESS","payment_amount":140.25,"payment_group":"upi"}
		]`))
	})

	p, err := GetCashfreeFor(models.ChefModeLive).SuccessfulPayment("o1")
	require.NoError(t, err)
	require.NotNil(t, p)
	require.Equal(t, "333", p.CFPaymentID.String())
	require.Equal(t, 14025, p.AmountPaise.Paise())
	require.True(t, p.IsCaptured())
}

// No captured payment ⇒ (nil, nil), which the verify path reads as "not paid".
// An error here would surface as a 500 instead of an honest "not completed".
func TestSuccessfulPayment_NilWhenNothingCaptured(t *testing.T) {
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"cf_payment_id":1,"payment_status":"FAILED","payment_amount":1.00}]`))
	})
	p, err := GetCashfreeFor(models.ChefModeLive).SuccessfulPayment("o1")
	require.NoError(t, err)
	require.Nil(t, p)
}

// An order with no attempts yet 404s at Cashfree; that is "not paid", not an error.
func TestFetchOrderPayments_404IsAnEmptyList(t *testing.T) {
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"order_not_found","message":"no such order"}`))
	})
	payments, err := GetCashfreeFor(models.ChefModeLive).FetchOrderPayments("o1")
	require.NoError(t, err)
	require.Empty(t, payments)
}

// payment_group is an instrument label; it must map onto the same short
// vocabulary the platform already stores so a receipt reads identically
// regardless of gateway. An unknown group passes through rather than vanishing
// into an "other" bucket.
func TestMethodLabel_MapsOntoThePlatformVocabulary(t *testing.T) {
	for group, want := range map[string]string{
		"upi": "upi", "credit_card": "card", "debit_card": "card",
		"net_banking": "netbanking", "wallet": "wallet", "pay_later": "paylater",
		"something_new": "something_new",
	} {
		p := &CashfreePayment{PaymentGroup: group}
		require.Equal(t, want, p.MethodLabel(), "group %q", group)
	}
}

// --- Orders ---

// Auth is a header pair (not HTTP Basic) plus a pinned API version. Getting any of
// the three wrong is a 401 at checkout.
func TestCashfreeRequest_SendsHeaderAuthAndPinnedVersion(t *testing.T) {
	var h http.Header
	withCashfreeServer(t, models.ChefModeTest, func(w http.ResponseWriter, r *http.Request) {
		h = r.Header.Clone()
		_, _ = w.Write([]byte(`{"order_id":"o","payment_session_id":"sess","order_status":"ACTIVE","order_amount":1.00}`))
	})

	_, err := GetCashfreeFor(models.ChefModeTest).FetchOrder("o")
	require.NoError(t, err)
	require.Equal(t, "app_test", h.Get("x-client-id"))
	require.Equal(t, "secret_test", h.Get("x-client-secret"))
	require.Equal(t, cashfreeAPIVersion, h.Get("x-api-version"))
	require.Empty(t, h.Get("Authorization"), "Cashfree does not use HTTP Basic")
}

// A duplicate order_id (409) means a previous attempt already created this order.
// It must be fetched and reused — minting a second id for the same purchase is how
// a payment arrives for an order id nothing recognises.
func TestCashfreeCreateOrder_409ReusesTheExistingOrder(t *testing.T) {
	var posts, gets int
	withCashfreeServer(t, models.ChefModeTest, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts++
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"code":"order_already_exists","message":"duplicate"}`))
			return
		}
		gets++
		_, _ = w.Write([]byte(`{"cf_order_id":77,"order_id":"ord-1","payment_session_id":"sess_live",
			"order_status":"ACTIVE","order_amount":140.25}`))
	})

	res, err := GetCashfreeFor(models.ChefModeTest).CreateOrder(&CashfreeOrderRequest{
		OrderID:     "ord-1",
		AmountPaise: 14025,
		Customer:    CashfreeCustomerDetails{CustomerID: "c1", CustomerPhone: "9999999999"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, posts)
	require.Equal(t, 1, gets)
	require.Equal(t, "sess_live", res.PaymentSessionID)
	require.True(t, res.IsPayable())
	require.Equal(t, 14025, res.AmountPaise.Paise())
}

// The order body must carry the rupee-decimal amount and default to INR.
func TestCashfreeCreateOrder_SendsRupeeAmountAndDefaultsCurrency(t *testing.T) {
	var body map[string]any
	withCashfreeServer(t, models.ChefModeTest, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_, _ = w.Write([]byte(`{"order_id":"o","payment_session_id":"s","order_status":"ACTIVE","order_amount":140.25}`))
	})

	_, err := GetCashfreeFor(models.ChefModeTest).CreateOrder(&CashfreeOrderRequest{
		OrderID:     "o",
		AmountPaise: 14025,
		Customer:    CashfreeCustomerDetails{CustomerID: "c1", CustomerPhone: "9999999999"},
	})
	require.NoError(t, err)
	require.Equal(t, 140.25, body["order_amount"])
	require.Equal(t, "INR", body["order_currency"])
	// order_meta must be absent — no return_url (the server-side fetch is the
	// authority) and no notify_url (webhooks go to the one signed endpoint).
	require.NotContains(t, body, "order_meta")
}

// Only ACTIVE + a session id is payable. Handing the client a token for an
// EXPIRED/TERMINATED/PAID order sends them into a checkout that cannot complete.
func TestCashfreeOrder_IsPayableOnlyWhenActiveWithASession(t *testing.T) {
	require.True(t, (&CashfreeOrderResponse{OrderStatus: CashfreeOrderActive, PaymentSessionID: "s"}).IsPayable())
	require.False(t, (&CashfreeOrderResponse{OrderStatus: CashfreeOrderActive}).IsPayable())
	require.False(t, (&CashfreeOrderResponse{OrderStatus: CashfreeOrderPaid, PaymentSessionID: "s"}).IsPayable())
	require.False(t, (&CashfreeOrderResponse{OrderStatus: CashfreeOrderExpired, PaymentSessionID: "s"}).IsPayable())
	require.False(t, (*CashfreeOrderResponse)(nil).IsPayable())
}

// --- Health check + error hygiene ---

// A 404 on the sentinel order proves auth worked; only 401/403 means bad keys.
// Reporting the 404 as unhealthy would tell every admin their working keys are broken.
func TestCashfreeHealthCheck_404IsHealthy401IsNot(t *testing.T) {
	status := http.StatusNotFound
	withCashfreeServer(t, models.ChefModeTest, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"code":"order_not_found","message":"nope"}`))
	})

	c := GetCashfreeFor(models.ChefModeTest)
	require.NoError(t, c.HealthCheck(), "404 on a nonexistent order means auth succeeded")

	status = http.StatusUnauthorized
	require.Error(t, c.HealthCheck())
}

// Gateway errors must surface only Cashfree's structured code/message, never the
// raw body — validation errors there can echo a submitted phone or bank detail
// straight into a log line or Sentry event.
func TestCashfreeError_LeaksOnlyStructuredFields(t *testing.T) {
	raw := []byte(`{"code":"invalid_request","message":"phone invalid","type":"validation",
		"raw_echo":"customer_phone=9812345678"}`)
	err := cashfreeError(http.StatusBadRequest, raw)
	require.Contains(t, err.Error(), "invalid_request")
	require.Contains(t, err.Error(), "phone invalid")
	require.NotContains(t, err.Error(), "9812345678", "a rejected value must never ride the error message")
}

// An unparseable error body still produces a usable error and still leaks nothing.
func TestCashfreeError_FallsBackToStatusOnly(t *testing.T) {
	err := cashfreeError(http.StatusBadGateway, []byte(`<html>gateway down 9812345678</html>`))
	require.Contains(t, err.Error(), "502")
	require.NotContains(t, err.Error(), "9812345678")
}

// --- Credential slots ---

// The two slots must be cached and invalidated INDEPENDENTLY: saving test keys
// must never knock a healthy live gateway offline.
func TestInvalidateCashfreeFor_IsScopedToOneSlot(t *testing.T) {
	live := NewCashfreeTestClient("", "live_app", "live_sk", "live_wh", models.ChefModeLive)
	test := NewCashfreeTestClient("", "test_app", "test_sk", "test_wh", models.ChefModeTest)
	withCashfreeClient(t, models.ChefModeLive, live)
	withCashfreeClient(t, models.ChefModeTest, test)

	InvalidateCashfreeFor(models.ChefModeTest)

	require.NotNil(t, snapshotCashfreeClient(models.ChefModeLive), "live slot must survive a test-slot invalidation")
	require.Nil(t, snapshotCashfreeClient(models.ChefModeTest))
}

// Secret names must be per-slot and product-scoped, and the exported form must
// agree with the internal one — a drift there is what once made admin-entered
// the retired gateway keys silently invisible to the app.
func TestCashfreeSecretNames_ArePerSlotAndConsistent(t *testing.T) {
	liveID, liveSecret, liveWH := CashfreeSecretNames(models.ChefModeLive)
	testID, testSecret, testWH := CashfreeSecretNames(models.ChefModeTest)

	require.Equal(t, SecretCashfreeAppID, liveID)
	require.Equal(t, SecretCashfreeSecretKey, liveSecret)
	require.Equal(t, SecretCashfreeWebhookSecret, liveWH)
	require.Equal(t, SecretCashfreeTestAppID, testID)
	require.Equal(t, SecretCashfreeTestSecretKey, testSecret)
	require.Equal(t, SecretCashfreeTestWebhookSecret, testWH)

	for _, pair := range [][2]string{{liveID, testID}, {liveSecret, testSecret}, {liveWH, testWH}} {
		require.NotEqual(t, pair[0], pair[1], "slots must not share a secret name")
	}
	for _, name := range []string{liveID, liveSecret, liveWH, testID, testSecret, testWH} {
		require.True(t, strings.HasPrefix(name, "prod-homechef-cashfree-"),
			"secret %q must be product-scoped so it cannot collide with another product", name)
	}
}

// An unconfigured slot returns nil, which every caller already handles — that is
// the contract the retired gateway accessor has and the Cashfree paths rely on it.
func TestGetCashfreeFor_NilWhenSlotUnconfigured(t *testing.T) {
	withCashfreeClient(t, models.ChefModeTest, nil)
	// No Secret Manager and no CASHFREE_* env in the test process, so the fetch
	// fails and there is no cached client to fall back to.
	require.Nil(t, GetCashfreeFor(models.ChefModeTest))
}

// A live-slot client reports production; a test-slot one reports sandbox. This is
// what the admin surface renders, and the alarm an operator needs if the two ever
// disagree with the slot name.
func TestIsSandbox_DerivesFromTheResolvedHost(t *testing.T) {
	require.True(t, (&CashfreeClient{mode: models.ChefModeTest}).IsSandbox())
	require.False(t, (&CashfreeClient{mode: models.ChefModeLive}).IsSandbox())
	require.False(t, NewCashfreeTestClient("http://127.0.0.1:1", "a", "b", "c", models.ChefModeTest).IsSandbox(),
		"an injected host is reported honestly, not as the mode implies")
}

// A guard so the retry-suffix helper stays inside Cashfree's 45-character
// order_id limit even after several retries.
func TestCashfreeOrderIDLimits(t *testing.T) {
	// A UUID is 36 chars; "-r10" takes it to 40, still under 45.
	base := "11111111-2222-3333-4444-555555555555"
	require.Len(t, base, 36)
	require.LessOrEqual(t, len(fmt.Sprintf("%s-r%d", base, 10)), 45)
}
