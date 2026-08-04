package services

// delivery_surge.go — the live surge layer for the self-delivery fee (#704 fuel,
// #705 traffic, #706 weather).
//
// Each factor is ≥ 1.0 and comes from a provider that may be absent, slow or
// wrong; every one degrades to a neutral 1.0 rather than blocking, because this
// sits on the checkout path.
//
// Whether surge reaches the CHARGED fee (not just the displayed estimate) is
// gated by DELIVERY_SURGE_CHARGE_ENABLED — see delivery_fee.go.

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
)

// maxSurgeMultiplier caps any single factor (and the combined multiplier) so a
// bad signal can't produce a runaway fee. 2× is already an extreme day.
const maxSurgeMultiplier = 2.0

// surgeBudget bounds the WHOLE resolution. The factors run concurrently, so a
// checkout waits for the slowest provider rather than their sum, and a blown
// budget yields neutral instead of stalling a customer at payment.
const surgeBudget = 2500 * time.Millisecond

const (
	// Fuel prices move at most daily and are refreshed by cron.
	fuelSurgeCacheTTL = 24 * time.Hour
	// Weather is live but regional — a storm doesn't stop at a street corner.
	weatherSurgeCacheTTL = 20 * time.Minute
	// Traffic shifts within minutes; cached only enough to dedupe a burst.
	trafficSurgeCacheTTL = 5 * time.Minute
	// A failed lookup is cached as neutral so an outage costs one call per cell
	// per minute instead of one per checkout.
	surgeFailureCacheTTL = 60 * time.Second
)

// Cache-cell precision. Traffic is genuinely local (~1 km); weather is regional,
// so a coarser ~11 km cell cuts weather calls sharply with no real loss.
const (
	trafficCellFormat = "surge:traffic:%.2f,%.2f>%.2f,%.2f"
	weatherCellFormat = "surge:weather:%.1f,%.1f"
)

// trafficProbeOffset builds a short synthetic origin when the real one is
// unknown — enough for the router to return a congestion-weighted duration for
// the drop's area. Only a fallback; a known kitchen is always preferred.
const trafficProbeOffset = 0.01

// FuelIndexProvider returns the fuel-cost multiplier for a country, relative to
// the baseline the chef's per-km rate assumes (1.0 = baseline).
type FuelIndexProvider interface {
	FuelMultiplier(ctx context.Context, country string) (float64, bool)
}

var fuelIndexProvider FuelIndexProvider

// SetFuelIndexProvider installs (or clears, with nil) the fuel-price source.
func SetFuelIndexProvider(p FuelIndexProvider) { fuelIndexProvider = p }

// WeatherProvider returns the weather-condition multiplier at a drop location
// (1.0 = clear; higher = rain/storm slows the drive).
type WeatherProvider interface {
	WeatherMultiplier(ctx context.Context, lat, lng float64) (float64, bool)
}

var weatherProvider WeatherProvider

// SetWeatherProvider installs (or clears, with nil) the weather source.
func SetWeatherProvider(p WeatherProvider) { weatherProvider = p }

// TrafficProvider returns the traffic-congestion multiplier for the ACTUAL leg
// (1.0 = free-flowing; higher = the drive takes longer at peak). It takes both
// endpoints because congestion is a property of the route driven, not of a point.
type TrafficProvider interface {
	TrafficMultiplier(ctx context.Context, fromLat, fromLng, toLat, toLng float64) (float64, bool)
}

var trafficProvider TrafficProvider

// SetTrafficProvider installs (or clears, with nil) the traffic source.
func SetTrafficProvider(p TrafficProvider) { trafficProvider = p }

// SurgeFactors is the breakdown of the current surge multipliers, each ≥ 1.0.
type SurgeFactors struct {
	Fuel     float64 `json:"fuel"`
	Traffic  float64 `json:"traffic"`
	Weather  float64 `json:"weather"`
	Combined float64 `json:"combined"`
}

// NeutralSurge is the no-signal result: every factor 1.0.
func NeutralSurge() SurgeFactors {
	return SurgeFactors{Fuel: 1, Traffic: 1, Weather: 1, Combined: 1}
}

// CurrentSurge resolves the live surge factors for the leg origin→drop. Never
// fails: a missing, erroring or slow provider yields a neutral 1.0. The combined
// multiplier is the product, clamped to [1.0, maxSurgeMultiplier].
//
// The origin is the chef's kitchen: traffic is measured on the road actually
// driven, and weather at the drop the driver has to reach.
func CurrentSurge(ctx context.Context, country string, originLat, originLng, dropLat, dropLng float64) SurgeFactors {
	ctx, cancel := context.WithTimeout(ctx, surgeBudget)
	defer cancel()

	f := NeutralSurge()
	var wg sync.WaitGroup
	var mu sync.Mutex

	run := func(assign func(float64), resolve func() float64) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := resolve()
			mu.Lock()
			assign(v)
			mu.Unlock()
		}()
	}

	run(func(v float64) { f.Fuel = v }, func() float64 { return fuelMultiplier(ctx, country) })
	run(func(v float64) { f.Traffic = v }, func() float64 { return trafficMultiplier(ctx, originLat, originLng, dropLat, dropLng) })
	run(func(v float64) { f.Weather = v }, func() float64 { return weatherMultiplier(ctx, dropLat, dropLng) })
	wg.Wait()

	f.Combined = clampSurge(f.Fuel * f.Traffic * f.Weather)
	return f
}

// cachedSurgeFactor is the shared resolve-with-cache path for every factor: hot
// cache, then the provider, then a negative entry so a failing provider isn't
// retried on every checkout. Always returns a usable multiplier.
func cachedSurgeFactor(ctx context.Context, key string, ttl time.Duration, record func(), fetch func() (float64, bool)) float64 {
	hot := redisKV{}
	if v, ok := hot.Get(ctx, key); ok {
		if m, err := strconv.ParseFloat(v, 64); err == nil {
			return clampSurge(m)
		}
	}
	m, ok := fetch()
	if !ok {
		hot.Set(ctx, key, "1.0", surgeFailureCacheTTL)
		return 1.0
	}
	if record != nil {
		record()
	}
	m = clampSurge(m)
	hot.Set(ctx, key, strconv.FormatFloat(m, 'f', 4, 64), ttl)
	return m
}

// fuelMultiplier returns the clamped fuel surge. The provider reads a
// cron-refreshed stored price, so this makes no network call.
func fuelMultiplier(ctx context.Context, country string) float64 {
	if fuelIndexProvider == nil {
		return 1.0
	}
	return cachedSurgeFactor(ctx, "surge:fuel:"+country, fuelSurgeCacheTTL, recordFuelProviderCall,
		func() (float64, bool) { return fuelIndexProvider.FuelMultiplier(ctx, country) })
}

// weatherMultiplier returns the clamped weather surge, cached per regional cell.
func weatherMultiplier(ctx context.Context, lat, lng float64) float64 {
	if weatherProvider == nil || (lat == 0 && lng == 0) {
		return 1.0
	}
	return cachedSurgeFactor(ctx, fmt.Sprintf(weatherCellFormat, lat, lng), weatherSurgeCacheTTL, RecordWeatherProviderCall,
		func() (float64, bool) { return weatherProvider.WeatherMultiplier(ctx, lat, lng) })
}

// trafficMultiplier returns the clamped traffic surge for the origin→drop leg,
// cached per ~1 km cell PAIR: congestion on a route is shared by nearby orders
// out of the same kitchen, but not by a different kitchen serving the same drop.
// A missing origin falls back to probing at the drop alone.
func trafficMultiplier(ctx context.Context, originLat, originLng, dropLat, dropLng float64) float64 {
	if trafficProvider == nil || (dropLat == 0 && dropLng == 0) {
		return 1.0
	}
	if originLat == 0 && originLng == 0 {
		originLat, originLng = dropLat-trafficProbeOffset, dropLng-trafficProbeOffset
	}
	key := fmt.Sprintf(trafficCellFormat, originLat, originLng, dropLat, dropLng)
	return cachedSurgeFactor(ctx, key, trafficSurgeCacheTTL, recordTrafficProviderCall,
		func() (float64, bool) {
			return trafficProvider.TrafficMultiplier(ctx, originLat, originLng, dropLat, dropLng)
		})
}

// clampSurge keeps a multiplier in [1.0, maxSurgeMultiplier]: surge only ever
// raises the fee (a cheap-fuel day doesn't discount the chef), never past the cap.
func clampSurge(m float64) float64 {
	if m < 1.0 {
		return 1.0
	}
	if m > maxSurgeMultiplier {
		return maxSurgeMultiplier
	}
	return m
}
