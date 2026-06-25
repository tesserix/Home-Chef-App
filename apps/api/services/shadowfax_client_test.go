package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
)

func newTestShadowfax(srv *httptest.Server) *shadowfaxClient {
	return newShadowfaxClient(&models.DeliveryProvider{APIBaseURL: srv.URL, APIKey: "tok123"})
}

func TestShadowfaxCreateTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/clients/orders/" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Token tok123" {
			t.Fatalf("auth header = %q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["order_type"] != "marketplace" {
			t.Fatalf("order_type = %v", body["order_type"])
		}
		w.Write([]byte(`{"message":"Success","data":{"awb_number":"SF999TST","customer_track_url":"https://exp.shadowfax.in/abc"}}`))
	}))
	defer srv.Close()

	resp, err := newTestShadowfax(srv).CreateTask(context.Background(), ProviderDeliveryRequest{
		OrderID: uuid.New(), ClientOrderID: "HC-1", OrderValue: 250,
		PickupName: "Chef", PickupPhone: "9000000000", PickupPincode: "560016",
		CustomerName: "Cust", CustomerPhone: "9111111111", DropoffPincode: "560017",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ExternalDeliveryID != "SF999TST" || resp.TrackingURL != "https://exp.shadowfax.in/abc" {
		t.Fatalf("got %+v", resp)
	}
}

func TestShadowfaxCancelTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/clients/orders/cancel/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["request_id"] != "HC-1" {
			t.Fatalf("request_id = %v", body["request_id"])
		}
		w.Write([]byte(`{"responseMsg":"Request has been marked as cancelled","responseCode":200}`))
	}))
	defer srv.Close()
	if err := newTestShadowfax(srv).CancelTask(context.Background(), "HC-1", "customer cancelled"); err != nil {
		t.Fatal(err)
	}
}

func TestShadowfaxGetQuoteServiceability(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/v1/clients/serviceability/") {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"data":[{"pincode":560017}]}`))
	}))
	defer srv.Close()
	q, err := newTestShadowfax(srv).GetQuote(context.Background(), QuoteRequest{DropoffPincode: "560017", City: "Bengaluru"})
	if err != nil || !q.Serviceable || q.Fee != 0 {
		t.Fatalf("quote = %+v err=%v (want serviceable, fee 0)", q, err)
	}
	// Empty drop pincode → not serviceable, no call needed.
	if q2, _ := newTestShadowfax(srv).GetQuote(context.Background(), QuoteRequest{}); q2.Serviceable {
		t.Fatal("empty pincode must be non-serviceable")
	}
}

func TestShadowfaxTrackTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v4/clients/orders/SF999TST/track/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Write([]byte(`{"message":"Success","order_details":{"status":"ofd","customer_track_url":"https://exp.shadowfax.in/xyz"}}`))
	}))
	defer srv.Close()
	tr, err := newTestShadowfax(srv).TrackTask(context.Background(), "SF999TST")
	if err != nil || tr.ProviderStatus != "ofd" || tr.TrackingURL != "https://exp.shadowfax.in/xyz" {
		t.Fatalf("track = %+v err=%v", tr, err)
	}
}
