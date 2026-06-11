package services

import (
	"context"

	"github.com/homechef/api/models"
)

// QuoteRequest asks a provider for a price + ETA for a pickup→dropoff leg.
type QuoteRequest struct {
	PickupLat  float64
	PickupLng  float64
	DropoffLat float64
	DropoffLng float64
	City       string
	Weight     float64 // estimated weight in kg
}

// QuoteResponse is a provider's price + ETA for a delivery leg.
type QuoteResponse struct {
	Fee         float64 // amount Fe3dr is charged (and passes to the customer)
	Currency    string
	ETAMinutes  int
	Serviceable bool // false → provider cannot serve this leg; caller falls back
}

// TrackResponse is the live state of a provider task: status + rider position.
type TrackResponse struct {
	ProviderStatus string // raw provider status (pre-mapping)
	RiderName      string
	RiderPhone     string
	RiderLat       float64
	RiderLng       float64
	TrackingURL    string
}

// DeliveryProviderClient is the OUTBOUND adapter every 3PL integration
// implements. The inbound webhook path (status + rider location push) lives in
// HandleProviderWebhook; this interface covers everything Fe3dr initiates.
//
// Implementations are keyed by DeliveryProvider.Code via ClientFor. Providers
// without an implementation fall back to the mock path in CreateProviderDelivery
// and a computed quote in GetProviderQuote, so the platform degrades gracefully
// rather than failing an order.
type DeliveryProviderClient interface {
	// GetQuote returns price + ETA for a leg, or Serviceable=false if the
	// provider cannot serve it. Errors are transport/auth failures only.
	GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error)
	// CreateTask books a delivery (pickup=chef, drop=customer) and returns the
	// provider's external IDs + tracking URL + cost.
	CreateTask(ctx context.Context, req ProviderDeliveryRequest) (*ProviderDeliveryResponse, error)
	// CancelTask cancels a previously-booked task by its external ID.
	CancelTask(ctx context.Context, externalID, reason string) error
	// TrackTask pulls the current status + rider position for a task. Used as a
	// reconciliation fallback when webhooks are missed.
	TrackTask(ctx context.Context, externalID string) (*TrackResponse, error)
}

// ClientFor returns the outbound adapter for a provider, keyed by its Code.
// The bool is false when no real adapter exists yet — callers then use the
// built-in mock/fallback path so onboarding a not-yet-coded provider still works.
func ClientFor(p *models.DeliveryProvider) (DeliveryProviderClient, bool) {
	switch p.Code {
	case "shadowfax":
		return newShadowfaxClient(p), true
	default:
		return nil, false
	}
}
