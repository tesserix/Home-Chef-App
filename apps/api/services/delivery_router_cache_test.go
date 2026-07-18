package services

// delivery_router_cache_test.go — the two-tier cost control (#699). A metered
// router must be called at most ONCE per (chef → drop) trip: Redis serves repeat
// requests in-process, Postgres serves them after a restart / Redis flush, and
// only a genuine first-ever trip reaches the provider.

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// countingRouter records how many times the (paid) provider is actually hit.
type countingRouter struct {
	km    float64
	ok    bool
	calls int
}

func (r *countingRouter) RoadDistanceKm(_ context.Context, _, _, _, _ float64) (float64, bool) {
	r.calls++
	return r.km, r.ok
}

// mapCache is an in-memory kvCache (stands in for Redis) so the tier logic is
// deterministic without a live Redis.
type mapCache struct {
	m     map[string]string
	sets  int
	down0 bool // when true, always miss + drop writes (Redis-down simulation)
}

func newMapCache() *mapCache { return &mapCache{m: map[string]string{}} }
func (c *mapCache) Get(_ context.Context, k string) (string, bool) {
	if c.down0 {
		return "", false
	}
	v, ok := c.m[k]
	return v, ok
}
func (c *mapCache) Set(_ context.Context, k, v string, _ time.Duration) {
	c.sets++
	if !c.down0 {
		c.m[k] = v
	}
}

func newCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: glogger.Default.LogMode(glogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_distance_cache (
		id text PRIMARY KEY, cache_key text UNIQUE NOT NULL,
		chef_lat real, chef_lng real, drop_lat real, drop_lng real,
		distance_km real NOT NULL, provider text, created_at datetime, updated_at datetime)`).Error)
	return db
}

func newTestCachedRouter(inner DeliveryRouter, hot kvCache, db *gorm.DB) *cachedRouter {
	return &cachedRouter{inner: inner, hot: hot, store: gormDistanceStore{db: db}, provider: "test", ttl: roadDistanceCacheTTL}
}

const cLat, cLng, dLat, dLng = 12.9716, 77.5946, 12.9352, 77.6245

func TestCachedRouter_FirstCallHitsProvider_ThenBothTiersServe(t *testing.T) {
	inner := &countingRouter{km: 7.2, ok: true}
	hot := newMapCache()
	db := newCacheTestDB(t)
	r := newTestCachedRouter(inner, hot, db)

	// First order for this trip → provider is called once, written to both tiers.
	km, ok := r.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	require.True(t, ok)
	require.Equal(t, 7.2, km)
	require.Equal(t, 1, inner.calls)

	// Repeat order → served from Redis, provider NOT called again.
	km, ok = r.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	require.True(t, ok)
	require.Equal(t, 7.2, km)
	require.Equal(t, 1, inner.calls, "a repeat trip must never re-hit the paid provider")

	// And it was persisted durably.
	var n int64
	db.Model(&models.DeliveryDistanceCache{}).Count(&n)
	require.Equal(t, int64(1), n, "the distance is persisted for future orders")
}

// After a Redis flush (hot cache empty), the durable Postgres layer still serves
// the trip without paying the provider — the whole point of the CNPG cache.
func TestCachedRouter_SurvivesRedisFlush_ViaPostgres(t *testing.T) {
	inner := &countingRouter{km: 5.0, ok: true}
	db := newCacheTestDB(t)

	// First order warms Postgres.
	r1 := newTestCachedRouter(inner, newMapCache(), db)
	r1.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	require.Equal(t, 1, inner.calls)

	// A fresh process (new empty Redis) sharing the same DB.
	freshHot := newMapCache()
	r2 := newTestCachedRouter(inner, freshHot, db)
	km, ok := r2.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	require.True(t, ok)
	require.Equal(t, 5.0, km)
	require.Equal(t, 1, inner.calls, "Postgres served it — the provider is not re-hit after a Redis flush")
	require.GreaterOrEqual(t, freshHot.sets, 1, "and the hot layer is warmed from Postgres for next time")
}

// A saved address geocodes to identical coords every time, so a repeat order is
// always a cache hit — the common regular-customer case. Sub-boundary GPS jitter
// also collapses onto the same key (the key rounds); jitter that happens to
// straddle a rounding boundary is a rare, harmless extra call, not a correctness
// issue — a cache miss just re-computes the same number.
func TestCachedRouter_SameAddress_AlwaysHits(t *testing.T) {
	inner := &countingRouter{km: 3.3, ok: true}
	r := newTestCachedRouter(inner, newMapCache(), newCacheTestDB(t))

	// Identical coords (a saved address) — must hit after the first call.
	r.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	r.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	// Jitter within the same rounding cell (matches the key-rounding test).
	r.RoadDistanceKm(context.Background(), cLat+0.0003, cLng, dLat, dLng-0.0002)
	require.Equal(t, 1, inner.calls, "a saved address (and in-cell jitter) must never re-hit the paid provider")
}

// Redis down must not break anything — Postgres still absorbs the repeat.
func TestCachedRouter_RedisDown_StillCachesInPostgres(t *testing.T) {
	inner := &countingRouter{km: 9.9, ok: true}
	db := newCacheTestDB(t)
	down := newMapCache()
	down.down0 = true
	r := newTestCachedRouter(inner, down, db)

	r.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	r.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	require.Equal(t, 1, inner.calls, "with Redis down, Postgres still prevents the second paid call")
}

// A provider that can't answer is not cached (we don't want to memoise a failure).
func TestCachedRouter_ProviderRefusal_NotCached(t *testing.T) {
	inner := &countingRouter{ok: false}
	db := newCacheTestDB(t)
	r := newTestCachedRouter(inner, newMapCache(), db)

	_, ok := r.RoadDistanceKm(context.Background(), cLat, cLng, dLat, dLng)
	require.False(t, ok)
	var n int64
	db.Model(&models.DeliveryDistanceCache{}).Count(&n)
	require.Equal(t, int64(0), n, "a failed lookup must not be memoised")
}

func TestRoadDistanceCacheKey_RoundsToSharedEntry(t *testing.T) {
	base := roadDistanceCacheKey(cLat, cLng, dLat, dLng)
	jitter := roadDistanceCacheKey(cLat+0.0003, cLng, dLat, dLng-0.0002)
	require.Equal(t, base, jitter, "~100 m jitter shares a key")

	far := roadDistanceCacheKey(cLat+0.01, cLng, dLat, dLng)
	require.NotEqual(t, base, far, "a genuinely different chef location is a different key")
}
