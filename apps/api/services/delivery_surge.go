package services

// delivery_surge.go — the surge layer for the self-delivery ESTIMATE (#704 fuel;
// #705 traffic and #706 weather slot in here later).
//
// Surge only affects the "approx max" ESTIMATE shown to the customer, never the
// amount an order is actually charged (that stays ComputeSelfDeliveryFee, the
// deterministic charge basis). The estimate is the worst-case ceiling the chef
// can only bring DOWN at accept, so factoring current conditions into it is
// exactly right: a high-fuel day quotes a higher max, and the chef still chooses
// the final fee within it.
//
// Every factor defaults to 1.0 (neutral) when its provider isn't configured, and
// a provider that can't answer degrades to neutral — the estimate never blocks or
// errors on an external signal.

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// maxSurgeMultiplier caps any single factor (and the combined multiplier) so a
// bad signal can't produce a runaway estimate. 2× is already an extreme day.
const maxSurgeMultiplier = 2.0

// fuelSurgeCacheTTL — fuel prices move at most daily, so a day-long cache keeps
// the (already cheap) fuel lookup near-free and stable within a day.
const fuelSurgeCacheTTL = 24 * time.Hour

// weatherSurgeCacheTTL — weather is a LIVE signal (a storm rolls in within the
// hour), so it's only briefly cached per location: fresh enough to be honest,
// still cheap enough not to hit the weather API on every quote for the same area.
const weatherSurgeCacheTTL = 20 * time.Minute

// trafficSurgeCacheTTL — traffic shifts within minutes, so it's cached only very
// briefly, just to dedupe a burst of quotes for the same area.
const trafficSurgeCacheTTL = 5 * time.Minute

// FuelIndexProvider returns the current fuel-cost multiplier for a country,
// relative to the chef's baseline per-km rate (1.0 = baseline). Implemented over
// a real fuel-price source (#700 provider choice); nil disables fuel surge.
type FuelIndexProvider interface {
	FuelMultiplier(ctx context.Context, country string) (float64, bool)
}

var fuelIndexProvider FuelIndexProvider

// SetFuelIndexProvider installs (or clears, with nil) the fuel-price source.
func SetFuelIndexProvider(p FuelIndexProvider) { fuelIndexProvider = p }

// WeatherProvider returns the current weather-condition multiplier at a drop
// location (1.0 = clear; higher = rain/storm slows and complicates the drive).
// Implemented over a weather API (#700 provider choice); nil disables it.
type WeatherProvider interface {
	WeatherMultiplier(ctx context.Context, lat, lng float64) (float64, bool)
}

var weatherProvider WeatherProvider

// SetWeatherProvider installs (or clears, with nil) the weather source.
func SetWeatherProvider(p WeatherProvider) { weatherProvider = p }

// TrafficProvider returns the current traffic-congestion multiplier for a drop
// location (1.0 = free-flowing; higher = the drive takes longer at peak/gridlock).
// Implemented over a traffic-aware routing/traffic API (#700); nil disables it.
type TrafficProvider interface {
	TrafficMultiplier(ctx context.Context, lat, lng float64) (float64, bool)
}

var trafficProvider TrafficProvider

// SetTrafficProvider installs (or clears, with nil) the traffic source.
func SetTrafficProvider(p TrafficProvider) { trafficProvider = p }

// SurgeFactors is the breakdown of the current surge multipliers. Each is ≥ 1.0.
// Traffic and Weather stay 1.0 until #705/#706 wire their providers.
type SurgeFactors struct {
	Fuel     float64 `json:"fuel"`
	Traffic  float64 `json:"traffic"`
	Weather  float64 `json:"weather"`
	Combined float64 `json:"combined"`
}

// CurrentSurge resolves the live surge factors for a country + drop location.
// Never fails — a missing/erroring provider yields a neutral 1.0. The combined
// multiplier is the product of the factors, clamped to [1.0, maxSurgeMultiplier].
func CurrentSurge(ctx context.Context, country string, dropLat, dropLng float64) SurgeFactors {
	f := SurgeFactors{
		Fuel:    fuelMultiplier(ctx, country),
		Traffic: trafficMultiplier(ctx, dropLat, dropLng),
		Weather: weatherMultiplier(ctx, dropLat, dropLng),
	}
	f.Combined = clampSurge(f.Fuel * f.Traffic * f.Weather)
	return f
}

// fuelMultiplier returns the clamped fuel surge, cached for a day so we hit the
// provider at most once per country per day. Neutral (1.0) without a provider.
func fuelMultiplier(ctx context.Context, country string) float64 {
	if fuelIndexProvider == nil {
		return 1.0
	}
	key := "surge:fuel:" + country
	hot := redisKV{}
	if v, ok := hot.Get(ctx, key); ok {
		if m, err := strconv.ParseFloat(v, 64); err == nil {
			return clampSurge(m)
		}
	}
	m, ok := fuelIndexProvider.FuelMultiplier(ctx, country)
	if !ok {
		return 1.0
	}
	recordFuelProviderCall()
	m = clampSurge(m)
	hot.Set(ctx, key, strconv.FormatFloat(m, 'f', 4, 64), fuelSurgeCacheTTL)
	return m
}

// weatherMultiplier returns the clamped weather surge at a drop location, briefly
// cached per ~1 km cell so a burst of quotes for the same area shares one live
// reading. Neutral (1.0) without a provider or on any error.
func weatherMultiplier(ctx context.Context, lat, lng float64) float64 {
	if weatherProvider == nil {
		return 1.0
	}
	// Coords missing → no location to check → neutral.
	if lat == 0 && lng == 0 {
		return 1.0
	}
	key := fmt.Sprintf("surge:weather:%.2f,%.2f", lat, lng) // ~1 km cell
	hot := redisKV{}
	if v, ok := hot.Get(ctx, key); ok {
		if m, err := strconv.ParseFloat(v, 64); err == nil {
			return clampSurge(m)
		}
	}
	m, ok := weatherProvider.WeatherMultiplier(ctx, lat, lng)
	if !ok {
		return 1.0
	}
	RecordWeatherProviderCall()
	m = clampSurge(m)
	hot.Set(ctx, key, strconv.FormatFloat(m, 'f', 4, 64), weatherSurgeCacheTTL)
	return m
}

// trafficMultiplier returns the clamped traffic surge at a drop location. Traffic
// shifts fast (a jam clears in minutes), so it's cached even more briefly than
// weather — just long enough to share a reading across a burst of quotes for the
// same area. Neutral (1.0) without a provider or on any error.
func trafficMultiplier(ctx context.Context, lat, lng float64) float64 {
	if trafficProvider == nil {
		return 1.0
	}
	if lat == 0 && lng == 0 {
		return 1.0
	}
	key := fmt.Sprintf("surge:traffic:%.2f,%.2f", lat, lng) // ~1 km cell
	hot := redisKV{}
	if v, ok := hot.Get(ctx, key); ok {
		if m, err := strconv.ParseFloat(v, 64); err == nil {
			return clampSurge(m)
		}
	}
	m, ok := trafficProvider.TrafficMultiplier(ctx, lat, lng)
	if !ok {
		return 1.0
	}
	recordTrafficProviderCall()
	m = clampSurge(m)
	hot.Set(ctx, key, strconv.FormatFloat(m, 'f', 4, 64), trafficSurgeCacheTTL)
	return m
}

// clampSurge keeps a multiplier in [1.0, maxSurgeMultiplier]: surge only ever
// raises the estimate (a cheaper-than-baseline signal doesn't discount the chef),
// and never past the cap.
func clampSurge(m float64) float64 {
	if m < 1.0 {
		return 1.0
	}
	if m > maxSurgeMultiplier {
		return maxSurgeMultiplier
	}
	return m
}
