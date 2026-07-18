package handlers

// mappls.go — Mappls (MapmyIndia) address search for autocomplete
// (#address-search). Mappls is India-native and has flat/house-level address
// data + precise coordinates ("eLoc") that Photon (OSM) lacks, so a real drop
// point resolves to the RIGHT delivery distance/fee. It's the primary provider;
// Photon stays as the no-credential fallback.
//
// Auth is OAuth2 client-credentials: exchange client_id/client_secret for a
// bearer token (valid ~24h) and cache it, refreshing just before expiry. Text
// Search returns places WITH coordinates in one call, matching our
// coordinate-bearing AddressSuggestion shape.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/homechef/api/config"
)

const (
	mapplsTokenURL  = "https://outpost.mappls.com/api/security/oauth/token"
	mapplsSearchURL = "https://atlas.mappls.com/api/places/textsearch/json"
)

var mapplsClient = &http.Client{Timeout: 4 * time.Second}

// mapplsConfigured reports whether both OAuth credentials are set.
func mapplsConfigured() bool {
	return config.AppConfig != nil &&
		config.AppConfig.MapplsClientID != "" &&
		config.AppConfig.MapplsClientSecret != ""
}

// mapplsToken caches the OAuth bearer token across requests so we mint one at
// most ~once/day instead of per lookup.
var mapplsToken struct {
	sync.Mutex
	value     string
	expiresAt time.Time
}

type mapplsTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// getMapplsToken returns a valid bearer token, minting (and caching) a new one
// when the cached token is missing or within 60s of expiry.
func getMapplsToken(reqCtx context.Context) (string, error) {
	mapplsToken.Lock()
	defer mapplsToken.Unlock()

	// Reuse while comfortably in-window. We can't call time.Now() at the top-of-
	// package level, but here inside a request it's fine.
	if mapplsToken.value != "" && time.Until(mapplsToken.expiresAt) > 60*time.Second {
		return mapplsToken.value, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", config.AppConfig.MapplsClientID)
	form.Set("client_secret", config.AppConfig.MapplsClientSecret)

	ctx, cancel := context.WithTimeout(reqCtx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, mapplsTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := mapplsClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mappls token status %d", resp.StatusCode)
	}
	var tok mapplsTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("mappls token empty")
	}
	ttl := time.Duration(tok.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = time.Hour // defensive: cache at least an hour if the API omits expiry.
	}
	mapplsToken.value = tok.AccessToken
	mapplsToken.expiresAt = time.Now().Add(ttl)
	return mapplsToken.value, nil
}

// flexFloat parses a JSON value that Mappls sends as either a number (20.29) or a
// string ("20.29").
type flexFloat float64

func (f *flexFloat) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), `"`)
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = flexFloat(v)
	return nil
}

type mapplsLocation struct {
	PlaceName      string    `json:"placeName"`
	PlaceAddress   string    `json:"placeAddress"`
	ELoc           string    `json:"eLoc"`
	Type           string    `json:"type"`
	Latitude       flexFloat `json:"latitude"`
	Longitude      flexFloat `json:"longitude"`
	EntryLatitude  flexFloat `json:"entryLatitude"`
	EntryLongitude flexFloat `json:"entryLongitude"`
}

type mapplsSearchResponse struct {
	SuggestedLocations []mapplsLocation `json:"suggestedLocations"`
}

// fetchMapplsSuggestions runs a Mappls Text Search for India and maps results to
// coordinate-bearing AddressSuggestions. Any transport/parse/auth error is
// returned so the caller can fall back to Photon.
func fetchMapplsSuggestions(reqCtx context.Context, q string) ([]AddressSuggestion, error) {
	token, err := getMapplsToken(reqCtx)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(mapplsSearchURL)
	if err != nil {
		return nil, err
	}
	qs := u.Query()
	qs.Set("query", q)
	qs.Set("region", "IND")
	qs.Set("lang", "en")
	u.RawQuery = qs.Encode()

	ctx, cancel := context.WithTimeout(reqCtx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "bearer "+token)

	resp, err := mapplsClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		// Token went stale early — drop it so the next call re-mints.
		mapplsToken.Lock()
		mapplsToken.value = ""
		mapplsToken.Unlock()
		return nil, fmt.Errorf("mappls search unauthorized")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mappls search status %d", resp.StatusCode)
	}
	var body mapplsSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make([]AddressSuggestion, 0, len(body.SuggestedLocations))
	for _, loc := range body.SuggestedLocations {
		s := mapplsToSuggestion(loc)
		if s.Description == "" || (s.Lat == 0 && s.Lon == 0) {
			continue // no usable point → an unlocatable row can't price delivery.
		}
		out = append(out, s)
	}
	return out, nil
}

// mapplsToSuggestion maps a Mappls location to our AddressSuggestion. Mappls
// returns a flat placeAddress string (no structured components), so city/state/
// postcode are parsed best-effort from the trailing comma-separated tokens
// ("<area>, <city>, <state> <PIN>").
func mapplsToSuggestion(loc mapplsLocation) AddressSuggestion {
	lat, lon := float64(loc.Latitude), float64(loc.Longitude)
	if lat == 0 && lon == 0 { // some POI rows carry only the entry point.
		lat, lon = float64(loc.EntryLatitude), float64(loc.EntryLongitude)
	}
	s := AddressSuggestion{
		Line1:   strings.TrimSpace(loc.PlaceName),
		Country: "IN",
		Lat:     lat,
		Lon:     lon,
	}
	city, region, postal := parseIndianAddressTail(loc.PlaceAddress)
	s.City, s.Region, s.Postal = city, region, postal

	desc := strings.TrimSpace(strings.Join(nonEmptyStrings(loc.PlaceName, loc.PlaceAddress), ", "))
	if desc == "" {
		desc = strings.TrimSpace(loc.PlaceAddress)
	}
	s.Description = desc
	return s
}

// parseIndianAddressTail pulls city / state / 6-digit PIN out of a Mappls flat
// address string, best-effort. Format is typically
// "<area>, <sub-locality>, <city>, <state> <PIN>" or with the PIN comma-split.
func parseIndianAddressTail(addr string) (city, region, postal string) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", "", ""
	}
	parts := strings.Split(addr, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	// A trailing token may be "<state> <PIN>" or a bare "<PIN>". Split the PIN off.
	if n := len(parts); n > 0 {
		last := parts[n-1]
		if pin := trailing6Digits(last); pin != "" {
			postal = pin
			last = strings.TrimSpace(strings.TrimSuffix(last, pin))
			parts[n-1] = last
		}
	}
	// Drop any now-empty tokens.
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	if n := len(cleaned); n >= 1 {
		region = cleaned[n-1] // last non-PIN token ≈ state
	}
	if n := len(cleaned); n >= 2 {
		city = cleaned[n-2] // token before state ≈ city/district
	}
	return city, region, postal
}

// trailing6Digits returns a 6-digit Indian PIN if the string ends in one.
func trailing6Digits(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 6 {
		return ""
	}
	tail := s[len(s)-6:]
	for _, r := range tail {
		if r < '0' || r > '9' {
			return ""
		}
	}
	// Guard against a longer run of digits (e.g. a phone number) being sliced.
	if len(s) > 6 {
		prev := rune(s[len(s)-7])
		if prev >= '0' && prev <= '9' {
			return ""
		}
	}
	return tail
}

func nonEmptyStrings(vals ...string) []string {
	out := make([]string, 0, len(vals))
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}
