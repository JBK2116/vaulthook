// Package cache provides cache functionality for the application
package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/JBK2116/vaulthook/internal/config"
)

//nolint:gochecknoglobals // Singleton init guard for the shared cache; intentional.
var once sync.Once

// RedisCache provides an interface for interacting with the application cache.
type RedisCache struct {
	cache *redis.Client
}

// Cache is an in-memory instance implementation of RedisCache.
//
//nolint:gochecknoglobals // Shared application-wide cache singleton; intentional.
var Cache = RedisCache{
	cache: nil,
}

// InitRedisCache initializes the application's Redis cache and in-memory instance.
func InitRedisCache(ctx context.Context) error {
	var err error
	once.Do(func() {
		cache, nErr := NewCache(ctx)
		if nErr != nil {
			err = nErr
			return
		}
		Cache.cache = cache
	})
	return err
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

// GetCache returns the underlying redis client.
func (r *RedisCache) GetCache() *redis.Client {
	return r.cache
}

// NewCache returns a new redis client for use.
func NewCache(ctx context.Context) (*redis.Client, error) {
	opts, parseErr := redis.ParseURL(config.Envs.RedisURL)
	if parseErr != nil {
		return nil, parseErr
	}
	cache := redis.NewClient(opts)
	if _, pingErr := cache.Ping(ctx).Result(); pingErr != nil {
		return nil, pingErr
	}
	return cache, nil
}
