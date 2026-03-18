package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

type LocationHandler struct{}

func NewLocationHandler() *LocationHandler {
	return &LocationHandler{}
}

// GetCountries returns all active countries
func (h *LocationHandler) GetCountries(c *gin.Context) {
	var countries []models.Country
	database.DB.Where("is_active = ?", true).Order("name").Find(&countries)
	c.JSON(http.StatusOK, countries)
}

// GetStates returns all active states for a country
func (h *LocationHandler) GetStates(c *gin.Context) {
	countryCode := c.Param("countryCode")

	var country models.Country
	if err := database.DB.Where("code = ?", countryCode).First(&country).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Country not found"})
		return
	}

	var states []models.State
	database.DB.Where("country_id = ? AND is_active = ?", country.ID, true).Order("name").Find(&states)
	c.JSON(http.StatusOK, states)
}

// GetCities returns all active cities for a state
func (h *LocationHandler) GetCities(c *gin.Context) {
	stateCode := c.Param("stateCode")
	countryCode := c.DefaultQuery("country", "IN")

	var state models.State
	if err := database.DB.
		Joins("JOIN countries ON countries.id = states.country_id").
		Where("states.code = ? AND countries.code = ?", stateCode, countryCode).
		First(&state).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "State not found"})
		return
	}

	var cities []models.City
	database.DB.Where("state_id = ? AND is_active = ?", state.ID, true).Order("is_major DESC, name").Find(&cities)
	c.JSON(http.StatusOK, cities)
}

// GetPostcodes returns all active postcodes for a city
func (h *LocationHandler) GetPostcodes(c *gin.Context) {
	cityName := c.Param("cityName")
	stateCode := c.Query("state")

	query := database.DB.Model(&models.City{})
	if stateCode != "" {
		query = query.Joins("JOIN states ON states.id = cities.state_id").
			Where("cities.name = ? AND states.code = ?", cityName, stateCode)
	} else {
		query = query.Where("name = ?", cityName)
	}

	var city models.City
	if err := query.First(&city).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "City not found"})
		return
	}

	var postcodes []models.Postcode
	database.DB.Where("city_id = ? AND is_active = ?", city.ID, true).Order("code").Find(&postcodes)
	c.JSON(http.StatusOK, postcodes)
}

// SearchPostcodes searches postcodes by code prefix or area name
func (h *LocationHandler) SearchPostcodes(c *gin.Context) {
	q := c.Query("q")
	if len(q) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query must be at least 2 characters"})
		return
	}

	type PostcodeResult struct {
		Code      string `json:"code"`
		AreaName  string `json:"areaName"`
		CityName  string `json:"cityName"`
		StateCode string `json:"stateCode"`
		StateName string `json:"stateName"`
	}

	var results []PostcodeResult
	database.DB.Raw(`
		SELECT p.code, p.area_name, c.name as city_name, s.code as state_code, s.name as state_name
		FROM postcodes p
		JOIN cities c ON c.id = p.city_id
		JOIN states s ON s.id = c.state_id
		WHERE p.is_active = true AND (p.code LIKE ? OR LOWER(p.area_name) LIKE LOWER(?))
		ORDER BY p.code
		LIMIT 20
	`, q+"%", "%"+q+"%").Scan(&results)

	c.JSON(http.StatusOK, results)
}
