// Package cache provides cache functionality for the application
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/redis/go-redis/v9"
)

// RedisCache provides an interface for interacting with the application cache.
type RedisCache struct {
	cache *redis.Client
}

// Cache is an in-memory instance implementation of RedisCache.
var Cache = RedisCache{
	cache: nil,
}

// InitRedisCache initializes the application's Redis cache and in-memory instance.
func InitRedisCache(ctx context.Context) error {
	opts, err := redis.ParseURL(config.Envs.RedisURL)
	if err != nil {
		return err
	}
	Cache.cache = redis.NewClient(opts)
	if _, err := Cache.cache.Ping(ctx).Result(); err != nil {
		return err
	}
	return nil
}

// Get retrieves a value from the redis cache.
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	val, err := r.cache.Get(ctx, key).Result()
	return val, err
}

// GetBytes retrieves a value from the redis cache in byte format.
func (r *RedisCache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	val, err := r.cache.Get(ctx, key).Bytes()
	return val, err
}

// Set stores a value into the provided key in the redis cache.
func (r *RedisCache) Set(ctx context.Context, key string, val any, ttl time.Duration) error {
	err := r.cache.Set(ctx, key, val, ttl).Err()
	return err
}

// Mset stores a collection of key value pairs into the redis cache atomically.
func (r *RedisCache) Mset(ctx context.Context, pairs ...any) error {
	if len(pairs)%2 != 0 {
		return fmt.Errorf("[Cache] odd number of arguments passed to Mset: got %d", len(pairs))
	}
	err := r.cache.MSet(ctx, pairs...).Err()
	return err
}
