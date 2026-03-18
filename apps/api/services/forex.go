package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const (
	forexCacheKey = "forex:rates:INR"
	forexCacheTTL = 12 * time.Hour

	// Tier 1: OpenExchangeRates (requires OPENEXCHANGERATES_APP_ID)
	openExchangeRatesURL = "https://openexchangerates.org/api/latest.json?base=USD"

	// Tier 2: frankfurter.dev (open source, ECB-backed, no API key)
	frankfurterURL = "https://api.frankfurter.dev/v1/latest?base=INR"

	// Tier 3: exchangeratesapi.io (requires EXCHANGERATES_API_KEY)
	exchangeRatesAPIURL = "https://api.exchangeratesapi.io/v1/latest?base=INR"
)

// --- Response types ---

type openExchangeRatesResponse struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

type frankfurterResponse struct {
	Base  string             `json:"base"`
	Rates map[string]float64 `json:"rates"`
}

type exchangeRatesAPIResponse struct {
	Success bool               `json:"success"`
	Base    string             `json:"base"`
	Rates   map[string]float64 `json:"rates"`
}

// GetRates returns exchange rates with INR as base currency.
// Lookup: Redis → DB (fresh) → OpenExchangeRates → frankfurter.dev → exchangeratesapi.io → stale DB.
func GetRates(ctx context.Context) (map[string]float64, error) {
	rc := GetRedisClient()

	// 1. Redis cache
	if rc.IsConnected() {
		var cached map[string]float64
		if err := rc.GetJSON(ctx, forexCacheKey, &cached); err == nil && len(cached) > 1 {
			return cached, nil
		}
	}

	// 2. Fresh DB rates (< 12h)
	if rates, ok := loadDBRates(forexCacheTTL); ok {
		if rc.IsConnected() {
			_ = rc.SetJSON(ctx, forexCacheKey, rates, forexCacheTTL)
		}
		return rates, nil
	}

	// 3. Fetch from external APIs
	return fetchAndStoreRates(ctx)
}

// RefreshRates bypasses all caches and fetches fresh rates.
func RefreshRates(ctx context.Context) (map[string]float64, error) {
	return fetchAndStoreRates(ctx)
}

// fetchAndStoreRates tries three APIs in priority order, persists results.
func fetchAndStoreRates(ctx context.Context) (map[string]float64, error) {
	// Tier 1: OpenExchangeRates
	rates, err := fetchFromOpenExchangeRates(ctx)
	if err != nil {
		log.Printf("forex: tier-1 OpenExchangeRates failed: %v", err)

		// Tier 2: frankfurter.dev
		rates, err = fetchFromFrankfurter(ctx)
	}
	if err != nil {
		log.Printf("forex: tier-2 frankfurter.dev failed: %v", err)

		// Tier 3: exchangeratesapi.io
		rates, err = fetchFromExchangeRatesAPI(ctx)
	}

	if err != nil {
		log.Printf("forex: all 3 tiers failed: %v — falling back to stale DB", err)
		if stale, ok := loadDBRates(0); ok {
			log.Println("forex: serving stale rates from database")
			return stale, nil
		}
		return map[string]float64{"INR": 1.0}, nil
	}

	persistRates(ctx, rates)
	return rates, nil
}

// --- Tier 1: OpenExchangeRates ---
// Free plan only supports USD base, so we fetch USD-based rates and convert to INR base.
func fetchFromOpenExchangeRates(ctx context.Context) (map[string]float64, error) {
	appID := config.AppConfig.OpenExchangeRatesAppID
	if appID == "" {
		return nil, fmt.Errorf("no OPENEXCHANGERATES_APP_ID configured")
	}

	url := fmt.Sprintf("%s&app_id=%s", openExchangeRatesURL, appID)
	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp openExchangeRatesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	// Convert from USD base to INR base
	inrPerUSD := resp.Rates["INR"]
	if inrPerUSD <= 0 {
		return nil, fmt.Errorf("INR rate missing from OpenExchangeRates response")
	}

	rates := make(map[string]float64, len(resp.Rates)+1)
	rates["INR"] = 1.0
	for code, usdRate := range resp.Rates {
		if code == "INR" || usdRate <= 0 {
			continue
		}
		// 1 INR = (usdRate / inrPerUSD) target currency
		rates[code] = usdRate / inrPerUSD
	}

	log.Printf("forex: fetched %d rates from OpenExchangeRates (tier-1)", len(rates)-1)
	return rates, nil
}

// --- Tier 2: frankfurter.dev ---
func fetchFromFrankfurter(ctx context.Context) (map[string]float64, error) {
	body, err := httpGet(ctx, frankfurterURL)
	if err != nil {
		return nil, err
	}

	var resp frankfurterResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	rates := make(map[string]float64, len(resp.Rates)+1)
	rates["INR"] = 1.0
	for code, rate := range resp.Rates {
		if rate > 0 {
			rates[code] = rate
		}
	}

	log.Printf("forex: fetched %d rates from frankfurter.dev (tier-2)", len(resp.Rates))
	return rates, nil
}

// --- Tier 3: exchangeratesapi.io ---
func fetchFromExchangeRatesAPI(ctx context.Context) (map[string]float64, error) {
	apiKey := config.AppConfig.ExchangeRatesAPIKey
	if apiKey == "" {
		return nil, fmt.Errorf("no EXCHANGERATES_API_KEY configured")
	}

	url := fmt.Sprintf("%s&access_key=%s", exchangeRatesAPIURL, apiKey)
	body, err := httpGet(ctx, url)
	if err != nil {
		return nil, err
	}

	var resp exchangeRatesAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if !resp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	rates := make(map[string]float64, len(resp.Rates)+1)
	rates["INR"] = 1.0
	for code, rate := range resp.Rates {
		if rate > 0 {
			rates[code] = rate
		}
	}

	log.Printf("forex: fetched %d rates from exchangeratesapi.io (tier-3)", len(resp.Rates))
	return rates, nil
}

// --- Helpers ---

func httpGet(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func persistRates(ctx context.Context, rates map[string]float64) {
	now := time.Now().UTC()

	for code, rate := range rates {
		if code == "INR" || rate <= 0 {
			continue
		}
		var existing models.ExchangeRate
		result := database.DB.Where("base_currency = ? AND target_currency = ?", "INR", code).First(&existing)
		if result.Error != nil {
			database.DB.Create(&models.ExchangeRate{
				BaseCurrency:   "INR",
				TargetCurrency: code,
				Rate:           rate,
				FetchedAt:      now,
			})
		} else {
			database.DB.Model(&existing).Updates(map[string]interface{}{
				"rate":       rate,
				"fetched_at": now,
			})
		}
	}

	rc := GetRedisClient()
	if rc.IsConnected() {
		_ = rc.SetJSON(ctx, forexCacheKey, rates, forexCacheTTL)
	}
}

func loadDBRates(maxAge time.Duration) (map[string]float64, bool) {
	var dbRates []models.ExchangeRate
	query := database.DB.Where("base_currency = ?", "INR")
	if maxAge > 0 {
		cutoff := time.Now().UTC().Add(-maxAge)
		query = query.Where("fetched_at > ?", cutoff)
	}

	result := query.Find(&dbRates)
	if result.Error != nil || len(dbRates) == 0 {
		return nil, false
	}

	rates := make(map[string]float64, len(dbRates)+1)
	rates["INR"] = 1.0
	for _, r := range dbRates {
		if r.Rate > 0 {
			rates[r.TargetCurrency] = r.Rate
		}
	}
	return rates, true
}
