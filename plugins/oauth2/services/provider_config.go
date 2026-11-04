package services

import (
	"golang.org/x/oauth2"

	"github.com/Authula/authula/plugins/oauth2/types"
)

// ProviderDefaults describes the built-in behaviour of a provider before config overrides.
type ProviderDefaults struct {
	Endpoint    oauth2.Endpoint
	Scopes      []string
	UserInfoURL string
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
