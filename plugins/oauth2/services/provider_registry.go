package services

import (
	"fmt"
	"sync"

	"github.com/GoBetterAuth/go-better-auth/plugins/oauth2/types"
)

// ProviderRegistry manages OAuth2 providers
type ProviderRegistry struct {
	providers map[string]types.OAuth2Provider
	mu        sync.RWMutex
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]types.OAuth2Provider),
	}
}

// Register registers a provider
func (r *ProviderRegistry) Register(name string, provider types.OAuth2Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %s already registered", name)
	}

	r.providers[name] = provider
	return nil
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (types.OAuth2Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	return provider, exists
}

// GetAll returns all registered providers
func (r *ProviderRegistry) GetAll() map[string]types.OAuth2Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]types.OAuth2Provider)
	for k, v := range r.providers {
		result[k] = v
	}
	return result
}

// IsProviderEnabled checks if a provider is registered
func (r *ProviderRegistry) IsProviderEnabled(name string) bool {
	_, exists := r.Get(name)
	return exists
}
