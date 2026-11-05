package services

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/Authula/authula/plugins/oauth2/types"
)

var providerNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// ValidateProviderName enforces a path-safe provider identifier.
func ValidateProviderName(name string) error {
	if !providerNamePattern.MatchString(name) {
		return fmt.Errorf("provider name %q must match %s", name, providerNamePattern.String())
	}
	return nil
}

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
	name = strings.ToLower(strings.TrimSpace(name))
	if err := ValidateProviderName(name); err != nil {
		return err
	}
	if provider == nil {
		return fmt.Errorf("provider %s is nil", name)
	}

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

// Names returns the registered provider names in deterministic order.
func (r *ProviderRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
