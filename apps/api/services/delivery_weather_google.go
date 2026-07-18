package services

// delivery_weather_google.go — the real Google Maps Platform Weather provider
// (#706), a cheap GCP source for current conditions at the drop location.
//
// It implements WeatherProvider over the Weather API currentConditions:lookup
// endpoint and maps precipitation into a surge multiplier: bad weather slows and
// complicates a self-delivery drive, so it raises the customer's "approx max"
// estimate (never the charge basis). It's short-TTL cached per ~1 km cell by the
// surge layer, so a burst of quotes for the same area shares one live reading.
//
// Never blocks or errors: any HTTP error, timeout, or unparseable body returns
// (0, false) and the weather factor degrades to a neutral 1.0.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/homechef/api/config"
)

const googleWeatherEndpoint = "https://weather.googleapis.com/v1/currentConditions:lookup"

// googleWeatherProvider calls the Weather API for current conditions.
type googleWeatherProvider struct {
	apiKey   string
	endpoint string
	client   *http.Client
}

func newGoogleWeatherProvider(apiKey string) *googleWeatherProvider {
	return newGoogleWeatherProviderWithEndpoint(apiKey, googleWeatherEndpoint, &http.Client{Timeout: 4 * time.Second})
}

func newGoogleWeatherProviderWithEndpoint(apiKey, endpoint string, client *http.Client) *googleWeatherProvider {
	return &googleWeatherProvider{apiKey: apiKey, endpoint: endpoint, client: client}
}

// weatherConditions is the slice of the Weather API response we use: how likely
// precipitation is and how much is falling.
type weatherConditions struct {
	Precipitation struct {
		Probability struct {
			Percent float64 `json:"percent"`
		} `json:"probability"`
		QPF struct {
			// Quantity is the accumulation (mm) — a proxy for intensity.
			Quantity float64 `json:"quantity"`
		} `json:"qpf"`
	} `json:"precipitation"`
}

// InitWeatherProvider installs the Google Weather provider when a key is set;
// otherwise weather surge stays neutral. Call once at startup.
func InitWeatherProvider() {
	key := config.AppConfig.GoogleWeatherAPIKey
	if key == "" {
		log.Println("weather surge: GOOGLE_WEATHER_API_KEY not set — weather factor neutral")
		return
	}
	SetWeatherProvider(newGoogleWeatherProvider(key))
	log.Println("weather surge: Google Weather provider enabled")
}

// WeatherMultiplier returns a surge factor in [1.0, ~1.5] derived from the
// precipitation outlook, or (0, false) on any failure so the factor degrades to
// neutral. The surge layer clamps to the global cap.
func (p *googleWeatherProvider) WeatherMultiplier(ctx context.Context, lat, lng float64) (float64, bool) {
	q := url.Values{}
	q.Set("key", p.apiKey)
	q.Set("location.latitude", fmt.Sprintf("%g", lat))
	q.Set("location.longitude", fmt.Sprintf("%g", lng))
	reqURL := p.endpoint + "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, false
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	var wc weatherConditions
	if err := json.NewDecoder(resp.Body).Decode(&wc); err != nil {
		return 0, false
	}
	return weatherMultiplierFromConditions(wc), true
}

// weatherMultiplierFromConditions maps precipitation to a surge factor. Dry ⇒
// 1.0. Otherwise a modest bump scaled by how likely AND how heavy the rain is,
// capped at 1.5 here (the surge layer applies the final global clamp). Kept pure
// so the mapping is unit-testable without HTTP.
func weatherMultiplierFromConditions(wc weatherConditions) float64 {
	prob := wc.Precipitation.Probability.Percent / 100.0 // 0..1
	if prob <= 0 {
		return 1.0
	}
	// Intensity: 0 mm ⇒ 0, ≥5 mm ⇒ full weight (heavy rain).
	intensity := wc.Precipitation.QPF.Quantity / 5.0
	if intensity > 1 {
		intensity = 1
	}
	// Up to +0.4 for a likely, heavy downpour; scaled down for lighter/less-likely.
	m := 1.0 + 0.4*prob*intensity
	if m < 1.0 {
		return 1.0
	}
	if m > 1.5 {
		return 1.5
	}
	return m
}
