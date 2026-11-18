package email_password

import (
	"slices"
	"testing"

	"github.com/Authula/authula/internal/plugins"
	inttests "github.com/Authula/authula/internal/tests"
	"github.com/Authula/authula/models"
	"github.com/Authula/authula/plugins/email-password/constants"
	"github.com/Authula/authula/plugins/email-password/types"
	"github.com/stretchr/testify/require"
)

type pluginInitTestCase struct {
	name                 string
	includeUserService   bool
	includeMailerService bool
	wantErr              string
	wantMailerService    bool
	wantAPI              bool
}

func TestEmailPasswordPlugin_Init(t *testing.T) {
	t.Parallel()

	tests := []pluginInitTestCase{
		{
			name:                 "allows missing mailer service",
			includeUserService:   true,
			includeMailerService: false,
			wantMailerService:    false,
			wantAPI:              true,
		},
		{
			name:                 "fails when user service is missing",
			includeUserService:   false,
			includeMailerService: false,
			wantErr:              "user service not available in service registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(types.EmailPasswordPluginConfig{})
			serviceRegistry := plugins.NewServiceRegistry()

			if tt.includeUserService {
				serviceRegistry.Register(models.ServiceUser.String(), &inttests.MockUserService{})
			}
			serviceRegistry.Register(models.ServiceAccount.String(), &inttests.MockAccountService{})
			serviceRegistry.Register(models.ServiceSession.String(), &inttests.MockSessionService{})
			serviceRegistry.Register(models.ServiceVerification.String(), &inttests.MockVerificationService{})
			serviceRegistry.Register(models.ServiceToken.String(), &inttests.MockTokenService{})
			serviceRegistry.Register(models.ServicePassword.String(), &inttests.MockPasswordService{})
			if tt.includeMailerService {
				serviceRegistry.Register(models.ServiceMailer.String(), &inttests.MockMailerService{})
			}

			ctx := &models.PluginContext{
				Logger:          &inttests.MockLogger{},
				ServiceRegistry: serviceRegistry,
				GetConfig: func() *models.Config {
					return &models.Config{BaseURL: "http://localhost", BasePath: "/auth"}
				},
			}

			err := plugin.Init(ctx)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantMailerService, plugin.mailerService != nil)
			require.Equal(t, tt.wantAPI, plugin.Api != nil)
			if tt.includeMailerService {
				require.NotNil(t, plugin.mailerService)
			}
			if !tt.includeMailerService {
				require.Nil(t, plugin.mailerService)
			}
		})
	}
}

func TestEmailPasswordPlugin_InitServiceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		serviceName string
		service     any
		wantErr     string
	}{
		{
			name:        "fails when account service is missing",
			serviceName: models.ServiceAccount.String(),
			wantErr:     "account service not available in service registry",
		},
		{
			name:        "fails when session service is missing",
			serviceName: models.ServiceSession.String(),
			wantErr:     "session service not available in service registry",
		},
		{
			name:        "fails when verification service is missing",
			serviceName: models.ServiceVerification.String(),
			wantErr:     "verification service not available in service registry",
		},
		{
			name:        "fails when token service is missing",
			serviceName: models.ServiceToken.String(),
			wantErr:     "token service not available in service registry",
		},
		{
			name:        "fails when password service is missing",
			serviceName: models.ServicePassword.String(),
			wantErr:     "password service not available in service registry",
		},
		{
			name:        "fails when user service has the wrong type",
			serviceName: models.ServiceUser.String(),
			service:     "not a user service",
			wantErr:     "user service not available in service registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			services := newInitTestServices()
			if tt.service == nil {
				delete(services, tt.serviceName)
			} else {
				services[tt.serviceName] = tt.service
			}

			plugin := New(types.EmailPasswordPluginConfig{})
			err := plugin.Init(newInitTestPluginContext(services, &inttests.MockLogger{}))

			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestEmailPasswordPlugin_InitMailerWarning(t *testing.T) {
	t.Parallel()

	const mailerWarning = "mailer service not available in service registry: automatic email sending will be disabled for the email/password plugin"

	tests := []struct {
		name                 string
		includeMailerService bool
		wantWarning          bool
	}{
		{
			name:                 "warns when mailer service is missing",
			includeMailerService: false,
			wantWarning:          true,
		},
		{
			name:                 "does not warn when mailer service is registered",
			includeMailerService: true,
			wantWarning:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			services := newInitTestServices()
			if tt.includeMailerService {
				services[models.ServiceMailer.String()] = &inttests.MockMailerService{}
			}
			logger := &inttests.LevelRecordingLogger{}

			plugin := New(types.EmailPasswordPluginConfig{})
			require.NoError(t, plugin.Init(newInitTestPluginContext(services, logger)))

			require.Equal(t, tt.wantWarning, slices.Contains(logger.Messages(), mailerWarning))
		})
	}
}

func TestEmailPasswordPlugin_InitTemplateOverrideError(t *testing.T) {
	t.Setenv(constants.EnvVerifyEmailSubject, "{{")

	plugin := New(types.EmailPasswordPluginConfig{})
	err := plugin.Init(newInitTestPluginContext(newInitTestServices(), &inttests.MockLogger{}))

	require.ErrorContains(t, err, `failed to initialize email templates: failed to parse subject template "verify_email"`)
}

func TestEmailPasswordPlugin_InitInvalidPluginConfig(t *testing.T) {
	t.Parallel()

	ctx := newInitTestPluginContext(newInitTestServices(), &inttests.MockLogger{})
	ctx.GetConfig = func() *models.Config {
		return &models.Config{
			Plugins: models.PluginsConfig{models.PluginEmailPassword.String(): "not a plugin config"},
		}
	}

	plugin := New(types.EmailPasswordPluginConfig{})

	require.Error(t, plugin.Init(ctx))
}

func TestEmailPasswordPlugin_Config(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		config                types.EmailPasswordPluginConfig
		wantMinPasswordLength int
	}{
		{
			name:                  "returns the configured minimum password length",
			config:                types.EmailPasswordPluginConfig{MinPasswordLength: 12},
			wantMinPasswordLength: 12,
		},
		{
			name:                  "returns the default minimum password length when unset",
			config:                types.EmailPasswordPluginConfig{},
			wantMinPasswordLength: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plugin := New(tt.config)

			config, ok := plugin.Config().(types.EmailPasswordPluginConfig)
			require.True(t, ok)
			require.Equal(t, tt.wantMinPasswordLength, config.MinPasswordLength)
		})
	}
}

func TestEmailPasswordPlugin_Close(t *testing.T) {
	t.Parallel()

	plugin := New(types.EmailPasswordPluginConfig{})

	require.NoError(t, plugin.Close())
}

func newInitTestServices() map[string]any {
	return map[string]any{
		models.ServiceUser.String():         &inttests.MockUserService{},
		models.ServiceAccount.String():      &inttests.MockAccountService{},
		models.ServiceSession.String():      &inttests.MockSessionService{},
		models.ServiceVerification.String(): &inttests.MockVerificationService{},
		models.ServiceToken.String():        &inttests.MockTokenService{},
		models.ServicePassword.String():     &inttests.MockPasswordService{},
	}
}

func newInitTestPluginContext(services map[string]any, logger models.Logger) *models.PluginContext {
	serviceRegistry := plugins.NewServiceRegistry()
	for name, service := range services {
		serviceRegistry.Register(name, service)
	}

	return &models.PluginContext{
		Logger:          logger,
		EventBus:        &inttests.MockEventBus{},
		ServiceRegistry: serviceRegistry,
		GetConfig: func() *models.Config {
			return &models.Config{BaseURL: "http://localhost", BasePath: "/auth"}
		},
	}
}

func newInitializedTestPlugin(t *testing.T) *EmailPasswordPlugin {
	t.Helper()

	plugin := New(types.EmailPasswordPluginConfig{})
	require.NoError(t, plugin.Init(newInitTestPluginContext(newInitTestServices(), &inttests.MockLogger{})))

	return plugin
}
