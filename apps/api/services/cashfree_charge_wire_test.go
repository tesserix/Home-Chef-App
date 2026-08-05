package services

// cashfree_charge_wire_test.go — what CreateCashfreeCharge actually puts on the
// wire.
//
// A charge that sent a slightly different body from the à la carte order path
// was the difference between orders minting every day and an FSSAI filing
// failing with Cashfree's generic "HTTP 500: request_failed". The order path is
// the one proven against the live merchant account, so these pin the charge
// request to its shape rather than to what the helper happens to do.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// captureCashfreeOrder runs fn against a stub gateway and returns the decoded
// request body and headers it received.
func captureCashfreeOrder(t *testing.T, fn func()) (map[string]any, http.Header) {
	t.Helper()
	var body map[string]any
	var headers http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		headers = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order_id":"x","payment_session_id":"sess","order_status":"ACTIVE"}`))
	}))
	defer srv.Close()

	prev := GetCashfreeFor(models.ChefModeLive)
	SetCashfreeClient(NewCashfreeTestClient(srv.URL, "TESTapp", "cf_test_secret", "wh", models.ChefModeLive))
	t.Cleanup(func() { SetCashfreeClient(prev) })

	fn()
	return body, headers
}

// The header the order path never sends must not be sent here either. Cashfree
// dedups on order_id — which is the charge row's own UUID — and CreateOrder
// already reads the existing order back on 409, so the header bought nothing.
func TestCreateCashfreeCharge_SendsNoIdempotencyHeader(t *testing.T) {
	rowID := uuid.New()
	_, headers := captureCashfreeOrder(t, func() {
		_, _ = CreateCashfreeCharge(models.ChefModeLive, rowID, 177.00, "INR",
			CashfreeCustomerDetails{CustomerID: "abc", CustomerPhone: "9845012345"},
			map[string]string{"type": "fssai_filing"}, "Fe3dr FSSAI filing", "fssai")
	})
	require.Empty(t, headers.Get(cashfreeHeaderIdempotency),
		"the proven order path sends no idempotency key; the charge path must match it")
}

// order_note is likewise absent from the order path. Anything the working
// request does not carry, this one does not carry.
func TestCreateCashfreeCharge_MatchesTheOrderPathBody(t *testing.T) {
	rowID := uuid.New()
	body, _ := captureCashfreeOrder(t, func() {
		_, _ = CreateCashfreeCharge(models.ChefModeLive, rowID, 177.00, "INR",
			CashfreeCustomerDetails{CustomerID: "abc", CustomerPhone: "9845012345"},
			map[string]string{"type": "fssai_filing"}, "Fe3dr FSSAI filing", "fssai")
	})

	_, hasNote := body["order_note"]
	require.False(t, hasNote, "order_note is not on the order path, so it is not sent here")

	// The order id IS the idempotency mechanism — it must be the row's own UUID
	// so a capture can never be attributed to another row.
	require.Equal(t, rowID.String(), body["order_id"])
	require.Equal(t, "INR", body["order_currency"])
	// Cashfree's wire format is rupee-decimal, not paise.
	require.Equal(t, 177.00, body["order_amount"])
	require.NotNil(t, body["customer_details"])
}
