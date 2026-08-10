package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
	"github.com/redis/go-redis/v9"
)

var (
	ErrRedis              = errors.New("[limiter] redis error occurred")
	ErrRedisTimeout       = errors.New("[limiter] redis timeout occurred")
	ErrRedisNotConfigured = errors.New("[limiter] redis client not configured")
)

// allowScript atomically leaks the bucket and permits the request if there's room.
// KEYS[1] = bucket key, ARGV[1] = max_req_second, ARGV[2] = now (unix seconds float)
var allowScript = redis.NewScript(`
local level = tonumber(redis.call("HGET", KEYS[1], "level") or "0")
local lastLeak = tonumber(redis.call("HGET", KEYS[1], "last_leak") or ARGV[2])
local maxReq = tonumber(ARGV[1])
local now = tonumber(ARGV[2])

local elapsed = now - lastLeak
level = level - (elapsed * maxReq)
if level < 0 then level = 0 end

local allowed = 0
if level + 1 <= maxReq then
    level = level + 1
    allowed = 1
end

redis.call("HSET", KEYS[1], "level", level, "last_leak", now)
redis.call("EXPIRE", KEYS[1], 60)

return allowed
`)

var once sync.Once

// limiter provides an interface for interacting with the redis rate limiter used in the webhook forwarding pipeline.
var limiter = &RateLimiter{}

type RateLimiter struct {
	rdb *redis.Client
}

// InitRateLimiter configures the rate limiter's internal state.
func InitRateLimiter(rdb *redis.Client) {
	once.Do(func() {
		if rdb != nil {
			limiter.rdb = rdb
		}
	})
}

// Allow checks if the request is permitted to be executed.
// prov.MaxReqSecond must already reflect the current cached value.
func (rl *RateLimiter) Allow(ctx context.Context, prov *model.Provider) (bool, error) {
	if prov.MaxReqSecond <= 0 {
		return false, nil
	}
	if rl.rdb == nil {
		return false, ErrRedisNotConfigured
	}
	key := fmt.Sprintf("ratelimit:%s", prov.Name)
	now := float64(time.Now().UnixMilli()) / 1000.0
	res, err := allowScript.Run(ctx, rl.rdb, []string{key}, prov.MaxReqSecond, now).Int()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return false, ErrRedisTimeout
		}
		return false, fmt.Errorf("%w %v", ErrRedis, err)
	}
	return res == 1, nil
}
