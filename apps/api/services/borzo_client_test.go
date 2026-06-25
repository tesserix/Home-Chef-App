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

func newTestBorzo(srv *httptest.Server) *borzoClient {
	return newBorzoClient(&models.DeliveryProvider{APIBaseURL: srv.URL, APIKey: "tok-abc"})
}

func TestBorzoGetQuote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calculate-order" || r.Method != http.MethodPost {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-DV-Auth-Token"); got != "tok-abc" {
			t.Fatalf("auth header = %q", got)
		}
		w.Write([]byte(`{"is_successful":true,"order":{"payment_amount":"73.50","delivery_fee_amount":"73.50"}}`))
	}))
	defer srv.Close()
	q, err := newTestBorzo(srv).GetQuote(context.Background(), QuoteRequest{PickupLat: 12.97, DropoffLat: 12.98, Weight: 1})
	if err != nil || !q.Serviceable || q.Fee != 73.5 {
		t.Fatalf("quote = %+v err=%v (want serviceable, fee 73.5)", q, err)
	}
}

func TestBorzoCreateTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/create-order" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["vehicle_type_id"] == nil {
			t.Fatal("missing vehicle_type_id")
		}
		w.Write([]byte(`{"order":{"order_id":99001,"order_name":"AB-1","payment_amount":"73.50","points":[{"tracking_url":"https://x/pickup"},{"tracking_url":"https://borzo.track/abc"}]}}`))
	}))
	defer srv.Close()
	resp, err := newTestBorzo(srv).CreateTask(context.Background(), ProviderDeliveryRequest{
		OrderID: uuid.New(), PickupName: "Chef", PickupPhone: "9000000000",
		CustomerName: "Cust", CustomerPhone: "9111111111", ItemDescription: "Order HC-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ExternalDeliveryID != "99001" || resp.TrackingURL != "https://borzo.track/abc" || resp.Cost != 73.5 {
		t.Fatalf("got %+v", resp)
	}
}

func TestBorzoCancelTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cancel-order" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		// numeric order id should round-trip as a JSON number
		if body["order_id"] == nil {
			t.Fatal("missing order_id")
		}
		w.Write([]byte(`{"is_successful":true}`))
	}))
	defer srv.Close()
	if err := newTestBorzo(srv).CancelTask(context.Background(), "99001", "customer cancelled"); err != nil {
		t.Fatal(err)
	}
}

func TestBorzoTrackTask(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/orders") {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("order_id") != "99001" {
			t.Fatalf("order_id query = %q", r.URL.Query().Get("order_id"))
		}
		w.Write([]byte(`{"orders":[{"status":"active","courier":{"phone":"9222222222","latitude":"12.9716","longitude":"77.5946"},"points":[{"tracking_url":"https://x"},{"tracking_url":"https://borzo.track/abc"}]}]}`))
	}))
	defer srv.Close()
	tr, err := newTestBorzo(srv).TrackTask(context.Background(), "99001")
	if err != nil || tr.ProviderStatus != "active" || tr.RiderLat != 12.9716 || tr.RiderPhone != "9222222222" {
		t.Fatalf("track = %+v err=%v", tr, err)
	}
}
