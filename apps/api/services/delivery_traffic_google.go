package services

// delivery_traffic_google.go — the real traffic surge provider (#705), derived
// from the SAME Google Routes API (and the same Routes-restricted key) already
// used for distance. No extra key/secret: a traffic-aware route returns both the
// live and the free-flow (static) duration, and their ratio is the congestion
// multiplier.
//
// It folds into the self-delivery ESTIMATE only (never the charge basis), is
// short-TTL cached per ~1 km cell by the surge layer, and degrades to a neutral
// 1.0 on any error — traffic is a nice-to-have signal, never a blocker.

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/homechef/api/config"
)

// googleTrafficProvider calls Routes with a traffic-aware preference and compares
// live vs static duration.
type googleTrafficProvider struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

func newGoogleTrafficProvider(apiKey string) *googleTrafficProvider {
	return newGoogleTrafficProviderWithEndpoint(apiKey, googleRoutesEndpoint, &http.Client{Timeout: 4 * time.Second})
}

func newGoogleTrafficProviderWithEndpoint(apiKey, endpoint string, client *http.Client) *googleTrafficProvider {
	return &googleTrafficProvider{apiKey: apiKey, endpoint: endpoint, client: client}
}

// trafficFieldMask asks for just the two durations we compare.
const trafficFieldMask = "routes.duration,routes.staticDuration"

type trafficRoutesRequest struct {
	Origin            routesWaypoint `json:"origin"`
	Destination       routesWaypoint `json:"destination"`
	TravelMode        string         `json:"travelMode"`
	RoutingPreference string         `json:"routingPreference"`
}

type trafficRoutesResponse struct {
	Routes []struct {
		Duration       string `json:"duration"`
		StaticDuration string `json:"staticDuration"`
	} `json:"routes"`
}

// InitTrafficProvider installs the Google traffic provider when the Routes key is
// configured (it reuses that key). Otherwise the traffic factor stays neutral.
func InitTrafficProvider() {
	key := config.AppConfig.GoogleMapsAPIKey
	if key == "" {
		log.Println("traffic surge: GOOGLE_MAPS_API_KEY not set — traffic factor neutral")
		return
	}
	SetTrafficProvider(newGoogleTrafficProvider(key))
	log.Println("traffic surge: Google Routes traffic provider enabled")
}

// TrafficMultiplier returns liveDuration / staticDuration (congestion factor), or
// (0, false) on any failure so the factor degrades to neutral. The surge layer
// applies the global clamp.
func (p *googleTrafficProvider) TrafficMultiplier(ctx context.Context, lat, lng float64) (float64, bool) {
	// Traffic is directional and needs both endpoints; we treat the chef→drop leg
	// as a short representative hop from just-south-west of the drop to the drop.
	// In practice the surge layer only has the drop location, so we probe traffic
	// AT the drop with a tiny synthetic origin offset — enough for the API to
	// return a congestion-weighted duration for that area.
	from := waypoint(lat-0.01, lng-0.01)
	to := waypoint(lat, lng)

	body, err := json.Marshal(trafficRoutesRequest{
		Origin:            from,
		Destination:       to,
		TravelMode:        "DRIVE",
		RoutingPreference: "TRAFFIC_AWARE",
	})
	if err != nil {
		return 0, false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", p.apiKey)
	req.Header.Set("X-Goog-FieldMask", trafficFieldMask)

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	var out trafficRoutesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, false
	}
	if len(out.Routes) == 0 {
		return 0, false
	}
	live, ok1 := parseRoutesDurationSeconds(out.Routes[0].Duration)
	static, ok2 := parseRoutesDurationSeconds(out.Routes[0].StaticDuration)
	if !ok1 || !ok2 || static <= 0 || live <= 0 {
		return 0, false
	}
	return live / static, true
}

// parseRoutesDurationSeconds parses the Routes API duration format ("753s") into
// seconds. Returns false for empty or malformed values.
func parseRoutesDurationSeconds(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "s") {
		return 0, false
	}
	n, err := strconv.ParseFloat(strings.TrimSuffix(s, "s"), 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
