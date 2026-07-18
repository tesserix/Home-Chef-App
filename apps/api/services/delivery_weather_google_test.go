package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Clear weather → neutral 1.0 (no surge).
func TestGoogleWeatherProvider_ClearIsNeutral(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// 0% precip, low intensity → clear.
		_, _ = w.Write([]byte(`{"precipitation":{"probability":{"percent":0},"qpf":{"quantity":0}}}`))
	}))
	defer srv.Close()

	p := newGoogleWeatherProviderWithEndpoint("k", srv.URL, srv.Client())
	m, ok := p.WeatherMultiplier(context.Background(), 12.97, 77.59)
	require.True(t, ok)
	require.InDelta(t, 1.0, m, 1e-9, "clear weather adds no surge")
}

// Heavy rain (high probability + high accumulation) → a surge above 1.0.
func TestGoogleWeatherProvider_HeavyRainSurges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"precipitation":{"probability":{"percent":90},"qpf":{"quantity":8}}}`))
	}))
	defer srv.Close()

	p := newGoogleWeatherProviderWithEndpoint("k", srv.URL, srv.Client())
	m, ok := p.WeatherMultiplier(context.Background(), 12.97, 77.59)
	require.True(t, ok)
	require.Greater(t, m, 1.0, "heavy rain raises the estimate")
	require.LessOrEqual(t, m, maxSurgeMultiplier, "still within the surge cap")
}

// The request carries the API key and the drop coordinates.
func TestGoogleWeatherProvider_RequestCarriesKeyAndCoords(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"precipitation":{"probability":{"percent":10},"qpf":{"quantity":0}}}`))
	}))
	defer srv.Close()

	p := newGoogleWeatherProviderWithEndpoint("secret-key", srv.URL, srv.Client())
	_, _ = p.WeatherMultiplier(context.Background(), 12.9716, 77.5946)
	require.Contains(t, gotQuery, "key=secret-key")
	require.Contains(t, gotQuery, "12.9716")
	require.Contains(t, gotQuery, "77.5946")
}

// Any HTTP error must fall through (no answer) → neutral, never blocks.
func TestGoogleWeatherProvider_ErrorFallsThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := newGoogleWeatherProviderWithEndpoint("k", srv.URL, srv.Client())
	_, ok := p.WeatherMultiplier(context.Background(), 1, 2)
	require.False(t, ok)
}
