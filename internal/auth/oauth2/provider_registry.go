package oauth2

import (
	"fmt"
)

type OAuth2ProviderRegistry struct {
	providers map[string]OAuth2Provider
}

func NewOAuth2ProviderRegistry() *OAuth2ProviderRegistry {
	return &OAuth2ProviderRegistry{
		providers: make(map[string]OAuth2Provider),
	}
}

func (r *OAuth2ProviderRegistry) Register(provider OAuth2Provider) {
	r.providers[provider.GetName()] = provider
}

func (r *OAuth2ProviderRegistry) Get(name string) (OAuth2Provider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}
