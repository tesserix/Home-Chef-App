package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A successful Routes response yields road distance in km, and the request is a
// well-formed computeRoutes call carrying the API key + field mask.
func TestGoogleRoutesRouter_ParsesDistance(t *testing.T) {
	var gotKey, gotMask, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Goog-Api-Key")
		gotMask = r.Header.Get("X-Goog-FieldMask")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		// 5300 m → 5.3 km.
		_, _ = w.Write([]byte(`{"routes":[{"distanceMeters":5300}]}`))
	}))
	defer srv.Close()

	r := newGoogleRoutesRouterWithEndpoint("test-key", srv.URL, srv.Client())
	km, ok := r.RoadDistanceKm(context.Background(), 12.9716, 77.5946, 12.9352, 77.6245)
	require.True(t, ok)
	require.InDelta(t, 5.3, km, 1e-9)

	require.Equal(t, "test-key", gotKey)
	require.Contains(t, gotMask, "distanceMeters", "field mask limits the response to what we need")
	// The body carries both endpoints as lat/lng.
	require.Contains(t, gotBody, "77.5946")
	require.Contains(t, gotBody, "12.9352")
	// A driving route, not walking/transit.
	require.Contains(t, strings.ToUpper(gotBody), "DRIVE")
}

// A non-200 (quota, auth, bad request) must NOT answer — the caller then falls
// back to the winding-factor haversine, never an error or a blocked checkout.
func TestGoogleRoutesRouter_HTTPErrorFallsThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":429}}`))
	}))
	defer srv.Close()

	r := newGoogleRoutesRouterWithEndpoint("k", srv.URL, srv.Client())
	_, ok := r.RoadDistanceKm(context.Background(), 1, 2, 3, 4)
	require.False(t, ok, "an API error yields no answer → fallback")
}

// An empty routes array (no route found) also falls through, not a bogus 0 km.
func TestGoogleRoutesRouter_NoRouteFallsThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"routes":[]}`))
	}))
	defer srv.Close()

	r := newGoogleRoutesRouterWithEndpoint("k", srv.URL, srv.Client())
	_, ok := r.RoadDistanceKm(context.Background(), 1, 2, 3, 4)
	require.False(t, ok, "no route → no answer, not 0 km")
}

// Sanity: the request body is valid JSON with the expected origin/destination
// shape (guards against a future refactor breaking the Routes contract).
func TestGoogleRoutesRouter_RequestShape(t *testing.T) {
	var payload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&payload)
		_, _ = w.Write([]byte(`{"routes":[{"distanceMeters":100}]}`))
	}))
	defer srv.Close()

	r := newGoogleRoutesRouterWithEndpoint("k", srv.URL, srv.Client())
	_, _ = r.RoadDistanceKm(context.Background(), 10, 20, 30, 40)
	require.Contains(t, payload, "origin")
	require.Contains(t, payload, "destination")
	require.Equal(t, "DRIVE", payload["travelMode"])
}
