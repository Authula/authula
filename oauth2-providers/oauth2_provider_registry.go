package oauth2providers

import (
	"fmt"
	"sync"

	"github.com/GoBetterAuth/go-better-auth/models"
)

type OAuth2ProviderRegistry struct {
	config    *models.Config
	mu        sync.RWMutex
	providers map[string]OAuth2Provider
}

func NewOAuth2ProviderRegistry(config *models.Config) *OAuth2ProviderRegistry {
	return &OAuth2ProviderRegistry{
		config:    config,
		providers: make(map[string]OAuth2Provider),
	}
}

func (r *OAuth2ProviderRegistry) Register(provider OAuth2Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[provider.GetName()] = provider
}

func (r *OAuth2ProviderRegistry) Get(name string) (OAuth2Provider, error) {
	r.RefreshOAuth2Providers()

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

func (r *OAuth2ProviderRegistry) RefreshOAuth2Providers() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]OAuth2Provider)

	for name, providerConfig := range r.config.SocialProviders {
		if !providerConfig.Enabled {
			continue
		}

		var provider OAuth2Provider
		switch name {
		case "google":
			provider = NewGoogleProvider(&providerConfig)
		case "github":
			provider = NewGitHubProvider(&providerConfig)
		case "discord":
			provider = NewDiscordProvider(&providerConfig)
		default:
			if providerConfig.AuthURL != "" && providerConfig.TokenURL != "" {
				provider = NewGenericProvider(name, &providerConfig)
			}
		}

		if provider != nil {
			r.providers[provider.GetName()] = provider
		}
	}
}
