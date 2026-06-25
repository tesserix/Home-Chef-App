# Shadowfax 3PL Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the real Shadowfax Unified (Forward) API into the existing provider-agnostic 3PL pipeline so a "Hand to a rider" delivery books, tracks, and cancels against the Shadowfax sandbox — gated dark until activated.

**Architecture:** The dispatch spine (Temporal `DeliveryWorkflow` → `DispatchOrderDelivery` → `ProviderService` → `ClientFor("shadowfax")`) and the webhook route already exist. We rewrite the Shadowfax adapter to the real contract, add a pure status mapper, parse the real Push Callback, thread the pincode/contact/amount fields the API needs, configure a dark provider row, and add a customer "Track live" link to Shadowfax's hosted page.

**Tech Stack:** Go 1.26 + Gin + GORM (backend); Expo/React Native + TypeScript (mobile-customer, mobile-vendor); Temporal (durable dispatch).

## Global Constraints

- Go: `gofmt` before commit; `go build ./...` + `go test ./...` green. Run Go cmds with absolute paths or `cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App/apps/api && …`.
- Mobile: `@babel/core ^7` + `tailwindcss ^3` pins; `npx tsc --noEmit` clean except the pre-existing vendor `app/reviews.tsx` router-typing error. Jest is broken repo-wide — frontend verifies via `tsc` + manual sim.
- Secrets: the Shadowfax token lives in GCP Secret Manager `shadowfax-api-token` (project `tesseracthub-480811`). NEVER commit it — the repo is PUBLIC. It reaches the process via env (`SHADOWFAX_API_TOKEN`) / ESO only.
- Never surface raw provider error codes to end users (`resolveAuthErrorMessage` / friendly copy).
- Single-line conventional commits, no signature. Feature branch `feat/shadowfax-3pl-integration`; owner authorizes merge.
- The provider row ships with `active=false` (dark) — no live behaviour change until the owner flips it.

## Shadowfax Unified API contract (reference — staging base `https://dale.staging.shadowfax.in/api/`, auth header `Authorization: Token <key>`)

- Serviceability (gate, no price): `GET /v1/clients/serviceability/?service=customer_delivery&pincodes=<csv>` → `{ "data": [ { "pincode": <n>, ... } ] }` (serviceable pincodes listed).
- Create: `POST /v3/clients/orders/` body `{ order_type:"marketplace", order_details:{client_order_id, product_value, cod_amount:0, payment_mode:"Prepaid", actual_weight}, customer_details:{drop}, pickup_details:{chef}, rts_details:{chef}, product_details:[sku] }` → `{ message, data:{ awb_number, customer_track_url } }` (awb auto-assigned when omitted).
- Cancel: `POST /v3/clients/orders/cancel/` body `{ request_id:<client_order_id or awb>, cancel_remarks }` → `{ responseMsg, responseCode }` (200 cancelled, 304 queued, 400 invalid/delivered).
- Track: `GET /v4/clients/orders/{awb}/track/` → `{ message, order_details:{ status, status_display, customer_track_url } }` (no rider GPS).
- Push Callback (webhook): `{ order_id (=client_order_id), status, event, current_location, event_timestamp, comments, otp_verified }`.
- Statuses: `new, ofp, picked, assigned_for_delivery, ofd, delivered, cancelled_by_seller, cancelled_by_customer, rts, rts_d, rto, lost, on_hold, pickup_on_hold, nc, na`.

---

### Task 1: Pure Shadowfax→Delivery status mapper

A single pure function both the track parser and the webhook use, so the mapping lives in one tested place.

**Files:**
- Create: `apps/api/services/shadowfax_status.go`
- Test: `apps/api/services/shadowfax_status_test.go`

**Interfaces:**
- Produces: `mapShadowfaxStatus(raw string) (models.DeliveryStatus, bool)` — returns the mapped status and `ok=false` for an unrecognised raw status (caller ignores/logs, never crashes).

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/homechef/api/models"
)

func TestMapShadowfaxStatus(t *testing.T) {
	cases := map[string]models.DeliveryStatus{
		"new":                    models.DeliveryAssigned,
		"ofp":                    models.DeliveryAssigned,
		"picked":                 models.DeliveryPickedUp,
		"assigned_for_delivery":  models.DeliveryInTransit,
		"ofd":                    models.DeliveryInTransit,
		"delivered":              models.DeliveryDelivered,
		"cancelled_by_seller":    models.DeliveryCancelled,
		"cancelled_by_customer":  models.DeliveryCancelled,
		"rts":                    models.DeliveryReturned,
		"rto":                    models.DeliveryReturned,
		"lost":                   models.DeliveryFailed,
		"on_hold":                models.DeliveryFailed,
	}
	for raw, want := range cases {
		got, ok := mapShadowfaxStatus(raw)
		if !ok || got != want {
			t.Fatalf("mapShadowfaxStatus(%q) = %q,%v; want %q,true", raw, got, ok, want)
		}
	}
	if _, ok := mapShadowfaxStatus("totally_unknown"); ok {
		t.Fatal("unknown status must return ok=false")
	}
	// Case-insensitive + whitespace tolerant (providers vary casing).
	if got, ok := mapShadowfaxStatus("  DELIVERED "); !ok || got != models.DeliveryDelivered {
		t.Fatalf("normalised match failed: %q %v", got, ok)
	}
}
```

- [ ] **Step 2: Run it, verify FAIL**

Run: `cd apps/api && go test ./services/ -run TestMapShadowfaxStatus -v`
Expected: FAIL (`mapShadowfaxStatus` undefined).

- [ ] **Step 3: Implement**

```go
package services

import (
	"strings"

	"github.com/homechef/api/models"
)

// mapShadowfaxStatus maps a Shadowfax Unified-API status string to our internal
// DeliveryStatus. We use the on-demand subset of Shadowfax's parcel vocabulary;
// hub-internal statuses (recd_at_fwd_hub, item_manifested, …) we don't surface
// collapse to the nearest customer-meaningful state. Returns ok=false for an
// unrecognised status so callers log-and-ignore rather than mis-transition.
func mapShadowfaxStatus(raw string) (models.DeliveryStatus, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "new", "ofp", "pickup_on_hold", "assigned_for_seller_pickup":
		return models.DeliveryAssigned, true
	case "picked":
		return models.DeliveryPickedUp, true
	case "assigned_for_delivery", "ofd", "recd_at_fwd_hub", "recd_at_fwd_dc":
		return models.DeliveryInTransit, true
	case "delivered":
		return models.DeliveryDelivered, true
	case "cancelled_by_seller", "cancelled_by_customer":
		return models.DeliveryCancelled, true
	case "rts", "rts_d", "rto", "rto_d", "rts_in_process", "rts_ofd":
		return models.DeliveryReturned, true
	case "lost", "on_hold", "nc", "na", "pickup_not_attempted":
		return models.DeliveryFailed, true
	default:
		return "", false
	}
}
```

- [ ] **Step 4: Run it, verify PASS**

Run: `cd apps/api && go test ./services/ -run TestMapShadowfaxStatus -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
gofmt -w apps/api/services/shadowfax_status.go apps/api/services/shadowfax_status_test.go
git add apps/api/services/shadowfax_status.go apps/api/services/shadowfax_status_test.go
git commit -m "feat(delivery): pure Shadowfax->Delivery status mapper"
```

---

### Task 2: Thread pincode + contact + amount onto the dispatch request

The real create-order needs pincodes/city/state for both legs, the chef as the pickup contact, and the order value. Extend `ProviderDeliveryRequest` and populate it in `DispatchOrderDelivery`.

**Files:**
- Modify: `apps/api/services/provider.go` (`ProviderDeliveryRequest` struct)
- Modify: `apps/api/services/provider_dispatch.go` (`DispatchOrderDelivery` request build)

**Interfaces:**
- Produces: `ProviderDeliveryRequest` gains `PickupName, PickupPhone, PickupCity, PickupState, PickupPincode, DropoffCity, DropoffState, DropoffPincode string` and `OrderValue float64` and `ClientOrderID string`. (Pincodes are strings — the API takes a 6-digit number but we carry the source string and convert in the adapter.)

- [ ] **Step 1: Add fields to `ProviderDeliveryRequest`**

In `apps/api/services/provider.go`, add to the struct (after `Weight`):

```go
	// Real-3PL fields (Shadowfax needs pincodes/city/state + a pickup contact +
	// the order value). PickupName/Phone identify the chef to the rider.
	ClientOrderID  string // our order id as the provider's client_order_id
	PickupName     string
	PickupPhone    string
	PickupCity     string
	PickupState    string
	PickupPincode  string
	DropoffCity    string
	DropoffState   string
	DropoffPincode string
	OrderValue     float64 // order subtotal — sent as product_value
```

- [ ] **Step 2: Populate them in `DispatchOrderDelivery`**

In `apps/api/services/provider_dispatch.go`, extend the `req := ProviderDeliveryRequest{...}` literal (the order already preloads `Chef` + `Customer`):

```go
	req := ProviderDeliveryRequest{
		OrderID:           order.ID,
		ClientOrderID:     order.OrderNumber,
		PickupAddress:     joinAddress(order.Chef.AddressLine1, order.Chef.City),
		PickupName:        order.Chef.BusinessName,
		PickupPhone:       order.Chef.Phone,
		PickupLat:         order.Chef.Latitude,
		PickupLng:         order.Chef.Longitude,
		PickupCity:        order.Chef.City,
		PickupState:       order.Chef.State,
		PickupPincode:     order.Chef.PostalCode,
		DropoffAddress:    joinAddress(order.DeliveryAddressLine1, order.DeliveryAddressCity),
		DropoffLat:        order.DeliveryLatitude,
		DropoffLng:        order.DeliveryLongitude,
		DropoffCity:       order.DeliveryAddressCity,
		DropoffState:      order.DeliveryAddressState,
		DropoffPincode:    order.DeliveryAddressPostalCode,
		CustomerName:      strings.TrimSpace(order.Customer.FirstName + " " + order.Customer.LastName),
		CustomerPhone:     order.Customer.Phone,
		ItemDescription:   fmt.Sprintf("Order %s", order.OrderNumber),
		OrderValue:        order.Subtotal,
		Weight:            1.0,
		ScheduledPickupAt: order.ScheduledFor,
	}
```

NOTE: if any of `order.Chef.Phone`, `order.Chef.State`, `order.DeliveryAddressState`, `order.DeliveryAddressPostalCode`, `order.Subtotal` do not exist with those exact names, grep the model (`apps/api/models/chef.go`, `apps/api/models/order.go`) and use the actual field; do not invent. (Chef has `PostalCode`/`State`/`Phone`; Order has `DeliveryAddressCity`/`DeliveryAddressCountry` already used above — confirm the `State`/`PostalCode`/`Subtotal` field names before compiling.)

- [ ] **Step 3: Build**

Run: `cd apps/api && go build ./...`
Expected: build OK. (If a field name is wrong, the compiler points at it — fix to the real model field.)

- [ ] **Step 4: Commit**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
gofmt -w apps/api/services/provider.go apps/api/services/provider_dispatch.go
git add apps/api/services/provider.go apps/api/services/provider_dispatch.go
git commit -m "feat(delivery): thread pincode/contact/value onto provider dispatch request"
```

---

### Task 3: Rewrite the Shadowfax adapter to the real contract

Replace the guessed endpoints/payloads in `shadowfax_client.go` with the real Unified API. `GetQuote` becomes a serviceability gate (Fee always 0). `CreateTask` posts the marketplace order and reads back the AWB + `customer_track_url`. `CancelTask` posts to the cancel endpoint with `request_id`. `TrackTask` reads status + track url (no rider GPS).

**Files:**
- Modify: `apps/api/services/shadowfax_client.go` (full rewrite of bodies/paths)
- Test: `apps/api/services/shadowfax_client_test.go` (new — httptest)

**Interfaces:**
- Consumes: `QuoteRequest`, `ProviderDeliveryRequest` (with Task 2 fields), `models.DeliveryProvider{APIBaseURL, APIKey}`.
- Produces: unchanged method set (`GetQuote/CreateTask/CancelTask/TrackTask`); `GetQuote` returns `{Serviceable, Fee:0}`; `CreateTask` returns `ProviderDeliveryResponse{ExternalDeliveryID:awb, TrackingURL:customer_track_url, Cost:0}`; `TrackTask` returns `{ProviderStatus, TrackingURL}` (rider fields empty).

- [ ] **Step 1: Write the failing httptest**

Create `apps/api/services/shadowfax_client_test.go`:

```go
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
	q, err := newTestShadowfax(srv).GetQuote(context.Background(), QuoteRequest{City: "Bengaluru"})
	if err != nil || !q.Serviceable || q.Fee != 0 {
		t.Fatalf("quote = %+v err=%v (want serviceable, fee 0)", q, err)
	}
}
```

NOTE: `GetQuote` reads the drop pincode from a new `QuoteRequest.DropoffPincode` field — add `DropoffPincode string` to `QuoteRequest` in `delivery_client.go` and set it in `QuoteCheckoutDeliveryFee`/dispatch where quotes are taken. The serviceability call sends `?service=customer_delivery&pincodes=<DropoffPincode>` and treats a non-empty `data` array as serviceable.

- [ ] **Step 2: Run it, verify FAIL**

Run: `cd apps/api && go test ./services/ -run TestShadowfax -v`
Expected: FAIL (old paths/payloads / missing `DropoffPincode`).

- [ ] **Step 3: Rewrite `shadowfax_client.go`**

Replace the path consts + the four method bodies. Key points: paths `/v1/clients/serviceability/`, `/v3/clients/orders/`, `/v3/clients/orders/cancel/`, `/v4/clients/orders/{awb}/track/`. Build the nested create payload from the Task-2 request fields:

```go
const (
	sfxPathServiceability = "/v1/clients/serviceability/"
	sfxPathCreateOrder    = "/v3/clients/orders/"
	sfxPathCancelOrder    = "/v3/clients/orders/cancel/"
	sfxPathTrackOrder     = "/v4/clients/orders/%s/track/"
)

func (c *shadowfaxClient) GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	// Serviceability is a yes/no pincode check — the Unified API returns no price,
	// so Fee stays 0 and the caller keeps the flat configured delivery fee.
	if req.DropoffPincode == "" {
		return &QuoteResponse{Serviceable: false}, nil
	}
	path := fmt.Sprintf("%s?service=customer_delivery&pincodes=%s", sfxPathServiceability, req.DropoffPincode)
	var out struct {
		Data []struct {
			Pincode int `json:"pincode"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &QuoteResponse{Serviceable: len(out.Data) > 0, Fee: 0, Currency: "INR"}, nil
}

func (c *shadowfaxClient) CreateTask(ctx context.Context, req ProviderDeliveryRequest) (*ProviderDeliveryResponse, error) {
	pin := func(s string) any { return s } // pincode sent as string; API accepts numeric-string
	leg := func(name, contact, addr, city, state, pc string, lat, lng float64) map[string]any {
		return map[string]any{
			"name": name, "contact": contact,
			"address_line_1": addr, "city": city, "state": state, "pincode": pin(pc),
			"latitude": fmt.Sprintf("%f", lat), "longitude": fmt.Sprintf("%f", lng),
		}
	}
	payload := map[string]any{
		"order_type": "marketplace",
		"order_details": map[string]any{
			"client_order_id": req.ClientOrderID,
			"product_value":   req.OrderValue,
			"cod_amount":      0,
			"payment_mode":    "Prepaid",
			"actual_weight":   req.Weight * 1000, // kg → grams
		},
		"customer_details": leg(req.CustomerName, req.CustomerPhone, req.DropoffAddress, req.DropoffCity, req.DropoffState, req.DropoffPincode, req.DropoffLat, req.DropoffLng),
		"pickup_details":   leg(req.PickupName, req.PickupPhone, req.PickupAddress, req.PickupCity, req.PickupState, req.PickupPincode, req.PickupLat, req.PickupLng),
		"rts_details":      leg(req.PickupName, req.PickupPhone, req.PickupAddress, req.PickupCity, req.PickupState, req.PickupPincode, req.PickupLat, req.PickupLng),
		"product_details": []map[string]any{{
			"sku_name": req.ItemDescription, "price": req.OrderValue, "category": "food",
		}},
	}
	var out struct {
		Message string `json:"message"`
		Data    struct {
			AWBNumber       string `json:"awb_number"`
			CustomerTrackURL string `json:"customer_track_url"`
		} `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, sfxPathCreateOrder, payload, &out); err != nil {
		return nil, err
	}
	if out.Data.AWBNumber == "" {
		return nil, fmt.Errorf("shadowfax: create returned empty awb (message=%q)", out.Message)
	}
	return &ProviderDeliveryResponse{
		ExternalDeliveryID: out.Data.AWBNumber,
		TrackingURL:        out.Data.CustomerTrackURL,
		Cost:               0, // Unified API quotes no per-order price
	}, nil
}

func (c *shadowfaxClient) CancelTask(ctx context.Context, externalID, reason string) error {
	if reason == "" {
		reason = "Cancelled by Fe3dr"
	}
	payload := map[string]any{"request_id": externalID, "cancel_remarks": reason}
	return c.do(ctx, http.MethodPost, sfxPathCancelOrder, payload, nil)
}

func (c *shadowfaxClient) TrackTask(ctx context.Context, externalID string) (*TrackResponse, error) {
	var out struct {
		OrderDetails struct {
			Status          string `json:"status"`
			CustomerTrackURL string `json:"customer_track_url"`
		} `json:"order_details"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf(sfxPathTrackOrder, externalID), nil, &out); err != nil {
		return nil, err
	}
	return &TrackResponse{
		ProviderStatus: out.OrderDetails.Status,
		TrackingURL:    out.OrderDetails.CustomerTrackURL,
		// No rider GPS in the Unified API — RiderLat/Lng/Name/Phone stay zero.
	}, nil
}
```

Leave the `do` helper as-is (auth header already `Token <key>`). Update the struct doc comment to drop the "guessed" language and the TODO(shadowfax-creds) markers (the contract is now confirmed). Add `DropoffPincode string` to `QuoteRequest` in `delivery_client.go`.

- [ ] **Step 4: Add the cancel-by-client-order wiring**

`CancelOrderDelivery` (provider_dispatch.go) currently passes `delivery.ExternalDeliveryID` (the awb) to `CancelTask` — that still works (`request_id` accepts awb OR client_order_id). No change needed; confirm it compiles.

- [ ] **Step 5: Run tests, verify PASS**

Run: `cd apps/api && go test ./services/ -run TestShadowfax -v`
Expected: PASS (all three).

- [ ] **Step 6: Build + full service tests**

Run: `cd apps/api && go build ./... && go test ./services/`
Expected: build OK, tests PASS.

- [ ] **Step 7: Commit**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
gofmt -w apps/api/services/shadowfax_client.go apps/api/services/delivery_client.go
git add apps/api/services/shadowfax_client.go apps/api/services/delivery_client.go apps/api/services/shadowfax_client_test.go
git commit -m "feat(delivery): rewrite Shadowfax adapter to the real Unified API contract"
```

---

### Task 4: Parse the real Shadowfax Push Callback in the webhook

Make `HandleWebhook` understand the real callback body for `:provider == "shadowfax"`: resolve the `Delivery` by `order_id` (our order number / client_order_id), map the status, update `Delivery` + order, idempotently.

**Files:**
- Modify: `apps/api/handlers/delivery_provider.go` (`HandleWebhook` Shadowfax branch)
- Test: `apps/api/handlers/delivery_webhook_hmac_test.go` (extend)

**Interfaces:**
- Consumes: `mapShadowfaxStatus` (Task 1); `models.Delivery`, `models.Order`.
- Produces: a webhook that, given `{order_id, status, event_timestamp}`, sets `Delivery.Status` (+ `DeliveredAt`/`PickedUpAt` where relevant) and propagates a terminal status to the order.

- [ ] **Step 1: Read the current `HandleWebhook`**

Run: `cd apps/api && sed -n '1,120p' handlers/delivery_provider.go` and locate the `HandleWebhook` body + how it currently resolves the delivery and the HMAC check. Reuse that resolution; only the Shadowfax payload parsing + status mapping changes.

- [ ] **Step 2: Write the failing test**

Add to `apps/api/handlers/delivery_webhook_hmac_test.go` a test that posts a Shadowfax callback for a known order and asserts the `Delivery.Status` flips to `in_transit` on `ofd` and `delivered` on `delivered`. Mirror the existing test's harness (DB setup + HMAC signing helper already present in that file — reuse them; do not invent a new signer).

```go
// Shape of the assertion (fill the harness from the existing test in this file):
//   seed an Order + Delivery{ExternalDeliveryID:"SF1", Status: assigned}
//   POST /api/webhooks/delivery/shadowfax  body {"order_id":"<order.OrderNumber>","status":"ofd","event":"ofd","event_timestamp":"2026-06-25 10:00:00"}
//   assert reloaded Delivery.Status == models.DeliveryInTransit
//   POST again with status "delivered" → assert Delivery.Status == models.DeliveryDelivered and order marked delivered
```

- [ ] **Step 3: Run it, verify FAIL**

Run: `cd apps/api && go test ./handlers/ -run TestDeliveryWebhook -v`
Expected: FAIL (status not mapped yet).

- [ ] **Step 4: Implement the Shadowfax branch**

In `HandleWebhook`, for the shadowfax provider decode:

```go
var cb struct {
	OrderID        string `json:"order_id"` // = our client_order_id (order number)
	Status         string `json:"status"`
	Event          string `json:"event"`
	EventTimestamp string `json:"event_timestamp"`
	OTPVerified    string `json:"otp_verified"`
}
```

Resolve the `Delivery` by joining the order: `Order.OrderNumber == cb.OrderID` → its `Delivery`. Then:

```go
status, ok := mapShadowfaxStatus(cb.Status)
if !ok {
	// Unknown/intermediate status — ack 200 so Shadowfax stops retrying, log it.
	c.JSON(http.StatusOK, gin.H{"message": "ignored"})
	return
}
updates := map[string]any{"status": status}
now := time.Now()
switch status {
case models.DeliveryPickedUp:
	updates["picked_up_at"] = now
case models.DeliveryDelivered:
	updates["delivered_at"] = now
}
database.DB.Model(&delivery).Updates(updates)
// Propagate terminal states to the order so the customer timeline + chef screen move.
if status == models.DeliveryDelivered {
	database.DB.Model(&models.Order{}).Where("id = ?", delivery.OrderID).Update("status", models.OrderStatusDelivered)
}
c.JSON(http.StatusOK, gin.H{"message": "ok"})
```

Keep the existing HMAC/secret verification ahead of this (reuse it). If the order/delivery isn't found, ack 200 + log (don't make Shadowfax retry forever). Use the real field names found in Step 1 (e.g. the delivery may already be loaded differently — adapt).

- [ ] **Step 5: Run tests, verify PASS**

Run: `cd apps/api && go test ./handlers/ -run TestDeliveryWebhook -v`
Expected: PASS.

- [ ] **Step 6: Build + handler tests**

Run: `cd apps/api && go build ./... && go test ./handlers/`
Expected: green.

- [ ] **Step 7: Commit**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
gofmt -w apps/api/handlers/delivery_provider.go apps/api/handlers/delivery_webhook_hmac_test.go
git add apps/api/handlers/delivery_provider.go apps/api/handlers/delivery_webhook_hmac_test.go
git commit -m "feat(delivery): parse real Shadowfax push callback -> Delivery/order status"
```

---

### Task 5: Configure the dark Shadowfax provider + secret plumbing

Seed a `shadowfax` `DeliveryProvider` row (staging base URL, serviceable city, webhook secret), reading the API token from env `SHADOWFAX_API_TOKEN` (sourced from Secret Manager). `active=false` so nothing changes until the owner flips it.

**Files:**
- Modify: `apps/api/config/config.go` (add `ShadowfaxAPIToken` from env)
- Create: `apps/api/database/seed_shadowfax.go` (idempotent upsert of the provider row, called at startup)
- Modify: `apps/api/database/database.go` (call the seed after AutoMigrate — match the existing seed pattern there)
- Modify: `apps/api/.env.example` (document `SHADOWFAX_API_TOKEN=` empty)

**Interfaces:**
- Consumes: `models.DeliveryProvider` fields (`Code, Name, APIBaseURL, APIKey, APISecret, WebhookSecret, IsActive/Active, ServiceableCities, …` — confirm exact field names in `models/delivery_provider.go`).
- Produces: `database.SeedShadowfaxProvider(db *gorm.DB, token, webhookSecret string)`.

- [ ] **Step 1: Add the env token to config**

In `apps/api/config/config.go`, add a field (match the existing config struct/loader style):

```go
ShadowfaxAPIToken string // from Secret Manager via env SHADOWFAX_API_TOKEN; empty = provider stays disabled
```
…and load it: `ShadowfaxAPIToken: os.Getenv("SHADOWFAX_API_TOKEN"),`

- [ ] **Step 2: Write the idempotent seed**

Create `apps/api/database/seed_shadowfax.go`:

```go
package database

import (
	"log"

	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// SeedShadowfaxProvider upserts the Shadowfax 3PL provider row. It ships DISABLED
// (Active=false) — the owner flips it on from the admin provider API once the
// sandbox is verified. The API token comes from Secret Manager via env, never the
// DB seed in source. A blank token leaves the row disabled regardless.
func SeedShadowfaxProvider(db *gorm.DB, token, webhookSecret string) {
	var existing models.DeliveryProvider
	err := db.Where("code = ?", "shadowfax").First(&existing).Error
	provider := models.DeliveryProvider{
		Code:          "shadowfax",
		Name:          "Shadowfax",
		APIBaseURL:    "https://dale.staging.shadowfax.in/api",
		APIKey:        token,
		WebhookSecret: webhookSecret,
		Active:        false, // dark until verified + flipped
	}
	if err == gorm.ErrRecordNotFound {
		if e := db.Create(&provider).Error; e != nil {
			log.Printf("seed shadowfax provider: %v", e)
		}
		return
	}
	// Refresh non-destructive fields (token/base url) but DO NOT auto-enable.
	db.Model(&existing).Updates(map[string]any{
		"api_base_url":   provider.APIBaseURL,
		"api_key":        token,
		"webhook_secret": webhookSecret,
	})
}
```

NOTE: confirm `models.DeliveryProvider`'s active-flag field name (`Active` vs `IsActive`) and serviceable-cities representation in `models/delivery_provider.go`; use the real names and add a serviceable city (e.g. `"Bengaluru"`) the way `FindAvailableProvider` expects (it filters by city — match its query). If the model requires serviceable cities to be non-empty for `FindAvailableProvider` to return it, seed the owner's test city.

- [ ] **Step 3: Call the seed at startup**

In `apps/api/database/database.go`, after `AutoMigrate` / alongside existing seeds, call `SeedShadowfaxProvider(DB, cfg.ShadowfaxAPIToken, cfg.ShadowfaxWebhookSecret)` — wire `cfg` the way other seeds receive config (grep how existing seeds get their values; if seeds run without `cfg`, read `os.Getenv` inside the seed instead and drop the param).

- [ ] **Step 4: Document the env var**

In `apps/api/.env.example`, add:

```
# Shadowfax 3PL (sandbox). Token from GCP Secret Manager shadowfax-api-token. Leave blank to keep disabled.
SHADOWFAX_API_TOKEN=
SHADOWFAX_WEBHOOK_SECRET=
```

- [ ] **Step 5: Build**

Run: `cd apps/api && go build ./...`
Expected: build OK.

- [ ] **Step 6: Commit**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
gofmt -w apps/api/config/config.go apps/api/database/seed_shadowfax.go apps/api/database/database.go
git add apps/api/config/config.go apps/api/database/seed_shadowfax.go apps/api/database/database.go apps/api/.env.example
git commit -m "feat(delivery): seed disabled Shadowfax provider + SHADOWFAX_API_TOKEN env wiring"
```

---

### Task 6: Add the dispatch serviceability gate + customer "Track live" link

Two thin pieces: (a) gate dispatch on serviceability so an unserviceable drop parks instead of erroring; (b) give the customer a "Track live" button that opens Shadowfax's `customer_track_url`.

**Files:**
- Modify: `apps/api/services/provider_dispatch.go` (`DispatchOrderDelivery` — serviceability gate before booking)
- Modify: `apps/api/services/provider_dispatch.go` (`QuoteCheckoutDeliveryFee` — keep flat fee when Fee<=0)
- Modify: `apps/mobile-customer/app/order/[id]/index.tsx` (Track-live button when a 3PL delivery exposes a tracking URL)

**Interfaces:**
- Consumes: `GetProviderQuote` returning `{Serviceable, Fee}`; the order-detail API field exposing `externalTrackingUrl` (already on `Delivery.ToResponse` as `externalTrackingUrl`).
- Produces: parked-on-unserviceable dispatch; a customer-facing track link.

- [ ] **Step 1: Serviceability gate in `DispatchOrderDelivery`**

After `provider` is found and before building `req`, add (uses the new `QuoteRequest.DropoffPincode`):

```go
	// Serviceability gate: the Unified API only books serviceable pincodes. If the
	// drop isn't serviceable, park the order for manual handling rather than erroring.
	if q, qerr := svc.GetProviderQuote(provider, QuoteRequest{
		DropoffPincode: order.DeliveryAddressPostalCode,
		City:           city,
	}); qerr == nil && q != nil && !q.Serviceable {
		log.Printf("dispatch: drop pincode %q not serviceable by %s — parked", order.DeliveryAddressPostalCode, provider.Code)
		return nil
	}
```

- [ ] **Step 2: Keep the flat fee in `QuoteCheckoutDeliveryFee`**

In `QuoteCheckoutDeliveryFee`, after the quote is fetched, treat a zero/absent price as "no quote" so the caller keeps the configured flat fee:

```go
	if err != nil || quote == nil || !quote.Serviceable || quote.Fee <= 0 {
		return 0, false
	}
	return quote.Fee, true
```

- [ ] **Step 3: Build + service tests**

Run: `cd apps/api && go build ./... && go test ./services/`
Expected: green.

- [ ] **Step 4: Commit the backend slice**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
gofmt -w apps/api/services/provider_dispatch.go
git add apps/api/services/provider_dispatch.go
git commit -m "feat(delivery): serviceability gate on dispatch + keep flat checkout fee"
```

- [ ] **Step 5: Customer "Track live" button**

In `apps/mobile-customer/app/order/[id]/index.tsx`, locate where the order's delivery/tracking data is available (the order-detail hook). When the order is a 3PL `delivery` order, it's active (picked_up/in transit), and a tracking URL is present (`order.delivery?.externalTrackingUrl` — confirm the field name the hook exposes; the backend ships `externalTrackingUrl` on the delivery), render a button that opens it. Reuse the existing in-app WebView pattern if one exists (the Razorpay checkout WebView in `apps/mobile-customer`); otherwise open externally:

```tsx
import * as WebBrowser from 'expo-web-browser';
// …
{trackUrl ? (
  <Pressable
    onPress={() => WebBrowser.openBrowserAsync(trackUrl)}
    accessibilityRole="button"
    accessibilityLabel="Track your delivery live"
    style={styles.trackLiveBtn}
  >
    <Text style={styles.trackLiveLabel}>Track live</Text>
  </Pressable>
) : null}
```

Confirm `expo-web-browser` is already a dependency (it is used elsewhere in the customer app); if not, prefer the existing WebView screen. Add `trackLiveBtn`/`trackLiveLabel` styles consistent with the screen's design tokens.

- [ ] **Step 6: Typecheck**

Run: `cd apps/mobile-customer && npx tsc --noEmit`
Expected: clean.

- [ ] **Step 7: Commit the customer slice**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
git add apps/mobile-customer/app/order/\[id\]/index.tsx
git commit -m "feat(customer): Track-live link to Shadowfax hosted tracking for 3PL orders"
```

---

### Task 7: Verify vendor 3PL status wording + final integration build

The vendor "Hand to a rider" → ready footer already reads "Waiting for a rider to pick up" and the History bucketing keeps 3PL `picked_up` in History. Confirm the chef sees the 3PL status progress (assigned → picked up → delivered) driven by the webhook, and run the whole suite.

**Files:**
- Verify (change only if wording is wrong): `apps/mobile-vendor/app/orders/[orderId].tsx` (`statusLabelFor` already returns "Picked up by rider"/"Ready · awaiting rider" for `delivery`)

- [ ] **Step 1: Confirm vendor wording**

Run: `cd apps/mobile-vendor && grep -n "awaiting rider\|Picked up by rider\|Waiting for a rider" "app/orders/[orderId].tsx"`
Expected: present (shipped in the chef-delivery streamline work). If a 3PL `delivered` order doesn't read "Delivered" for the chef, fix `statusLabelFor`; otherwise no change.

- [ ] **Step 2: Full backend suite**

Run: `cd apps/api && go build ./... && go test ./...`
Expected: build OK; tests PASS (Shadowfax + webhook + existing).

- [ ] **Step 3: Mobile typechecks**

Run: `cd apps/mobile-customer && npx tsc --noEmit` (clean) and `cd ../mobile-vendor && npx tsc --noEmit 2>&1 | grep -v "app/reviews.tsx"` (clean).

- [ ] **Step 4: Commit any verify-fixes**

```bash
cd /Users/Mahesh.Sangawar/personal/tesserix-new/Home-Chef-App
git add -A
git commit -m "test(delivery): verify vendor 3PL status wording + full Shadowfax suite green"
```

(If nothing changed, skip the commit.)

---

## Self-review notes

- **Spec coverage:** adapter rewrite (T3) ✓; status mapping (T1) ✓; webhook parse→status (T4) ✓; provider config + secret dark (T5) ✓; customer status timeline (existing carrier-neutral steps) + Track-live link (T6) ✓; vendor 3PL status (T7) ✓; serviceability gate + flat fee (T6) ✓; live-rider-map + live-fee correctly DROPPED (not built) ✓; reconciliation DEFERRED (not built) ✓.
- **Secrets:** token only via env from Secret Manager; provider ships `active=false`; nothing live until the owner flips it + deploys (Kargo). ✓
- **Type consistency:** `ProviderDeliveryRequest` fields added in T2 are consumed in T3; `QuoteRequest.DropoffPincode` added in T3 Step 1 note + used in T6 gate; `mapShadowfaxStatus` (T1) consumed in T4. Confirm real model field names (`Chef.Phone/State/PostalCode`, `Order.DeliveryAddressState/PostalCode/Subtotal`, `DeliveryProvider.Active`, `Delivery.ExternalTrackingURL`) at compile time — flagged inline where they must be checked against the model.
- **Activation/flip (post-merge):** deploy backend (Kargo), set the Shadowfax token in Secret Manager→env, register the webhook URL `https://fe3dr.com/api/webhooks/delivery/shadowfax` in the Shadowfax Client Portal, then enable the provider row (`active=true`) for the owner's test city. Sandbox test is owner-run.
