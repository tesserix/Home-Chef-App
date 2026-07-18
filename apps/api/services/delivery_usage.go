package services

// delivery_usage.go — cost telemetry for the delivery-intelligence providers
// (#699). The admin surface in tesserix-home needs to answer "how many paid
// calls are we making, and what is it costing us?" without a metrics backend.
//
// These are process-local counters (reset on restart) — cheap, lock-free, and
// enough for an at-a-glance operational view. The DURABLE picture of distinct
// trips paid for lives in delivery_distance_cache (one row = one trip ever paid),
// so the admin handler combines this live counter with that row count.

import (
	"sync/atomic"

	"github.com/homechef/api/config"
)

// deliveryUsage holds lock-free counters for the metered delivery providers.
// A cache HIT (Redis or Postgres) is free; only a provider call costs money.
type deliveryUsageCounters struct {
	distanceProviderCalls int64 // paid routing-provider calls (a cache miss)
	distanceHotHits       int64 // served from Redis — free
	distanceDurableHits   int64 // served from Postgres/CNPG — free
	weatherProviderCalls  int64 // weather API calls (short-TTL / uncached by design)
}

var deliveryUsage deliveryUsageCounters

func recordDistanceProviderCall() { atomic.AddInt64(&deliveryUsage.distanceProviderCalls, 1) }
func recordDistanceHotHit()       { atomic.AddInt64(&deliveryUsage.distanceHotHits, 1) }
func recordDistanceDurableHit()   { atomic.AddInt64(&deliveryUsage.distanceDurableHits, 1) }

// RecordWeatherProviderCall is exported so the weather signal (#706) increments
// it wherever it lives, keeping all delivery cost telemetry in one place.
func RecordWeatherProviderCall() { atomic.AddInt64(&deliveryUsage.weatherProviderCalls, 1) }

// DeliveryUsageSnapshot is a point-in-time read of the live counters plus the
// derived cache-hit ratio and estimated spend, for the admin cost view.
type DeliveryUsageSnapshot struct {
	DistanceProviderCalls int64   `json:"distanceProviderCalls"`
	DistanceHotHits       int64   `json:"distanceHotHits"`
	DistanceDurableHits   int64   `json:"distanceDurableHits"`
	DistanceCacheHitRatio float64 `json:"distanceCacheHitRatio"` // hits / (hits + provider calls)
	WeatherProviderCalls  int64   `json:"weatherProviderCalls"`

	// Cost estimate in USD, using the configured per-call prices. This is the
	// spend SINCE THE LAST RESTART; the all-time distance spend is derived by the
	// handler from the persisted cache row count.
	DistancePricePerCall float64 `json:"distancePricePerCall"`
	WeatherPricePerCall  float64 `json:"weatherPricePerCall"`
	EstimatedSpendUSD    float64 `json:"estimatedSpendUsd"`
}

// GetDeliveryUsageSnapshot reads the counters atomically and derives the ratio
// and spend. Safe to call concurrently.
func GetDeliveryUsageSnapshot() DeliveryUsageSnapshot {
	calls := atomic.LoadInt64(&deliveryUsage.distanceProviderCalls)
	hot := atomic.LoadInt64(&deliveryUsage.distanceHotHits)
	durable := atomic.LoadInt64(&deliveryUsage.distanceDurableHits)
	weather := atomic.LoadInt64(&deliveryUsage.weatherProviderCalls)

	hits := hot + durable
	ratio := 0.0
	if total := hits + calls; total > 0 {
		ratio = float64(hits) / float64(total)
	}

	distPrice := config.AppConfig.DeliveryDistancePricePerCallUSD
	wxPrice := config.AppConfig.DeliveryWeatherPricePerCallUSD

	return DeliveryUsageSnapshot{
		DistanceProviderCalls: calls,
		DistanceHotHits:       hot,
		DistanceDurableHits:   durable,
		DistanceCacheHitRatio: ratio,
		WeatherProviderCalls:  weather,
		DistancePricePerCall:  distPrice,
		WeatherPricePerCall:   wxPrice,
		EstimatedSpendUSD:     float64(calls)*distPrice + float64(weather)*wxPrice,
	}
}
