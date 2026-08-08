package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func geoProvider(t *testing.T, body string, status int) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	prev := geoProviderURL
	geoProviderURL = srv.URL + "/%s"
	t.Cleanup(func() {
		geoProviderURL = prev
		srv.Close()
	})
	return srv, &hits
}

func TestLookupIPLocation_ResolvesCountryAndCity(t *testing.T) {
	withMiniredis(t)
	geoProvider(t, `{"status":"success","country":"India","countryCode":"IN","city":"Bengaluru"}`, http.StatusOK)

	loc := LookupIPLocation(context.Background(), "49.207.1.1")

	require.Equal(t, "India", loc.Country)
	require.Equal(t, "Bengaluru", loc.City)
}

// A login must never be delayed or failed to geolocate it, so private and
// malformed addresses short-circuit without touching the provider.
func TestLookupIPLocation_SkipsPrivateAndMalformedAddresses(t *testing.T) {
	withMiniredis(t)
	_, hits := geoProvider(t, `{"status":"success","country":"India"}`, http.StatusOK)

	for _, ip := range []string{"", "10.0.0.4", "192.168.1.9", "127.0.0.1", "::1", "not-an-ip"} {
		loc := LookupIPLocation(context.Background(), ip)
		require.True(t, loc.Empty(), "expected no location for %q", ip)
	}
	require.Equal(t, int32(0), *hits)
}

func TestLookupIPLocation_ProviderFailureDegradesQuietly(t *testing.T) {
	withMiniredis(t)
	geoProvider(t, `upstream exploded`, http.StatusInternalServerError)

	loc := LookupIPLocation(context.Background(), "49.207.1.1")

	require.True(t, loc.Empty())
}

func TestLookupIPLocation_CachesTheAnswer(t *testing.T) {
	withMiniredis(t)
	_, hits := geoProvider(t, `{"status":"success","country":"India","countryCode":"IN","city":"Bengaluru"}`, http.StatusOK)

	first := LookupIPLocation(context.Background(), "49.207.1.1")
	second := LookupIPLocation(context.Background(), "49.207.1.1")

	require.Equal(t, first, second)
	require.Equal(t, int32(1), *hits, "a repeat lookup must be served from cache")
}

// Describe() is what the email prints, so it must never render a bare comma or
// an empty line when the provider knew nothing.
func TestIPLocationDescribe(t *testing.T) {
	require.Equal(t, "Bengaluru, India", IPLocation{City: "Bengaluru", Country: "India"}.Describe())
	require.Equal(t, "India", IPLocation{Country: "India"}.Describe())
	require.Equal(t, "Bengaluru", IPLocation{City: "Bengaluru"}.Describe())
	require.Equal(t, "Unknown location", IPLocation{}.Describe())
}
