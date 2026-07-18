package services

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
)

// The snapshot must derive the cache-hit ratio and spend correctly from the raw
// counters — this is what the admin cost view shows, so the arithmetic matters.
func TestGetDeliveryUsageSnapshot_RatioAndSpend(t *testing.T) {
	config.AppConfig = &config.Config{
		DeliveryDistancePricePerCallUSD: 0.005,
		DeliveryWeatherPricePerCallUSD:  0.001,
	}
	// Reset counters so the test is independent of run order.
	deliveryUsage = deliveryUsageCounters{}

	// 2 paid distance calls, 6 hot hits, 2 durable hits → 8 hits of 10 = 0.8.
	recordDistanceProviderCall()
	recordDistanceProviderCall()
	for i := 0; i < 6; i++ {
		recordDistanceHotHit()
	}
	recordDistanceDurableHit()
	recordDistanceDurableHit()
	// 3 weather calls (uncached by design).
	RecordWeatherProviderCall()
	RecordWeatherProviderCall()
	RecordWeatherProviderCall()

	s := GetDeliveryUsageSnapshot()
	require.Equal(t, int64(2), s.DistanceProviderCalls)
	require.Equal(t, int64(8), s.DistanceHotHits+s.DistanceDurableHits)
	require.InDelta(t, 0.8, s.DistanceCacheHitRatio, 1e-9, "8 hits of 10 lookups")
	// spend = 2×0.005 + 3×0.001 = 0.013
	require.InDelta(t, 0.013, s.EstimatedSpendUSD, 1e-9)
}

// With no lookups yet, the ratio must be 0 (not NaN from a divide-by-zero).
func TestGetDeliveryUsageSnapshot_EmptyIsZeroNotNaN(t *testing.T) {
	config.AppConfig = &config.Config{}
	deliveryUsage = deliveryUsageCounters{}
	s := GetDeliveryUsageSnapshot()
	require.Equal(t, 0.0, s.DistanceCacheHitRatio)
	require.Equal(t, 0.0, s.EstimatedSpendUSD)
}
