package services

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	geocodePhotonAPI  = "https://photon.komoot.io/api/"
	geocodeUserAgent  = "homechef-api (+https://fe3dr.com)"
)

var geocodeClient = &http.Client{Timeout: 4 * time.Second}

// parsePhotonGeocode pulls lat/lng from the top Photon feature.
// Photon geometry coordinates are [lon, lat].
func parsePhotonGeocode(body []byte) (lat, lng float64, ok bool) {
	var resp struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, 0, false
	}
	for _, f := range resp.Features {
		if len(f.Geometry.Coordinates) == 2 {
			return f.Geometry.Coordinates[1], f.Geometry.Coordinates[0], true
		}
	}
	return 0, 0, false
}

// GeocodeAddress forward-geocodes a free-text address via Photon. Best-effort:
// returns ok=false on blank input, transport error, or no match. Never panics —
// callers treat a miss as "no coordinates yet".
func GeocodeAddress(query string) (lat, lng float64, ok bool) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, 0, false
	}
	pu, err := url.Parse(geocodePhotonAPI)
	if err != nil {
		return 0, 0, false
	}
	params := url.Values{}
	params.Set("q", q)
	params.Set("limit", "1")
	pu.RawQuery = params.Encode()

	req, err := http.NewRequest(http.MethodGet, pu.String(), nil)
	if err != nil {
		return 0, 0, false
	}
	req.Header.Set("User-Agent", geocodeUserAgent)

	resp, err := geocodeClient.Do(req)
	if err != nil {
		return 0, 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, false
	}
	var buf [1 << 16]byte
	n, _ := resp.Body.Read(buf[:])
	return parsePhotonGeocode(buf[:n])
}
