package services

// easy_split_after_payment_test.go — #1091 / ADR-0003. The split moves off the
// create-order call and onto its own call, made after the release governor has
// decided. These pin the wire shape and the retry contract, because a split
// re-sent after an ambiguous failure is the difference between paying a chef
// once and paying them twice.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

type splitCall struct {
	Method  string
	Path    string
	Headers http.Header
	Body    map[string]any
}

func splitStub(t *testing.T, handler func(i int, w http.ResponseWriter)) (*CashfreeClient, *[]splitCall) {
	t.Helper()
	calls := &[]splitCall{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		call := splitCall{Method: r.Method, Path: r.URL.Path, Headers: r.Header.Clone()}
		_ = json.Unmarshal(raw, &call.Body)
		*calls = append(*calls, call)
		w.Header().Set("Content-Type", "application/json")
		handler(len(*calls)-1, w)
	}))
	t.Cleanup(srv.Close)
	return NewCashfreeTestClient(srv.URL, "TESTapp", "cf_test_secret", "wh", models.ChefModeTest), calls
}

func splitOK(w http.ResponseWriter) {
	_, _ = w.Write([]byte(`{"status":"OK","message":"Order split created"}`))
}

func TestSplitOrderAfterPayment_SendsTheVendorShareInRupees(t *testing.T) {
	cf, calls := splitStub(t, func(_ int, w http.ResponseWriter) { splitOK(w) })

	err := cf.SplitOrderAfterPayment("cf-order-1",
		[]CashfreeVendorSplit{{VendorID: "hc_vendor1", AmountPaise: CashfreeAmountFromPaise(38000)}},
		"idem-1")

	require.NoError(t, err)
	require.Len(t, *calls, 1)
	call := (*calls)[0]
	require.Equal(t, http.MethodPost, call.Method)
	require.Equal(t, "/easy-split/orders/cf-order-1/split", call.Path)
	require.Equal(t, "idem-1", call.Headers.Get("x-idempotency-key"),
		"without this a retry after a timeout can split twice")

	entries, _ := call.Body["split"].([]any)
	require.Len(t, entries, 1)
	entry, _ := entries[0].(map[string]any)
	require.Equal(t, "hc_vendor1", entry["vendor_id"])
	require.EqualValues(t, 380, entry["amount"], "Cashfree's wire format is rupee-decimal")

	// This order will never be split again, so the window can close now rather
	// than leaving the vendor balance provisional until it lapses.
	require.Equal(t, true, call.Body["disable_split"])
}

// Cashfree answers a repeat of a split it already applied with "transaction
// already processed". That is the idempotent success case: the chef has been
// paid, and reporting it as a failure would send the order down Rail B and pay
// them a second time.
func TestSplitOrderAfterPayment_TreatsAnAlreadyProcessedSplitAsDone(t *testing.T) {
	for _, code := range []int{http.StatusConflict, http.StatusBadRequest} {
		cf, _ := splitStub(t, func(_ int, w http.ResponseWriter) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"message":"Transaction already processed, not eligible for split","code":"api_request_failed"}`))
		})

		err := cf.SplitOrderAfterPayment("cf-order-1",
			[]CashfreeVendorSplit{{VendorID: "hc_v", AmountPaise: CashfreeAmountFromPaise(1000)}}, "idem-1")

		require.NoError(t, err, "status %d must read as already-split", code)
	}
}

// Cashfree needs a couple of minutes to sync a payment before it can be split.
// That is a "come back later", not a refusal, so the caller must retry rather
// than fall through to the other rail.
func TestSplitOrderAfterPayment_ReportsANotYetSyncedPaymentAsRetryable(t *testing.T) {
	cf, _ := splitStub(t, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Transaction is not synced, please retry after 2 mins","code":"invalid_request_error"}`))
	})

	err := cf.SplitOrderAfterPayment("cf-order-1",
		[]CashfreeVendorSplit{{VendorID: "hc_v", AmountPaise: CashfreeAmountFromPaise(1000)}}, "idem-1")

	require.ErrorIs(t, err, ErrEasySplitRetryable)
}

// A refusal Cashfree will give the same answer to forever must not be retried
// as though it were transient.
func TestSplitOrderAfterPayment_ReportsARealRefusalAsTerminal(t *testing.T) {
	cf, _ := splitStub(t, func(_ int, w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"vendor does not exist","code":"invalid_request_error"}`))
	})

	err := cf.SplitOrderAfterPayment("cf-order-1",
		[]CashfreeVendorSplit{{VendorID: "hc_v", AmountPaise: CashfreeAmountFromPaise(1000)}}, "idem-1")

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrEasySplitRetryable)
}

func TestSplitOrderAfterPayment_RefusesToCallWithNothingToSplit(t *testing.T) {
	cf, calls := splitStub(t, func(_ int, w http.ResponseWriter) { splitOK(w) })

	require.Error(t, cf.SplitOrderAfterPayment("cf-order-1", nil, "idem-1"))
	require.Empty(t, *calls, "an empty split would disable the window for nothing")
}
