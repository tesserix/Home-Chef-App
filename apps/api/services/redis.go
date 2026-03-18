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
	redisOnce    sync.Once
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
