package services

// delivery_fuel_cron.go — the daily pump-price refresh (#704).
//
// Indian OMC prices are published as web pages, not feeds, so this fetches the
// configured page and reads the ₹/litre out of it: a regex fast path first, and
// the support-platform SLM only when that finds nothing (markup changes silently
// break regexes — the model reads prose the pattern no longer matches).
//
// The model never produces a fee. It returns one number, which must then clear
// the same validation any scraped number does: plausible pump band, and no
// implausible jump from the last observation. A rejected reading leaves the
// previous price in place.

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
)

const fuelRefreshInterval = 12 * time.Hour

// fuelMaxDailyMovePercent rejects a reading that moved more than this from the
// last known price. Indian pump prices move in paise; a 15% jump is a mis-parse.
const fuelMaxDailyMovePercent = 15.0

// fuelPricePattern matches "₹ 104.21" / "Rs. 104.21" / "104.21 per litre" style
// figures. Deliberately loose — the numeric gate does the real filtering.
var fuelPricePattern = regexp.MustCompile(`(?i)(?:₹|rs\.?|inr)\s*([0-9]{2,3}\.[0-9]{1,2})|([0-9]{2,3}\.[0-9]{1,2})\s*(?:/|per\s*)(?:l|lit)`)

// StartFuelPriceCron launches the pump-price refresh loop. Fires on start so a
// deploy picks up a price immediately, then every fuelRefreshInterval.
func StartFuelPriceCron(ctx context.Context) {
	if config.AppConfig.DeliveryFuelSourceURL == "" {
		log.Println("fuel-price: DELIVERY_FUEL_SOURCE_URL not set — fuel factor neutral")
		return
	}
	go func() {
		runFuelPriceRefresh(ctx)
		ticker := time.NewTicker(fuelRefreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("fuel-price: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runFuelPriceRefresh(ctx)
			}
		}
	}()
	log.Printf("fuel-price: cron started (interval=%s)", fuelRefreshInterval)
}

// runFuelPriceRefresh fetches, extracts, validates and stores one observation.
// Every failure path is a no-op that keeps the previous price.
func runFuelPriceRefresh(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("fuel-price: refresh panicked: %v", r)
		}
	}()

	body, err := fetchFuelPage(ctx, config.AppConfig.DeliveryFuelSourceURL)
	if err != nil {
		log.Printf("fuel-price: fetch failed: %v", err)
		return
	}

	price, source, ok := extractFuelPrice(ctx, body)
	if !ok {
		log.Printf("fuel-price: no plausible price found in page (%d bytes)", len(body))
		return
	}
	if !fuelMoveAcceptable(price) {
		log.Printf("fuel-price: rejected %.2f from %s — implausible move from last known", price, source)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err := writeFuelSetting(database.DB, fuelPriceKey, strconv.FormatFloat(price, 'f', 2, 64)); err != nil {
		log.Printf("fuel-price: persist price failed: %v", err)
		return
	}
	if err := writeFuelSetting(database.DB, fuelPriceAtKey, now); err != nil {
		log.Printf("fuel-price: persist timestamp failed: %v", err)
		return
	}
	// The surge layer caches the multiplier for a day; drop it so the new price
	// takes effect now rather than up to 24h later.
	redisKV{}.Set(ctx, "surge:fuel:IN", "", time.Millisecond)
	log.Printf("fuel-price: refreshed to ₹%.2f/L via %s", price, source)
}

func fetchFuelPage(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	// A plain UA: several price pages return a stub body to unidentified clients.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; fe3dr-fuel/1.0)")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fuel page returned %d", resp.StatusCode)
	}
	var sb strings.Builder
	// Bounded read: a price page is small, and an unbounded body would let a
	// misconfigured URL exhaust memory.
	if _, err := io.CopyN(&sb, resp.Body, 2<<20); err != nil && err != io.EOF {
		return "", err
	}
	return sb.String(), nil
}

// htmlTagPattern strips markup so the price regex (and the model) see prose
// rather than attributes, which otherwise yield false numeric matches.
var htmlTagPattern = regexp.MustCompile(`(?s)<(script|style)\b.*?</(script|style)>|<[^>]*>`)

func stripHTMLTags(body string) string {
	return strings.Join(strings.Fields(htmlTagPattern.ReplaceAllString(body, " ")), " ")
}

// extractFuelPrice tries the regex first, then the SLM. Returns the price and
// which path produced it, for the log line.
func extractFuelPrice(ctx context.Context, body string) (float64, string, bool) {
	if price, ok := parseFuelFromMarkup(body); ok {
		return price, "regex", true
	}
	if price, ok := extractFuelPriceViaSLM(ctx, body); ok {
		return price, "slm", true
	}
	return 0, "", false
}

// parseFuelFromMarkup pulls candidate figures out of the page and returns the
// first that lands in the plausible pump band.
func parseFuelFromMarkup(body string) (float64, bool) {
	text := stripHTMLTags(body)
	for _, m := range fuelPricePattern.FindAllStringSubmatch(text, 40) {
		for _, g := range m[1:] {
			if g == "" {
				continue
			}
			if price, ok := parseFuelPrice(g); ok {
				return price, true
			}
		}
	}
	return 0, false
}

// fuelMoveAcceptable guards against a reading that parsed cleanly but grabbed the
// wrong number. No previous price ⇒ accept (the band check already passed).
func fuelMoveAcceptable(price float64) bool {
	raw, ok := readFuelSetting(database.DB, fuelPriceKey)
	if !ok {
		return true
	}
	prev, valid := parseFuelPrice(raw)
	if !valid || prev <= 0 {
		return true
	}
	return math.Abs(price-prev)/prev*100 <= fuelMaxDailyMovePercent
}
