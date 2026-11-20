package oauth2

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Authula/authula/env"
	internalplugins "github.com/Authula/authula/internal/plugins"
	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/oauth2/constants"
	"github.com/Authula/authula/plugins/oauth2/repositories"
	"github.com/Authula/authula/plugins/oauth2/services"
	oauth2tests "github.com/Authula/authula/plugins/oauth2/tests"
	"github.com/Authula/authula/plugins/oauth2/types"
	"github.com/Authula/authula/plugins/oauth2/usecases"
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

func newOAuth2ServiceRegistryWithout(missing models.ServiceID) *internalplugins.DefaultServiceRegistry {
	registry := internalplugins.NewServiceRegistry()
	servicesByName := map[models.ServiceID]any{
		models.ServiceUser:    new(internaltests.MockUserService),
		models.ServiceAccount: new(internaltests.MockAccountService),
		models.ServiceSession: new(internaltests.MockSessionService),
		models.ServiceToken:   new(internaltests.MockTokenService),
		models.ServiceJWT:     new(internaltests.MockJWTService),
	}
	for name, service := range servicesByName {
		if name == missing {
			continue
		}
		registry.Register(name.String(), service)
	}
	return registry
}

func newOAuth2PluginContext(registry models.ServiceRegistry, config *models.Config) *models.PluginContext {
	if config == nil {
		config = &models.Config{BaseURL: "https://api.example.com", BasePath: "/auth"}
	}
	return &models.PluginContext{
		Logger:          &internaltests.MockLogger{},
		ServiceRegistry: registry,
		GetConfig:       func() *models.Config { return config },
	}
}

func newInitializedOAuth2Plugin(t *testing.T) *OAuth2Plugin {
	t.Helper()

	clearProviderEnv(t)
	plugin := New(types.OAuth2PluginConfig{})
	require.NoError(t, plugin.Init(newOAuth2PluginContext(newOAuth2ServiceRegistry(true), nil)))
	t.Cleanup(func() { _ = plugin.Close() })
	return plugin
}

func TestOAuth2Plugin_MetadataAndConfig(t *testing.T) {
	plugin := New(types.OAuth2PluginConfig{HTTPTimeout: 3 * time.Second})

	metadata := plugin.Metadata()
	assert.Equal(t, models.PluginOAuth2.String(), metadata.ID)
	assert.Equal(t, "1.1.0", metadata.Version)
	assert.NotEmpty(t, metadata.Description)

	raw := plugin.Config()
	config, ok := raw.(types.OAuth2PluginConfig)
	require.True(t, ok)
	assert.Equal(t, 3*time.Second, config.HTTPTimeout)
	assert.NotNil(t, config.Providers)
}

func TestOAuth2Plugin_Routes(t *testing.T) {
	plugin := newInitializedOAuth2Plugin(t)

	routes := plugin.Routes()
	require.Len(t, routes, 2)
	assert.Equal(t, "GET", routes[0].Method)
	assert.Equal(t, "/oauth2/authorize/{provider}", routes[0].Path)
	assert.NotNil(t, routes[0].Handler)
	assert.Equal(t, "GET", routes[1].Method)
	assert.Equal(t, "/oauth2/callback/{provider}", routes[1].Path)
	assert.NotNil(t, routes[1].Handler)
}

func TestOAuth2Plugin_InitSuccess(t *testing.T) {
	plugin := newInitializedOAuth2Plugin(t)

	assert.NotNil(t, plugin.logger)
	assert.NotNil(t, plugin.ctx)
	assert.NotNil(t, plugin.globalConfig)
	assert.NotNil(t, plugin.httpClient)
	assert.Equal(t, plugin.pluginConfig.HTTPTimeout, plugin.httpClient.Timeout)
	assert.NotNil(t, plugin.providerRegistry)
	assert.NotNil(t, plugin.useCases)
	assert.NotNil(t, plugin.Api)
}

func TestOAuth2Plugin_InitFailsWhenCoreServiceMissing(t *testing.T) {
	tests := []struct {
		name    string
		missing models.ServiceID
		wantMsg string
	}{
		{name: "user service", missing: models.ServiceUser, wantMsg: "user service not available"},
		{name: "account service", missing: models.ServiceAccount, wantMsg: "account service not available"},
		{name: "session service", missing: models.ServiceSession, wantMsg: "session service not available"},
		{name: "token service", missing: models.ServiceToken, wantMsg: "token service not available"},
		{name: "jwt service", missing: models.ServiceJWT, wantMsg: "enable the jwt plugin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearProviderEnv(t)
			plugin := New(types.OAuth2PluginConfig{})

			err := plugin.Init(newOAuth2PluginContext(newOAuth2ServiceRegistryWithout(tt.missing), nil))

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantMsg)
		})
	}
}

func TestOAuth2Plugin_InitRejectsInvalidProviderConfig(t *testing.T) {
	clearProviderEnv(t)
	plugin := New(types.OAuth2PluginConfig{
		Providers: map[string]types.ProviderConfig{
			"custom": {ClientID: "id", ClientSecret: "secret", RedirectURL: "https://app.example.com/cb"},
		},
	})

	err := plugin.Init(newOAuth2PluginContext(newOAuth2ServiceRegistry(true), nil))

	require.ErrorIs(t, err, constants.ErrProviderConfigInvalid)
}

func TestOAuth2Plugin_InitializeRepositories(t *testing.T) {
	plugin := New(types.OAuth2PluginConfig{})
	plugin.db = internaltests.NewSQLiteIntegrationDB(t)

	plugin.initializeRepositories()

	assert.NotNil(t, plugin.oauthClientRepo)
	assert.NotNil(t, plugin.authRequestRepo)
	assert.NotNil(t, plugin.consentRequestRepo)
	assert.NotNil(t, plugin.consentRepo)
	assert.NotNil(t, plugin.accessGrantRepo)
	assert.NotNil(t, plugin.refreshTokenRepo)
	assert.NotNil(t, plugin.flowStateRepo)
}

func TestOAuth2Plugin_CloseIsIdempotent(t *testing.T) {
	clearProviderEnv(t)
	plugin := New(types.OAuth2PluginConfig{AutoCleanup: true, CleanupInterval: time.Hour})
	require.NoError(t, plugin.Init(newOAuth2PluginContext(newOAuth2ServiceRegistry(true), nil)))

	require.NoError(t, plugin.Close())
	require.NoError(t, plugin.Close())
	assert.Nil(t, plugin.stopCleanup)
	assert.Nil(t, plugin.done)
}

type cleanupAuthRequestRepo struct {
	repositories.AuthRequestRepository
	calls atomic.Int32
	err   error
}

func (r *cleanupAuthRequestRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	r.calls.Add(1)
	return 0, r.err
}

type cleanupConsentRequestRepo struct {
	repositories.ConsentRequestRepository
	calls atomic.Int32
	err   error
}

func (r *cleanupConsentRequestRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	r.calls.Add(1)
	return 0, r.err
}

type cleanupAccessGrantRepo struct {
	repositories.AccessGrantRepository
	calls atomic.Int32
	err   error
}

func (r *cleanupAccessGrantRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	r.calls.Add(1)
	return 0, r.err
}

type cleanupRefreshTokenRepo struct {
	repositories.RefreshTokenRepository
	calls atomic.Int32
	err   error
}

func (r *cleanupRefreshTokenRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	r.calls.Add(1)
	return 0, r.err
}

type cleanupFlowStateRepo struct {
	repositories.FlowStateRepository
	calls atomic.Int32
	err   error
}

func (r *cleanupFlowStateRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	r.calls.Add(1)
	return 0, r.err
}

func TestOAuth2Plugin_RunCleanupLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cleanupErr := errors.New("cleanup failed")
		logger := new(internaltests.LevelRecordingLogger)
		authRequests := &cleanupAuthRequestRepo{err: cleanupErr}
		consentRequests := &cleanupConsentRequestRepo{err: cleanupErr}
		accessGrants := &cleanupAccessGrantRepo{err: cleanupErr}
		refreshTokens := &cleanupRefreshTokenRepo{err: cleanupErr}
		flowStates := &cleanupFlowStateRepo{err: cleanupErr}
		plugin := &OAuth2Plugin{
			logger:             logger,
			authRequestRepo:    authRequests,
			consentRequestRepo: consentRequests,
			accessGrantRepo:    accessGrants,
			refreshTokenRepo:   refreshTokens,
			flowStateRepo:      flowStates,
			stopCleanup:        make(chan struct{}),
			done:               make(chan struct{}),
		}

		go plugin.runCleanupLoop(10 * time.Millisecond)
		synctest.Wait()
		assert.Zero(t, authRequests.calls.Load(), "cleanup must not run before the first interval")

		time.Sleep(15 * time.Millisecond)
		synctest.Wait()
		assert.Equal(t, int32(1), authRequests.calls.Load())
		assert.Equal(t, int32(1), consentRequests.calls.Load())
		assert.Equal(t, int32(1), accessGrants.calls.Load())
		assert.Equal(t, int32(1), refreshTokens.calls.Load())
		assert.Equal(t, int32(1), flowStates.calls.Load())

		assert.ElementsMatch(t, []string{
			"oauth2 auth request cleanup failed",
			"oauth2 consent request cleanup failed",
			"oauth2 access grant cleanup failed",
			"oauth2 refresh token cleanup failed",
			"oauth2 flow state cleanup failed",
		}, logger.Messages())

		close(plugin.stopCleanup)
		<-plugin.done
	})
}

func TestOAuth2Plugin_RegisterProviderEdgeCases(t *testing.T) {
	clearProviderEnv(t)

	tests := []struct {
		name      string
		providers map[string]types.ProviderConfig
		prepare   func(*OAuth2Plugin)
		call      func(*OAuth2Plugin) error
		wantErr   string
		verify    func(*testing.T, *OAuth2Plugin)
	}{
		{
			name: "duplicate canonical names",
			providers: map[string]types.ProviderConfig{
				"Google": {ClientID: "id", ClientSecret: "secret"},
				"google": {ClientID: "id", ClientSecret: "secret"},
			},
			call:    func(p *OAuth2Plugin) error { return p.registerProviders() },
			wantErr: "duplicate provider google",
		},
		{
			name: "unsupported built-in",
			call: func(p *OAuth2Plugin) error {
				return p.registerBuiltInProvider("not-a-built-in", types.ProviderConfig{})
			},
			wantErr: "unsupported built-in provider",
		},
		{
			name: "registry duplicate",
			providers: map[string]types.ProviderConfig{
				"google": {ClientID: "id", ClientSecret: "secret"},
			},
			prepare: func(p *OAuth2Plugin) {
				_ = p.providerRegistry.Register("google", &oauth2tests.MockProvider{ProviderName: "google"})
			},
			call:    func(p *OAuth2Plugin) error { return p.registerProviders() },
			wantErr: "already registered",
		},
		{
			name: "invalid endpoint url",
			providers: map[string]types.ProviderConfig{
				"custom": {
					ClientID:     "id",
					ClientSecret: "secret",
					RedirectURL:  "https://app.example.com/cb",
					AuthURL:      "not-a-url",
					TokenURL:     "https://custom.example.com/token",
					UserInfoURL:  "https://custom.example.com/user",
				},
			},
			call:    func(p *OAuth2Plugin) error { return p.registerProviders() },
			wantErr: "must be an absolute URL",
		},
		{
			name: "non-local http endpoint rejected",
			providers: map[string]types.ProviderConfig{
				"custom": {
					ClientID:     "id",
					ClientSecret: "secret",
					RedirectURL:  "https://app.example.com/cb",
					AuthURL:      "http://custom.example.com/authorize",
					TokenURL:     "https://custom.example.com/token",
					UserInfoURL:  "https://custom.example.com/user",
				},
			},
			call:    func(p *OAuth2Plugin) error { return p.registerProviders() },
			wantErr: "must use https except for localhost",
		},
		{
			name: "localhost http endpoint accepted",
			providers: map[string]types.ProviderConfig{
				"custom": {
					ClientID:     "id",
					ClientSecret: "secret",
					RedirectURL:  "https://app.example.com/cb",
					AuthURL:      "http://localhost:8080/authorize",
					TokenURL:     "http://127.0.0.1:8080/token",
					UserInfoURL:  "https://custom.example.com/user",
				},
			},
			call: func(p *OAuth2Plugin) error { return p.registerProviders() },
			verify: func(t *testing.T, p *OAuth2Plugin) {
				t.Helper()
				_, ok := p.providerRegistry.Get("custom")
				assert.True(t, ok)
			},
		},
		{
			name: "generic provider registry duplicate",
			providers: map[string]types.ProviderConfig{
				"custom": {
					ClientID:     "id",
					ClientSecret: "secret",
					RedirectURL:  "https://app.example.com/cb",
					AuthURL:      "https://custom.example.com/authorize",
					TokenURL:     "https://custom.example.com/token",
					UserInfoURL:  "https://custom.example.com/user",
				},
			},
			prepare: func(p *OAuth2Plugin) {
				_ = p.providerRegistry.Register("custom", &oauth2tests.MockProvider{ProviderName: "custom"})
			},
			call:    func(p *OAuth2Plugin) error { return p.registerProviders() },
			wantErr: "already registered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := newRegistrationPlugin(tt.providers)
			if tt.prepare != nil {
				tt.prepare(plugin)
			}

			err := tt.call(plugin)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.verify != nil {
				tt.verify(t, plugin)
			}
		})
	}
}

func TestValidateEndpointURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		endpoint string
		wantErr  string
	}{
		{name: "https", endpoint: "https://provider.example.com/token"},
		{name: "http localhost", endpoint: "http://localhost:8080/token"},
		{name: "http loopback", endpoint: "http://127.0.0.1/token"},
		{name: "relative", endpoint: "/token", wantErr: "must be an absolute URL"},
		{name: "unsupported scheme", endpoint: "ftp://provider.example.com/token", wantErr: "must use http or https"},
		{name: "remote http", endpoint: "http://provider.example.com/token", wantErr: "must use https except for localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateEndpointURL("token_url", tt.endpoint)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

type stubAuthorizeUseCase struct {
	result *usecases.AuthorizeResult
	err    error
}

func (s *stubAuthorizeUseCase) Authorize(context.Context, *types.AuthorizeRequest) (*usecases.AuthorizeResult, error) {
	return s.result, s.err
}

type stubCallbackUseCase struct {
	result *types.CallbackResult
	err    error
}

func (s *stubCallbackUseCase) Callback(context.Context, *types.CallbackRequest, *string, *string) (*types.CallbackResult, error) {
	return s.result, s.err
}

func TestOAuth2API_DelegatesToUseCases(t *testing.T) {
	t.Parallel()

	authorizeResult := &usecases.AuthorizeResult{AuthorizationURL: "https://provider.example.com/authorize"}
	callbackResult := &types.CallbackResult{RedirectTo: "/welcome"}
	plugin := &OAuth2Plugin{useCases: &usecases.UseCases{
		AuthorizeUseCase: &stubAuthorizeUseCase{result: authorizeResult},
		CallbackUseCase:  &stubCallbackUseCase{result: callbackResult},
	}}

	api := BuildAPI(plugin)
	require.Same(t, plugin.useCases, api.UseCases)

	authorized, err := api.Authorize(context.Background(), &types.AuthorizeRequest{ProviderID: "google"})
	require.NoError(t, err)
	assert.Same(t, authorizeResult, authorized)

	callback, err := api.Callback(context.Background(), &types.CallbackRequest{ProviderID: "google"}, nil, nil)
	require.NoError(t, err)
	assert.Same(t, callbackResult, callback)
}
