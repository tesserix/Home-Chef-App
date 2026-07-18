package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Free-flowing traffic (live duration ≈ static) → neutral 1.0.
func TestGoogleTrafficProvider_FreeFlowIsNeutral(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"routes":[{"duration":"600s","staticDuration":"600s"}]}`))
	}))
	defer srv.Close()

	p := newGoogleTrafficProviderWithEndpoint("k", srv.URL, srv.Client())
	m, ok := p.TrafficMultiplier(context.Background(), 12.97, 77.59)
	require.True(t, ok)
	require.InDelta(t, 1.0, m, 1e-9)
}

// Heavy traffic (live duration well above static) → surge above 1.0.
func TestGoogleTrafficProvider_CongestionSurges(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request must ask for a traffic-aware route to get a live duration.
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		require.Contains(t, strings.ToUpper(string(body)), "TRAFFIC_AWARE")
		// 900s live vs 600s static → 1.5×.
		_, _ = w.Write([]byte(`{"routes":[{"duration":"900s","staticDuration":"600s"}]}`))
	}))
	defer srv.Close()

	p := newGoogleTrafficProviderWithEndpoint("k", srv.URL, srv.Client())
	m, ok := p.TrafficMultiplier(context.Background(), 12.97, 77.59)
	require.True(t, ok)
	require.InDelta(t, 1.5, m, 1e-9)
}

// Missing/zero static duration can't yield a ratio → fall through to neutral.
func TestGoogleTrafficProvider_NoStaticFallsThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"routes":[{"duration":"600s"}]}`))
	}))
	defer srv.Close()

	p := newGoogleTrafficProviderWithEndpoint("k", srv.URL, srv.Client())
	_, ok := p.TrafficMultiplier(context.Background(), 1, 2)
	require.False(t, ok)
}

func TestGoogleTrafficProvider_HTTPErrorFallsThrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := newGoogleTrafficProviderWithEndpoint("k", srv.URL, srv.Client())
	_, ok := p.TrafficMultiplier(context.Background(), 1, 2)
	require.False(t, ok)
}

// The duration-string parser handles the Routes "123s" format and rejects junk.
func TestParseRoutesDurationSeconds(t *testing.T) {
	secs, ok := parseRoutesDurationSeconds("753s")
	require.True(t, ok)
	require.InDelta(t, 753, secs, 1e-9)

	_, ok = parseRoutesDurationSeconds("")
	require.False(t, ok)
	_, ok = parseRoutesDurationSeconds("abc")
	require.False(t, ok)
}
