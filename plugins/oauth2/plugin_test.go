package oauth2

import (
	"errors"
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

func TestRegisterProviders_EnvCredentials(t *testing.T) {
	clearProviderEnv(t)
	t.Setenv(env.EnvGoogleClientID, "env-id")
	t.Setenv(env.EnvGoogleClientSecret, "env-secret")

	plugin := newRegistrationPlugin(nil)
	require.NoError(t, plugin.registerProviders())

	google, ok := plugin.providerRegistry.Get("google")
	require.True(t, ok)
	assert.Equal(t, "env-id", google.GetConfig().ClientID)
	assert.Equal(t, "https://api.example.com/auth/oauth2/callback/google", google.GetConfig().RedirectURL)
}

func TestRegisterProviders_ConfigOnly(t *testing.T) {
	clearProviderEnv(t)

	plugin := newRegistrationPlugin(map[string]types.ProviderConfig{
		"google": {ClientID: "config-id", ClientSecret: "config-secret"},
	})
	require.NoError(t, plugin.registerProviders())

	google, ok := plugin.providerRegistry.Get("google")
	require.True(t, ok)
	assert.Equal(t, "config-id", google.GetConfig().ClientID)
}

func TestRegisterProviders_EnvOverridesOnlyCredentials(t *testing.T) {
	clearProviderEnv(t)
	t.Setenv(env.EnvGoogleClientID, "env-id")
	t.Setenv(env.EnvGoogleClientSecret, "env-secret")

	plugin := newRegistrationPlugin(map[string]types.ProviderConfig{
		"google": {
			ClientID:     "config-id",
			ClientSecret: "config-secret",
			RedirectURL:  "https://custom.example.com/callback",
			Scopes:       []string{"custom-scope"},
		},
	})
	require.NoError(t, plugin.registerProviders())

	google, ok := plugin.providerRegistry.Get("google")
	require.True(t, ok)
	assert.Equal(t, "env-id", google.GetConfig().ClientID)
	assert.Equal(t, "env-secret", google.GetConfig().ClientSecret)
	assert.Equal(t, "https://custom.example.com/callback", google.GetConfig().RedirectURL)
	assert.Equal(t, []string{"custom-scope"}, google.GetConfig().Scopes)
}

func TestRegisterProviders_ConfigErrors(t *testing.T) {
	tests := []struct {
		name      string
		env       map[string]string
		providers map[string]types.ProviderConfig
		wantErr   error
	}{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearProviderEnv(t)
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			plugin := newRegistrationPlugin(tt.providers)
			if tt.name == "invalid base url" {
				plugin.globalConfig.BaseURL = "not-a-url"
			}

			err := plugin.registerProviders()
			require.Error(t, err)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestRegisterProviders_DisabledProviderIsSkipped(t *testing.T) {
	clearProviderEnv(t)
	t.Setenv(env.EnvGoogleClientID, "env-id")
	t.Setenv(env.EnvGoogleClientSecret, "env-secret")

	plugin := newRegistrationPlugin(map[string]types.ProviderConfig{
		"google": {Enabled: new(false)},
	})
	require.NoError(t, plugin.registerProviders())

	_, ok := plugin.providerRegistry.Get("google")
	assert.False(t, ok)
}

func TestRegisterProviders_CustomProvider(t *testing.T) {
	clearProviderEnv(t)

	plugin := newRegistrationPlugin(map[string]types.ProviderConfig{
		"Okta": {
			ClientID:     "id",
			ClientSecret: "secret",
			RedirectURL:  "https://app.example.com/cb",
			AuthURL:      "https://okta.example.com/authorize",
			TokenURL:     "https://okta.example.com/token",
			UserInfoURL:  "https://okta.example.com/user",
			Scopes:       []string{"openid"},
		},
	})
	require.NoError(t, plugin.registerProviders())

	names := plugin.providerRegistry.Names()
	assert.Equal(t, []string{"okta"}, names)
	provider, ok := plugin.providerRegistry.Get("okta")
	require.True(t, ok)
	_, ok = provider.(*services.GenericOAuth2Provider)
	assert.True(t, ok)
}
