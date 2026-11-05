package types

import "time"

const defaultHTTPTimeout = 10 * time.Second

// OAuth2PluginConfig represents the OAuth2 plugin configuration
type OAuth2PluginConfig struct {
	Enabled     bool                      `json:"enabled" toml:"enabled"`
	HTTPTimeout time.Duration             `json:"http_timeout" toml:"http_timeout"`
	Providers   map[string]ProviderConfig `json:"providers" toml:"providers"`
}

// ApplyDefaults applies default values to the config
func (c *OAuth2PluginConfig) ApplyDefaults() {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = defaultHTTPTimeout
	}
}

// ProviderConfig represents configuration for an OAuth2 provider
type ProviderConfig struct {
	// Enabled defaults to true when absent. Set it explicitly to false to disable a provider.
	Enabled      *bool    `json:"enabled,omitempty" toml:"enabled,omitempty"`
	ClientID     string   `json:"client_id" toml:"client_id"`
	ClientSecret string   `json:"client_secret" toml:"client_secret"`
	RedirectURL  string   `json:"redirect_url" toml:"redirect_url"`
	Scopes       []string `json:"scopes" toml:"scopes"`
	AuthURL      string   `json:"auth_url" toml:"auth_url"`
	TokenURL     string   `json:"token_url" toml:"token_url"`
	UserInfoURL  string   `json:"user_info_url" toml:"user_info_url"`
	UserIDField  string   `json:"user_id_field" toml:"user_id_field"`
	EmailField   string   `json:"email_field" toml:"email_field"`
	NameField    string   `json:"name_field" toml:"name_field"`
	PictureField string   `json:"picture_field" toml:"picture_field"`
	// EmailVerifiedField is the field that carries the provider's email verification flag.
	EmailVerifiedField string `json:"email_verified_field" toml:"email_verified_field"`
	// PKCE controls whether this provider must use PKCE with S256. It defaults to true.
	// Setting it to false is an explicit opt-out and is not OAuth 2.1 compliant.
	PKCE *bool `json:"pkce,omitempty" toml:"pkce,omitempty"`
}

// IsEnabled reports whether the provider is enabled. Absent means enabled.
func (c ProviderConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

// PKCEEnabled reports whether PKCE is required for this provider. Absent means enabled.
func (c ProviderConfig) PKCEEnabled() bool {
	return c.PKCE == nil || *c.PKCE
}
