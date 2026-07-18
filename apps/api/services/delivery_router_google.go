package services

// delivery_router_google.go — the real Google Routes distance provider (#701),
// the recommended GCP option for accurate road distance.
//
// It implements DeliveryRouter over the Routes API v2 computeRoutes endpoint. It
// is wrapped by NewCachedRouter, so a given (chef → address) trip is billed to
// Google at most once ever — the two-tier cache absorbs every repeat order, which
// is what keeps Routes (≈ $5 / 1000 calls) near-free at our volume.
//
// It never blocks or errors the caller: any HTTP error, quota rejection, timeout,
// or missing route returns (0, false), and RoadDistanceKm's winding-factor
// haversine fallback answers instead. So enabling it (by setting GOOGLE_MAPS_API_KEY)
// is a pure accuracy upgrade with a guaranteed floor.

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/homechef/api/config"
)

const googleRoutesEndpoint = "https://routes.googleapis.com/directions/v2:computeRoutes"

// routesFieldMask limits the response to the one number we use — smaller, faster,
// and (on the Routes API) a cheaper billing SKU than requesting full routes.
const routesFieldMask = "routes.distanceMeters"

// InitDeliveryRouter installs the real Google Routes distance provider when a
// Maps API key is configured, wrapped in the two-tier cache so a trip is billed
// at most once ever. With no key it's a no-op and RoadDistanceKm keeps using the
// winding-factor haversine fallback — so distance always works, keys just make it
// exact. Call once at startup, after Redis + DB are up (the cache uses both).
func InitDeliveryRouter() {
	key := config.AppConfig.GoogleMapsAPIKey
	if key == "" {
		log.Println("delivery router: GOOGLE_MAPS_API_KEY not set — using haversine fallback")
		return
	}
	SetDeliveryRouter(NewCachedRouter(newGoogleRoutesRouter(key), "google"))
	log.Println("delivery router: Google Routes provider enabled (cached)")
}

// googleRoutesRouter calls the Routes API for point-to-point driving distance.
type googleRoutesRouter struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

// newGoogleRoutesRouter builds the production router (real endpoint, bounded
// timeout). Wrap it with NewCachedRouter before installing via SetDeliveryRouter.
func newGoogleRoutesRouter(apiKey string) *googleRoutesRouter {
	return newGoogleRoutesRouterWithEndpoint(apiKey, googleRoutesEndpoint, &http.Client{Timeout: 4 * time.Second})
}

// newGoogleRoutesRouterWithEndpoint is the injectable constructor used by tests
// to point at an httptest server.
func newGoogleRoutesRouterWithEndpoint(apiKey, endpoint string, client *http.Client) *googleRoutesRouter {
	return &googleRoutesRouter{apiKey: apiKey, endpoint: endpoint, client: client}
}

type routesLatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type routesWaypoint struct {
	Location struct {
		LatLng routesLatLng `json:"latLng"`
	} `json:"location"`
}

type routesRequest struct {
	Origin      routesWaypoint `json:"origin"`
	Destination routesWaypoint `json:"destination"`
	TravelMode  string         `json:"travelMode"`
}

type routesResponse struct {
	Routes []struct {
		DistanceMeters float64 `json:"distanceMeters"`
	} `json:"routes"`
}

func waypoint(lat, lng float64) routesWaypoint {
	var w routesWaypoint
	w.Location.LatLng = routesLatLng{Latitude: lat, Longitude: lng}
	return w
}

// RoadDistanceKm returns the driving distance in km, or (0, false) on any failure
// so the caller falls back. Never returns an error.
func (r *googleRoutesRouter) RoadDistanceKm(ctx context.Context, fromLat, fromLng, toLat, toLng float64) (float64, bool) {
	body, err := json.Marshal(routesRequest{
		Origin:      waypoint(fromLat, fromLng),
		Destination: waypoint(toLat, toLng),
		TravelMode:  "DRIVE",
	})
	if err != nil {
		return 0, false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", r.apiKey)
	req.Header.Set("X-Goog-FieldMask", routesFieldMask)

	resp, err := r.client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	var out routesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, false
	}
	if len(out.Routes) == 0 || out.Routes[0].DistanceMeters <= 0 {
		return 0, false
	}
	return out.Routes[0].DistanceMeters / 1000.0, true
}
