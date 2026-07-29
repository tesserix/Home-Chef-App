package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	geocodePhotonAPI = "https://photon.komoot.io/api/"
	geocodeUserAgent = "homechef-api (+https://fe3dr.com)"
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
	// Read the WHOLE body: a single Read returns only what happens to be buffered,
	// so a chunked response was silently truncated into invalid JSON and parsed as
	// "no match". Capped so a hostile/huge response can't balloon memory.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, 0, false
	}
	return parsePhotonGeocode(body)
}

// GeocodeAddressParts forward-geocodes a STRUCTURED address, trying progressively
// coarser variants until one matches.
//
// Photon indexes streets and localities, not flats. A full Indian address like
// "F 104 Ashima Residency, Kalarahanga, Patia, Bhubaneswar, Odisha, 751024"
// returns ZERO features, while the same address minus the unit line resolves
// immediately. The old single-shot call passed the full string and treated the
// empty result as "no coordinates", so every kitchen whose address carried a
// flat/house number stayed at lat/lng 0 — invisible to the bounding-box filter in
// GetChefs, i.e. "No chefs found" for every customer with a location set.
//
// Variants run most-precise first so a resolvable full address still wins.
// Returns ok=false only when even the city/state form matches nothing.
func GeocodeAddressParts(line1, line2, city, state, postal string) (lat, lng float64, ok bool) {
	for _, q := range geocodeCandidates(line1, line2, city, state, postal) {
		if lat, lng, ok = GeocodeAddress(q); ok {
			return lat, lng, true
		}
	}
	return 0, 0, false
}

// geocodeCandidates builds the ordered, de-duplicated query list for
// GeocodeAddressParts. Pure, so the fallback ordering is unit-tested without
// touching the network.
func geocodeCandidates(line1, line2, city, state, postal string) []string {
	join := func(parts ...string) string {
		kept := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				kept = append(kept, p)
			}
		}
		return strings.Join(kept, ", ")
	}
	// Dropping line1 (the flat/house line) is the high-value fallback; dropping
	// line2 instead covers the inverse convention, where the locality sits on
	// line1 and the unit on line2.
	out := make([]string, 0, 5)
	seen := map[string]bool{}
	for _, q := range []string{
		join(line1, line2, city, state, postal),
		join(line2, city, state, postal),
		join(line1, city, state, postal),
		join(city, state, postal),
		join(city, state),
	} {
		if q == "" || seen[q] {
			continue
		}
		seen[q] = true
		out = append(out, q)
	}
	return out
}
