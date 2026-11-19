package oauth2

import (
	"fmt"
	"net/url"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
	internalplugins "github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/services"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/util"
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
		env.EnvDropboxClientID,
		env.EnvDropboxClientSecret,
		env.EnvMicrosoftClientID,
		env.EnvMicrosoftClientSecret,
		env.EnvMicrosoftTenantID,
		env.EnvNotionClientID,
		env.EnvNotionClientSecret,
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
			name: "reserved auth url param is rejected",
			providers: map[string]types.ProviderConfig{
				"google": {
					ClientID:      "config-id",
					ClientSecret:  "config-secret",
					AuthURLParams: map[string]string{"state": "fixed"},
				},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "blank auth url param is rejected",
			providers: map[string]types.ProviderConfig{
				"google": {
					ClientID:      "config-id",
					ClientSecret:  "config-secret",
					AuthURLParams: map[string]string{"   ": "value"},
				},
			},
			wantErr: constants.ErrProviderConfigInvalid,
		},
		{
			name: "auth url params are applied to the authorization url",
			providers: map[string]types.ProviderConfig{
				"google": {
					ClientID:      "config-id",
					ClientSecret:  "config-secret",
					AuthURLParams: map[string]string{"prompt": "consent"},
				},
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				google, ok := plugin.providerRegistry.Get("google")
				require.True(t, ok)

				parsed, err := url.Parse(google.GetAuthURL("state-value"))
				require.NoError(t, err)
				assert.Equal(t, "consent", parsed.Query().Get("prompt"))
			},
		},
		{
			name: "dropbox env credentials",
			env: map[string]string{
				env.EnvDropboxClientID:     "env-id",
				env.EnvDropboxClientSecret: "env-secret",
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				dropbox, ok := plugin.providerRegistry.Get("dropbox")
				require.True(t, ok)
				_, ok = dropbox.(*services.DropboxOAuth2Provider)
				assert.True(t, ok)
				assert.Equal(t, "env-id", dropbox.GetConfig().ClientID)
				assert.Equal(t, "https://api.example.com/auth/oauth2/callback/dropbox", dropbox.GetConfig().RedirectURL)
				assert.Equal(t, []string{"openid", "profile", "email"}, dropbox.GetConfig().Scopes)
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
			name: "notion env credentials",
			env: map[string]string{
				env.EnvNotionClientID:     "env-id",
				env.EnvNotionClientSecret: "env-secret",
			},
			verify: func(t *testing.T, plugin *OAuth2Plugin) {
				t.Helper()

				notion, ok := plugin.providerRegistry.Get("notion")
				require.True(t, ok)
				_, ok = notion.(*services.NotionOAuth2Provider)
				assert.True(t, ok)
				assert.Equal(t, "env-id", notion.GetConfig().ClientID)
				assert.Equal(t, "env-secret", notion.GetConfig().ClientSecret)
				assert.Equal(t, "https://api.notion.com/v1/oauth/authorize", notion.GetConfig().Endpoint.AuthURL)
				assert.Equal(t, "https://api.notion.com/v1/oauth/token", notion.GetConfig().Endpoint.TokenURL)
				assert.Equal(t, "https://api.example.com/auth/oauth2/callback/notion", notion.GetConfig().RedirectURL)
				assert.False(t, notion.RequiresPKCE())
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

func providerScopesConfigTOML(name, extraConfig string) string {
	return fmt.Sprintf(`
[plugins.oauth2.providers.%s]
client_id = "client-id"
client_secret = "client-secret"
scopes = ["openid", "email"]
%s
`, name, extraConfig)
}

func TestProviderScopesFromTOMLConfig(t *testing.T) {
	clearProviderEnv(t)

	tests := []struct {
		name        string
		extraConfig string
	}{
		{name: "discord"},
		{name: "dropbox"},
		{name: "github"},
		{name: "google"},
		{name: "microsoft"},
		{
			name: "custom",
			extraConfig: `redirect_url = "https://app.example.com/callback"
auth_url = "https://custom.example.com/authorize"
token_url = "https://custom.example.com/token"
user_info_url = "https://custom.example.com/userinfo"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var globalConfig models.Config
			_, err := toml.Decode(providerScopesConfigTOML(tt.name, tt.extraConfig), &globalConfig)
			require.NoError(t, err)

			var pluginConfig types.OAuth2PluginConfig
			require.NoError(t, util.LoadPluginConfig(&globalConfig, "oauth2", &pluginConfig))

			providerConfig, ok := pluginConfig.Providers[tt.name]
			require.True(t, ok)
			require.Equal(t, []string{"openid", "email"}, providerConfig.Scopes)

			plugin := newRegistrationPlugin(pluginConfig.Providers)
			require.NoError(t, plugin.registerProviders())

			provider, ok := plugin.providerRegistry.Get(tt.name)
			require.True(t, ok)
			require.Equal(t, []string{"openid", "email"}, provider.GetConfig().Scopes)
		})
	}
}

func TestOAuth2PluginImplementsMigrationInterface(t *testing.T) {
	t.Parallel()

	var plugin models.Plugin = New(types.OAuth2PluginConfig{})
	migratable, ok := plugin.(models.PluginWithMigrations)
	require.True(t, ok)
	require.Len(t, migratable.Migrations("sqlite"), 1)
	require.Equal(t, []string{models.PluginJWT.String()}, migratable.DependsOn())
}

func TestOAuth2Plugin_DependsOn(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{models.PluginJWT.String()}, New(types.OAuth2PluginConfig{}).DependsOn())
}

func newOAuth2ServiceRegistry(withJWTService bool) *internalplugins.DefaultServiceRegistry {
	registry := internalplugins.NewServiceRegistry()
	registry.Register(models.ServiceUser.String(), new(internaltests.MockUserService))
	registry.Register(models.ServiceAccount.String(), new(internaltests.MockAccountService))
	registry.Register(models.ServiceSession.String(), new(internaltests.MockSessionService))
	registry.Register(models.ServiceToken.String(), new(internaltests.MockTokenService))
	if withJWTService {
		registry.Register(models.ServiceJWT.String(), new(internaltests.MockJWTService))
	}
	return registry
}

func TestOAuth2Plugin_initializeServicesRequiresJWTService(t *testing.T) {
	t.Parallel()

	plugin := New(types.OAuth2PluginConfig{})
	err := plugin.initializeServices(&models.PluginContext{ServiceRegistry: newOAuth2ServiceRegistry(false)})

	require.Error(t, err)
	assert.ErrorContains(t, err, "enable the jwt plugin")
}

func TestOAuth2Plugin_initializeServicesResolvesJWTService(t *testing.T) {
	t.Parallel()

	plugin := New(types.OAuth2PluginConfig{})
	err := plugin.initializeServices(&models.PluginContext{ServiceRegistry: newOAuth2ServiceRegistry(true)})

	require.NoError(t, err)
	require.NotNil(t, plugin.jwtService)
}
