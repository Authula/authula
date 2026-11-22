package types

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProviderConfig_PKCEEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config ProviderConfig
		want   bool
	}{
		{name: "absent defaults to enabled", config: ProviderConfig{}, want: true},
		{name: "explicit true", config: ProviderConfig{PKCE: new(true)}, want: true},
		{name: "explicit false opts out", config: ProviderConfig{PKCE: new(false)}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.config.PKCEEnabled(); got != tt.want {
				t.Fatalf("PKCEEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPKCEConfigSurface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		typ         reflect.Type
		wantPresent bool
	}{
		{name: "plugin config has no global PKCE toggle", typ: reflect.TypeFor[OAuth2PluginConfig](), wantPresent: false},
		{name: "provider config exposes per-provider PKCE", typ: reflect.TypeFor[ProviderConfig](), wantPresent: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			present := false
			for field := range tt.typ.Fields() {
				if strings.EqualFold(field.Name, "PKCE") {
					present = true
					break
				}
			}
			if present != tt.wantPresent {
				t.Fatalf("%s PKCE field present = %v, want %v", tt.typ.Name(), present, tt.wantPresent)
			}
		})
	}
}

func TestOAuth2PluginConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	config := OAuth2PluginConfig{}
	config.ApplyDefaults()

	if config.HTTPTimeout != defaultHTTPTimeout {
		t.Fatalf("HTTPTimeout = %v, want %v", config.HTTPTimeout, defaultHTTPTimeout)
	}
	if config.AuthorizationCodeTTL != defaultAuthorizationCodeTTL {
		t.Fatalf("AuthorizationCodeTTL = %v, want %v", config.AuthorizationCodeTTL, defaultAuthorizationCodeTTL)
	}
	if config.ConsentRequestTTL != defaultConsentRequestTTL {
		t.Fatalf("ConsentRequestTTL = %v, want %v", config.ConsentRequestTTL, defaultConsentRequestTTL)
	}
	if config.FlowStateTTL != defaultFlowStateTTL {
		t.Fatalf("FlowStateTTL = %v, want %v", config.FlowStateTTL, defaultFlowStateTTL)
	}
	if config.IDTokenTTL != defaultIDTokenTTL {
		t.Fatalf("IDTokenTTL = %v, want %v", config.IDTokenTTL, defaultIDTokenTTL)
	}
	if config.CleanupInterval != defaultCleanupInterval {
		t.Fatalf("CleanupInterval = %v, want %v", config.CleanupInterval, defaultCleanupInterval)
	}
	if !reflect.DeepEqual(config.AllowedScopes, defaultAllowedScopes()) {
		t.Fatalf("AllowedScopes = %v, want %v", config.AllowedScopes, defaultAllowedScopes())
	}
	if config.Providers == nil {
		t.Fatal("Providers should be initialized")
	}
}

func TestOAuth2PluginConfig_ApplyDefaultsPreservesExplicitValues(t *testing.T) {
	t.Parallel()

	config := OAuth2PluginConfig{
		AllowedScopes:        []string{"openid", "custom"},
		AuthorizationCodeTTL: time.Minute,
		ConsentRequestTTL:    2 * time.Minute,
		FlowStateTTL:         6 * time.Minute,
		IDTokenTTL:           3 * time.Minute,
		CleanupInterval:      4 * time.Minute,
	}
	config.ApplyDefaults()

	if !reflect.DeepEqual(config.AllowedScopes, []string{"openid", "custom"}) {
		t.Fatalf("AllowedScopes = %v", config.AllowedScopes)
	}
	if config.AuthorizationCodeTTL != time.Minute || config.ConsentRequestTTL != 2*time.Minute ||
		config.FlowStateTTL != 6*time.Minute || config.IDTokenTTL != 3*time.Minute ||
		config.CleanupInterval != 4*time.Minute {
		t.Fatalf("explicit config values were overwritten: %+v", config)
	}
}

func TestOAuth2PluginConfig_ClientCredentialsDefaults(t *testing.T) {
	t.Parallel()

	config := OAuth2PluginConfig{}
	config.ApplyDefaults()

	if config.ClientCredentialsTokenTTL != 15*time.Minute {
		t.Fatalf("ClientCredentialsTokenTTL = %v, want 15m", config.ClientCredentialsTokenTTL)
	}
	if config.ClientSecretValuePrefix != "authula_cs_" {
		t.Fatalf("ClientSecretValuePrefix = %q, want authula_cs_", config.ClientSecretValuePrefix)
	}

	explicit := OAuth2PluginConfig{
		ClientCredentialsTokenTTL: time.Minute,
		ClientSecretValuePrefix:   "custom_",
	}
	explicit.ApplyDefaults()

	if explicit.ClientCredentialsTokenTTL != time.Minute || explicit.ClientSecretValuePrefix != "custom_" {
		t.Fatalf("explicit client credentials config values were overwritten: %+v", explicit)
	}
}

func TestOAuth2PluginConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  OAuth2PluginConfig
		wantErr bool
	}{
		{
			name: "auth server disabled with OIDC only ceiling",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{"openid", "profile", "email"},
				ClientSecretValuePrefix:   "authula_cs_",
				ClientCredentialsTokenTTL: time.Minute,
			},
		},
		{
			name: "auth server enabled with non-OIDC scope",
			config: OAuth2PluginConfig{
				AuthorizationServerEnabled: true,
				AllowedScopes:              []string{"openid", "read", "write"},
				ClientSecretValuePrefix:    "authula_cs_",
				ClientCredentialsTokenTTL:  time.Minute,
			},
		},
		{
			name: "auth server enabled with OIDC only ceiling",
			config: OAuth2PluginConfig{
				AuthorizationServerEnabled: true,
				AllowedScopes:              []string{"openid", "profile"},
				ClientSecretValuePrefix:    "authula_cs_",
				ClientCredentialsTokenTTL:  time.Minute,
			},
			wantErr: true,
		},
		{
			name: "auth server enabled with sub-second ttl",
			config: OAuth2PluginConfig{
				AuthorizationServerEnabled: true,
				AllowedScopes:              []string{"read"},
				ClientSecretValuePrefix:    "authula_cs_",
				ClientCredentialsTokenTTL:  time.Millisecond,
			},
			wantErr: true,
		},
		{
			name: "empty prefix",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{"read"},
				ClientCredentialsTokenTTL: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "prefix too long",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{"read"},
				ClientSecretValuePrefix:   strings.Repeat("a", 33),
				ClientCredentialsTokenTTL: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "prefix with invalid characters",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{"read"},
				ClientSecretValuePrefix:   "bad prefix!",
				ClientCredentialsTokenTTL: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "duplicate allowed scope",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{"read", "read"},
				ClientSecretValuePrefix:   "authula_cs_",
				ClientCredentialsTokenTTL: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "wildcard allowed scope",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{"*"},
				ClientSecretValuePrefix:   "authula_cs_",
				ClientCredentialsTokenTTL: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "allowed scope with space",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{"read write"},
				ClientSecretValuePrefix:   "authula_cs_",
				ClientCredentialsTokenTTL: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "empty allowed scope",
			config: OAuth2PluginConfig{
				AllowedScopes:             []string{""},
				ClientSecretValuePrefix:   "authula_cs_",
				ClientCredentialsTokenTTL: time.Minute,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
