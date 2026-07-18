package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// LocationHandler exposes the public, India-only reference-data API:
// countries, states, cities, and PIN codes. The endpoints follow
// mark8ly's platform-api/location shape — ISO codes throughout, JSON
// envelope of {"data": ...}, simple GET semantics, no auth.
//
// All data is preloaded by the seeder at boot (see database.go), so
// these handlers do nothing more than translate path params into
// indexed queries.
type LocationHandler struct{}

func NewLocationHandler() *LocationHandler { return &LocationHandler{} }

// dataEnvelope is the shared response shape. Lists are always
// non-nil (`[]` rather than `null`) so clients can iterate without
// a null guard.
type dataEnvelope struct {
	Data any `json:"data"`
}

func ok[T any](c *gin.Context, items []T) {
	if items == nil {
		items = []T{}
	}
	c.JSON(http.StatusOK, dataEnvelope{Data: items})
}

// GetCountries returns every seeded country, sorted by name.
func (h *LocationHandler) GetCountries(c *gin.Context) {
	var countries []models.Country
	if err := database.DB.WithContext(c.Request.Context()).
		Order("name ASC").
		Find(&countries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_countries_failed", "message": err.Error()})
		return
	}
	ok(c, countries)
}

// GetStates returns every state for a country, sorted by name. Country
// code is matched case-insensitively (callers commonly send "in" or "IN").
func (h *LocationHandler) GetStates(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("countryCode")))
	if len(code) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_country_code", "message": "country code must be a 2-letter ISO code"})
		return
	}
	var states []models.State
	if err := database.DB.WithContext(c.Request.Context()).
		Where("country_code = ?", code).
		Order("name ASC").
		Find(&states).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_states_failed", "message": err.Error()})
		return
	}
	ok(c, states)
}

// GetCities returns every city for a state, ordered with the IsMajor
// flag first so big metros surface at the top of the picker.
//
// Route is GET /locations/states/:stateCode/cities. The stateCode path
// param is the local state code (e.g. "MH"); callers can override the
// country with ?country=IN (defaults to IN since this is the only
// seeded country today).
func (h *LocationHandler) GetCities(c *gin.Context) {
	stateCode := strings.ToUpper(strings.TrimSpace(c.Param("stateCode")))
	countryCode := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("country", "IN")))
	if stateCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_state_code", "message": "state code is required"})
		return
	}
	stateID := countryCode + "-" + stateCode

	// Ensure the state exists so we can return a clean 404 rather than an
	// empty list that the caller can't distinguish from "no cities yet".
	var state models.State
	if err := database.DB.WithContext(c.Request.Context()).
		Where("id = ?", stateID).
		First(&state).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "state_not_found", "message": "state " + stateID + " does not exist"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup_state_failed", "message": err.Error()})
		return
	}

	var cities []models.City
	if err := database.DB.WithContext(c.Request.Context()).
		Where("state_id = ?", state.ID).
		Order("is_major DESC, name ASC").
		Find(&cities).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_cities_failed", "message": err.Error()})
		return
	}
	ok(c, cities)
}

// GetPostcodes returns every PIN for a named city. The mobile client
// builds this URL by URL-encoding the city name; an optional ?state=
// query disambiguates same-named cities (rare in India but defensive).
func (h *LocationHandler) GetPostcodes(c *gin.Context) {
	cityName := strings.TrimSpace(c.Param("cityName"))
	stateCode := strings.ToUpper(strings.TrimSpace(c.Query("state")))
	if cityName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_city_name", "message": "city name is required"})
		return
	}

	q := database.DB.WithContext(c.Request.Context()).Model(&models.City{}).
		Where("LOWER(cities.name) = LOWER(?)", cityName)
	if stateCode != "" {
		// The state_id encodes the parent country, so a trailing -<stateCode>
		// match against either "IN-MH" or "MH" works equally well via LIKE.
		q = q.Where("cities.state_id LIKE ?", "%-"+stateCode)
	}
	var city models.City
	if err := q.First(&city).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "city_not_found", "message": "city " + cityName + " does not exist"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup_city_failed", "message": err.Error()})
		return
	}

	var postcodes []models.Postcode
	if err := database.DB.WithContext(c.Request.Context()).
		Where("city_id = ?", city.ID).
		Order("code ASC").
		Find(&postcodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_postcodes_failed", "message": err.Error()})
		return
	}
	ok(c, postcodes)
}

// PostcodeSearchResult flattens the postcode → city → state chain into
// a single autocomplete row. The mobile chip-picker shows
// `560034 — Koramangala, Bengaluru, Karnataka` so all three names
// need to come back in one round-trip.
type PostcodeSearchResult struct {
	Code      string `json:"code"`
	AreaName  string `json:"areaName"`
	CityID    string `json:"cityId"`
	CityName  string `json:"cityName"`
	StateID   string `json:"stateId"`
	StateName string `json:"stateName"`
}

// SearchPostcodes powers the PIN-code autocomplete on the mobile
// address forms. It matches the query as a code prefix (typical: user
// types "5600") or a substring of the area name (typical: user types
// "Koramang"). Capped at 20 results so the mobile picker isn't
// overwhelmed.
func (h *LocationHandler) SearchPostcodes(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query_too_short", "message": "search query must be at least 2 characters"})
		return
	}

	var results []PostcodeSearchResult
	if err := database.DB.WithContext(c.Request.Context()).Raw(`
		SELECT p.code,
		       p.area_name,
		       c.id   AS city_id,
		       c.name AS city_name,
		       s.id   AS state_id,
		       s.name AS state_name
		FROM postcodes p
		JOIN cities c ON c.id = p.city_id
		JOIN states s ON s.id = c.state_id
		WHERE p.code LIKE ? OR LOWER(p.area_name) LIKE LOWER(?)
		ORDER BY p.code ASC
		LIMIT 20
	`, q+"%", "%"+q+"%").Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search_postcodes_failed", "message": err.Error()})
		return
	}
	if results == nil {
		results = []PostcodeSearchResult{}
	}
	c.JSON(http.StatusOK, dataEnvelope{Data: results})
}

// photonClient is reused across requests so connection pooling kicks in.
var photonClient = &http.Client{Timeout: 4 * time.Second}

const (
	photonAPI       = "https://photon.komoot.io/api/"
	photonUserAgent = "homechef-api (+https://fe3dr.com)"
)

// photonProperties is the subset of Photon's per-feature properties we
// consume. Photon's response shape: see https://photon.komoot.io/
type photonProperties struct {
	Name        string `json:"name,omitempty"`
	Street      string `json:"street,omitempty"`
	HouseNumber string `json:"housenumber,omitempty"`
	Postcode    string `json:"postcode,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"countrycode,omitempty"`
}

// photonGeometry is GeoJSON Point geometry. Coordinates are [lon, lat].
type photonGeometry struct {
	Coordinates []float64 `json:"coordinates"`
}

type photonFeature struct {
	Properties photonProperties `json:"properties"`
	Geometry   photonGeometry   `json:"geometry"`
}

type photonResponse struct {
	Features []photonFeature `json:"features"`
}

// AddressSuggestion is the flattened shape the mobile autocomplete
// renders. Mirrors mark8ly/apps/storefront/app/api/locations/autocomplete
// so a shared mobile-shared address picker can target both products.
// Lat/Lon come from Photon's geometry so the client can persist real
// coordinates on the address (powers delivery-zone checks + 3PL quotes).
type AddressSuggestion struct {
	Description string  `json:"description"`
	Line1       string  `json:"line1"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Postal      string  `json:"postal"`
	Country     string  `json:"country"`
	Lat         float64 `json:"lat,omitempty"`
	Lon         float64 `json:"lon,omitempty"`
}

// AutocompleteAddresses powers the mobile address picker's autocomplete on top
// of our seeded PIN registry.
//
// Provider order (India-only — HomeChef serves only India today, so surfacing
// foreign matches would let a customer pick an unusable drop point):
//  1. Mappls (MapmyIndia) when credentials are set — India-native, flat/house-
//     level accuracy + precise coordinates, so a real drop resolves to the RIGHT
//     delivery distance/fee. This is why an accurate geocoder matters: the
//     coordinates it returns are what the delivery-fee quote is priced on.
//  2. Photon (OpenStreetMap) fallback — free, no key. It indexes streets/POIs,
//     not flat numbers, so we try progressively-broader query variants to still
//     land the customer on a real area to refine.
//
// A provider that errors or returns nothing degrades to the next; an empty final
// result keeps the seeded /postcodes/search fallback usable.
func (h *LocationHandler) AutocompleteAddresses(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 3 {
		c.JSON(http.StatusOK, dataEnvelope{Data: []AddressSuggestion{}})
		return
	}

	// Primary: Mappls. One call returns flat-level matches with coordinates.
	if mapplsConfigured() {
		if out, err := fetchMapplsSuggestions(c.Request.Context(), q); err == nil && len(out) > 0 {
			c.JSON(http.StatusOK, dataEnvelope{Data: out})
			return
		}
		// Mappls errored or found nothing — fall through to Photon.
	}

	// Fallback: Photon. Try progressively-simpler variants (strip the flat/house
	// number, then fall back to the trailing city/state) until one finds the area,
	// so a full "Flat 104 Asima Residency Kalarahanga Bhubaneswar Odisha" still
	// yields a real, coordinate-bearing match instead of an empty box.
	for _, variant := range photonQueryVariants(q) {
		out, err := fetchPhotonSuggestions(c.Request.Context(), variant)
		if err != nil {
			continue // Photon hiccup on this variant — try the next.
		}
		if len(out) > 0 {
			c.JSON(http.StatusOK, dataEnvelope{Data: out})
			return
		}
	}
	// Nothing matched (or both providers are down) — empty so the seeded
	// /postcodes/search fallback stays usable.
	c.JSON(http.StatusOK, dataEnvelope{Data: []AddressSuggestion{}})
}

// photonNoisePrefix matches a leading flat/house-number token ("Flat 104",
// "No. 5", "H.No 3", "Plot 12", "#7") that Photon can't match on.
var photonNoisePrefix = regexp.MustCompile(`(?i)^(flat|plot|house|door|shop|h\.?\s*no\.?|no\.?|#)\.?\s*\d*[a-z]?[-/]?\d*\b[\s,]*`)

// photonQueryVariants returns the query plus up to a few progressively-broader
// fallbacks (most-specific first): drop a leading flat/house number, then the
// trailing city+state, then just the city. First non-empty wins in the caller.
func photonQueryVariants(q string) []string {
	norm := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	q = norm(q)
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		s = norm(s)
		k := strings.ToLower(s)
		if len(s) >= 3 && !seen[k] {
			seen[k] = true
			out = append(out, s)
		}
	}
	add(q)
	stripped := norm(photonNoisePrefix.ReplaceAllString(q, ""))
	add(stripped)
	// Area fallbacks from the trailing tokens (usually "<city> <state>", then "<city>").
	tokens := strings.Fields(stripped)
	if len(tokens) >= 3 {
		add(strings.Join(tokens[len(tokens)-3:], " "))
	}
	if len(tokens) >= 2 {
		add(strings.Join(tokens[len(tokens)-2:], " "))
	}
	if len(tokens) >= 1 {
		add(tokens[len(tokens)-1])
	}
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

// fetchPhotonSuggestions queries Photon for one term and returns India-only,
// coordinate-bearing suggestions. Any transport/parse error is returned so the
// caller can try the next variant.
func fetchPhotonSuggestions(reqCtx context.Context, q string) ([]AddressSuggestion, error) {
	pu, err := url.Parse(photonAPI)
	if err != nil {
		return nil, err
	}
	qs := pu.Query()
	qs.Set("q", q)
	qs.Set("limit", "8")
	qs.Set("lang", "en")
	pu.RawQuery = qs.Encode()

	ctx, cancel := context.WithTimeout(reqCtx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pu.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", photonUserAgent)

	resp, err := photonClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("photon status %d", resp.StatusCode)
	}
	var body photonResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	out := make([]AddressSuggestion, 0, len(body.Features))
	for _, f := range body.Features {
		p := f.Properties
		if !strings.EqualFold(p.CountryCode, "IN") { // India-only guard.
			continue
		}
		s := AddressSuggestion{
			Line1:   buildPhotonLine1(p),
			City:    strings.TrimSpace(p.City),
			Region:  strings.TrimSpace(p.State),
			Postal:  strings.TrimSpace(p.Postcode),
			Country: strings.ToUpper(p.CountryCode),
		}
		s.Description = buildPhotonDescription(s, p)
		if s.Description == "" || (s.Line1 == "" && s.City == "") {
			continue
		}
		if len(f.Geometry.Coordinates) >= 2 { // GeoJSON [lon, lat].
			s.Lon = f.Geometry.Coordinates[0]
			s.Lat = f.Geometry.Coordinates[1]
		}
		out = append(out, s)
	}
	return out, nil
}

func buildPhotonLine1(p photonProperties) string {
	num := strings.TrimSpace(p.HouseNumber)
	street := strings.TrimSpace(p.Street)
	if street == "" {
		street = strings.TrimSpace(p.Name)
	}
	if num != "" && street != "" {
		return fmt.Sprintf("%s %s", num, street)
	}
	if street != "" {
		return street
	}
	return num
}

func buildPhotonDescription(s AddressSuggestion, p photonProperties) string {
	parts := []string{s.Line1, s.City, s.Region, s.Postal, strings.TrimSpace(p.Country)}
	cleaned := parts[:0]
	for _, x := range parts {
		x = strings.TrimSpace(x)
		if x != "" {
			cleaned = append(cleaned, x)
		}
	}
	return strings.Join(cleaned, ", ")
}
