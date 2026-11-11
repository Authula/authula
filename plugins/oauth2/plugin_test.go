package oauth2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
)

func clearProviderEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		env.EnvGoogleClientID,
		env.EnvGoogleClientSecret,
		env.EnvGithubClientID,
		env.EnvGithubClientSecret,
		env.EnvDiscordClientID,
		env.EnvDiscordClientSecret,
		env.EnvMicrosoftClientID,
		env.EnvMicrosoftClientSecret,
		env.EnvMicrosoftTenantID,
	} {
		t.Setenv(key, "")
	}
}

func newRegistrationPlugin(providers map[string]types.ProviderConfig) *OAuth2Plugin {
	return &OAuth2Plugin{
		globalConfig: &models.Config{
			BaseURL:  "https://api.example.com",
			BasePath: "/auth",
		},
		pluginConfig:     types.OAuth2PluginConfig{Providers: providers},
		logger:           &internaltests.MockLogger{},
		providerRegistry: services.NewProviderRegistry(),
	}
}

func TestRegisterProviders(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		providers map[string]types.ProviderConfig
		configure func(*OAuth2Plugin)
		wantErr   error
		verify    func(*testing.T, *OAuth2Plugin)
	}{
		{
			name: "env credentials",
			env: map[string]string{
				env.EnvGoogleClientID:     "env-id",
				env.EnvGoogleClientSecret: "env-secret",
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				google, ok := plugin.providerRegistry.Get("google")
				require.True(t, ok)
				assert.Equal(t, "env-id", google.GetConfig().ClientID)
				assert.Equal(t, "https://api.example.com/auth/oauth2/callback/google", google.GetConfig().RedirectURL)
			},
		},
		{
			name: "config only",
			providers: map[string]types.ProviderConfig{
				"google": {ClientID: "config-id", ClientSecret: "config-secret"},
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				google, ok := plugin.providerRegistry.Get("google")
				require.True(t, ok)
				assert.Equal(t, "config-id", google.GetConfig().ClientID)
			},
		},
		{
			name: "env overrides only credentials",
			env: map[string]string{
				env.EnvGoogleClientID:     "env-id",
				env.EnvGoogleClientSecret: "env-secret",
			},
			providers: map[string]types.ProviderConfig{
				"google": {
					ClientID:     "config-id",
					ClientSecret: "config-secret",
					RedirectURL:  "https://custom.example.com/callback",
					Scopes:       []string{"custom-scope"},
				},
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				google, ok := plugin.providerRegistry.Get("google")
				require.True(t, ok)
				assert.Equal(t, "env-id", google.GetConfig().ClientID)
				assert.Equal(t, "env-secret", google.GetConfig().ClientSecret)
				assert.Equal(t, "https://custom.example.com/callback", google.GetConfig().RedirectURL)
				assert.Equal(t, []string{"custom-scope"}, google.GetConfig().Scopes)
			},
		},
		{
			name:    "partial env credentials",
			env:     map[string]string{env.EnvGoogleClientID: "id"},
			wantErr: constants.ErrProviderConfigurationPartial,
		},
		{
			name: "invalid base url",
			providers: map[string]types.ProviderConfig{
				"google": {ClientID: "id", ClientSecret: "secret"},
			},
			configure: func(plugin *OAuth2Plugin) {
				plugin.globalConfig.BaseURL = "not-a-url"
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "custom provider missing endpoints",
			providers: map[string]types.ProviderConfig{
				"custom": {ClientID: "id", ClientSecret: "secret", RedirectURL: "https://app.example.com/cb"},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "insecure endpoint rejected",
			providers: map[string]types.ProviderConfig{
				"custom": {
					ClientID:     "id",
					ClientSecret: "secret",
					RedirectURL:  "https://app.example.com/cb",
					AuthURL:      "http://evil.example.com/authorize",
					TokenURL:     "https://evil.example.com/token",
					UserInfoURL:  "https://evil.example.com/user",
				},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "disabled provider is skipped",
			env: map[string]string{
				env.EnvGoogleClientID:     "env-id",
				env.EnvGoogleClientSecret: "env-secret",
			},
			providers: map[string]types.ProviderConfig{
				"google": {Enabled: new(false)},
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				_, ok := plugin.providerRegistry.Get("google")
				assert.False(t, ok)
			},
		},
		{
			name: "invalid redirect wildcard",
			providers: map[string]types.ProviderConfig{
				"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "https://*.example.com/callback"},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "invalid redirect fragment",
			providers: map[string]types.ProviderConfig{
				"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "https://app.example.com/callback#frag"},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "insecure redirect rejected",
			providers: map[string]types.ProviderConfig{
				"google": {ClientID: "id", ClientSecret: "secret", RedirectURL: "http://app.example.com/callback"},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "unsafe provider name rejected",
			providers: map[string]types.ProviderConfig{
				"bad name": {ClientID: "id", ClientSecret: "secret", RedirectURL: "https://app.example.com/cb"},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "per provider PKCE opt out is honoured",
			providers: map[string]types.ProviderConfig{
				"google": {ClientID: "config-id", ClientSecret: "config-secret", PKCE: new(false)},
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				google, ok := plugin.providerRegistry.Get("google")
				require.True(t, ok)
				assert.False(t, google.RequiresPKCE())
			},
		},
		{
			name: "microsoft env credentials and tenant",
			env: map[string]string{
				env.EnvMicrosoftClientID:     "env-id",
				env.EnvMicrosoftClientSecret: "env-secret",
				env.EnvMicrosoftTenantID:     "organizations",
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				microsoft, ok := plugin.providerRegistry.Get("microsoft")
				require.True(t, ok)
				_, ok = microsoft.(*services.MicrosoftOAuth2Provider)
				assert.True(t, ok)
				assert.Equal(t, "env-id", microsoft.GetConfig().ClientID)
				assert.Equal(t, "https://api.example.com/auth/oauth2/callback/microsoft", microsoft.GetConfig().RedirectURL)
				assert.Equal(t, "https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize", microsoft.GetConfig().Endpoint.AuthURL)
			},
		},
		{
			name: "microsoft config tenant",
			providers: map[string]types.ProviderConfig{
				"microsoft": {ClientID: "config-id", ClientSecret: "config-secret", TenantID: new("contoso.onmicrosoft.com")},
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				microsoft, ok := plugin.providerRegistry.Get("microsoft")
				require.True(t, ok)
				assert.Equal(t, "https://login.microsoftonline.com/contoso.onmicrosoft.com/oauth2/v2.0/token", microsoft.GetConfig().Endpoint.TokenURL)
			},
		},
		{
			name: "custom provider",
			providers: map[string]types.ProviderConfig{
				"Okta": {
					ClientID:     "id",
					ClientSecret: "secret",
					RedirectURL:  "https://app.example.com/cb",
					AuthURL:      "https://okta.example.com/authorize",
					TokenURL:     "https://okta.example.com/token",
					UserInfoURL:  "https://okta.example.com/user",
					Scopes:       []string{"openid"},
				},
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				assert.Equal(t, []string{"okta"}, plugin.providerRegistry.Names())
				provider, ok := plugin.providerRegistry.Get("okta")
				require.True(t, ok)
				_, ok = provider.(*services.GenericOAuth2Provider)
				assert.True(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearProviderEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			plugin := newRegistrationPlugin(tt.providers)
			if tt.configure != nil {
				tt.configure(plugin)
			}

			err := plugin.registerProviders()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			if tt.verify != nil {
				tt.verify(t, plugin)
			}
		})
	}
}

func TestOAuth2PluginImplementsMigrationInterface(t *testing.T) {
	t.Parallel()

	var plugin models.Plugin = New(types.OAuth2PluginConfig{})
	migratable, ok := plugin.(models.PluginWithMigrations)
	require.True(t, ok)
	require.Len(t, migratable.Migrations("sqlite"), 1)
	require.Nil(t, migratable.DependsOn())
}
