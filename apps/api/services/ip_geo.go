package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// IPLocation is a coarse, best-effort origin for a sign-in. Only ever used to
// tell a human "this login came from roughly here" — never for authorization.
type IPLocation struct {
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
}

func (l IPLocation) Empty() bool { return l.Country == "" && l.City == "" }

func (l IPLocation) Describe() string {
	parts := make([]string, 0, 2)
	for _, p := range []string{l.City, l.Country} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return "Unknown location"
	}
	return strings.Join(parts, ", ")
}

// geoProviderURL is a format string taking the IP. Overridable so the provider
// can be swapped without a rebuild, and so tests can point at a local server.
var geoProviderURL = envOr("GEOIP_PROVIDER_URL", "http://ip-api.com/json/%s?fields=status,country,countryCode,city")

const geoCacheTTL = 24 * time.Hour

var geoHTTP = &http.Client{Timeout: 2 * time.Second}

// LookupIPLocation resolves an approximate origin for an IP, cached in Redis.
// It never returns an error: geolocation decorates a security email, and no
// failure of it may delay or break a sign-in.
func LookupIPLocation(ctx context.Context, ip string) IPLocation {
	ip = strings.TrimSpace(ip)
	if !isPublicIP(ip) {
		return IPLocation{}
	}

	cacheKey := "geoip:" + ip
	redis := GetRedisClient()
	if redis.IsConnected() {
		var cached IPLocation
		if err := redis.GetJSON(ctx, cacheKey, &cached); err == nil {
			return cached
		}
	}

	loc, ok := fetchIPLocation(ctx, ip)
	if !ok {
		return IPLocation{}
	}
	if redis.IsConnected() {
		_ = redis.SetJSON(ctx, cacheKey, loc, geoCacheTTL)
	}
	return loc
}

func fetchIPLocation(ctx context.Context, ip string) (IPLocation, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf(geoProviderURL, ip), nil)
	if err != nil {
		return IPLocation{}, false
	}
	resp, err := geoHTTP.Do(req)
	if err != nil {
		return IPLocation{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return IPLocation{}, false
	}
	var out struct {
		Status string `json:"status"`
		IPLocation
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return IPLocation{}, false
	}
	if out.Status != "success" || out.IPLocation.Empty() {
		return IPLocation{}, false
	}
	return out.IPLocation, true
}

func isPublicIP(raw string) bool {
	parsed := net.ParseIP(raw)
	if parsed == nil {
		return false
	}
	return !parsed.IsLoopback() && !parsed.IsPrivate() &&
		!parsed.IsLinkLocalUnicast() && !parsed.IsUnspecified()
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
