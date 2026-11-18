package magiclink

import (
	"testing"
	"time"

	internaltests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	magiclinkconstants "github.com/Authula/authula/plugins/magic-link/constants"
	types "github.com/Authula/authula/plugins/magic-link/types"
	"github.com/Authula/authula/plugins/magic-link/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingLogger captures warning messages so tests can assert on them.
type recordingLogger struct {
	warnings []string
}

func (l *recordingLogger) Debug(string, ...any) {}
func (l *recordingLogger) Info(string, ...any)  {}
func (l *recordingLogger) Warn(msg string, _ ...any) {
	l.warnings = append(l.warnings, msg)
}
func (l *recordingLogger) Error(string, ...any) {}

type pluginDeps struct {
	user         *internaltests.MockUserService
	account      *internaltests.MockAccountService
	session      *internaltests.MockSessionService
	verification *internaltests.MockVerificationService
	token        *internaltests.MockTokenService
	mailer       *internaltests.MockMailerService
}

func newPluginDeps() *pluginDeps {
	return &pluginDeps{
		user:         &internaltests.MockUserService{},
		account:      &internaltests.MockAccountService{},
		session:      &internaltests.MockSessionService{},
		verification: &internaltests.MockVerificationService{},
		token:        &internaltests.MockTokenService{},
		mailer:       &internaltests.MockMailerService{},
	}
}

// serviceMap returns the registry contents for the required services plus an
// optional mailer. Tests delete or replace entries to exercise failure paths.
func (d *pluginDeps) serviceMap(includeMailer bool) map[string]any {
	services := map[string]any{
		models.ServiceUser.String():         d.user,
		models.ServiceAccount.String():      d.account,
		models.ServiceSession.String():      d.session,
		models.ServiceVerification.String(): d.verification,
		models.ServiceToken.String():        d.token,
	}
	if includeMailer {
		services[models.ServiceMailer.String()] = d.mailer
	}
	return services
}

func newPluginContext(cfg *models.Config, logger models.Logger, services map[string]any) *models.PluginContext {
	if logger == nil {
		logger = &internaltests.MockLogger{}
	}
	return &models.PluginContext{
		Logger:          logger,
		ServiceRegistry: &internaltests.TestServiceRegistry{Services: services},
		GetConfig:       func() *models.Config { return cfg },
	}
}

func newInitializedPlugin(t *testing.T) *MagicLinkPlugin {
	t.Helper()

	deps := newPluginDeps()
	p := New(types.MagicLinkPluginConfig{Enabled: true})
	require.NoError(t, p.Init(newPluginContext(&models.Config{}, nil, deps.serviceMap(true))))
	return p
}

func TestNew_AppliesDefaults(t *testing.T) {
	tests := []struct {
		name          string
		config        types.MagicLinkPluginConfig
		wantExpiresIn time.Duration
	}{
		{
			name:          "zero expiry falls back to the default",
			config:        types.MagicLinkPluginConfig{},
			wantExpiresIn: 15 * time.Minute,
		},
		{
			name:          "negative expiry falls back to the default",
			config:        types.MagicLinkPluginConfig{ExpiresIn: -time.Minute},
			wantExpiresIn: 15 * time.Minute,
		},
		{
			name:          "explicit expiry is preserved",
			config:        types.MagicLinkPluginConfig{ExpiresIn: 5 * time.Minute},
			wantExpiresIn: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New(tt.config)

			require.NotNil(t, p.pluginConfig)
			assert.Equal(t, tt.wantExpiresIn, p.pluginConfig.ExpiresIn)
			assert.Nil(t, p.Api, "API must not be built until Init runs")
		})
	}
}

func TestMetadata(t *testing.T) {
	md := New(types.MagicLinkPluginConfig{}).Metadata()

	assert.Equal(t, models.PluginMagicLink.String(), md.ID)
	assert.Equal(t, "1.0.0", md.Version)
	assert.NotEmpty(t, md.Description)
}

func TestConfig_ReturnsTheSameConfig(t *testing.T) {
	p := New(types.MagicLinkPluginConfig{Enabled: true, ExpiresIn: 10 * time.Minute})

	cfg, ok := p.Config().(*types.MagicLinkPluginConfig)
	require.True(t, ok)
	assert.Same(t, p.pluginConfig, cfg)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, 10*time.Minute, cfg.ExpiresIn)
}

func TestInit_MissingServices(t *testing.T) {
	tests := []struct {
		name     string
		service  models.ServiceID
		override any
		wantErr  string
	}{
		{
			name:    "user service missing",
			service: models.ServiceUser,
			wantErr: "user service not available in service registry",
		},
		{
			name:    "account service missing",
			service: models.ServiceAccount,
			wantErr: "account service not available in service registry",
		},
		{
			name:    "session service missing",
			service: models.ServiceSession,
			wantErr: "session service not available in service registry",
		},
		{
			name:    "verification service missing",
			service: models.ServiceVerification,
			wantErr: "verification service not available in service registry",
		},
		{
			name:    "token service missing",
			service: models.ServiceToken,
			wantErr: "token service not available in service registry",
		},
		{
			name:     "user service registered with the wrong type",
			service:  models.ServiceUser,
			override: "not a user service",
			wantErr:  "user service not available in service registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			services := newPluginDeps().serviceMap(true)
			if tt.override != nil {
				services[tt.service.String()] = tt.override
			} else {
				delete(services, tt.service.String())
			}

			p := New(types.MagicLinkPluginConfig{})
			err := p.Init(newPluginContext(&models.Config{}, nil, services))

			require.EqualError(t, err, tt.wantErr)
			assert.Nil(t, p.Api)
		})
	}
}

func TestInit_WiresServices(t *testing.T) {
	deps := newPluginDeps()
	p := New(types.MagicLinkPluginConfig{Enabled: true})

	require.NoError(t, p.Init(newPluginContext(&models.Config{}, nil, deps.serviceMap(true))))

	assert.Same(t, deps.user, p.userService)
	assert.Same(t, deps.account, p.accountService)
	assert.Same(t, deps.session, p.sessionService)
	assert.Same(t, deps.verification, p.verificationService)
	assert.Same(t, deps.token, p.tokenService)
	assert.Same(t, deps.mailer, p.mailerService)
	assert.NotNil(t, p.emailTemplateManager)
	require.NotNil(t, p.Api)
	assert.NotNil(t, p.Api.useCases)
	assert.NotNil(t, p.Api.useCases.SignInUseCase)
	assert.NotNil(t, p.Api.useCases.VerifyUseCase)
	assert.NotNil(t, p.Api.useCases.ExchangeUseCase)
}

func TestInit_MailerIsOptional(t *testing.T) {
	deps := newPluginDeps()
	logger := &recordingLogger{}
	p := New(types.MagicLinkPluginConfig{})

	require.NoError(t, p.Init(newPluginContext(&models.Config{}, logger, deps.serviceMap(false))))

	assert.Nil(t, p.mailerService)
	signInUseCase, ok := p.Api.useCases.SignInUseCase.(*usecases.SignInUseCaseImpl)
	require.True(t, ok)
	assert.Nil(t, signInUseCase.MailerService)
	assert.Contains(t, logger.warnings, "mailer service not available in service registry: automatic email sending will be disabled for the magic link plugin")
}

func TestInit_ConfigLoadFailureKeepsDefaults(t *testing.T) {
	deps := newPluginDeps()
	logger := &recordingLogger{}
	cfg := &models.Config{
		Plugins: models.PluginsConfig{
			models.PluginMagicLink.String(): "not-a-config-map",
		},
	}
	p := New(types.MagicLinkPluginConfig{Enabled: true})

	require.NoError(t, p.Init(newPluginContext(cfg, logger, deps.serviceMap(true))))

	assert.True(t, p.pluginConfig.Enabled)
	assert.Equal(t, 15*time.Minute, p.pluginConfig.ExpiresIn)
	assert.Contains(t, logger.warnings, "failed to load magic link plugin config, using defaults")
}

func TestInit_LoadsConfigFromPluginsMap(t *testing.T) {
	deps := newPluginDeps()
	cfg := &models.Config{
		Plugins: models.PluginsConfig{
			models.PluginMagicLink.String(): map[string]any{
				"enabled":    false,
				"expires_in": "30m",
			},
		},
	}
	p := New(types.MagicLinkPluginConfig{Enabled: true})

	require.NoError(t, p.Init(newPluginContext(cfg, nil, deps.serviceMap(true))))

	assert.False(t, p.pluginConfig.Enabled)
	assert.Equal(t, 30*time.Minute, p.pluginConfig.ExpiresIn)
}

func TestInit_PrefersPreParsedConfig(t *testing.T) {
	deps := newPluginDeps()
	cfg := &models.Config{
		Plugins: models.PluginsConfig{},
		PreParsedConfigs: map[string]any{
			models.PluginMagicLink.String(): types.MagicLinkPluginConfig{
				Enabled:   true,
				ExpiresIn: time.Hour,
			},
		},
	}
	p := New(types.MagicLinkPluginConfig{})

	require.NoError(t, p.Init(newPluginContext(cfg, nil, deps.serviceMap(true))))

	assert.True(t, p.pluginConfig.Enabled)
	assert.Equal(t, time.Hour, p.pluginConfig.ExpiresIn)
}

func TestClose(t *testing.T) {
	p := New(types.MagicLinkPluginConfig{})

	assert.NoError(t, p.Close())
}

func TestInit_TemplateManagerFailure(t *testing.T) {
	t.Setenv(magiclinkconstants.EnvMagicLinkSignInHTML, "{{.Broken")

	deps := newPluginDeps()
	p := New(types.MagicLinkPluginConfig{})

	err := p.Init(newPluginContext(&models.Config{}, nil, deps.serviceMap(true)))

	require.ErrorContains(t, err, "failed to initialize magic link email templates")
	assert.Nil(t, p.Api)
}
