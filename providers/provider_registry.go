package providers

import (
	"fmt"
	"sync"
)

type OAuth2ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]OAuth2Provider
}

func NewOAuth2ProviderRegistry() *OAuth2ProviderRegistry {
	return &OAuth2ProviderRegistry{
		providers: make(map[string]OAuth2Provider),
	}
}

func (r *OAuth2ProviderRegistry) Register(provider OAuth2Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.GetName()] = provider
}

func (r *OAuth2ProviderRegistry) Get(name string) (OAuth2Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

func (r *OAuth2ProviderRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers = make(map[string]OAuth2Provider)
}
