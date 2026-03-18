package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type CurrencyHandler struct{}

func NewCurrencyHandler() *CurrencyHandler {
	return &CurrencyHandler{}
}

// ListCurrencies returns all active currencies.
func (h *CurrencyHandler) ListCurrencies(c *gin.Context) {
	var currencies []models.Currency
	database.DB.Where("is_active = ?", true).Order("code").Find(&currencies)
	c.JSON(http.StatusOK, currencies)
}

// GetRates returns exchange rates map with INR as base.
func (h *CurrencyHandler) GetRates(c *gin.Context) {
	rates, err := services.GetRates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch exchange rates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"base":  "INR",
		"rates": rates,
	})
}

// DetectCurrency detects user's currency based on IP geolocation.
func (h *CurrencyHandler) DetectCurrency(c *gin.Context) {
	ip := getClientIP(c)

	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		c.JSON(http.StatusOK, gin.H{
			"countryCode":    "IN",
			"currencyCode":   "INR",
			"currencySymbol": "₹",
		})
		return
	}

	// Call ip-api.com for geolocation
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=countryCode,currency", ip))
	if err != nil {
		// Fallback to INR
		c.JSON(http.StatusOK, gin.H{
			"countryCode":    "IN",
			"currencyCode":   "INR",
			"currencySymbol": "₹",
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusOK, defaultCurrencyDetection())
		return
	}

	var ipData struct {
		CountryCode string `json:"countryCode"`
		Currency    string `json:"currency"`
	}
	if err := json.Unmarshal(body, &ipData); err != nil || ipData.Currency == "" {
		c.JSON(http.StatusOK, defaultCurrencyDetection())
		return
	}

	// Look up currency symbol from our currencies table
	symbol := getCurrencySymbol(ipData.Currency)

	c.JSON(http.StatusOK, gin.H{
		"countryCode":    ipData.CountryCode,
		"currencyCode":   ipData.Currency,
		"currencySymbol": symbol,
	})
}

// SetPreferredCurrency updates the logged-in user's preferred currency.
func (h *CurrencyHandler) SetPreferredCurrency(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		CurrencyCode string `json:"currencyCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "currencyCode is required"})
		return
	}

	code := strings.ToUpper(req.CurrencyCode)

	// Verify currency exists and is active
	var currency models.Currency
	if err := database.DB.Where("code = ? AND is_active = ?", code, true).First(&currency).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or unsupported currency"})
		return
	}

	// Get or create customer profile
	var profile models.CustomerProfile
	result := database.DB.Where("user_id = ?", userID).First(&profile)
	if result.Error != nil {
		profile = models.CustomerProfile{
			UserID:            userID,
			PreferredCurrency: code,
		}
		database.DB.Create(&profile)
	} else {
		database.DB.Model(&profile).Update("preferred_currency", code)
	}

	c.JSON(http.StatusOK, gin.H{
		"currencyCode":   currency.Code,
		"currencySymbol": currency.Symbol,
		"decimalPlaces":  currency.DecimalPlaces,
	})
}

func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For (set by reverse proxies / load balancers)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip := strings.TrimSpace(parts[0])
		if parsedIP := net.ParseIP(ip); parsedIP != nil {
			return ip
		}
	}
	// Check X-Real-IP
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		if parsedIP := net.ParseIP(xri); parsedIP != nil {
			return xri
		}
	}
	// Fall back to RemoteAddr
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	return ip
}

func defaultCurrencyDetection() gin.H {
	return gin.H{
		"countryCode":    "IN",
		"currencyCode":   "INR",
		"currencySymbol": "₹",
	}
}

// getCurrencySymbol looks up the symbol for a currency code from the DB,
// falling back to a built-in map for common currencies.
func getCurrencySymbol(code string) string {
	var currency models.Currency
	if err := database.DB.Where("code = ?", code).First(&currency).Error; err == nil {
		return currency.Symbol
	}

	// Fallback for common currencies
	symbols := map[string]string{
		"INR": "₹", "USD": "$", "EUR": "€", "GBP": "£",
		"AED": "د.إ", "SGD": "S$", "AUD": "A$", "CAD": "C$",
		"JPY": "¥", "CNY": "¥", "KRW": "₩", "THB": "฿",
	}
	if s, ok := symbols[code]; ok {
		return s
	}
	return code
}
