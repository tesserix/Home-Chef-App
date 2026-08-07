package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homechef/api/models"
)

// captureRefundBody stands a fake Cashfree up and returns the decoded body of
// the refund request the client actually sent.
func captureRefundBody(t *testing.T, order *models.Order, refundPaise int) map[string]any {
	t.Helper()

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode refund body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"refund_id":"r1","refund_status":"SUCCESS","refund_amount":1}`))
	}))
	defer srv.Close()

	c := NewCashfreeTestClient(srv.URL, "TESTapp", "cf_test_secret", "wh", models.ChefModeLive)
	if _, err := c.CreateRefund("cf-order-1", &CashfreeRefundRequest{
		AmountPaise:    CashfreeAmountFromPaise(refundPaise),
		IdempotencyKey: "refund:test:1",
		Splits:         BuildRefundSplits(order, 0, refundPaise),
	}); err != nil {
		t.Fatalf("CreateRefund: %v", err)
	}
	return body
}

func TestRefundWireCarriesTheVendorSplit(t *testing.T) {
	order := splitOrder(500, 38000)

	body := captureRefundBody(t, order, 50000)

	raw, ok := body["refund_splits"]
	if !ok {
		t.Fatalf("refund_splits absent from the request body: %v", body)
	}
	splits, ok := raw.([]any)
	if !ok || len(splits) != 1 {
		t.Fatalf("refund_splits = %v, want one entry", raw)
	}
	entry, _ := splits[0].(map[string]any)
	if entry["vendor_id"] != "hc_vendor1" {
		t.Errorf("vendor_id = %v, want hc_vendor1", entry["vendor_id"])
	}
	// Cashfree's wire format is rupee-decimal, like every other amount.
	if amount, _ := entry["amount"].(float64); amount != 380 {
		t.Errorf("amount = %v, want 380 rupees", entry["amount"])
	}
}

func TestRefundWireOmitsSplitsOnAFullCaptureOrder(t *testing.T) {
	order := splitOrder(500, 0)

	body := captureRefundBody(t, order, 50000)

	if _, present := body["refund_splits"]; present {
		t.Errorf("refund_splits must be absent for a full-capture order, got %v", body["refund_splits"])
	}
}
