package providers

import (
	"context"
	"sync"

	"github.com/JBK2116/vaulthook/internal/model"
)

// ProviderCache is an in-memory cache used by workers when managing wehooks
type ProviderCache struct {
	mu        sync.RWMutex
	providers map[model.ProviderName]*model.Provider
}

// Cache is the global default instance of ProviderCache
var Cache = ProviderCache{
	mu:        sync.RWMutex{},
	providers: make(map[model.ProviderName]*model.Provider, 0),
}

// InitProviderCache loads the cache with the details of the providers in the database
func InitProviderCache(ctx context.Context, repo *ProviderRepo) error {
	provs, err := repo.getAll(ctx)
	if err != nil {
		return err
	}
	for _, val := range provs {
		v := val
		Cache.providers[model.ProviderName(val.Name)] = &v
	}
	return nil
}

// Get returns a pointer to the provider with the supplied name
func (c *ProviderCache) Get(key model.ProviderName) *model.Provider {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.providers[key]
	if !ok {
		return nil
	}
	return v
}

// Set updates the value of the provided key to point to the new value
func (c *ProviderCache) Set(key model.ProviderName, val *model.Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers[key] = val
}
