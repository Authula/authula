package types

import "time"

const (
	defaultHTTPTimeout          = 10 * time.Second
	defaultAuthorizationCodeTTL = 90 * time.Second
	defaultConsentRequestTTL    = 10 * time.Minute
	defaultIDTokenTTL           = 5 * time.Minute
	defaultCleanupInterval      = time.Minute
	defaultRateLimitWindow      = time.Minute
	defaultRateLimitMax         = 60
)

func defaultAllowedScopes() []string {
	return []string{ScopeOpenID, ScopeProfile, ScopeEmail}
}

// OAuth2PluginConfig represents the OAuth2 plugin configuration
type OAuth2PluginConfig struct {
	Enabled     bool                      `json:"enabled" toml:"enabled"`
	HTTPTimeout time.Duration             `json:"http_timeout" toml:"http_timeout"`
	Providers   map[string]ProviderConfig `json:"providers" toml:"providers"`

	AuthorizationServerEnabled bool     `json:"authorization_server_enabled" toml:"authorization_server_enabled"`
	ClientManagementEnabled    bool     `json:"client_management_enabled" toml:"client_management_enabled"`
	LoginURL                   string   `json:"login_url" toml:"login_url"`
	ConsentURL                 string   `json:"consent_url" toml:"consent_url"`
	ConsentOrigins             []string `json:"consent_origins" toml:"consent_origins"`
	AllowedScopes              []string `json:"allowed_scopes" toml:"allowed_scopes"`
	AllowOrgClients            bool     `json:"allow_org_clients" toml:"allow_org_clients"`

	AuthorizationCodeTTL time.Duration `json:"authorization_code_ttl" toml:"authorization_code_ttl"`
	ConsentRequestTTL    time.Duration `json:"consent_request_ttl" toml:"consent_request_ttl"`
	IDTokenTTL           time.Duration `json:"id_token_ttl" toml:"id_token_ttl"`

	AutoCleanup     bool          `json:"auto_cleanup" toml:"auto_cleanup"`
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`

	RateLimitWindow time.Duration `json:"rate_limit_window" toml:"rate_limit_window"`
	RateLimitMax    int           `json:"rate_limit_max" toml:"rate_limit_max"`
}

// ApplyDefaults applies default values to the config
func (c *OAuth2PluginConfig) ApplyDefaults() {
	if c.Providers == nil {
		c.Providers = make(map[string]ProviderConfig)
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = defaultHTTPTimeout
	}
	if len(c.AllowedScopes) == 0 {
		c.AllowedScopes = defaultAllowedScopes()
	}
	if c.AuthorizationCodeTTL <= 0 {
		c.AuthorizationCodeTTL = defaultAuthorizationCodeTTL
	}
	if c.ConsentRequestTTL <= 0 {
		c.ConsentRequestTTL = defaultConsentRequestTTL
	}
	if c.IDTokenTTL <= 0 {
		c.IDTokenTTL = defaultIDTokenTTL
	}
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = defaultCleanupInterval
	}
	if c.RateLimitWindow <= 0 {
		c.RateLimitWindow = defaultRateLimitWindow
	}
	if c.RateLimitMax <= 0 {
		c.RateLimitMax = defaultRateLimitMax
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
	PKCE     *bool  `json:"pkce,omitempty" toml:"pkce,omitempty"`
	TenantID string `json:"tenant_id,omitempty" toml:"tenant_id,omitempty"` // Microsoft only: common (default), organizations, consumers or a tenant ID
}

// IsEnabled reports whether the provider is enabled. Absent means enabled.
func (c ProviderConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

// PKCEEnabled reports whether PKCE is required for this provider. Absent means enabled.
func (c ProviderConfig) PKCEEnabled() bool {
	return c.PKCE == nil || *c.PKCE
}
