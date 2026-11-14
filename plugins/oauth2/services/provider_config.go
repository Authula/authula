package services

import (
	"fmt"
	"strings"

	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// ProviderDefaults describes the built-in behaviour of a provider before config overrides.
type ProviderDefaults struct {
	Endpoint      oauth2.Endpoint
	Scopes        []string
	UserInfoURL   string
	AuthURLParams map[string]string
}

// reservedAuthURLParams are protocol-managed authorization parameters that
// provider configuration must never override. Overriding state would bypass
// CSRF protection, and the remaining values are set by the OAuth2 config.
var reservedAuthURLParams = map[string]struct{}{
	"response_type":         {},
	"client_id":             {},
	"redirect_uri":          {},
	"state":                 {},
	"code_challenge":        {},
	"code_challenge_method": {},
	"scope":                 {},
}

func isReservedAuthURLParam(name string) bool {
	_, reserved := reservedAuthURLParams[strings.ToLower(strings.TrimSpace(name))]
	return reserved
}

// ResolveAuthURLParams merges provider defaults with operator configuration.
// Configuration overrides defaults, and an empty configured value removes a
// default. The defaults map is never mutated.
func ResolveAuthURLParams(cfg types.ProviderConfig, defaults ProviderDefaults) map[string]string {
	resolved := make(map[string]string, len(defaults.AuthURLParams)+len(cfg.AuthURLParams))

	for name, value := range defaults.AuthURLParams {
		name = strings.TrimSpace(name)
		if name == "" || isReservedAuthURLParam(name) {
			continue
		}
		resolved[name] = value
	}

	for name, value := range cfg.AuthURLParams {
		name = strings.TrimSpace(name)
		if name == "" || isReservedAuthURLParam(name) {
			continue
		}
		if strings.TrimSpace(value) == "" {
			delete(resolved, name)
			continue
		}
		resolved[name] = value
	}

	return resolved
}

// ValidateAuthURLParams rejects blank and protocol-managed parameter names.
// Values are strings, so URL query encoding is the only handling they need.
func ValidateAuthURLParams(params map[string]string) error {
	for name := range params {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return fmt.Errorf("auth_url_params contains a blank parameter name")
		}
		if isReservedAuthURLParam(trimmed) {
			return fmt.Errorf("auth_url_params parameter %q is managed by the OAuth2 protocol and cannot be overridden", trimmed)
		}
	}
	return nil
}

// ResolveOAuth2Config applies a provider config on top of built-in defaults.
func ResolveOAuth2Config(cfg types.ProviderConfig, defaults ProviderDefaults) *oauth2.Config {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = defaults.Scopes
	}

	endpoint := defaults.Endpoint
	if cfg.AuthURL != "" {
		endpoint.AuthURL = cfg.AuthURL
	}
	if cfg.TokenURL != "" {
		endpoint.TokenURL = cfg.TokenURL
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint:     endpoint,
	}
}

// ResolveUserInfoURL returns the configured user info URL or the provider default.
func ResolveUserInfoURL(cfg types.ProviderConfig, defaultURL string) string {
	if cfg.UserInfoURL != "" {
		return cfg.UserInfoURL
	}
	return defaultURL
}

type providerFields struct {
	userID        string
	email         string
	name          string
	picture       string
	emailVerified string
}

func defaultField(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
