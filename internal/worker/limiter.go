package worker

import (
	"sync"
	"time"

	"github.com/JBK2116/vaulthook/internal/model"
)

// RateLimiter implements per provider rate-limiting using the leaky bucket algorithm
type RateLimiter struct {
	buckets sync.Map
}

// leakyBucket is used for rate limiting per provider
type leakyBucket struct {
	mu       sync.Mutex
	level    int
	lastLeak time.Time
}

// limiter handles rate limiting throughout the application
var limiter = &RateLimiter{}

// InitRateLimiter configures a RateLimiter with a bucket for each provider
func InitRateLimiter() {
	for _, v := range model.ProviderNames {
		buck := &leakyBucket{
			mu:       sync.Mutex{},
			level:    0,
			lastLeak: time.Now(),
		}
		limiter.buckets.Store(v, buck)
	}
}

// Allow checks if the request is permitted to be executed
func (rl *RateLimiter) Allow(prov *model.Provider) bool {
	val, ok := rl.buckets.Load(model.ProviderName(prov.Name))
	if !ok {
		// Should never happen in prod realistically
		return false
	}
	b := val.(*leakyBucket)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastLeak).Seconds()
	if prov.MaxReqSecond <= 0 {
		return false
	}
	b.level -= int(elapsed) * prov.MaxReqSecond
	if b.level < 0 {
		b.level = 0
	}
	b.lastLeak = now
	if b.level+1 <= prov.MaxReqSecond {
		b.level++
		return true
	}
	return false
}
