package types

import (
	"fmt"
	"time"
)

const (
	defaultHTTPTimeout               = 10 * time.Second
	defaultAuthorizationCodeTTL      = 90 * time.Second
	defaultConsentRequestTTL         = 10 * time.Minute
	defaultFlowStateTTL              = 5 * time.Minute
	defaultIDTokenTTL                = 5 * time.Minute
	defaultCleanupInterval           = time.Minute
	defaultClientCredentialsTokenTTL = 15 * time.Minute
	defaultClientSecretValuePrefix   = "authula_cs_"
	maxClientSecretValuePrefixLength = 32
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
	FlowStateTTL         time.Duration `json:"flow_state_ttl" toml:"flow_state_ttl"`
	IDTokenTTL           time.Duration `json:"id_token_ttl" toml:"id_token_ttl"`

	ClientCredentialsTokenTTL time.Duration `json:"client_credentials_token_ttl" toml:"client_credentials_token_ttl"`
	ClientSecretValuePrefix   string        `json:"client_secret_value_prefix" toml:"client_secret_value_prefix"`

	AutoCleanup     bool          `json:"auto_cleanup" toml:"auto_cleanup"`
	CleanupInterval time.Duration `json:"cleanup_interval" toml:"cleanup_interval"`
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
	if c.FlowStateTTL <= 0 {
		c.FlowStateTTL = defaultFlowStateTTL
	}
	if c.IDTokenTTL <= 0 {
		c.IDTokenTTL = defaultIDTokenTTL
	}
	if c.CleanupInterval <= 0 {
		c.CleanupInterval = defaultCleanupInterval
	}
	if c.ClientCredentialsTokenTTL <= 0 {
		c.ClientCredentialsTokenTTL = defaultClientCredentialsTokenTTL
	}
	if c.ClientSecretValuePrefix == "" {
		c.ClientSecretValuePrefix = defaultClientSecretValuePrefix
	}
}

// Validate checks the OAuth2 plugin configuration after defaults are applied.
func (c *OAuth2PluginConfig) Validate() error {
	if c.ClientSecretValuePrefix == "" {
		return fmt.Errorf("client_secret_value_prefix must not be empty")
	}
	if len(c.ClientSecretValuePrefix) > maxClientSecretValuePrefixLength {
		return fmt.Errorf("client_secret_value_prefix must be at most %d characters", maxClientSecretValuePrefixLength)
	}
	for _, char := range c.ClientSecretValuePrefix {
		if !isClientSecretPrefixRune(char) {
			return fmt.Errorf("client_secret_value_prefix may only contain [A-Za-z0-9_-]")
		}
	}

	if err := validateAllowedScopes(c.AllowedScopes); err != nil {
		return err
	}

	if !c.AuthorizationServerEnabled {
		return nil
	}

	if c.ClientCredentialsTokenTTL < time.Second {
		return fmt.Errorf("client_credentials_token_ttl must be at least one second")
	}

	for _, scope := range c.AllowedScopes {
		if !IsOIDCScope(scope) {
			return nil
		}
	}

	return fmt.Errorf("allowed_scopes must contain at least one non-OIDC scope when the authorization server is enabled")
}

func validateAllowedScopes(scopes []string) error {
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		parsed, ok := ParseScope(scope)
		if !ok || len(parsed) != 1 || parsed[0] != scope {
			return fmt.Errorf("invalid allowed scope %q", scope)
		}
		if _, exists := seen[scope]; exists {
			return fmt.Errorf("duplicate allowed scope %q", scope)
		}
		seen[scope] = struct{}{}
	}
	return nil
}

func isClientSecretPrefixRune(char rune) bool {
	if char >= 'a' && char <= 'z' {
		return true
	}
	if char >= 'A' && char <= 'Z' {
		return true
	}
	if char >= '0' && char <= '9' {
		return true
	}
	return char == '_' || char == '-'
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
	PKCE     *bool   `json:"pkce,omitempty" toml:"pkce,omitempty"`
	TenantID *string `json:"tenant_id,omitempty" toml:"tenant_id,omitempty"` // Microsoft only: common (default), organizations, consumers or a tenant ID
	// AuthURLParams are extra query parameters appended to the provider's
	// authorization URL. Provider configuration overrides provider defaults.
	// Empty values remove a default parameter. Protocol-managed parameters are
	// rejected during configuration validation.
	AuthURLParams map[string]string `json:"auth_url_params,omitempty" toml:"auth_url_params,omitempty"`
}

// IsEnabled reports whether the provider is enabled. Absent means enabled.
func (c ProviderConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

// PKCEEnabled reports whether PKCE is required for this provider. Absent means enabled.
func (c ProviderConfig) PKCEEnabled() bool {
	return c.PKCE == nil || *c.PKCE
}
