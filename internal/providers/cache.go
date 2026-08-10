package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/JBK2116/vaulthook/internal/cache"
	"github.com/JBK2116/vaulthook/internal/model"
)

const (
	RedisInfiniteTTL    = 0
	RedisProviderPrefix = "provider+"
)

// ProviderCache is a Redis-backed cache used by workers when managing webooks
type ProviderCache struct {
}

// Cache is the global default instance of ProviderCache
var Cache = ProviderCache{}

var once sync.Once

// InitProviderCache loads the cache with the details of the providers in the database
func InitProviderCache(ctx context.Context, repo *ProviderRepo) error {
	var err error
	once.Do(func() {
		provs, getErr := repo.getAll(ctx)
		if getErr != nil {
			err = getErr
			return
		}
		kvPairs := make([]any, 0, len(provs)*2)
		for _, val := range provs {
			b, sErr := serialize(val)
			if sErr != nil {
				err = fmt.Errorf("[Providers] error marshaling provider during cache insertion: %w", sErr)
				return
			}
			kvPairs = append(kvPairs, RedisProviderPrefix+val.Name, b)
		}
		if len(kvPairs) == 0 {
			err = fmt.Errorf("[Providers] no providers found in database")
			return
		}
		if msetErr := cache.Cache.Mset(ctx, kvPairs...); msetErr != nil {
			err = msetErr
			return
		}
	})
	return err
}

// Get returns a provider from the cache
func (c *ProviderCache) Get(ctx context.Context, key model.ProviderName) (model.Provider, error) {
	val, gErr := cache.Cache.GetBytes(ctx, RedisProviderPrefix+string(key))
	if gErr != nil {
		return model.Provider{}, gErr
	}
	var prov model.Provider
	err := json.Unmarshal(val, &prov)
	if err != nil {
		return model.Provider{}, err
	}
	return prov, nil
}

// Set updates the value of the provided key to point to the new value
func (c *ProviderCache) Set(ctx context.Context, key model.ProviderName, val model.Provider) error {
	b, sErr := serialize(val)
	if sErr != nil {
		return sErr
	}
	err := cache.Cache.Set(ctx, RedisProviderPrefix+string(key), b, RedisInfiniteTTL)
	if err != nil {
		return err
	}
	return nil
}

// serialize converts an in-memory object into a serialized json object
func serialize(val any) ([]byte, error) {
	b, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}
	return b, nil
}
