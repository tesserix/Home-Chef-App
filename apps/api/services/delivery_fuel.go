package services

// delivery_fuel.go — the fuel-price surge signal (#704).
//
// A chef's per-km rate assumes some pump price; when petrol moves, the same km
// costs more to drive. This turns a live ₹/litre into a multiplier against that
// assumed baseline. The price is refreshed by a daily cron (delivery_fuel_cron.go),
// never on the request path — this provider only reads the stored figure.

import (
	"context"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const (
	// fuelPriceKey holds the last VALIDATED pump price (₹/litre), written by the cron.
	fuelPriceKey = "delivery.fuel_price_per_litre"
	// fuelPriceAtKey is when that price was stored, RFC3339 — drives staleness.
	fuelPriceAtKey = "delivery.fuel_price_observed_at"
	// fuelOverrideKey lets an admin pin the price when a scrape goes wrong. Wins
	// over the scraped value; clearing it returns to the automatic figure.
	fuelOverrideKey = "delivery.fuel_price_override_per_litre"
	// fuelBaselineKey is the price the chefs' per-km rates were set against.
	fuelBaselineKey = "delivery.fuel_baseline_per_litre"
)

const (
	// Sanity band for an Indian pump price (₹/litre). Anything outside is a
	// mis-parse (a phone number, a percentage, a date), not a price.
	fuelPriceMinPerLitre = 50.0
	fuelPriceMaxPerLitre = 200.0
	// A validated price older than this is not trustworthy enough to move money.
	fuelPriceMaxAge = 7 * 24 * time.Hour
)

// omcFuelProvider resolves the fuel multiplier from the stored pump price. It
// does no network I/O: the cron owns refreshing, so a checkout never waits on it.
type omcFuelProvider struct {
	db *gorm.DB
}

// InitFuelProvider installs the fuel surge provider. Enabled whenever a baseline
// is resolvable; the factor stays neutral until the cron stores a price.
func InitFuelProvider() {
	if database.DB == nil {
		return
	}
	SetFuelIndexProvider(&omcFuelProvider{db: database.DB})
}

// FuelMultiplier returns currentPrice / baseline for the country, or (0, false)
// when there is no trustworthy price — the surge layer then stays neutral.
// Only India is priced today; other countries have no configured baseline.
func (p *omcFuelProvider) FuelMultiplier(_ context.Context, country string) (float64, bool) {
	if country != "" && country != "IN" {
		return 0, false
	}
	price, ok := p.currentPrice()
	if !ok {
		return 0, false
	}
	baseline := fuelBaseline(p.db)
	if baseline <= 0 {
		return 0, false
	}
	return price / baseline, true
}

// currentPrice prefers an admin override, else the last validated scraped price
// provided it is recent enough to price against.
func (p *omcFuelProvider) currentPrice() (float64, bool) {
	if v, ok := readFuelSetting(p.db, fuelOverrideKey); ok {
		if price, valid := parseFuelPrice(v); valid {
			return price, true
		}
	}
	raw, ok := readFuelSetting(p.db, fuelPriceKey)
	if !ok {
		return 0, false
	}
	price, valid := parseFuelPrice(raw)
	if !valid || fuelPriceStale(p.db) {
		return 0, false
	}
	return price, true
}

// fuelPriceStale reports whether the stored observation is too old to price on.
// A missing/unparseable timestamp counts as stale — we never guess freshness.
func fuelPriceStale(db *gorm.DB) bool {
	raw, ok := readFuelSetting(db, fuelPriceAtKey)
	if !ok {
		return true
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return true
	}
	return time.Since(at) > fuelPriceMaxAge
}

// parseFuelPrice validates a stored/scraped price into the plausible pump band.
// This is the gate every automatically-obtained number must clear before it can
// move a customer's fee.
func parseFuelPrice(raw string) (float64, bool) {
	price, err := strconv.ParseFloat(raw, 64)
	if err != nil || price < fuelPriceMinPerLitre || price > fuelPriceMaxPerLitre {
		return 0, false
	}
	return price, true
}

// fuelBaseline resolves the pump price the per-km rates assume, from
// PlatformSettings then config. Admin-tunable so rates can be rebased without a
// deploy.
func fuelBaseline(db *gorm.DB) float64 {
	if v, ok := readFuelSetting(db, fuelBaselineKey); ok {
		if b, err := strconv.ParseFloat(v, 64); err == nil && b > 0 {
			return b
		}
	}
	return config.AppConfig.DeliveryFuelBaselinePerLitre
}

func readFuelSetting(db *gorm.DB, key string) (string, bool) {
	if db == nil {
		return "", false
	}
	var s models.PlatformSettings
	if err := db.Where("key = ?", key).First(&s).Error; err != nil {
		return "", false
	}
	if s.Value == "" {
		return "", false
	}
	return s.Value, true
}

// writeFuelSetting upserts one fuel key. Best-effort: a write failure leaves the
// previous value in place, which is the safe outcome.
func writeFuelSetting(db *gorm.DB, key, value string) error {
	if db == nil {
		return nil
	}
	var s models.PlatformSettings
	err := db.Where("key = ?", key).First(&s).Error
	if err != nil {
		return db.Create(&models.PlatformSettings{Key: key, Value: value}).Error
	}
	return db.Model(&s).Update("value", value).Error
}
