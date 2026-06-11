package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/homechef/api/models"
)

// shadowfaxClient is the OUTBOUND adapter for Shadowfax (Hyperlocal / on-demand).
//
// SWAP-READY: built against Shadowfax's documented hyperlocal REST shape, but
// no sandbox credentials were available at build time. Every place the exact
// contract matters — endpoint paths, auth header, request/response field names —
// is isolated in the marked helpers below and tagged TODO(shadowfax-creds).
// When sandbox access lands, only those helpers should need edits; the wiring
// (registry, dispatch, webhook, quote, cancel) is provider-agnostic and stays put.
//
// Auth: Shadowfax uses token auth — `Authorization: Token <api_key>`. The token
// is the provider's APIKey; APIBaseURL is the environment root (sandbox vs prod).
type shadowfaxClient struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

func newShadowfaxClient(p *models.DeliveryProvider) *shadowfaxClient {
	return &shadowfaxClient{
		baseURL:    strings.TrimRight(p.APIBaseURL, "/"),
		apiKey:     p.APIKey,
		apiSecret:  p.APISecret,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// --- endpoint paths (TODO(shadowfax-creds): confirm against partner docs) ---

const (
	sfxPathServiceability = "/api/v3/clients/serviceability/"
	sfxPathCreateOrder    = "/api/v3/clients/order/"
	sfxPathCancelOrder    = "/api/v3/clients/order/cancel/"
	sfxPathTrackOrder     = "/api/v3/clients/order/track/"
)

// GetQuote asks Shadowfax for price + ETA + serviceability for a leg.
func (c *shadowfaxClient) GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	// TODO(shadowfax-creds): confirm serviceability/price payload + field names.
	payload := map[string]any{
		"pickup_details":  map[string]any{"latitude": req.PickupLat, "longitude": req.PickupLng},
		"drop_details":    map[string]any{"latitude": req.DropoffLat, "longitude": req.DropoffLng},
		"city":            req.City,
		"order_weight_kg": req.Weight,
	}

	var out struct {
		Serviceable bool    `json:"serviceable"`
		Price       float64 `json:"price"`
		Currency    string  `json:"currency"`
		ETAMinutes  int     `json:"eta_minutes"`
	}
	if err := c.do(ctx, http.MethodPost, sfxPathServiceability, payload, &out); err != nil {
		return nil, err
	}

	currency := out.Currency
	if currency == "" {
		currency = "INR"
	}
	return &QuoteResponse{
		Fee:         out.Price,
		Currency:    currency,
		ETAMinutes:  out.ETAMinutes,
		Serviceable: out.Serviceable,
	}, nil
}

// CreateTask books a delivery with Shadowfax (pickup=chef, drop=customer).
func (c *shadowfaxClient) CreateTask(ctx context.Context, req ProviderDeliveryRequest) (*ProviderDeliveryResponse, error) {
	// TODO(shadowfax-creds): confirm create-order payload + field names.
	payload := map[string]any{
		"client_order_id": req.OrderID.String(),
		"pickup_details": map[string]any{
			"address":   req.PickupAddress,
			"latitude":  req.PickupLat,
			"longitude": req.PickupLng,
		},
		"drop_details": map[string]any{
			"address":   req.DropoffAddress,
			"latitude":  req.DropoffLat,
			"longitude": req.DropoffLng,
			"name":      req.CustomerName,
			"phone":     req.CustomerPhone,
		},
		"order_details": map[string]any{
			"description": req.ItemDescription,
			"weight_kg":   req.Weight,
		},
	}

	// TODO(shadowfax-creds): confirm create-order response field names.
	var out struct {
		OrderID     string  `json:"order_id"`      // Shadowfax's task id
		TrackingID  string  `json:"tracking_id"`   //
		TrackingURL string  `json:"tracking_url"`  //
		Price       float64 `json:"price"`         // what Shadowfax charges Fe3dr
		PickupETA   int     `json:"pickup_eta_min"`
		DropETA     int     `json:"drop_eta_min"`
	}
	if err := c.do(ctx, http.MethodPost, sfxPathCreateOrder, payload, &out); err != nil {
		return nil, err
	}
	if out.OrderID == "" {
		return nil, fmt.Errorf("shadowfax: create order returned empty order_id")
	}

	now := time.Now()
	return &ProviderDeliveryResponse{
		ExternalDeliveryID: out.OrderID,
		ExternalTrackingID: out.TrackingID,
		TrackingURL:        out.TrackingURL,
		EstimatedPickup:    now.Add(time.Duration(out.PickupETA) * time.Minute),
		EstimatedDelivery:  now.Add(time.Duration(out.PickupETA+out.DropETA) * time.Minute),
		Cost:               out.Price,
	}, nil
}

// CancelTask cancels a Shadowfax order by its external id.
func (c *shadowfaxClient) CancelTask(ctx context.Context, externalID, reason string) error {
	// TODO(shadowfax-creds): confirm cancel payload + field names.
	payload := map[string]any{
		"order_id":      externalID,
		"cancel_reason": reason,
	}
	return c.do(ctx, http.MethodPost, sfxPathCancelOrder, payload, nil)
}

// TrackTask pulls current status + rider position for a Shadowfax order.
func (c *shadowfaxClient) TrackTask(ctx context.Context, externalID string) (*TrackResponse, error) {
	// TODO(shadowfax-creds): confirm track endpoint shape (query vs path) + fields.
	path := fmt.Sprintf("%s?order_id=%s", sfxPathTrackOrder, externalID)

	var out struct {
		Status      string  `json:"status"`
		RiderName   string  `json:"rider_name"`
		RiderPhone  string  `json:"rider_phone"`
		RiderLat    float64 `json:"rider_latitude"`
		RiderLng    float64 `json:"rider_longitude"`
		TrackingURL string  `json:"tracking_url"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return &TrackResponse{
		ProviderStatus: out.Status,
		RiderName:      out.RiderName,
		RiderPhone:     out.RiderPhone,
		RiderLat:       out.RiderLat,
		RiderLng:       out.RiderLng,
		TrackingURL:    out.TrackingURL,
	}, nil
}

// do executes an authenticated JSON request and decodes the response into out
// (when non-nil). It centralizes auth, encoding, and non-2xx handling so the
// per-method helpers stay focused on payload/field mapping.
func (c *shadowfaxClient) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("shadowfax: marshal request: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("shadowfax: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// TODO(shadowfax-creds): confirm auth scheme (Token vs Bearer vs api-key header).
	httpReq.Header.Set("Authorization", "Token "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("shadowfax: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shadowfax: %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("shadowfax: decode response: %w", err)
	}
	return nil
}
