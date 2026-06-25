package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/homechef/api/models"
)

// borzoClient is the OUTBOUND adapter for the Borzo (ex-WeFast) Business API —
// an on-demand intracity courier with self-serve token auth.
//
// Contract from https://borzodelivery.com/in/business-api/doc (v1.8). Auth: a
// single static header `X-DV-Auth-Token: <api_key>`. APIBaseURL is the env root
// incl. version (sandbox `https://robotapitest-in.borzodelivery.com/api/business/1.8`,
// prod `https://robot-in.borzodelivery.com/api/business/1.8`).
//
// Unlike the Shadowfax marketplace API, Borzo is point-to-point (no warehouse
// return), quotes a price (`payment_amount`), and exposes the courier's live
// lat/lng (`order.courier.latitude/longitude`) for the in-app tracking map.
type borzoClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func newBorzoClient(p *models.DeliveryProvider) *borzoClient {
	return &borzoClient{
		baseURL:    strings.TrimRight(p.APIBaseURL, "/"),
		apiKey:     p.APIKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

const (
	borzoVehicleMotorbike = 8 // default vehicle_type_id — bikes suit hot food
	borzoPathCalculate    = "/calculate-order"
	borzoPathCreate       = "/create-order"
	borzoPathCancel       = "/cancel-order"
	borzoPathOrders       = "/orders"
)

// borzoPoint is a pickup or drop stop in a Borzo order.
func borzoPoint(addr, name, phone string, lat, lng float64) map[string]any {
	return map[string]any{
		"address":   addr,
		"latitude":  lat,
		"longitude": lng,
		"contact_person": map[string]any{
			"name":  name,
			"phone": phone,
		},
	}
}

// GetQuote prices a pickup→drop leg via calculate-order (validate-only, no order
// placed). Returns the live delivery price + serviceability.
func (c *borzoClient) GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	payload := map[string]any{
		"matter":          "Food order",
		"vehicle_type_id": borzoVehicleMotorbike,
		"total_weight_kg": req.Weight,
		"points": []map[string]any{
			borzoPoint("Pickup", "Kitchen", "0000000000", req.PickupLat, req.PickupLng),
			borzoPoint("Drop", "Customer", "0000000000", req.DropoffLat, req.DropoffLng),
		},
	}
	var out struct {
		IsSuccessful bool `json:"is_successful"`
		Order        struct {
			PaymentAmount     json.Number `json:"payment_amount"`
			DeliveryFeeAmount json.Number `json:"delivery_fee_amount"`
		} `json:"order"`
	}
	if err := c.do(ctx, http.MethodPost, borzoPathCalculate, payload, &out); err != nil {
		return nil, err
	}
	fee, _ := out.Order.PaymentAmount.Float64()
	return &QuoteResponse{Serviceable: out.IsSuccessful, Fee: fee, Currency: "INR"}, nil
}

// CreateTask books a point-to-point order (point 0 = chef pickup, point 1 =
// customer drop) and returns Borzo's order id + recipient tracking URL + price.
func (c *borzoClient) CreateTask(ctx context.Context, req ProviderDeliveryRequest) (*ProviderDeliveryResponse, error) {
	payload := map[string]any{
		"matter":          req.ItemDescription,
		"vehicle_type_id": borzoVehicleMotorbike,
		"total_weight_kg": req.Weight,
		"points": []map[string]any{
			borzoPoint(req.PickupAddress, req.PickupName, req.PickupPhone, req.PickupLat, req.PickupLng),
			borzoPoint(req.DropoffAddress, req.CustomerName, req.CustomerPhone, req.DropoffLat, req.DropoffLng),
		},
	}
	var out struct {
		Order struct {
			OrderID       int64       `json:"order_id"`
			OrderName     string      `json:"order_name"`
			PaymentAmount json.Number `json:"payment_amount"`
			Points        []struct {
				TrackingURL string `json:"tracking_url"`
			} `json:"points"`
		} `json:"order"`
	}
	if err := c.do(ctx, http.MethodPost, borzoPathCreate, payload, &out); err != nil {
		return nil, err
	}
	if out.Order.OrderID == 0 {
		return nil, fmt.Errorf("borzo: create returned empty order_id")
	}
	// Recipient tracking URL is on the drop point (last point).
	trackURL := ""
	if n := len(out.Order.Points); n > 0 {
		trackURL = out.Order.Points[n-1].TrackingURL
	}
	cost, _ := out.Order.PaymentAmount.Float64()
	return &ProviderDeliveryResponse{
		ExternalDeliveryID: strconv.FormatInt(out.Order.OrderID, 10),
		ExternalTrackingID: out.Order.OrderName,
		TrackingURL:        trackURL,
		Cost:               cost,
	}, nil
}

// CancelTask cancels a Borzo order by its order_id.
func (c *borzoClient) CancelTask(ctx context.Context, externalID, reason string) error {
	payload := map[string]any{"order_id": borzoOrderID(externalID)}
	return c.do(ctx, http.MethodPost, borzoPathCancel, payload, nil)
}

// TrackTask reads current status + live courier position for an order.
func (c *borzoClient) TrackTask(ctx context.Context, externalID string) (*TrackResponse, error) {
	path := fmt.Sprintf("%s?order_id=%s", borzoPathOrders, externalID)
	var out struct {
		Orders []struct {
			Status  string `json:"status"`
			Courier struct {
				Phone     string      `json:"phone"`
				Latitude  json.Number `json:"latitude"`
				Longitude json.Number `json:"longitude"`
			} `json:"courier"`
			Points []struct {
				TrackingURL string `json:"tracking_url"`
			} `json:"points"`
		} `json:"orders"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	if len(out.Orders) == 0 {
		return nil, fmt.Errorf("borzo: track returned no orders for %s", externalID)
	}
	o := out.Orders[0]
	lat, _ := o.Courier.Latitude.Float64()
	lng, _ := o.Courier.Longitude.Float64()
	trackURL := ""
	if n := len(o.Points); n > 0 {
		trackURL = o.Points[n-1].TrackingURL
	}
	return &TrackResponse{
		ProviderStatus: o.Status,
		RiderPhone:     o.Courier.Phone,
		RiderLat:       lat,
		RiderLng:       lng,
		TrackingURL:    trackURL,
	}, nil
}

// borzoOrderID returns the int order id when externalID is numeric, else the
// raw string — Borzo's cancel body wants an int but we never want to crash.
func borzoOrderID(externalID string) any {
	if n, err := strconv.ParseInt(externalID, 10, 64); err == nil {
		return n
	}
	return externalID
}

// do executes an authenticated JSON request against Borzo, decoding into out.
func (c *borzoClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("borzo: marshal request: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("borzo: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-DV-Auth-Token", c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("borzo: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("borzo: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("borzo: decode response: %w", err)
	}
	return nil
}
