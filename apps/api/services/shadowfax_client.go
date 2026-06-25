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

// shadowfaxClient is the OUTBOUND adapter for the Shadowfax Unified (Forward) API.
//
// Contract confirmed against https://sfxunifiedapi.docs.apiary.io/ (marketplace
// seller model). Auth: header `Authorization: Token <api_key>`. Base URL is the
// environment root incl. `/api` (sandbox `https://dale.staging.shadowfax.in/api`,
// prod `https://dale.shadowfax.in/api`).
//
// Capability notes baked into this adapter:
//   - Serviceability is a yes/no PINCODE check; the API returns no per-order price,
//     so GetQuote reports Serviceable only (Fee stays 0; caller keeps the flat fee).
//   - Create auto-assigns an AWB when omitted; we read it back as ExternalDeliveryID.
//   - Track/webhook expose status + a hosted customer_track_url, never rider GPS.
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

const (
	sfxPathServiceability = "/v1/clients/serviceability/"
	sfxPathCreateOrder    = "/v3/clients/orders/"
	sfxPathCancelOrder    = "/v3/clients/orders/cancel/"
	sfxPathTrackOrder     = "/v4/clients/orders/%s/track/"
)

// GetQuote performs a serviceability (pincode) check. The Unified API quotes no
// price, so Fee stays 0 and the caller keeps the flat configured delivery fee.
func (c *shadowfaxClient) GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
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

// CreateTask books a marketplace delivery (pickup=chef, drop=customer) and reads
// back the assigned AWB + Shadowfax-hosted customer tracking URL.
func (c *shadowfaxClient) CreateTask(ctx context.Context, req ProviderDeliveryRequest) (*ProviderDeliveryResponse, error) {
	leg := func(name, contact, addr, city, state, pc string, lat, lng float64) map[string]any {
		return map[string]any{
			"name": name, "contact": contact,
			"address_line_1": addr, "city": city, "state": state, "pincode": pc,
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
			AWBNumber        string `json:"awb_number"`
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

// CancelTask cancels an order by AWB or client_order_id (request_id accepts either).
func (c *shadowfaxClient) CancelTask(ctx context.Context, externalID, reason string) error {
	if reason == "" {
		reason = "Cancelled by Fe3dr"
	}
	payload := map[string]any{"request_id": externalID, "cancel_remarks": reason}
	return c.do(ctx, http.MethodPost, sfxPathCancelOrder, payload, nil)
}

// TrackTask pulls current status + the hosted track URL for an order by AWB. The
// Unified API exposes no live rider coordinates, so the rider fields stay zero.
func (c *shadowfaxClient) TrackTask(ctx context.Context, externalID string) (*TrackResponse, error) {
	var out struct {
		OrderDetails struct {
			Status           string `json:"status"`
			CustomerTrackURL string `json:"customer_track_url"`
		} `json:"order_details"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf(sfxPathTrackOrder, externalID), nil, &out); err != nil {
		return nil, err
	}
	return &TrackResponse{
		ProviderStatus: out.OrderDetails.Status,
		TrackingURL:    out.OrderDetails.CustomerTrackURL,
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
