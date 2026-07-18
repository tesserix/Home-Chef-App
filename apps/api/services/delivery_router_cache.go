package services

// delivery_router_cache.go — cache road distances so a paid router is called at
// most once per (chef → drop) pair (#701/#699 cost control).
//
// The road distance between two fixed points does not change. So the FIRST time
// a customer orders from a chef to an address we call the provider; every repeat
// order from that same address reuses the cached value and makes ZERO paid calls.
// This is what keeps Google Routes (or any metered provider) near-free at our
// volume — only genuinely time-varying signals (weather, live traffic) are
// re-fetched per order; the distance is not.
//
// The cache is behind a tiny interface so the behaviour is testable without a
// live Redis, and so Redis being down degrades to "just call the router",
// never an error.

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// roadDistanceCacheTTL — distances are effectively permanent (roads rarely
// change), so a long TTL maximises reuse. A month bounds staleness from the odd
// new road / one-way without ever meaningfully mispricing.
const roadDistanceCacheTTL = 30 * 24 * time.Hour

// kvCache is the minimal string cache the router needs. Implemented over the app
// Redis client in prod; faked in tests. All methods are best-effort — a cache
// miss or a down cache must never fail the caller.
type kvCache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration)
}

// redisKV adapts the shared Redis client and is safe when Redis is absent/down
// (Get misses, Set no-ops) — so the pricing path never depends on Redis being up.
type redisKV struct{}

func (redisKV) Get(ctx context.Context, key string) (string, bool) {
	rc := GetRedisClient()
	if rc == nil || !rc.IsConnected() {
		return "", false
	}
	v, err := rc.Get(ctx, key)
	if err != nil {
		return "", false
	}
	return v, true
}

func (redisKV) Set(ctx context.Context, key, value string, ttl time.Duration) {
	rc := GetRedisClient()
	if rc == nil || !rc.IsConnected() {
		return
	}
	_ = rc.Set(ctx, key, value, ttl)
}

// distanceStore is the DURABLE cache (Postgres/CNPG): it survives restarts and
// Redis evictions, so a (chef → address) distance is paid for at most once ever.
// An interface so the two-tier logic is testable without a live DB.
type distanceStore interface {
	lookup(ctx context.Context, key string) (float64, bool)
	save(ctx context.Context, key string, fromLat, fromLng, toLat, toLng, km float64, provider string)
}

// gormDistanceStore persists to delivery_distance_cache. Best-effort: a DB error
// on lookup or save must never fail the fee — it just means a cache miss / a lost
// write, and the provider (or fallback) still answers.
type gormDistanceStore struct{ db *gorm.DB }

func (s gormDistanceStore) lookup(ctx context.Context, key string) (float64, bool) {
	if s.db == nil {
		return 0, false
	}
	var row models.DeliveryDistanceCache
	if err := s.db.WithContext(ctx).Select("distance_km").
		First(&row, "cache_key = ?", key).Error; err != nil {
		return 0, false
	}
	return row.DistanceKm, true
}

func (s gormDistanceStore) save(ctx context.Context, key string, fromLat, fromLng, toLat, toLng, km float64, provider string) {
	if s.db == nil {
		return
	}
	// DoNothing on conflict: two first-orders for the same trip race to insert;
	// the first wins, the second is a harmless no-op (the distance is identical).
	_ = s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&models.DeliveryDistanceCache{
		CacheKey: key, ChefLat: fromLat, ChefLng: fromLng, DropLat: toLat, DropLng: toLng,
		DistanceKm: km, Provider: provider,
	}).Error
}

// cachedRouter fronts a metered DeliveryRouter with two cache tiers:
//
//	Redis (hot, TTL)  →  Postgres (durable, permanent)  →  the provider
//
// so a given (chef → address) trip costs a provider call at most once ever, and
// repeat orders — the common case for a regular customer — are free.
type cachedRouter struct {
	inner    DeliveryRouter
	hot      kvCache
	store    distanceStore
	provider string
	ttl      time.Duration
}

// NewCachedRouter wraps a metered router with the Redis + Postgres cache. Install
// it via SetDeliveryRouter once a real provider (#700) is chosen; the free
// haversine fallback needs no caching. `provider` labels persisted rows so a
// later provider swap can invalidate selectively.
func NewCachedRouter(inner DeliveryRouter, provider string) DeliveryRouter {
	return &cachedRouter{
		inner:    inner,
		hot:      redisKV{},
		store:    gormDistanceStore{db: database.DB},
		provider: provider,
		ttl:      roadDistanceCacheTTL,
	}
}

func (c *cachedRouter) RoadDistanceKm(ctx context.Context, fromLat, fromLng, toLat, toLng float64) (float64, bool) {
	key := roadDistanceCacheKey(fromLat, fromLng, toLat, toLng)

	// 1. Redis hot layer — fastest, no DB round-trip.
	if v, ok := c.hot.Get(ctx, key); ok {
		if km, err := strconv.ParseFloat(v, 64); err == nil && km >= 0 {
			recordDistanceHotHit()
			return km, true
		}
	}
	// 2. Postgres durable layer — survives a Redis flush; warm the hot layer on hit.
	if c.store != nil {
		if km, ok := c.store.lookup(ctx, key); ok && km >= 0 {
			recordDistanceDurableHit()
			c.hot.Set(ctx, key, strconv.FormatFloat(km, 'f', 3, 64), c.ttl)
			return km, true
		}
	}
	// 3. The provider — the only path that costs money; write through to both tiers.
	recordDistanceProviderCall()
	km, ok := c.inner.RoadDistanceKm(ctx, fromLat, fromLng, toLat, toLng)
	if ok && km >= 0 {
		c.hot.Set(ctx, key, strconv.FormatFloat(km, 'f', 3, 64), c.ttl)
		if c.store != nil {
			c.store.save(ctx, key, fromLat, fromLng, toLat, toLng, km, c.provider)
		}
	}
	return km, ok
}

// roadDistanceCacheKey rounds coordinates to ~3 decimal places (~110 m) so that
// the same customer address (which geocodes to the same point) and the same chef
// share ONE cache entry — that is what makes a repeat order a cache hit. Rounding
// also collapses trivially-different GPS jitter onto one key.
func roadDistanceCacheKey(fromLat, fromLng, toLat, toLng float64) string {
	return fmt.Sprintf("roaddist:%.3f,%.3f:%.3f,%.3f", fromLat, fromLng, toLat, toLng)
}
