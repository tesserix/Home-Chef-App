package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/homechef/api/config"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

var (
	redisInstance *RedisClient
	redisOnce     sync.Once
)

func GetRedisClient() *RedisClient {
	redisOnce.Do(func() {
		redisInstance = &RedisClient{}
	})
	return redisInstance
}

func (r *RedisClient) Connect() error {
	opts, err := redis.ParseURL(config.AppConfig.RedisURL)
	if err != nil {
		return err
	}
	r.client = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return err
	}

	log.Println("Redis connection established")
	return nil
}

func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

func (r *RedisClient) IsConnected() bool {
	if r.client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return r.client.Ping(ctx).Err() == nil
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (r *RedisClient) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, string(data), ttl).Err()
}

// IncrAndExpire bumps a counter and sets its TTL on first hit. Used by
// the rate-limiter — INCR is atomic so two pods racing the same key
// agree on the count. EXPIRE only fires on the first hit (count == 1)
// so we don't reset the window on every request. Errors propagate so
// the caller can fail-open per the Redis fail-mode decision.
func (r *RedisClient) IncrAndExpire(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if r.client == nil {
		return 0, redis.ErrClosed
	}
	count, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if count == 1 {
		// Best-effort — if EXPIRE fails the key will simply persist
		// forever, capped by Redis maxmemory eviction. We accept the
		// risk rather than punish the request on a partial failure.
		_ = r.client.Expire(ctx, key, ttl).Err()
	}
	return count, nil
}

// SetNX is a thin wrapper around Redis SET NX EX — atomic create-if-
// absent with TTL. Used by the idempotency middleware to register a
// pending key before the handler runs, so concurrent retries see the
// in-flight marker and either wait or return the cached body.
func (r *RedisClient) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	if r.client == nil {
		return false, redis.ErrClosed
	}
	return r.client.SetNX(ctx, key, value, ttl).Result()
}

// Del deletes a single key. Used by the idempotency middleware to
// clear the in-flight marker after a 5xx so a retry can re-execute.
func (r *RedisClient) Del(ctx context.Context, key string) error {
	if r.client == nil {
		return redis.ErrClosed
	}
	return r.client.Del(ctx, key).Err()
}
