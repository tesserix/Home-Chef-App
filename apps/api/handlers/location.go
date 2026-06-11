package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

// AutocompleteAddresses proxies the Photon (OpenStreetMap-backed)
// geocoder so the mobile address picker has world-coverage autocomplete
// suggestions on top of our seeded PIN registry. Photon is free,
// requires no API key, and is the same backend mark8ly's address
// fieldset uses.
//
// Country is forced to "IN" since HomeChef serves only India today —
// surfacing foreign matches would let chefs pick an unusable address.
// The mobile client doesn't get a knob to override.
func (h *LocationHandler) AutocompleteAddresses(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 3 {
		c.JSON(http.StatusOK, dataEnvelope{Data: []AddressSuggestion{}})
		return
	}

	// Build the Photon request. limit=8 mirrors mark8ly; any more and
	// the mobile dropdown starts feeling overwhelming.
	pu, err := url.Parse(photonAPI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "photon_url", "message": err.Error()})
		return
	}
	qs := pu.Query()
	qs.Set("q", q)
	qs.Set("limit", "8")
	qs.Set("lang", "en")
	pu.RawQuery = qs.Encode()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pu.String(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "photon_request", "message": err.Error()})
		return
	}
	req.Header.Set("User-Agent", photonUserAgent)

	resp, err := photonClient.Do(req)
	if err != nil {
		// Photon outages shouldn't break our search box — return empty
		// so the seeded /postcodes/search results stay usable.
		c.JSON(http.StatusOK, dataEnvelope{Data: []AddressSuggestion{}})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusOK, dataEnvelope{Data: []AddressSuggestion{}})
		return
	}

	var body photonResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		c.JSON(http.StatusOK, dataEnvelope{Data: []AddressSuggestion{}})
		return
	}

	out := make([]AddressSuggestion, 0, len(body.Features))
	for _, f := range body.Features {
		p := f.Properties
		// India-only guard.
		if !strings.EqualFold(p.CountryCode, "IN") {
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
		// GeoJSON coordinates are [lon, lat]; surface them so the client can
		// persist real coordinates on the address.
		if len(f.Geometry.Coordinates) >= 2 {
			s.Lon = f.Geometry.Coordinates[0]
			s.Lat = f.Geometry.Coordinates[1]
		}
		out = append(out, s)
	}
	c.JSON(http.StatusOK, dataEnvelope{Data: out})
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
